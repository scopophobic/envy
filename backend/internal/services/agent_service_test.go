package services

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestGenerateAgentTokenIsOpaqueAndParseable(t *testing.T) {
	id, raw, hash, display, err := GenerateAgentToken()
	if err != nil {
		t.Fatalf("GenerateAgentToken() error = %v", err)
	}
	if !strings.HasPrefix(raw, agentTokenPrefix) {
		t.Fatalf("token %q does not have agent prefix", raw)
	}
	parsed, err := credentialIDFromToken(raw)
	if err != nil || parsed != id {
		t.Fatalf("credentialIDFromToken() = %s, %v; want %s", parsed, err, id)
	}
	wantHash := sha256.Sum256([]byte(raw))
	if hash != hex.EncodeToString(wantHash[:]) {
		t.Fatal("stored token hash does not match SHA-256 of raw token")
	}
	if strings.Contains(hash, raw) || display == raw || len(display) > 24 {
		t.Fatal("safe credential fields expose too much of the raw token")
	}
}

func TestGenerateAgentTokenProducesIndependentCredentials(t *testing.T) {
	idA, tokenA, hashA, _, err := GenerateAgentToken()
	if err != nil {
		t.Fatal(err)
	}
	idB, tokenB, hashB, _, err := GenerateAgentToken()
	if err != nil {
		t.Fatal(err)
	}
	if idA == idB || tokenA == tokenB || hashA == hashB {
		t.Fatal("two generated agent credentials were not independent")
	}
}

func TestCredentialIDFromTokenRejectsMalformedValues(t *testing.T) {
	values := []string{"", "Bearer token", "envo_agent_bad_secret", "envo_agent_" + uuid.NewString() + "_short"}
	for _, value := range values {
		if _, err := credentialIDFromToken(value); err == nil {
			t.Fatalf("credentialIDFromToken(%q) unexpectedly succeeded", value)
		}
	}
}

func TestNormalizeKeysTrimsDeduplicatesAndSorts(t *testing.T) {
	got := normalizeKeys([]string{" Z_KEY ", "A_KEY", "", "A_KEY"})
	want := []string{"A_KEY", "Z_KEY"}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("normalizeKeys() = %v, want %v", got, want)
	}
}
