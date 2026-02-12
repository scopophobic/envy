package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/envo/backend/internal/config"
	"github.com/envo/backend/internal/database"
	"github.com/envo/backend/internal/models"
	"github.com/envo/backend/internal/utils"
	"github.com/google/uuid"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"gorm.io/gorm"
)

// GoogleUserInfo represents user info from Google
type GoogleUserInfo struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Picture       string `json:"picture"`
}

// AuthService handles authentication logic
type AuthService struct {
	oauth2Config *oauth2.Config
	jwtManager   *utils.JWTManager
}

// NewAuthService creates a new auth service
func NewAuthService(cfg *config.Config, jwtManager *utils.JWTManager) *AuthService {
	oauth2Config := &oauth2.Config{
		ClientID:     cfg.GoogleClientID,
		ClientSecret: cfg.GoogleClientSecret,
		RedirectURL:  cfg.GoogleRedirectURL,
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
		Endpoint: google.Endpoint,
	}

	return &AuthService{
		oauth2Config: oauth2Config,
		jwtManager:   jwtManager,
	}
}

// GetAuthURL returns the Google OAuth URL
func (s *AuthService) GetAuthURL(state string) string {
	return s.oauth2Config.AuthCodeURL(state, oauth2.AccessTypeOffline)
}

// HandleCallback handles the OAuth callback
func (s *AuthService) HandleCallback(ctx context.Context, code string) (*models.User, string, string, error) {
	// Exchange code for token
	token, err := s.oauth2Config.Exchange(ctx, code)
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to exchange code: %w", err)
	}

	// Get user info from Google
	userInfo, err := s.getUserInfo(ctx, token)
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to get user info: %w", err)
	}

	// Find or create user
	user, err := s.findOrCreateUser(userInfo)
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to find or create user: %w", err)
	}

	// Generate tokens
	accessToken, refreshToken, err := s.generateTokens(user)
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to generate tokens: %w", err)
	}

	return user, accessToken, refreshToken, nil
}

// GenerateTokensForUser is used by the CLI exchange flow to mint tokens after a one-time login code.
func (s *AuthService) GenerateTokensForUser(user *models.User) (string, string, error) {
	return s.generateTokens(user)
}

// getUserInfo fetches user info from Google
func (s *AuthService) getUserInfo(ctx context.Context, token *oauth2.Token) (*GoogleUserInfo, error) {
	client := s.oauth2Config.Client(ctx, token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var userInfo GoogleUserInfo
	if err := json.Unmarshal(body, &userInfo); err != nil {
		return nil, err
	}

	return &userInfo, nil
}

// findOrCreateUser finds or creates a user
func (s *AuthService) findOrCreateUser(userInfo *GoogleUserInfo) (*models.User, error) {
	db := database.GetDB()

	var user models.User
	err := db.Where("oauth_provider = ? AND oauth_id = ?", "google", userInfo.ID).First(&user).Error

	if err == gorm.ErrRecordNotFound {
		// Create new user
		user = models.User{
			Email:              userInfo.Email,
			Name:               userInfo.Name,
			OAuthProvider:      "google",
			OAuthID:            userInfo.ID,
			SubscriptionTier:   string(models.TierFree),
			SubscriptionStatus: string(models.StatusActive),
		}

		if err := db.Create(&user).Error; err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}

	return &user, nil
}

// generateTokens generates access and refresh tokens
func (s *AuthService) generateTokens(user *models.User) (string, string, error) {
	db := database.GetDB()

	// Load user with permissions (if Preload fails, continue with empty permissions so login still works)
	_ = db.Preload("OrgMemberships.Role.Permissions").First(user, user.ID)

	// Get user permissions (may be empty for new users)
	var permissions []string
	if user.OrgMemberships != nil {
		for _, membership := range user.OrgMemberships {
			if membership.Role.ID == uuid.Nil {
				continue
			}
			for _, permission := range membership.Role.Permissions {
				found := false
				for _, p := range permissions {
					if p == permission.Name {
						found = true
						break
					}
				}
				if !found {
					permissions = append(permissions, permission.Name)
				}
			}
		}
	}

	// Generate access token
	accessToken, err := s.jwtManager.GenerateAccessToken(user.ID, user.Email, permissions)
	if err != nil {
		return "", "", err
	}

	// Generate refresh token
	refreshTokenString, expiresAt, err := s.jwtManager.GenerateRefreshToken(user.ID)
	if err != nil {
		return "", "", err
	}

	// Save refresh token to database
	refreshToken := models.RefreshToken{
		UserID:    user.ID,
		Token:     refreshTokenString,
		ExpiresAt: expiresAt,
		Revoked:   false,
	}

	if err := db.Create(&refreshToken).Error; err != nil {
		return "", "", err
	}

	return accessToken, refreshTokenString, nil
}

// RefreshAccessToken generates a new access token from a refresh token
func (s *AuthService) RefreshAccessToken(ctx context.Context, refreshTokenString string) (string, error) {
	db := database.GetDB()

	// Find refresh token
	var refreshToken models.RefreshToken
	if err := db.Where("token = ?", refreshTokenString).First(&refreshToken).Error; err != nil {
		return "", fmt.Errorf("invalid refresh token")
	}

	// Check if valid
	if !refreshToken.IsValid() {
		return "", fmt.Errorf("refresh token expired or revoked")
	}

	// Load user
	var user models.User
	if err := db.Preload("OrgMemberships.Role.Permissions").First(&user, refreshToken.UserID).Error; err != nil {
		return "", err
	}

	// Get permissions
	var permissions []string
	for _, membership := range user.OrgMemberships {
		for _, permission := range membership.Role.Permissions {
			found := false
			for _, p := range permissions {
				if p == permission.Name {
					found = true
					break
				}
			}
			if !found {
				permissions = append(permissions, permission.Name)
			}
		}
	}

	// Generate new access token
	accessToken, err := s.jwtManager.GenerateAccessToken(user.ID, user.Email, permissions)
	if err != nil {
		return "", err
	}

	return accessToken, nil
}

// Logout revokes a refresh token
func (s *AuthService) Logout(ctx context.Context, refreshTokenString string) error {
	db := database.GetDB()

	return db.Model(&models.RefreshToken{}).
		Where("token = ?", refreshTokenString).
		Update("revoked", true).Error
}
