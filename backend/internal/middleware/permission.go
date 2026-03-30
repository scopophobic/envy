package middleware

import (
	"net/http"

	"github.com/envo/backend/internal/database"
	"github.com/envo/backend/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// RequirePermission checks if the user has the required permission.
// For personal workspaces, the owner is granted all permissions automatically.
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

// CheckWorkspaceAccess verifies a user has access to a workspace (org).
// Personal workspaces short-circuit: only the owner can access, with full permissions.
func CheckWorkspaceAccess(user *models.User, orgID uuid.UUID) (bool, bool) {
	db := database.GetDB()
	var org models.Organization
	if err := db.First(&org, orgID).Error; err != nil {
		return false, false
	}

	if org.IsPersonal() {
		return org.OwnerID == user.ID, org.OwnerID == user.ID
	}

	// Org workspace: check membership
	for _, m := range user.OrgMemberships {
		if m.OrgID == orgID {
			return true, false
		}
	}
	return false, false
}

// RejectIfPersonalWorkspace is middleware for routes that should be blocked on personal workspaces
// (e.g. invite, member management). Expects :id param to be the org/workspace ID.
func RejectIfPersonalWorkspace() gin.HandlerFunc {
	return func(c *gin.Context) {
		orgIDStr := c.Param("id")
		orgID, err := uuid.Parse(orgIDStr)
		if err != nil {
			c.Next()
			return
		}

		db := database.GetDB()
		var org models.Organization
		if err := db.First(&org, orgID).Error; err != nil {
			c.Next()
			return
		}

		if org.IsPersonal() {
			c.JSON(http.StatusForbidden, gin.H{"error": "This action is not available for personal workspaces"})
			c.Abort()
			return
		}

		c.Next()
	}
}
