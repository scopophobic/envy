package utils

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestJWTTokenTypesAreNotInterchangeable(t *testing.T) {
	manager, err := NewJWTManager("test-secret-that-is-long-enough", "15m", "24h")
	if err != nil {
		t.Fatal(err)
	}
	userID := uuid.New()
	access, err := manager.GenerateAccessToken(userID, "owner@example.com", []string{"secrets.read"})
	if err != nil {
		t.Fatal(err)
	}
	refresh, _, err := manager.GenerateRefreshToken(userID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manager.ValidateAccessToken(access); err != nil {
		t.Fatalf("access token rejected: %v", err)
	}
	if _, err := manager.ValidateRefreshToken(refresh); err != nil {
		t.Fatalf("refresh token rejected: %v", err)
	}
	if _, err := manager.ValidateAccessToken(refresh); err == nil {
		t.Fatal("refresh token accepted as access token")
	}
	if _, err := manager.ValidateRefreshToken(access); err == nil {
		t.Fatal("access token accepted as refresh token")
	}
}

func TestJWTRejectsExpiredAccessToken(t *testing.T) {
	manager := &JWTManager{secretKey: "test-secret", accessTokenDuration: -time.Second, refreshTokenDuration: time.Hour}
	token, err := manager.GenerateAccessToken(uuid.New(), "owner@example.com", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manager.ValidateAccessToken(token); err == nil {
		t.Fatal("expired access token was accepted")
	}
}
