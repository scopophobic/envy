package handlers

import (
	"context"
	"net/http"

	"github.com/envo/backend/internal/database"
	"github.com/envo/backend/internal/middleware"
	"github.com/envo/backend/internal/models"
	"github.com/envo/backend/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// SecretHandler handles secret endpoints
type SecretHandler struct {
	secretService *services.SecretService
}

// NewSecretHandler creates a new secret handler
func NewSecretHandler(secretService *services.SecretService) *SecretHandler {
	return &SecretHandler{
		secretService: secretService,
	}
}

// userHasAccessToEnv ensures the user belongs to the org that owns the environment
func userHasAccessToEnv(user *models.User, envID uuid.UUID) (uuid.UUID, bool) {
	db := database.GetDB()
	var env models.Environment
	if err := db.Preload("Project").First(&env, envID).Error; err != nil {
		return uuid.Nil, false
	}
	if userHasAccessToOrg(user, env.Project.OrgID) {
		return env.Project.OrgID, true
	}
	return uuid.Nil, false
}

// CreateSecret creates a new secret
// POST /api/v1/environments/:envId/secrets
func (h *SecretHandler) CreateSecret(c *gin.Context) {
	envID, err := uuid.Parse(c.Param("envId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid environment ID"})
		return
	}

	user, err := middleware.GetCurrentUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	if _, ok := userHasAccessToEnv(user, envID); !ok {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	var req struct {
		Key   string `json:"key" binding:"required"`
		Value string `json:"value" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request", "details": err.Error()})
		return
	}

	ip := c.ClientIP()
	secret, err := h.secretService.CreateSecret(c.Request.Context(), user.ID, envID, req.Key, req.Value, ip)
	if err != nil {
		if err.Error() == "secret limit reached for this environment" {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create secret", "details": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, secret)
}

// ListSecrets lists secrets (metadata) for an environment
// GET /api/v1/environments/:envId/secrets
func (h *SecretHandler) ListSecrets(c *gin.Context) {
	envID, err := uuid.Parse(c.Param("envId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid environment ID"})
		return
	}

	user, err := middleware.GetCurrentUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	if _, ok := userHasAccessToEnv(user, envID); !ok {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	secrets, err := h.secretService.ListSecrets(c.Request.Context(), envID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list secrets"})
		return
	}

	c.JSON(http.StatusOK, secrets)
}

// UpdateSecret updates a secret
// PATCH /api/v1/secrets/:id
func (h *SecretHandler) UpdateSecret(c *gin.Context) {
	secretID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid secret ID"})
		return
	}

	user, err := middleware.GetCurrentUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	// Ensure user has access to the environment via the secret
	db := database.GetDB()
	var secret models.Secret
	if err := db.First(&secret, secretID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Secret not found"})
		return
	}

	if _, ok := userHasAccessToEnv(user, secret.EnvironmentID); !ok {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	var req struct {
		Key   *string `json:"key"`
		Value *string `json:"value"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	if req.Key == nil && req.Value == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "At least one of key or value must be provided"})
		return
	}

	ip := c.ClientIP()
	updated, err := h.secretService.UpdateSecret(c.Request.Context(), user.ID, secretID, req.Key, req.Value, ip)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update secret", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, updated)
}

// DeleteSecret deletes a secret
// DELETE /api/v1/secrets/:id
func (h *SecretHandler) DeleteSecret(c *gin.Context) {
	secretID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid secret ID"})
		return
	}

	user, err := middleware.GetCurrentUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	// Ensure user has access to the environment via the secret
	db := database.GetDB()
	var secret models.Secret
	if err := db.First(&secret, secretID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Secret not found"})
		return
	}

	if _, ok := userHasAccessToEnv(user, secret.EnvironmentID); !ok {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	ip := c.ClientIP()
	if err := h.secretService.DeleteSecret(c.Request.Context(), user.ID, secretID, ip); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete secret"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Secret deleted successfully"})
}

// ExportEnvironmentSecrets exports decrypted secrets for CLI
// GET /api/v1/environments/:envId/secrets/export
func (h *SecretHandler) ExportEnvironmentSecrets(c *gin.Context) {
	envID, err := uuid.Parse(c.Param("envId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid environment ID"})
		return
	}

	user, err := middleware.GetCurrentUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	if _, ok := userHasAccessToEnv(user, envID); !ok {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	ip := c.ClientIP()
	secrets, orgID, err := h.secretService.ExportEnvironmentSecrets(context.Background(), user.ID, envID, ip)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to export secrets", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"org_id":        orgID,
		"environment_id": envID,
		"secrets":       secrets,
	})
}

