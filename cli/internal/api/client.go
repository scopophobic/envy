package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/envo/cli/internal/store"
)

type Client struct {
	baseURL string
	http    *http.Client
	tokens  *store.Tokens
}

func NewClient(baseURL string, tokens *store.Tokens) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		http: &http.Client{
			Timeout: 30 * time.Second,
		},
		tokens: tokens,
	}
}

func (c *Client) SetTokens(t *store.Tokens) {
	c.tokens = t
}

func (c *Client) Tokens() *store.Tokens {
	return c.tokens
}

func (c *Client) buildURL(path string, q url.Values) string {
	u := c.baseURL + path
	if q == nil {
		return u
	}
	if enc := q.Encode(); enc != "" {
		return u + "?" + enc
	}
	return u
}

func (c *Client) do(ctx context.Context, method, path string, body any, out any, auth bool) (*http.Response, error) {
	var r io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		r = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.buildURL(path, nil), r)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if auth {
		if c.tokens == nil || c.tokens.AccessToken == "" {
			return nil, fmt.Errorf("not logged in (missing access token)")
		}
		req.Header.Set("Authorization", "Bearer "+c.tokens.AccessToken)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}

	defer func() {
		// only drain if we are not decoding
		if out == nil {
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}
	}()

	if out == nil {
		return resp, nil
	}

	b, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return resp, fmt.Errorf("api error %s: %s", resp.Status, strings.TrimSpace(string(b)))
	}

	if err := json.Unmarshal(b, out); err != nil {
		return resp, fmt.Errorf("failed to parse response: %w", err)
	}
	return resp, nil
}

// -------- Auth --------

type googleLoginResp struct {
	URL string `json:"url"`
}

func (c *Client) GoogleLoginURL(ctx context.Context) (string, error) {
	var out googleLoginResp
	_, err := c.do(ctx, http.MethodGet, "/api/v1/auth/google/login", nil, &out, false)
	if err != nil {
		return "", err
	}
	if out.URL == "" {
		return "", fmt.Errorf("backend did not return login url")
	}
	return out.URL, nil
}

type refreshReq struct {
	RefreshToken string `json:"refresh_token"`
}

type refreshResp struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

func (c *Client) Refresh(ctx context.Context) (*store.Tokens, error) {
	if c.tokens == nil || c.tokens.RefreshToken == "" {
		return nil, fmt.Errorf("missing refresh token; run `envo login`")
	}

	var out refreshResp
	_, err := c.do(ctx, http.MethodPost, "/api/v1/auth/refresh", refreshReq{RefreshToken: c.tokens.RefreshToken}, &out, false)
	if err != nil {
		return nil, err
	}

	t := *c.tokens
	t.AccessToken = out.AccessToken
	t.TokenType = out.TokenType
	t.ExpiresAt = time.Now().Add(time.Duration(out.ExpiresIn) * time.Second)
	return &t, nil
}

func (c *Client) EnsureAccessToken(ctx context.Context) (*store.Tokens, error) {
	if c.tokens == nil {
		return nil, fmt.Errorf("not logged in; run `envo login`")
	}
	// Refresh a bit early
	if c.tokens.AccessToken == "" || time.Until(c.tokens.ExpiresAt) < 30*time.Second {
		t, err := c.Refresh(ctx)
		if err != nil {
			return nil, err
		}
		c.tokens = t
	}
	return c.tokens, nil
}

// -------- Domain models (minimal) --------

type Org struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Project struct {
	ID    string `json:"id"`
	OrgID string `json:"org_id"`
	Name  string `json:"name"`
}

type Environment struct {
	ID        string `json:"id"`
	ProjectID string `json:"project_id"`
	Name      string `json:"name"`
}

// -------- List endpoints --------

func (c *Client) ListOrgs(ctx context.Context) ([]Org, error) {
	if _, err := c.EnsureAccessToken(ctx); err != nil {
		return nil, err
	}
	var out []Org
	_, err := c.do(ctx, http.MethodGet, "/api/v1/orgs", nil, &out, true)
	return out, err
}

func (c *Client) ListOrgProjects(ctx context.Context, orgID string) ([]Project, error) {
	if _, err := c.EnsureAccessToken(ctx); err != nil {
		return nil, err
	}
	var out []Project
	_, err := c.do(ctx, http.MethodGet, "/api/v1/orgs/"+orgID+"/projects", nil, &out, true)
	return out, err
}

func (c *Client) ListProjectEnvironments(ctx context.Context, projectID string) ([]Environment, error) {
	if _, err := c.EnsureAccessToken(ctx); err != nil {
		return nil, err
	}
	var out []Environment
	_, err := c.do(ctx, http.MethodGet, "/api/v1/projects/"+projectID+"/environments", nil, &out, true)
	return out, err
}

// -------- Secrets export --------

type exportResp struct {
	OrgID         string            `json:"org_id"`
	EnvironmentID string            `json:"environment_id"`
	Secrets       map[string]string `json:"secrets"`
}

func (c *Client) ExportEnvironmentSecrets(ctx context.Context, envID string) (map[string]string, error) {
	if _, err := c.EnsureAccessToken(ctx); err != nil {
		return nil, err
	}
	var out exportResp
	_, err := c.do(ctx, http.MethodGet, "/api/v1/environments/"+envID+"/secrets/export", nil, &out, true)
	if err != nil {
		return nil, err
	}
	if out.Secrets == nil {
		out.Secrets = map[string]string{}
	}
	return out.Secrets, nil
}

