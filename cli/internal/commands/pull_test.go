package commands

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/envo/cli/internal/api"
	"github.com/envo/cli/internal/store"
)

func orgClient(t *testing.T, response string) *api.Client {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/orgs" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, response)
	}))
	t.Cleanup(server.Close)

	return api.NewClient(server.URL, &store.Tokens{
		AccessToken: "test-token",
		ExpiresAt:   time.Now().Add(time.Hour),
	})
}

func TestResolveOrgIDDefaultsToPersonalWorkspace(t *testing.T) {
	client := orgClient(t, `[
		{"id":"personal-id","name":"Asha's workspace","owner_type":"personal"},
		{"id":"team-id","name":"Acme","owner_type":"org"}
	]`)

	id, err := resolveOrgID(context.Background(), client, "")
	if err != nil {
		t.Fatalf("resolveOrgID returned an error: %v", err)
	}
	if id != "personal-id" {
		t.Fatalf("resolveOrgID = %q, want personal-id", id)
	}
}

func TestResolveOrgIDStillSelectsExplicitTeam(t *testing.T) {
	client := orgClient(t, `[
		{"id":"personal-id","name":"Asha's workspace","owner_type":"personal"},
		{"id":"team-id","name":"Acme","owner_type":"org"}
	]`)

	id, err := resolveOrgID(context.Background(), client, "acme")
	if err != nil {
		t.Fatalf("resolveOrgID returned an error: %v", err)
	}
	if id != "team-id" {
		t.Fatalf("resolveOrgID = %q, want team-id", id)
	}
}

func TestResolveOrgIDExplainsMissingPersonalWorkspace(t *testing.T) {
	client := orgClient(t, `[{"id":"team-id","name":"Acme","owner_type":"org"}]`)

	_, err := resolveOrgID(context.Background(), client, "")
	if err == nil || !strings.Contains(err.Error(), "no personal workspace found") {
		t.Fatalf("resolveOrgID error = %v, want missing personal workspace error", err)
	}
}
