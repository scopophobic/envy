package handlers

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestAllowedFrontendRedirectUsesExactOrigin(t *testing.T) {
	handler := NewAuthHandler(nil, nil, "https://app.example.com", true)

	allowed := []string{
		"https://app.example.com/auth/callback",
		"https://app.example.com/invite/accept?token=abc",
	}
	for _, candidate := range allowed {
		if !handler.isAllowedFrontendRedirect(candidate) {
			t.Errorf("expected %q to be allowed", candidate)
		}
	}

	rejected := []string{
		"https://evil.example/auth/callback",
		"http://app.example.com/auth/callback",
		"https://app.example.com.evil.example/auth/callback",
		"https://user@app.example.com/auth/callback",
		"//evil.example/auth/callback",
	}
	for _, candidate := range rejected {
		if handler.isAllowedFrontendRedirect(candidate) {
			t.Errorf("expected %q to be rejected", candidate)
		}
	}
}

func TestProductionOAuthCookieAttributes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewAuthHandler(nil, nil, "https://app.example.com", true)
	response := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(response)

	handler.setOAuthCookie(context, "oauth_state", "state", 600)
	cookie := response.Header().Get("Set-Cookie")
	for _, attribute := range []string{"HttpOnly", "Secure", "SameSite=Lax"} {
		if !strings.Contains(cookie, attribute) {
			t.Errorf("cookie %q does not contain %s", cookie, attribute)
		}
	}
}
