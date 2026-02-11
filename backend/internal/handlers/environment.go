package handlers

import (
	"net/http"

	"github.com/envo/backend/internal/database"
	"github.com/envo/backend/internal/middleware"
	"github.com/envo/backend/internal/models"
	"github.com/envo/backend/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// EnvironmentHandler handles environment endpoints
type EnvironmentHandler struct {
	envService     *services.EnvironmentService
	projectService *services.ProjectService
}

// NewEnvironmentHandler creates a new environment handler
func NewEnvironmentHandler(envService *services.EnvironmentService, projectService *services.ProjectService) *EnvironmentHandler {
	return &EnvironmentHandler{
		envService:     envService,
		projectService: projectService,
	}
}

// userHasAccessToProjectOrg ensures the user belongs to the org that owns the project
func userHasAccessToProjectOrg(user *models.User, projectID uuid.UUID) bool {
	db := database.GetDB()
	var project models.Project
	if err := db.First(&project, projectID).Error; err != nil {
		return false
	}
	return userHasAccessToOrg(user, project.OrgID)
}

// userHasAccessToEnvOrg ensures the user belongs to the org that owns the environment
func userHasAccessToEnvOrg(user *models.User, envID uuid.UUID) bool {
	db := database.GetDB()
	var env models.Environment
	if err := db.Preload("Project").First(&env, envID).Error; err != nil {
		return false
	}
	return userHasAccessToOrg(user, env.Project.OrgID)
}

// CreateEnvironment creates a new environment for a project
// POST /api/v1/projects/:projectId/environments
func (h *EnvironmentHandler) CreateEnvironment(c *gin.Context) {
	projectID, err := uuid.Parse(c.Param("projectId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}

	user, err := middleware.GetCurrentUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	if !userHasAccessToProjectOrg(user, projectID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	var req struct {
		Name string `json:"name" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request", "details": err.Error()})
		return
	}

	env, err := h.envService.CreateEnvironment(projectID, req.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create environment"})
		return
	}

	c.JSON(http.StatusCreated, env)
}

// ListProjectEnvironments lists environments for a project
// GET /api/v1/projects/:projectId/environments
func (h *EnvironmentHandler) ListProjectEnvironments(c *gin.Context) {
	projectID, err := uuid.Parse(c.Param("projectId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}

	user, err := middleware.GetCurrentUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	if !userHasAccessToProjectOrg(user, projectID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	envs, err := h.envService.ListProjectEnvironments(projectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list environments"})
		return
	}

	c.JSON(http.StatusOK, envs)
}

// UpdateEnvironment updates an environment
// PATCH /api/v1/environments/:id
func (h *EnvironmentHandler) UpdateEnvironment(c *gin.Context) {
	envID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid environment ID"})
		return
	}

	user, err := middleware.GetCurrentUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	if !userHasAccessToEnvOrg(user, envID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	var req struct {
		Name string `json:"name" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	updated, err := h.envService.UpdateEnvironment(envID, req.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update environment"})
		return
	}

	c.JSON(http.StatusOK, updated)
}

// DeleteEnvironment deletes an environment
// DELETE /api/v1/environments/:id
func (h *EnvironmentHandler) DeleteEnvironment(c *gin.Context) {
	envID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid environment ID"})
		return
	}

	user, err := middleware.GetCurrentUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	if !userHasAccessToEnvOrg(user, envID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	if err := h.envService.DeleteEnvironment(envID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete environment"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Environment deleted successfully"})
}

