package middleware

import (
	"net/http"

	"github.com/envo/backend/internal/models"
	"github.com/gin-gonic/gin"
)

// RequirePermission checks if the user has the required permission
func RequirePermission(permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, err := GetCurrentUser(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
			c.Abort()
			return
		}

		if !HasPermission(user, permission) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireAnyPermission checks if the user has at least one of the required permissions
func RequireAnyPermission(permissions ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, err := GetCurrentUser(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
			c.Abort()
			return
		}

		for _, permission := range permissions {
			if HasPermission(user, permission) {
				c.Next()
				return
			}
		}

		c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
		c.Abort()
	}
}

// HasPermission checks if a user has a specific permission
func HasPermission(user *models.User, permissionName string) bool {
	// Load user's org memberships with roles and permissions
	for _, membership := range user.OrgMemberships {
		for _, permission := range membership.Role.Permissions {
			if permission.Name == permissionName {
				return true
			}
		}
	}
	return false
}

// GetUserPermissions returns all permissions for a user across all orgs
func GetUserPermissions(user *models.User) []string {
	permissionSet := make(map[string]bool)
	
	for _, membership := range user.OrgMemberships {
		for _, permission := range membership.Role.Permissions {
			permissionSet[permission.Name] = true
		}
	}

	permissions := make([]string, 0, len(permissionSet))
	for permission := range permissionSet {
		permissions = append(permissions, permission)
	}

	return permissions
}
