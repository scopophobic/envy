package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/envo/backend/internal/middleware"
	"github.com/envo/backend/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type PlatformHandler struct {
	platformService *services.PlatformService
}

func NewPlatformHandler(platformService *services.PlatformService) *PlatformHandler {
	return &PlatformHandler{platformService: platformService}
}

func (h *PlatformHandler) ListConnections(c *gin.Context) {
	userID, err := middleware.GetCurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	rows, err := h.platformService.ListConnections(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list platform connections"})
		return
	}

	out := make([]gin.H, 0, len(rows))
	for _, row := range rows {
		meta := map[string]any{}
		if len(row.Metadata) > 0 {
			_ = json.Unmarshal(row.Metadata, &meta)
		}
		out = append(out, gin.H{
			"id":           row.ID,
			"platform":     row.Platform,
			"name":         row.Name,
			"token_prefix": row.TokenPrefix,
			"metadata":     meta,
			"created_at":   row.CreatedAt,
			"updated_at":   row.UpdatedAt,
		})
	}
	c.JSON(http.StatusOK, out)
}

func (h *PlatformHandler) CreateConnection(c *gin.Context) {
	userID, err := middleware.GetCurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	var req struct {
		Platform string         `json:"platform" binding:"required"`
		Name     string         `json:"name"`
		Token    string         `json:"token" binding:"required"`
		Metadata map[string]any `json:"metadata"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request", "details": err.Error()})
		return
	}

	conn, err := h.platformService.CreateConnection(c.Request.Context(), userID, services.CreatePlatformConnectionInput{
		Platform: req.Platform,
		Name:     req.Name,
		Token:    req.Token,
		Metadata: req.Metadata,
	})
	if err != nil {
		status := http.StatusInternalServerError
		msg := strings.ToLower(err.Error())
		if strings.Contains(msg, "unsupported platform") || strings.Contains(msg, "required") || strings.Contains(msg, "validation failed") {
			status = http.StatusBadRequest
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":           conn.ID,
		"platform":     conn.Platform,
		"name":         conn.Name,
		"token_prefix": conn.TokenPrefix,
		"metadata":     req.Metadata,
		"created_at":   conn.CreatedAt,
		"updated_at":   conn.UpdatedAt,
	})
}

func (h *PlatformHandler) DeleteConnection(c *gin.Context) {
	userID, err := middleware.GetCurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid connection ID"})
		return
	}
	if err := h.platformService.DeleteConnection(c.Request.Context(), userID, id); err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete connection"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Platform connection removed"})
}

func (h *PlatformHandler) SyncEnvironment(c *gin.Context) {
	user, err := middleware.GetCurrentUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	envID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid environment ID"})
		return
	}
	if !userHasAccessToEnvOrg(user, envID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	var req struct {
		PlatformConnectionID string `json:"platform_connection_id" binding:"required"`
		TargetProjectID      string `json:"target_project_id" binding:"required"`
		TargetEnvironment    string `json:"target_environment" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request", "details": err.Error()})
		return
	}

	connID, err := uuid.Parse(req.PlatformConnectionID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid platform_connection_id"})
		return
	}

	res, err := h.platformService.SyncEnvironment(c.Request.Context(), user.ID, services.SyncEnvironmentInput{
		EnvironmentID: envID,
		ConnectionID:  connID,
		TargetProject: req.TargetProjectID,
		TargetEnv:     req.TargetEnvironment,
	}, c.ClientIP())
	if err != nil {
		msg := strings.ToLower(err.Error())
		code := http.StatusInternalServerError
		if strings.Contains(msg, "unsupported platform") || strings.Contains(msg, "required") || strings.Contains(msg, "not found") {
			code = http.StatusBadRequest
		}
		c.JSON(code, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, res)
}
