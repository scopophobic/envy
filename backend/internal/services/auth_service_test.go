package services

import (
	"net/url"
	"testing"

	"golang.org/x/oauth2"
)

func TestRefreshTokenHash(t *testing.T) {
	raw := "header.payload.signature"
	got := refreshTokenHash(raw)

	if got == raw {
		t.Fatal("refresh token hash must not equal the raw token")
	}
	if len(got) != 64 {
		t.Fatalf("refresh token hash length = %d, want 64", len(got))
	}
	if got != refreshTokenHash(raw) {
		t.Fatal("refresh token hash must be deterministic")
	}
	if got == refreshTokenHash(raw+"-different") {
		t.Fatal("different refresh tokens must produce different hashes")
	}
}

func TestGetAuthURLCanRequireAccountSelection(t *testing.T) {
	service := &AuthService{oauth2Config: &oauth2.Config{
		ClientID:    "client-id",
		RedirectURL: "https://app.example.com/callback",
		Endpoint: oauth2.Endpoint{
			AuthURL: "https://accounts.example.com/auth",
		},
	}}

	raw := service.GetAuthURL("state", true)
	parsed, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("parse auth URL: %v", err)
	}
	if got := parsed.Query().Get("prompt"); got != "select_account" {
		t.Fatalf("prompt = %q, want select_account", got)
	}
}
