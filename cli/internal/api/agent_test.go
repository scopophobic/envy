package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestResolveAgentSecretsUsesAgentBearerAndRequestBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/agent/secrets/resolve" || r.Method != http.MethodPost {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer envo_agent_test" {
			t.Fatalf("Authorization = %q", got)
		}
		var request ResolveAgentSecretsRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		if request.Project != "api" || request.Environment != "production" || len(request.Keys) != 1 || request.Keys[0] != "DATABASE_URL" {
			t.Fatalf("request body = %+v", request)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"agent_id":"a","environment_id":"e","lease_id":"l","expires_at":"2026-08-04T00:00:00Z","secrets":{"DATABASE_URL":"secret"}}`))
	}))
	defer server.Close()

	client := NewAgentClient(server.URL, "envo_agent_test")
	result, err := client.ResolveAgentSecrets(context.Background(), ResolveAgentSecretsRequest{
		Project: "api", Environment: "production", Keys: []string{"DATABASE_URL"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Secrets["DATABASE_URL"] != "secret" {
		t.Fatalf("secrets = %v", result.Secrets)
	}
}
