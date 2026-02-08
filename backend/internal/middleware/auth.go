package middleware

import (
	"net/http"

	"github.com/envo/backend/internal/database"
	"github.com/envo/backend/internal/models"
	"github.com/envo/backend/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// AuthMiddleware validates JWT tokens and attaches user to context
func AuthMiddleware(jwtManager *utils.JWTManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		// Extract token
		tokenString, err := utils.ExtractToken(authHeader)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header format"})
			c.Abort()
			return
		}

		// Validate token
		claims, err := jwtManager.ValidateToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		// Load user from database
		var user models.User
		if err := database.GetDB().Preload("OrgMemberships.Role.Permissions").First(&user, "id = ?", claims.UserID).Error; err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
			c.Abort()
			return
		}

		// Attach user to context
		c.Set("user_id", user.ID)
		c.Set("user", &user)
		c.Set("email", user.Email)

		c.Next()
	}
}

// OptionalAuthMiddleware is like AuthMiddleware but doesn't abort if no token
func OptionalAuthMiddleware(jwtManager *utils.JWTManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Next()
			return
		}

		tokenString, err := utils.ExtractToken(authHeader)
		if err != nil {
			c.Next()
			return
		}

		claims, err := jwtManager.ValidateToken(tokenString)
		if err != nil {
			c.Next()
			return
		}

		var user models.User
		if err := database.GetDB().Preload("OrgMemberships.Role.Permissions").First(&user, "id = ?", claims.UserID).Error; err != nil {
			c.Next()
			return
		}

		c.Set("user_id", user.ID)
		c.Set("user", &user)
		c.Set("email", user.Email)

		c.Next()
	}
}

// GetCurrentUser retrieves the current user from context
func GetCurrentUser(c *gin.Context) (*models.User, error) {
	userInterface, exists := c.Get("user")
	if !exists {
		return nil, http.ErrNoCookie
	}

	user, ok := userInterface.(*models.User)
	if !ok {
		return nil, http.ErrNoCookie
	}

	return user, nil
}

// GetCurrentUserID retrieves the current user ID from context
func GetCurrentUserID(c *gin.Context) (uuid.UUID, error) {
	userIDInterface, exists := c.Get("user_id")
	if !exists {
		return uuid.Nil, http.ErrNoCookie
	}

	userID, ok := userIDInterface.(uuid.UUID)
	if !ok {
		return uuid.Nil, http.ErrNoCookie
	}

	return userID, nil
}
