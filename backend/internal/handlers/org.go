package handlers

import (
	"net/http"
	"strings"

	"github.com/envo/backend/internal/middleware"
	"github.com/envo/backend/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// OrgHandler handles organization endpoints
type OrgHandler struct {
	orgService *services.OrgService
}

// NewOrgHandler creates a new organization handler
func NewOrgHandler(orgService *services.OrgService) *OrgHandler {
	return &OrgHandler{
		orgService: orgService,
	}
}

// CreateOrganization creates a new organization
// POST /api/v1/orgs
func (h *OrgHandler) CreateOrganization(c *gin.Context) {
	var req struct {
		Name string `json:"name" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request", "details": err.Error()})
		return
	}

	userID, err := middleware.GetCurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	org, err := h.orgService.CreateOrganization(userID, req.Name)
	if err != nil {
		if err.Error() == "organization limit reached for your tier" {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create organization", "details": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, org)
}

// GetOrganization retrieves an organization
// GET /api/v1/orgs/:id
func (h *OrgHandler) GetOrganization(c *gin.Context) {
	orgID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid organization ID"})
		return
	}

	userID, err := middleware.GetCurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	// Check access
	hasAccess, err := h.orgService.CheckUserAccess(userID, orgID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check access"})
		return
	}
	if !hasAccess {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	org, err := h.orgService.GetOrganization(orgID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Organization not found"})
		return
	}

	c.JSON(http.StatusOK, org)
}

// ListOrganizations lists user's organizations
// GET /api/v1/orgs
func (h *OrgHandler) ListOrganizations(c *gin.Context) {
	userID, err := middleware.GetCurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	orgs, err := h.orgService.ListUserOrganizations(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list organizations"})
		return
	}

	c.JSON(http.StatusOK, orgs)
}

// UpdateOrganization updates an organization
// PATCH /api/v1/orgs/:id
func (h *OrgHandler) UpdateOrganization(c *gin.Context) {
	orgID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid organization ID"})
		return
	}

	var req struct {
		Name string `json:"name" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	userID, err := middleware.GetCurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}
	hasAccess, err := h.orgService.CheckUserAccess(userID, orgID)
	if err != nil || !hasAccess {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	org, err := h.orgService.UpdateOrganization(orgID, req.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update organization"})
		return
	}

	c.JSON(http.StatusOK, org)
}

// DeleteOrganization deletes an organization
// DELETE /api/v1/orgs/:id
func (h *OrgHandler) DeleteOrganization(c *gin.Context) {
	orgID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid organization ID"})
		return
	}

	userID, err := middleware.GetCurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}
	hasAccess, err := h.orgService.CheckUserAccess(userID, orgID)
	if err != nil || !hasAccess {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	if err := h.orgService.DeleteOrganization(orgID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete organization"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Organization deleted successfully"})
}

// InviteMember invites a user to the organization
// POST /api/v1/orgs/:id/members
func (h *OrgHandler) InviteMember(c *gin.Context) {
	orgID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid organization ID"})
		return
	}

	var req struct {
		Email  string `json:"email" binding:"required,email"`
		Role   string `json:"role"`
		RoleID string `json:"role_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request", "details": err.Error()})
		return
	}

	userID, err := middleware.GetCurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}
	hasAccess, err := h.orgService.CheckUserAccess(userID, orgID)
	if err != nil || !hasAccess {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}
	var roleID *uuid.UUID
	if strings.TrimSpace(req.RoleID) != "" {
		parsed, parseErr := uuid.Parse(req.RoleID)
		if parseErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role_id"})
			return
		}
		roleID = &parsed
	}
	invitation, inviteURL, emailWarning, err := h.orgService.InviteMember(orgID, userID, req.Email, roleID, req.Role)
	if err != nil {
		if err.Error() == "member limit reached for this organization" {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"invitation": invitation,
		"invite_url": inviteURL,
		"warning":    emailWarning,
	})
}

// UpdateMemberRole updates a member's role
// PATCH /api/v1/orgs/:id/members/:memberId
func (h *OrgHandler) UpdateMemberRole(c *gin.Context) {
	memberID, err := uuid.Parse(c.Param("memberId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid member ID"})
		return
	}

	var req struct {
		Role   string `json:"role"`
		RoleID string `json:"role_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	orgID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid organization ID"})
		return
	}
	userID, err := middleware.GetCurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}
	hasAccess, err := h.orgService.CheckUserAccess(userID, orgID)
	if err != nil || !hasAccess {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	var roleID *uuid.UUID
	if strings.TrimSpace(req.RoleID) != "" {
		parsed, parseErr := uuid.Parse(req.RoleID)
		if parseErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role_id"})
			return
		}
		roleID = &parsed
	}

	member, err := h.orgService.UpdateMemberRole(memberID, roleID, req.Role)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, member)
}

// RemoveMember removes a member from the organization
// DELETE /api/v1/orgs/:id/members/:memberId
func (h *OrgHandler) RemoveMember(c *gin.Context) {
	orgID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid organization ID"})
		return
	}
	memberID, err := uuid.Parse(c.Param("memberId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid member ID"})
		return
	}

	userID, err := middleware.GetCurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}
	hasAccess, err := h.orgService.CheckUserAccess(userID, orgID)
	if err != nil || !hasAccess {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	if err := h.orgService.RemoveMember(memberID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove member"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Member removed successfully"})
}

// ListRoles returns all available roles for an org (system + custom).
// GET /api/v1/orgs/:id/roles
func (h *OrgHandler) ListRoles(c *gin.Context) {
	orgID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid organization ID"})
		return
	}
	roles, err := h.orgService.ListRoles(orgID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, roles)
}

// CreateRole creates a custom role in an organization.
// POST /api/v1/orgs/:id/roles
func (h *OrgHandler) CreateRole(c *gin.Context) {
	orgID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid organization ID"})
		return
	}
	var req struct {
		Name            string   `json:"name" binding:"required"`
		PermissionNames []string `json:"permission_names"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request", "details": err.Error()})
		return
	}
	role, err := h.orgService.CreateRole(orgID, req.Name, req.PermissionNames)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, role)
}

// UpdateRole updates custom role metadata and permissions.
// PATCH /api/v1/orgs/:id/roles/:roleId
func (h *OrgHandler) UpdateRole(c *gin.Context) {
	orgID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid organization ID"})
		return
	}
	roleID, err := uuid.Parse(c.Param("roleId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role ID"})
		return
	}
	var req struct {
		Name            string   `json:"name"`
		PermissionNames []string `json:"permission_names"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request", "details": err.Error()})
		return
	}
	role, err := h.orgService.UpdateRole(orgID, roleID, req.Name, req.PermissionNames)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, role)
}

// DeleteRole deletes a custom role, optionally reassigning members.
// DELETE /api/v1/orgs/:id/roles/:roleId
func (h *OrgHandler) DeleteRole(c *gin.Context) {
	orgID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid organization ID"})
		return
	}
	roleID, err := uuid.Parse(c.Param("roleId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role ID"})
		return
	}
	var req struct {
		ReplacementRoleID string `json:"replacement_role_id"`
	}
	_ = c.ShouldBindJSON(&req)
	var replacement *uuid.UUID
	if strings.TrimSpace(req.ReplacementRoleID) != "" {
		parsed, parseErr := uuid.Parse(req.ReplacementRoleID)
		if parseErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid replacement_role_id"})
			return
		}
		replacement = &parsed
	}
	if err := h.orgService.DeleteRole(orgID, roleID, replacement); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Role deleted"})
}

// ListInvitations lists invitations for an organization.
// GET /api/v1/orgs/:id/invites
func (h *OrgHandler) ListInvitations(c *gin.Context) {
	orgID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid organization ID"})
		return
	}
	invites, err := h.orgService.ListInvitations(orgID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, invites)
}

// ResendInvitation resends a pending invitation email.
// POST /api/v1/orgs/:id/invites/:inviteId/resend
func (h *OrgHandler) ResendInvitation(c *gin.Context) {
	orgID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid organization ID"})
		return
	}
	inviteID, err := uuid.Parse(c.Param("inviteId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid invitation ID"})
		return
	}
	userID, err := middleware.GetCurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}
	inviteURL, err := h.orgService.ResendInvitation(orgID, inviteID, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"invite_url": inviteURL})
}

// RevokeInvitation revokes a pending invitation.
// DELETE /api/v1/orgs/:id/invites/:inviteId
func (h *OrgHandler) RevokeInvitation(c *gin.Context) {
	orgID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid organization ID"})
		return
	}
	inviteID, err := uuid.Parse(c.Param("inviteId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid invitation ID"})
		return
	}
	if err := h.orgService.RevokeInvitation(orgID, inviteID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Invitation revoked"})
}

// AcceptInvitation accepts an invitation token for the signed-in user.
// POST /api/v1/invites/accept
func (h *OrgHandler) AcceptInvitation(c *gin.Context) {
	userID, err := middleware.GetCurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}
	var req struct {
		Token string `json:"token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "token is required"})
		return
	}
	member, err := h.orgService.AcceptInvitation(userID, req.Token)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, member)
}

// ListMyInvitations lists pending invitations for the signed-in user.
// GET /api/v1/invites/mine
func (h *OrgHandler) ListMyInvitations(c *gin.Context) {
	userID, err := middleware.GetCurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}
	invites, err := h.orgService.ListInvitationsForUser(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, invites)
}

// AcceptMyInvitation accepts a pending invite by id for signed-in user.
// POST /api/v1/invites/:inviteId/accept
func (h *OrgHandler) AcceptMyInvitation(c *gin.Context) {
	userID, err := middleware.GetCurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}
	inviteID, err := uuid.Parse(c.Param("inviteId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid invite id"})
		return
	}
	member, err := h.orgService.AcceptInvitationByID(userID, inviteID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, member)
}
