package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// RequireSuperAdmin guards platform-level administration APIs.
func RequireSuperAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		user, err := GetCurrentUser(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
			c.Abort()
			return
		}
		if !user.IsSuperAdmin {
			c.JSON(http.StatusForbidden, gin.H{"error": "Super admin access required"})
			c.Abort()
			return
		}
		c.Next()
	}
}

