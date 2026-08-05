package config

import (
	"strings"
	"testing"
	"time"
)

func validProductionConfig() *Config {
	return &Config{
		Env:                            "production",
		DBPassword:                     "database-password",
		JWTSecret:                      strings.Repeat("a", 64),
		JWTAccessTokenExpiry:           "15m",
		JWTRefreshTokenExpiry:          "720h",
		GoogleClientID:                 "client-id",
		GoogleClientSecret:             "client-secret",
		GoogleRedirectURL:              "https://api.example.com/api/v1/auth/google/callback",
		FrontendURL:                    "https://app.example.com",
		AWSKMSKeyID:                    "arn:aws:kms:region:account:key/id",
		RateLimitEnabled:               true,
		AuthRateLimitPerMinute:         30,
		SecretExportRateLimitPerMinute: 30,
		PlatformSyncRateLimitPerMinute: 10,
		AgentResolveRateLimitPerMinute: 60,
		DBMaxOpenConns:                 25,
		DBMaxIdleConns:                 10,
		DBConnMaxLifetime:              30 * time.Minute,
		DBConnMaxIdleTime:              5 * time.Minute,
		HTTPReadHeaderTimeout:          5 * time.Second,
		HTTPReadTimeout:                15 * time.Second,
		HTTPWriteTimeout:               60 * time.Second,
		HTTPIdleTimeout:                2 * time.Minute,
		HTTPShutdownTimeout:            15 * time.Second,
		HTTPMaxHeaderBytes:             1 << 20,
		MaxRequestBodyBytes:            1 << 20,
		TierCacheTTL:                   5 * time.Minute,
		SecretDecryptConcurrency:       8,
		AgentUsageWriteInterval:        time.Minute,
	}
}

func TestProductionConfigRejectsWeakJWTSecret(t *testing.T) {
	cfg := validProductionConfig()
	cfg.JWTSecret = "change_me"
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "JWT_SECRET") {
		t.Fatalf("Validate() error = %v, want weak JWT secret error", err)
	}
}

func TestProductionConfigRequiresHTTPS(t *testing.T) {
	cfg := validProductionConfig()
	cfg.FrontendURL = "http://app.example.com"
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "FRONTEND_URL") {
		t.Fatalf("Validate() error = %v, want FRONTEND_URL HTTPS error", err)
	}
}

func TestProductionConfigRequiresKMSByDefault(t *testing.T) {
	cfg := validProductionConfig()
	cfg.AWSKMSKeyID = ""
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "AWS_KMS_KEY_ID") {
		t.Fatalf("Validate() error = %v, want KMS error", err)
	}

	cfg.AllowLocalEncryptionInProduction = true
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() with explicit local-encryption override returned %v", err)
	}
}

func TestProductionConfigAcceptsHardenedSettings(t *testing.T) {
	if err := validProductionConfig().Validate(); err != nil {
		t.Fatalf("Validate() returned %v", err)
	}
}

func TestConfigRejectsUnsafePoolAndConcurrencySettings(t *testing.T) {
	cfg := validProductionConfig()
	cfg.DBMaxIdleConns = cfg.DBMaxOpenConns + 1
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "database pool") {
		t.Fatalf("Validate() error = %v, want database pool error", err)
	}

	cfg = validProductionConfig()
	cfg.SecretDecryptConcurrency = 65
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "performance") {
		t.Fatalf("Validate() error = %v, want performance settings error", err)
	}
}
