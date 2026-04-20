package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/envo/backend/internal/database"
	"github.com/envo/backend/internal/models"
	"github.com/google/uuid"
	"gorm.io/datatypes"
)

const platformVercel = "vercel"

type CreatePlatformConnectionInput struct {
	Platform string
	Name     string
	Token    string
	Metadata map[string]any
}

type SyncEnvironmentInput struct {
	EnvironmentID uuid.UUID
	ConnectionID  uuid.UUID
	TargetProject string
	TargetEnv     string
}

type SyncResult struct {
	Platform       string `json:"platform"`
	ConnectionName string `json:"connection_name"`
	TargetProject  string `json:"target_project_id"`
	TargetEnv      string `json:"target_environment"`
	Synced         int    `json:"synced"`
}

// PlatformService manages deploy platform connections and manual env sync.
type PlatformService struct {
	encryptor      Encryptor
	localEncryptor Encryptor
	secretService  *SecretService
	httpClient     *http.Client
}

func NewPlatformService(encryptor Encryptor, localEncryptor Encryptor, secretService *SecretService) *PlatformService {
	return &PlatformService{
		encryptor:      encryptor,
		localEncryptor: localEncryptor,
		secretService:  secretService,
		httpClient: &http.Client{
			Timeout: 25 * time.Second,
		},
	}
}

func normalizePlatform(p string) string {
	return strings.ToLower(strings.TrimSpace(p))
}

func tokenPrefix(token string) string {
	t := strings.TrimSpace(token)
	if len(t) <= 6 {
		return t
	}
	return t[:6]
}

func (s *PlatformService) CreateConnection(ctx context.Context, userID uuid.UUID, in CreatePlatformConnectionInput) (*models.PlatformConnection, error) {
	if s.encryptor == nil {
		return nil, fmt.Errorf("secret encryption is not configured")
	}

	platform := normalizePlatform(in.Platform)
	if platform != platformVercel {
		return nil, fmt.Errorf("unsupported platform %q; currently supported: vercel", platform)
	}

	token := strings.TrimSpace(in.Token)
	if token == "" {
		return nil, fmt.Errorf("platform token is required")
	}

	name := strings.TrimSpace(in.Name)
	if name == "" {
		name = platform
	}

	if err := s.validateConnection(ctx, platform, token); err != nil {
		return nil, err
	}

	ciphertext, err := s.encryptor.Encrypt(ctx, token, userID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt platform token: %w", err)
	}

	metaRaw, err := json.Marshal(in.Metadata)
	if err != nil {
		return nil, fmt.Errorf("invalid metadata: %w", err)
	}

	conn := &models.PlatformConnection{
		UserID:         userID,
		Platform:       platform,
		Name:           name,
		EncryptedToken: ciphertext,
		KeyID:          s.encryptor.KeyID(),
		TokenPrefix:    tokenPrefix(token),
		Metadata:       datatypes.JSON(metaRaw),
	}
	if string(conn.Metadata) == "null" {
		conn.Metadata = datatypes.JSON([]byte("{}"))
	}

	if err := database.GetDB().WithContext(ctx).Create(conn).Error; err != nil {
		return nil, fmt.Errorf("failed to save platform connection: %w", err)
	}
	return conn, nil
}

func (s *PlatformService) ListConnections(ctx context.Context, userID uuid.UUID) ([]models.PlatformConnection, error) {
	var rows []models.PlatformConnection
	err := database.GetDB().WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}

func (s *PlatformService) DeleteConnection(ctx context.Context, userID, connectionID uuid.UUID) error {
	db := database.GetDB().WithContext(ctx)
	res := db.Where("id = ? AND user_id = ?", connectionID, userID).Delete(&models.PlatformConnection{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return fmt.Errorf("platform connection not found")
	}
	return nil
}

func (s *PlatformService) SyncEnvironment(ctx context.Context, userID uuid.UUID, in SyncEnvironmentInput, ip string) (*SyncResult, error) {
	db := database.GetDB().WithContext(ctx)

	var conn models.PlatformConnection
	if err := db.Where("id = ? AND user_id = ?", in.ConnectionID, userID).First(&conn).Error; err != nil {
		return nil, fmt.Errorf("platform connection not found")
	}

	token, err := s.decryptToken(ctx, &conn, userID.String())
	if err != nil {
		return nil, err
	}

	secrets, _, err := s.secretService.ExportEnvironmentSecrets(ctx, userID, in.EnvironmentID, ip)
	if err != nil {
		return nil, err
	}

	targetProject := strings.TrimSpace(in.TargetProject)
	targetEnv := strings.TrimSpace(in.TargetEnv)
	if targetProject == "" || targetEnv == "" {
		return nil, fmt.Errorf("target_project_id and target_environment are required")
	}

	switch conn.Platform {
	case platformVercel:
		if err := s.syncVercel(ctx, token, targetProject, targetEnv, secrets); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported platform %q; currently supported: vercel", conn.Platform)
	}

	return &SyncResult{
		Platform:       conn.Platform,
		ConnectionName: conn.Name,
		TargetProject:  targetProject,
		TargetEnv:      targetEnv,
		Synced:         len(secrets),
	}, nil
}

func (s *PlatformService) decryptToken(ctx context.Context, conn *models.PlatformConnection, scope string) (string, error) {
	dec := s.encryptor
	alt := s.localEncryptor
	if s.localEncryptor != nil && (conn.KeyID == "local" || strings.HasPrefix(conn.EncryptedToken, "local:")) {
		dec = s.localEncryptor
		alt = s.encryptor
	}

	token, err := dec.Decrypt(ctx, conn.EncryptedToken, scope)
	if err == nil {
		return token, nil
	}
	if alt != nil && alt != dec {
		token2, err2 := alt.Decrypt(ctx, conn.EncryptedToken, scope)
		if err2 == nil {
			return token2, nil
		}
	}
	return "", fmt.Errorf("failed to decrypt platform token")
}

func (s *PlatformService) validateConnection(ctx context.Context, platform, token string) error {
	switch platform {
	case platformVercel:
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.vercel.com/v2/user", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		resp, err := s.httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("failed to verify vercel token: %w", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("vercel token validation failed: %s", strings.TrimSpace(string(body)))
		}
		return nil
	default:
		return fmt.Errorf("unsupported platform %q; currently supported: vercel", platform)
	}
}

func (s *PlatformService) syncVercel(ctx context.Context, token, projectID, env string, secrets map[string]string) error {
	target := normalizeVercelEnv(env)
	for key, value := range secrets {
		payload := map[string]any{
			"key":    key,
			"value":  value,
			"type":   "encrypted",
			"target": []string{target},
		}
		b, _ := json.Marshal(payload)

		u := fmt.Sprintf("https://api.vercel.com/v10/projects/%s/env", projectID)
		req, _ := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(b))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")

		resp, err := s.httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("failed syncing %s: %w", key, err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			// Conflict means key exists. Update instead.
			if resp.StatusCode == http.StatusConflict {
				if err := s.updateVercelEnv(ctx, token, projectID, target, key, value); err == nil {
					continue
				}
			}
			return fmt.Errorf("vercel sync failed for key %s: %s", key, strings.TrimSpace(string(body)))
		}
	}
	return nil
}

func normalizeVercelEnv(env string) string {
	switch strings.ToLower(strings.TrimSpace(env)) {
	case "prod", "production":
		return "production"
	case "preview", "staging":
		return "preview"
	default:
		return "development"
	}
}

func (s *PlatformService) updateVercelEnv(ctx context.Context, token, projectID, target, key, value string) error {
	listURL := fmt.Sprintf("https://api.vercel.com/v10/projects/%s/env?decrypt=false", projectID)
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, listURL, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("failed listing vercel env vars")
	}

	var out struct {
		Envs []struct {
			ID     string   `json:"id"`
			Key    string   `json:"key"`
			Target []string `json:"target"`
		} `json:"envs"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return err
	}

	var envID string
	for _, e := range out.Envs {
		if e.Key != key {
			continue
		}
		for _, t := range e.Target {
			if t == target {
				envID = e.ID
				break
			}
		}
		if envID != "" {
			break
		}
	}
	if envID == "" {
		return fmt.Errorf("existing key not found")
	}

	payload := map[string]any{
		"key":   key,
		"value": value,
		"type":  "encrypted",
	}
	b, _ := json.Marshal(payload)
	updateURL := fmt.Sprintf("https://api.vercel.com/v10/projects/%s/env/%s", projectID, envID)
	upReq, _ := http.NewRequestWithContext(ctx, http.MethodPatch, updateURL, bytes.NewReader(b))
	upReq.Header.Set("Authorization", "Bearer "+token)
	upReq.Header.Set("Content-Type", "application/json")
	upResp, err := s.httpClient.Do(upReq)
	if err != nil {
		return err
	}
	defer upResp.Body.Close()
	if upResp.StatusCode < 200 || upResp.StatusCode >= 300 {
		body, _ := io.ReadAll(upResp.Body)
		return fmt.Errorf("vercel update failed: %s", strings.TrimSpace(string(body)))
	}
	return nil
}
