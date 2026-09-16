package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/envo/backend/internal/middleware"
	"github.com/envo/backend/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// AuditHandler handles audit log endpoints
type AuditHandler struct {
	auditService *services.AuditService
}

// NewAuditHandler creates a new audit handler
func NewAuditHandler(auditService *services.AuditService) *AuditHandler {
	return &AuditHandler{
		auditService: auditService,
	}
}

// ListOrgAuditLogs lists recent audit logs for an organization
// GET /api/v1/orgs/:orgId/audit-logs
func (h *AuditHandler) ListOrgAuditLogs(c *gin.Context) {
	orgID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid organization ID"})
		return
	}

	_, err = middleware.GetCurrentUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	var before *time.Time
	if raw := strings.TrimSpace(c.Query("before")); raw != "" {
		parsed, parseErr := time.Parse(time.RFC3339, raw)
		if parseErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "before must be RFC3339"})
			return
		}
		before = &parsed
	}
	logs, err := h.auditService.QueryOrgLogs(c.Request.Context(), orgID, services.AuditQuery{Limit: limit, ActorType: c.Query("actor_type"), Action: c.Query("action"), ResourceType: c.Query("resource_type"), Before: before})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list audit logs"})
		return
	}

	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, logs)
}
