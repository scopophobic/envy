package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	// Server
	Port string
	Env  string

	// Database
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string

	// JWT
	JWTSecret              string
	JWTAccessTokenExpiry   string
	JWTRefreshTokenExpiry  string

	// Google OAuth
	GoogleClientID     string
	GoogleClientSecret string
	GoogleRedirectURL  string

	// AWS KMS
	AWSRegion          string
	AWSKMSKeyID        string
	AWSAccessKeyID     string
	AWSSecretAccessKey string

	// Frontend
	FrontendURL string

	// Razorpay
	RazorpayKeyID        string
	RazorpayKeySecret    string
	RazorpayWebhookSecret string
	RazorpayPlanStarter  string
	RazorpayPlanTeam     string

	// Email (team invitations)
	SMTPHost      string
	SMTPPort      string
	SMTPUsername  string
	SMTPPassword  string
	SMTPFromEmail string
	SMTPFromName  string

	InviteTokenTTLHours int

	// Rate Limiting
	RateLimitEnabled bool
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	// Try common locations so the API picks up Razorpay keys whether you start from repo root or backend/
	_ = godotenv.Load(".env", "backend/.env", "../backend/.env", "../.env")

	cfg := &Config{
		Port: getEnv("PORT", "8080"),
		Env:  getEnv("ENV", "development"),

		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "envo"),
		DBPassword: getEnv("DB_PASSWORD", ""),
		DBName:     getEnv("DB_NAME", "envo_db"),
		DBSSLMode:  getEnv("DB_SSLMODE", "disable"),

		JWTSecret:              getEnv("JWT_SECRET", ""),
		JWTAccessTokenExpiry:   getEnv("JWT_ACCESS_TOKEN_EXPIRY", "15m"),
		JWTRefreshTokenExpiry:  getEnv("JWT_REFRESH_TOKEN_EXPIRY", "720h"),

		GoogleClientID:     strings.TrimSpace(getEnv("GOOGLE_CLIENT_ID", "")),
		GoogleClientSecret: strings.TrimSpace(getEnv("GOOGLE_CLIENT_SECRET", "")),
		GoogleRedirectURL:  strings.TrimSpace(getEnv("GOOGLE_REDIRECT_URL", "")),

		AWSRegion:          getEnv("AWS_REGION", "us-east-1"),
		AWSKMSKeyID:        getEnv("AWS_KMS_KEY_ID", ""),
		AWSAccessKeyID:     getEnv("AWS_ACCESS_KEY_ID", ""),
		AWSSecretAccessKey: getEnv("AWS_SECRET_ACCESS_KEY", ""),

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

		RateLimitEnabled: getEnvBool("RATE_LIMIT_ENABLED", true),
	}

	// Validate required fields
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate checks if required configuration values are set
func (c *Config) Validate() error {
	if c.JWTSecret == "" {
		return fmt.Errorf("JWT_SECRET is required")
	}

	if c.DBPassword == "" && c.Env == "production" && strings.TrimSpace(os.Getenv("DB_URL")) == "" {
		return fmt.Errorf("DB_PASSWORD is required in production")
	}

	// OAuth validation only in production
	if c.Env == "production" {
		if c.GoogleClientID == "" {
			return fmt.Errorf("GOOGLE_CLIENT_ID is required")
		}
		if c.GoogleClientSecret == "" {
			return fmt.Errorf("GOOGLE_CLIENT_SECRET is required")
		}
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
