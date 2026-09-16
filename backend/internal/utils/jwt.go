package utils

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Claims represents JWT claims
type Claims struct {
	UserID      uuid.UUID `json:"user_id"`
	Email       string    `json:"email"`
	Permissions []string  `json:"permissions,omitempty"`
	TokenType   string    `json:"token_type"`
	jwt.RegisteredClaims
}

// JWTManager handles JWT token generation and validation
type JWTManager struct {
	secretKey            string
	accessTokenDuration  time.Duration
	refreshTokenDuration time.Duration
}

// NewJWTManager creates a new JWT manager
func NewJWTManager(secretKey string, accessTokenExpiry, refreshTokenExpiry string) (*JWTManager, error) {
	accessDuration, err := time.ParseDuration(accessTokenExpiry)
	if err != nil {
		return nil, fmt.Errorf("invalid access token expiry: %w", err)
	}

	refreshDuration, err := time.ParseDuration(refreshTokenExpiry)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token expiry: %w", err)
	}

	return &JWTManager{
		secretKey:            secretKey,
		accessTokenDuration:  accessDuration,
		refreshTokenDuration: refreshDuration,
	}, nil
}

// GenerateAccessToken generates a new access token
func (m *JWTManager) GenerateAccessToken(userID uuid.UUID, email string, permissions []string) (string, error) {
	claims := Claims{
		UserID:      userID,
		Email:       email,
		Permissions: permissions,
		TokenType:   "access",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(m.accessTokenDuration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(m.secretKey))
}

// GenerateRefreshToken generates a new refresh token (unique per call via jti)
func (m *JWTManager) GenerateRefreshToken(userID uuid.UUID) (string, time.Time, error) {
	expiresAt := time.Now().Add(m.refreshTokenDuration)
	now := time.Now()
	claims := Claims{
		UserID:    userID,
		TokenType: "refresh",
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.New().String(), // unique so DB unique constraint on token never collides
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(m.secretKey))
	if err != nil {
		return "", time.Time{}, err
	}

	return tokenString, expiresAt, nil
}

// ValidateToken validates a JWT token and returns the claims
func (m *JWTManager) validateToken(tokenString, expectedType string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(m.secretKey), nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	if claims.TokenType != expectedType {
		return nil, fmt.Errorf("invalid token type")
	}

	return claims, nil
}

func (m *JWTManager) ValidateAccessToken(tokenString string) (*Claims, error) {
	return m.validateToken(tokenString, "access")
}

func (m *JWTManager) ValidateRefreshToken(tokenString string) (*Claims, error) {
	return m.validateToken(tokenString, "refresh")
}

// ValidateToken remains an access-token validator for compatibility with
// existing callers. Refresh tokens are deliberately rejected here.
func (m *JWTManager) ValidateToken(tokenString string) (*Claims, error) {
	return m.ValidateAccessToken(tokenString)
}

// ExtractToken extracts the token from Authorization header
// Expected format: "Bearer <token>"
func ExtractToken(authHeader string) (string, error) {
	if authHeader == "" {
		return "", fmt.Errorf("authorization header is empty")
	}

	const bearerPrefix = "Bearer "
	if len(authHeader) < len(bearerPrefix) {
		return "", fmt.Errorf("invalid authorization header format")
	}

	if authHeader[:len(bearerPrefix)] != bearerPrefix {
		return "", fmt.Errorf("authorization header must start with 'Bearer '")
	}

	return authHeader[len(bearerPrefix):], nil
}
