package services

import "testing"

func TestInvitationTokenHashDeterministic(t *testing.T) {
	token := "abc123"
	h1 := invitationTokenHash(token)
	h2 := invitationTokenHash(token)
	if h1 != h2 {
		t.Fatalf("expected stable hash, got %q and %q", h1, h2)
	}
	if h1 == "" {
		t.Fatalf("expected non-empty hash")
	}
}

func TestGenerateInviteToken(t *testing.T) {
	raw, hashed, err := generateInviteToken()
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) < 32 {
		t.Fatalf("unexpected raw token length: %d", len(raw))
	}
	if len(hashed) != 64 {
		t.Fatalf("unexpected hash length: %d", len(hashed))
	}
	if invitationTokenHash(raw) != hashed {
		t.Fatalf("hash mismatch")
	}
}

