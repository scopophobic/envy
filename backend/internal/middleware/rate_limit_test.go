package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestRateLimiterRejectsRequestsOverLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	limiter := NewRateLimiter(2, time.Minute)
	router := gin.New()
	router.GET("/sensitive", limiter.Middleware(ClientIPRateLimitKey), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	for i, want := range []int{http.StatusNoContent, http.StatusNoContent, http.StatusTooManyRequests} {
		req := httptest.NewRequest(http.MethodGet, "/sensitive", nil)
		req.RemoteAddr = "192.0.2.10:1234"
		response := httptest.NewRecorder()
		router.ServeHTTP(response, req)
		if response.Code != want {
			t.Fatalf("request %d status = %d, want %d", i+1, response.Code, want)
		}
		if response.Header().Get("RateLimit-Limit") != "2" {
			t.Fatalf("request %d missing rate limit headers", i+1)
		}
	}
}

func TestRateLimiterSeparatesRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	limiter := NewRateLimiter(1, time.Minute)
	router := gin.New()
	handler := limiter.Middleware(ClientIPRateLimitKey)
	router.GET("/first", handler, func(c *gin.Context) { c.Status(http.StatusNoContent) })
	router.GET("/second", handler, func(c *gin.Context) { c.Status(http.StatusNoContent) })

	for _, path := range []string{"/first", "/second"} {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != http.StatusNoContent {
			t.Fatalf("%s status = %d, want %d", path, response.Code, http.StatusNoContent)
		}
	}
}
