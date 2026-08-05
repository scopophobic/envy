package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	// Server
	Port           string
	Env            string
	TrustedProxies []string

	// Database
	DBHost            string
	DBPort            string
	DBUser            string
	DBPassword        string
	DBName            string
	DBSSLMode         string
	DBMaxOpenConns    int
	DBMaxIdleConns    int
	DBConnMaxLifetime time.Duration
	DBConnMaxIdleTime time.Duration

	// HTTP server reliability limits
	HTTPReadHeaderTimeout time.Duration
	HTTPReadTimeout       time.Duration
	HTTPWriteTimeout      time.Duration
	HTTPIdleTimeout       time.Duration
	HTTPShutdownTimeout   time.Duration
	HTTPMaxHeaderBytes    int
	MaxRequestBodyBytes   int64

	// JWT
	JWTSecret             string
	JWTAccessTokenExpiry  string
	JWTRefreshTokenExpiry string

	// Google OAuth
	GoogleClientID     string
	GoogleClientSecret string
	GoogleRedirectURL  string

	// AWS KMS
	AWSRegion                        string
	AWSKMSKeyID                      string
	AWSAccessKeyID                   string
	AWSSecretAccessKey               string
	AllowLocalEncryptionInProduction bool

	// Frontend
	FrontendURL string

	// Razorpay
	RazorpayKeyID         string
	RazorpayKeySecret     string
	RazorpayWebhookSecret string
	RazorpayPlanStarter   string
	RazorpayPlanTeam      string

	// Email (team invitations)
	SMTPHost      string
	SMTPPort      string
	SMTPUsername  string
	SMTPPassword  string
	SMTPFromEmail string
	SMTPFromName  string

	InviteTokenTTLHours int

	// Rate Limiting
	RateLimitEnabled               bool
	AuthRateLimitPerMinute         int
	SecretExportRateLimitPerMinute int
	PlatformSyncRateLimitPerMinute int
	AgentResolveRateLimitPerMinute int

	// Performance controls
	TierCacheTTL             time.Duration
	SecretDecryptConcurrency int
	AgentUsageWriteInterval  time.Duration
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	// Try common locations so the API picks up Razorpay keys whether you start from repo root or backend/
	_ = godotenv.Load(".env", "backend/.env", "../backend/.env", "../.env")

	cfg := &Config{
		Port:           getEnv("PORT", "8080"),
		Env:            getEnv("ENV", "development"),
		TrustedProxies: splitCSV(getEnv("TRUSTED_PROXIES", "")),

		DBHost:            getEnv("DB_HOST", "localhost"),
		DBPort:            getEnv("DB_PORT", "5432"),
		DBUser:            getEnv("DB_USER", "envo"),
		DBPassword:        getEnv("DB_PASSWORD", ""),
		DBName:            getEnv("DB_NAME", "envo_db"),
		DBSSLMode:         getEnv("DB_SSLMODE", "disable"),
		DBMaxOpenConns:    getEnvInt("DB_MAX_OPEN_CONNS", 25),
		DBMaxIdleConns:    getEnvInt("DB_MAX_IDLE_CONNS", 10),
		DBConnMaxLifetime: getEnvDuration("DB_CONN_MAX_LIFETIME", 30*time.Minute),
		DBConnMaxIdleTime: getEnvDuration("DB_CONN_MAX_IDLE_TIME", 5*time.Minute),

		HTTPReadHeaderTimeout: getEnvDuration("HTTP_READ_HEADER_TIMEOUT", 5*time.Second),
		HTTPReadTimeout:       getEnvDuration("HTTP_READ_TIMEOUT", 15*time.Second),
		HTTPWriteTimeout:      getEnvDuration("HTTP_WRITE_TIMEOUT", 60*time.Second),
		HTTPIdleTimeout:       getEnvDuration("HTTP_IDLE_TIMEOUT", 2*time.Minute),
		HTTPShutdownTimeout:   getEnvDuration("HTTP_SHUTDOWN_TIMEOUT", 15*time.Second),
		HTTPMaxHeaderBytes:    getEnvInt("HTTP_MAX_HEADER_BYTES", 1<<20),
		MaxRequestBodyBytes:   int64(getEnvInt("MAX_REQUEST_BODY_BYTES", 1<<20)),

		JWTSecret:             getEnv("JWT_SECRET", ""),
		JWTAccessTokenExpiry:  getEnv("JWT_ACCESS_TOKEN_EXPIRY", "15m"),
		JWTRefreshTokenExpiry: getEnv("JWT_REFRESH_TOKEN_EXPIRY", "720h"),

		GoogleClientID:     strings.TrimSpace(getEnv("GOOGLE_CLIENT_ID", "")),
		GoogleClientSecret: strings.TrimSpace(getEnv("GOOGLE_CLIENT_SECRET", "")),
		GoogleRedirectURL:  strings.TrimSpace(getEnv("GOOGLE_REDIRECT_URL", "")),

		AWSRegion:                        getEnv("AWS_REGION", "us-east-1"),
		AWSKMSKeyID:                      getEnv("AWS_KMS_KEY_ID", ""),
		AWSAccessKeyID:                   getEnv("AWS_ACCESS_KEY_ID", ""),
		AWSSecretAccessKey:               getEnv("AWS_SECRET_ACCESS_KEY", ""),
		AllowLocalEncryptionInProduction: getEnvBool("ALLOW_LOCAL_ENCRYPTION_IN_PRODUCTION", false),

		FrontendURL: getEnv("FRONTEND_URL", "http://localhost:3000"),

		RazorpayKeyID:         getEnv("RAZORPAY_KEY_ID", ""),
		RazorpayKeySecret:     getEnv("RAZORPAY_KEY_SECRET", ""),
		RazorpayWebhookSecret: getEnv("RAZORPAY_WEBHOOK_SECRET", ""),
		RazorpayPlanStarter:   getEnv("RAZORPAY_PLAN_STARTER", ""),
		RazorpayPlanTeam:      getEnv("RAZORPAY_PLAN_TEAM", ""),

		SMTPHost:      strings.TrimSpace(getEnv("SMTP_HOST", "")),
		SMTPPort:      strings.TrimSpace(getEnv("SMTP_PORT", "587")),
		SMTPUsername:  strings.TrimSpace(getEnv("SMTP_USERNAME", "")),
		SMTPPassword:  strings.TrimSpace(getEnv("SMTP_PASSWORD", "")),
		SMTPFromEmail: strings.TrimSpace(getEnv("SMTP_FROM_EMAIL", "")),
		SMTPFromName:  strings.TrimSpace(getEnv("SMTP_FROM_NAME", "Envo")),

		InviteTokenTTLHours: getEnvInt("INVITE_TOKEN_TTL_HOURS", 168),

		RateLimitEnabled:               getEnvBool("RATE_LIMIT_ENABLED", true),
		AuthRateLimitPerMinute:         getEnvInt("AUTH_RATE_LIMIT_PER_MINUTE", 30),
		SecretExportRateLimitPerMinute: getEnvInt("SECRET_EXPORT_RATE_LIMIT_PER_MINUTE", 30),
		PlatformSyncRateLimitPerMinute: getEnvInt("PLATFORM_SYNC_RATE_LIMIT_PER_MINUTE", 10),
		AgentResolveRateLimitPerMinute: getEnvInt("AGENT_RESOLVE_RATE_LIMIT_PER_MINUTE", 60),

		TierCacheTTL:             getEnvDuration("TIER_CACHE_TTL", 5*time.Minute),
		SecretDecryptConcurrency: getEnvInt("SECRET_DECRYPT_CONCURRENCY", 8),
		AgentUsageWriteInterval:  getEnvDuration("AGENT_USAGE_WRITE_INTERVAL", time.Minute),
	}

	// Validate required fields
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate checks if required configuration values are set
func (c *Config) Validate() error {
	if strings.TrimSpace(c.JWTSecret) == "" {
		return fmt.Errorf("JWT_SECRET is required")
	}
	accessExpiry, err := time.ParseDuration(c.JWTAccessTokenExpiry)
	if err != nil {
		return fmt.Errorf("JWT_ACCESS_TOKEN_EXPIRY is invalid: %w", err)
	}
	if accessExpiry <= 0 {
		return fmt.Errorf("JWT_ACCESS_TOKEN_EXPIRY must be positive")
	}
	refreshExpiry, err := time.ParseDuration(c.JWTRefreshTokenExpiry)
	if err != nil {
		return fmt.Errorf("JWT_REFRESH_TOKEN_EXPIRY is invalid: %w", err)
	}
	if refreshExpiry <= 0 {
		return fmt.Errorf("JWT_REFRESH_TOKEN_EXPIRY must be positive")
	}
	if refreshExpiry <= accessExpiry {
		return fmt.Errorf("JWT_REFRESH_TOKEN_EXPIRY must be longer than JWT_ACCESS_TOKEN_EXPIRY")
	}
	if c.RateLimitEnabled && (c.AuthRateLimitPerMinute <= 0 || c.SecretExportRateLimitPerMinute <= 0 || c.PlatformSyncRateLimitPerMinute <= 0 || c.AgentResolveRateLimitPerMinute <= 0) {
		return fmt.Errorf("rate limits must be positive when RATE_LIMIT_ENABLED is true")
	}
	if c.DBMaxOpenConns <= 0 || c.DBMaxIdleConns < 0 || c.DBMaxIdleConns > c.DBMaxOpenConns || c.DBConnMaxLifetime <= 0 || c.DBConnMaxIdleTime <= 0 {
		return fmt.Errorf("database pool settings are invalid")
	}
	if c.HTTPReadHeaderTimeout <= 0 || c.HTTPReadTimeout <= 0 || c.HTTPWriteTimeout <= 0 || c.HTTPIdleTimeout <= 0 || c.HTTPShutdownTimeout <= 0 || c.HTTPMaxHeaderBytes <= 0 || c.MaxRequestBodyBytes <= 0 {
		return fmt.Errorf("HTTP server limits and timeouts must be positive")
	}
	if c.TierCacheTTL <= 0 || c.SecretDecryptConcurrency <= 0 || c.SecretDecryptConcurrency > 64 || c.AgentUsageWriteInterval <= 0 {
		return fmt.Errorf("performance settings are invalid")
	}

	if c.DBPassword == "" && c.Env == "production" && strings.TrimSpace(os.Getenv("DB_URL")) == "" {
		return fmt.Errorf("DB_PASSWORD is required in production")
	}

	// OAuth validation only in production
	if c.Env == "production" {
		secretLower := strings.ToLower(c.JWTSecret)
		if len(c.JWTSecret) < 32 || strings.Contains(secretLower, "change_me") || strings.Contains(secretLower, "your_jwt") {
			return fmt.Errorf("JWT_SECRET must be a random value of at least 32 characters in production")
		}
		if c.GoogleClientID == "" {
			return fmt.Errorf("GOOGLE_CLIENT_ID is required")
		}
		if c.GoogleClientSecret == "" {
			return fmt.Errorf("GOOGLE_CLIENT_SECRET is required")
		}
		if err := requireHTTPSURL("FRONTEND_URL", c.FrontendURL); err != nil {
			return err
		}
		if err := requireHTTPSURL("GOOGLE_REDIRECT_URL", c.GoogleRedirectURL); err != nil {
			return err
		}
		if strings.TrimSpace(c.AWSKMSKeyID) == "" && !c.AllowLocalEncryptionInProduction {
			return fmt.Errorf("AWS_KMS_KEY_ID is required in production unless ALLOW_LOCAL_ENCRYPTION_IN_PRODUCTION=true is explicitly set")
		}
		if c.RazorpayKeyID != "" && c.RazorpayKeySecret != "" && strings.TrimSpace(c.RazorpayWebhookSecret) == "" {
			return fmt.Errorf("RAZORPAY_WEBHOOK_SECRET is required when billing is enabled in production")
		}
	}
	if (c.RazorpayKeyID == "") != (c.RazorpayKeySecret == "") {
		return fmt.Errorf("RAZORPAY_KEY_ID and RAZORPAY_KEY_SECRET must be configured together")
	}
	if (c.AWSAccessKeyID == "") != (c.AWSSecretAccessKey == "") {
		return fmt.Errorf("AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY must be configured together; leave both empty to use an IAM role")
	}

	return nil
}

func requireHTTPSURL(name, raw string) error {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil {
		return fmt.Errorf("%s must be an absolute HTTPS URL in production", name)
	}
	return nil
}

// GetDSN returns the database connection string
func (c *Config) GetDSN() string {
	// Prefer DB_URL when provided (e.g. Supabase).
	// Note: this project uses GORM's postgres driver which expects a DSN string.
	if dbURL := strings.TrimSpace(os.Getenv("DB_URL")); dbURL != "" {
		dsn := dbURL

		// Supabase commonly requires sslmode=require. If sslmode isn't already
		// present in DB_URL, append it from DB_SSLMODE (or default to require).
		sslMode := strings.TrimSpace(c.DBSSLMode)
		if sslMode == "" {
			sslMode = "require"
		}

		// Only append when DB_URL doesn't already contain sslmode=...
		if !strings.Contains(strings.ToLower(dsn), "sslmode=") {
			if strings.Contains(dsn, "?") {
				dsn = dsn + "&sslmode=" + sslMode
			} else {
				dsn = dsn + "?sslmode=" + sslMode
			}
		}

		return dsn
	}

	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.DBHost, c.DBPort, c.DBUser, c.DBPassword, c.DBName, c.DBSSLMode,
	)
}

// IsDevelopment returns true if running in development mode
func (c *Config) IsDevelopment() bool {
	return c.Env == "development"
}

// IsProduction returns true if running in production mode
func (c *Config) IsProduction() bool {
	return c.Env == "production"
}

// Helper functions
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		boolValue, err := strconv.ParseBool(value)
		if err != nil {
			return defaultValue
		}
		return boolValue
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		intValue, err := strconv.Atoi(value)
		if err != nil {
			return defaultValue
		}
		return intValue
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		parsed, err := time.ParseDuration(value)
		if err == nil {
			return parsed
		}
	}
	return defaultValue
}

func splitCSV(value string) []string {
	var values []string
	for _, item := range strings.Split(value, ",") {
		if trimmed := strings.TrimSpace(item); trimmed != "" {
			values = append(values, trimmed)
		}
	}
	return values
}
