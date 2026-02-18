package handlers

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/envo/backend/internal/database"
	"github.com/envo/backend/internal/middleware"
	"github.com/envo/backend/internal/models"
	"github.com/envo/backend/internal/services"
	"github.com/google/uuid"
	"github.com/gin-gonic/gin"
)

// AuthHandler handles authentication endpoints
type AuthHandler struct {
	authService *services.AuthService
	tierService *services.TierService
	frontendURL string
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(authService *services.AuthService, tierService *services.TierService, frontendURL string) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		tierService: tierService,
		frontendURL: frontendURL,
	}
}

// GoogleLogin initiates Google OAuth flow
// GET /api/v1/auth/google/login
func (h *AuthHandler) GoogleLogin(c *gin.Context) {
	// Generate random state
	b := make([]byte, 32)
	rand.Read(b)
	state := base64.URLEncoding.EncodeToString(b)

	// Store state in session/cookie (for production, use secure session storage)
	c.SetCookie("oauth_state", state, 600, "/", "", false, true)

	// Get auth URL
	url := h.authService.GetAuthURL(state)

	c.JSON(http.StatusOK, gin.H{
		"url": url,
	})
}

// GoogleLoginRedirect initiates Google OAuth flow and redirects the browser.
// GET /api/v1/auth/google/redirect
func (h *AuthHandler) GoogleLoginRedirect(c *gin.Context) {
	// Generate random state
	b := make([]byte, 32)
	rand.Read(b)
	state := base64.URLEncoding.EncodeToString(b)

	// Store state in cookie
	c.SetCookie("oauth_state", state, 600, "/", "", false, true)
	
	// Mark this as a web flow (not CLI)
	c.SetCookie("oauth_flow", "web", 600, "/", "", false, true)

	// Optional: capture intended post-login redirect (frontend or CLI flow)
	if next := strings.TrimSpace(c.Query("next")); next != "" {
		// only allow http(s) to reduce footguns; CLI uses localhost callback via its own cookie below
		if u, err := url.Parse(next); err == nil && (u.Scheme == "http" || u.Scheme == "https") {
			c.SetCookie("post_login_next", next, 600, "/", "", false, true)
		}
	}

	c.Redirect(http.StatusFound, h.authService.GetAuthURL(state))
}

// CLIGoogleStart starts a CLI browser login session.
// The CLI runs a local HTTP server and opens this URL in the browser.
// GET /api/v1/auth/cli/google/start?callback=http://127.0.0.1:53682/callback
func (h *AuthHandler) CLIGoogleStart(c *gin.Context) {
	callback := strings.TrimSpace(c.Query("callback"))
	if callback == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "callback is required"})
		return
	}

	u, err := url.Parse(callback)
	if err != nil || u.Scheme != "http" || (u.Hostname() != "127.0.0.1" && u.Hostname() != "localhost") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "callback must be a localhost http URL"})
		return
	}

	// Store callback in cookie (browser-scoped, survives OAuth redirect)
	c.SetCookie("cli_callback", callback, 600, "/", "", false, true)
	
	// Mark this as a CLI flow (not web)
	c.SetCookie("oauth_flow", "cli", 600, "/", "", false, true)

	// Generate state and redirect to Google
	b := make([]byte, 32)
	rand.Read(b)
	state := base64.URLEncoding.EncodeToString(b)
	c.SetCookie("oauth_state", state, 600, "/", "", false, true)

	c.Redirect(http.StatusFound, h.authService.GetAuthURL(state))
}

// GoogleCallback handles Google OAuth callback
// GET /api/v1/auth/google/callback
func (h *AuthHandler) GoogleCallback(c *gin.Context) {
	// Verify state (must match cookie set when we redirected to Google)
	state := c.Query("state")
	storedState, cookieErr := c.Cookie("oauth_state")
	if cookieErr != nil || state == "" || state != storedState {
		msg := "Invalid state parameter. Ensure GOOGLE_REDIRECT_URL in backend .env uses the same host you use to open the login page (e.g. both http://localhost:8080 or both http://127.0.0.1:8080)."
		c.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return
	}

	// Clear state cookie
	c.SetCookie("oauth_state", "", -1, "/", "", false, true)

	// Get code
	code := c.Query("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Code not provided"})
		return
	}

	// Handle callback
	user, accessToken, refreshToken, err := h.authService.HandleCallback(c.Request.Context(), code)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to authenticate", "details": err.Error()})
		return
	}

	// Check flow type from cookie to distinguish CLI vs web
	flowType, _ := c.Cookie("oauth_flow")
	c.SetCookie("oauth_flow", "", -1, "/", "", false, true) // Clear flow cookie

	// If this is a CLI login flow, create a short-lived exchange code and redirect to the CLI callback.
	if flowType == "cli" {
		cliCallback, err := c.Cookie("cli_callback")
		if err == nil && strings.TrimSpace(cliCallback) != "" {
			// Clear cookie
			c.SetCookie("cli_callback", "", -1, "/", "", false, true)

			// Create exchange code
			exchangeCode := uuid.NewString()
			expiresAt := time.Now().Add(2 * time.Minute)
			rec := &models.CLILoginCode{
				Code:      exchangeCode,
				UserID:    user.ID,
				ExpiresAt: expiresAt,
			}

			if err := database.GetDB().Create(rec).Error; err != nil {
				// fallback to JSON if DB write fails
				c.JSON(http.StatusOK, gin.H{
					"access_token":  accessToken,
					"refresh_token": refreshToken,
					"token_type":    "Bearer",
					"expires_in":    900,
				})
				return
			}

			redir, _ := url.Parse(cliCallback)
			q := redir.Query()
			q.Set("code", exchangeCode)
			redir.RawQuery = q.Encode()

			c.Redirect(http.StatusFound, redir.String())
			return
		}
	}

	// Default to web flow: redirect to frontend callback URL with tokens in hash
	var frontendCallback string
	if frontendNext, err := c.Cookie("post_login_next"); err == nil && strings.TrimSpace(frontendNext) != "" {
		// Use cookie value if set
		frontendCallback = strings.TrimSpace(frontendNext)
		c.SetCookie("post_login_next", "", -1, "/", "", false, true)
	} else {
		// Default to FRONTEND_URL/auth/callback
		frontendCallback = h.frontendURL + "/auth/callback"
	}

	// Redirect to frontend with tokens in URL hash (not query param for security)
	// Note: HTTP redirects don't preserve fragments, so we construct the full URL manually
	fragment := fmt.Sprintf("access_token=%s&refresh_token=%s&token_type=Bearer&expires_in=900",
		url.QueryEscape(accessToken),
		url.QueryEscape(refreshToken))
	redirectURL := frontendCallback + "#" + fragment
	c.Redirect(http.StatusFound, redirectURL)
}

// CLIExchange exchanges a short-lived CLI login code for tokens.
// POST /api/v1/auth/cli/exchange
func (h *AuthHandler) CLIExchange(c *gin.Context) {
	var req struct {
		Code string `json:"code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request", "details": err.Error()})
		return
	}

	db := database.GetDB()
	var rec models.CLILoginCode
	if err := db.Where("code = ?", req.Code).First(&rec).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid code"})
		return
	}

	now := time.Now()
	if !rec.IsValid(now) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Code expired or already used"})
		return
	}

	usedAt := now
	if err := db.Model(&models.CLILoginCode{}).Where("id = ?", rec.ID).Update("used_at", usedAt).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to mark code used"})
		return
	}

	// Load user
	var user models.User
	if err := db.First(&user, "id = ?", rec.UserID).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	// Generate tokens (reuse existing auth service logic)
	accessToken, refreshToken, err := h.authService.GenerateTokensForUser(&user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to generate tokens",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"token_type":    "Bearer",
		"expires_in":    900,
	})
}

// RefreshToken generates a new access token
// POST /api/v1/auth/refresh
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request", "details": err.Error()})
		return
	}

	accessToken, err := h.authService.RefreshAccessToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired refresh token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token": accessToken,
		"token_type":   "Bearer",
		"expires_in":   900, // 15 minutes
	})
}

// Logout revokes the refresh token
// POST /api/v1/auth/logout
func (h *AuthHandler) Logout(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	if err := h.authService.Logout(c.Request.Context(), req.RefreshToken); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to logout"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}

// GetCurrentUser returns the current authenticated user
// GET /api/v1/auth/me
func (h *AuthHandler) GetCurrentUser(c *gin.Context) {
	user, err := middleware.GetCurrentUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":    user.ID,
		"email": user.Email,
		"name":  user.Name,
		"tier":  user.SubscriptionTier,
		"oauth_provider": user.OAuthProvider,
		"created_at": user.CreatedAt,
	})
}

// GetTierInfo returns the current user's tier limits and per-org usage.
// Hierarchy: org >> project >> team member >> env >> secrets
// GET /api/v1/auth/tier-info
func (h *AuthHandler) GetTierInfo(c *gin.Context) {
	user, err := middleware.GetCurrentUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	db := database.GetDB()
	tier := user.SubscriptionTier

	maxOrgs, _ := h.tierService.GetLimit(tier, models.LimitTypeMaxOrgs)
	maxProjects, _ := h.tierService.GetLimit(tier, models.LimitTypeMaxProjects)
	maxDevs, _ := h.tierService.GetLimit(tier, models.LimitTypeMaxDevs)
	maxSecrets, _ := h.tierService.GetLimit(tier, models.LimitTypeMaxSecretsPerEnv)

	// Global: owned orgs
	var ownedOrgs int64
	db.Model(&models.Organization{}).Where("owner_id = ?", user.ID).Count(&ownedOrgs)

	// Per-org usage breakdown
	var orgs []models.Organization
	db.Where("owner_id = ?", user.ID).Find(&orgs)

	type orgUsage struct {
		ID       string `json:"id"`
		Name     string `json:"name"`
		Projects int64  `json:"projects"`
		Members  int64  `json:"members"`
		Secrets  int64  `json:"secrets"`
	}

	perOrg := make([]orgUsage, 0, len(orgs))
	for _, org := range orgs {
		var projects int64
		db.Model(&models.Project{}).Where("org_id = ?", org.ID).Count(&projects)

		var members int64
		db.Model(&models.OrgMember{}).Where("org_id = ?", org.ID).Count(&members)

		var secrets int64
		db.Model(&models.Secret{}).
			Joins("JOIN environments ON environments.id = secrets.environment_id").
			Joins("JOIN projects ON projects.id = environments.project_id").
			Where("projects.org_id = ?", org.ID).
			Count(&secrets)

		perOrg = append(perOrg, orgUsage{
			ID:       org.ID.String(),
			Name:     org.Name,
			Projects: projects,
			Members:  members,
			Secrets:  secrets,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"tier": tier,
		"limits": gin.H{
			"max_orgs":             maxOrgs,
			"max_projects_per_org": maxProjects,
			"max_devs_per_org":     maxDevs,
			"max_secrets_per_env":  maxSecrets,
		},
		"usage": gin.H{
			"owned_orgs": ownedOrgs,
			"orgs":       perOrg,
		},
	})
}
