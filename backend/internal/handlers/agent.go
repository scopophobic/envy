package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/envo/backend/internal/middleware"
	"github.com/envo/backend/internal/models"
	"github.com/envo/backend/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type AgentHandler struct {
	agents  *services.AgentService
	secrets *services.SecretService
	audit   *services.AuditService
}

func NewAgentHandler(agents *services.AgentService, secrets *services.SecretService, audit *services.AuditService) *AgentHandler {
	return &AgentHandler{agents: agents, secrets: secrets, audit: audit}
}

func agentRouteIDs(c *gin.Context) (uuid.UUID, uuid.UUID, bool) {
	orgID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid organization ID"})
		return uuid.Nil, uuid.Nil, false
	}
	agentID, err := uuid.Parse(c.Param("agentId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid agent ID"})
		return uuid.Nil, uuid.Nil, false
	}
	return orgID, agentID, true
}

func currentUserID(c *gin.Context) (uuid.UUID, bool) {
	userID, err := middleware.GetCurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return uuid.Nil, false
	}
	return userID, true
}

func (h *AgentHandler) List(c *gin.Context) {
	orgID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid organization ID"})
		return
	}
	agents, err := h.agents.ListAgents(c.Request.Context(), orgID)
	if err != nil {
		respondInternalError(c, "Failed to list agents", err)
		return
	}
	c.JSON(http.StatusOK, agents)
}

func (h *AgentHandler) Create(c *gin.Context) {
	orgID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid organization ID"})
		return
	}
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Agent name is required"})
		return
	}
	agent, err := h.agents.CreateAgent(c.Request.Context(), userID, orgID, req.Name, req.Description, c.ClientIP())
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, agent)
}

func (h *AgentHandler) Update(c *gin.Context) {
	orgID, agentID, ok := agentRouteIDs(c)
	if !ok {
		return
	}
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Status is required"})
		return
	}
	agent, err := h.agents.UpdateAgentStatus(c.Request.Context(), userID, orgID, agentID, req.Status, c.ClientIP())
	if errors.Is(err, services.ErrAgentNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, agent)
}

func (h *AgentHandler) ListCredentials(c *gin.Context) {
	orgID, agentID, ok := agentRouteIDs(c)
	if !ok {
		return
	}
	credentials, err := h.agents.ListCredentials(c.Request.Context(), orgID, agentID)
	if err != nil {
		respondInternalError(c, "Failed to list agent credentials", err)
		return
	}
	c.JSON(http.StatusOK, credentials)
}

func (h *AgentHandler) CreateCredential(c *gin.Context) {
	orgID, agentID, ok := agentRouteIDs(c)
	if !ok {
		return
	}
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	var req struct {
		Name      string     `json:"name" binding:"required"`
		ExpiresAt *time.Time `json:"expires_at"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Credential name is required"})
		return
	}
	credential, raw, err := h.agents.CreateCredential(c.Request.Context(), userID, orgID, agentID, req.Name, req.ExpiresAt, c.ClientIP())
	if errors.Is(err, services.ErrAgentNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusCreated, gin.H{"credential": credential, "token": raw, "warning": "Copy this token now. Envo cannot show it again."})
}

func (h *AgentHandler) RevokeCredential(c *gin.Context) {
	orgID, agentID, ok := agentRouteIDs(c)
	if !ok {
		return
	}
	credentialID, err := uuid.Parse(c.Param("credentialId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid credential ID"})
		return
	}
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	if err := h.agents.RevokeCredential(c.Request.Context(), userID, orgID, agentID, credentialID, c.ClientIP()); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Credential not found or already revoked"})
			return
		}
		respondInternalError(c, "Failed to revoke credential", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *AgentHandler) ListGrants(c *gin.Context) {
	orgID, agentID, ok := agentRouteIDs(c)
	if !ok {
		return
	}
	grants, err := h.agents.ListGrants(c.Request.Context(), orgID, agentID)
	if err != nil {
		respondInternalError(c, "Failed to list agent grants", err)
		return
	}
	c.JSON(http.StatusOK, grants)
}

func (h *AgentHandler) CreateGrant(c *gin.Context) {
	orgID, agentID, ok := agentRouteIDs(c)
	if !ok {
		return
	}
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	var req struct {
		EnvironmentID   uuid.UUID  `json:"environment_id" binding:"required"`
		AllowedKeys     []string   `json:"allowed_keys"`
		AllowAllSecrets bool       `json:"allow_all_secrets"`
		ExpiresAt       *time.Time `json:"expires_at"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "A valid environment and access policy are required"})
		return
	}
	grant, err := h.agents.CreateGrant(c.Request.Context(), userID, orgID, agentID, req.EnvironmentID, req.AllowedKeys, req.AllowAllSecrets, req.ExpiresAt, c.ClientIP())
	if errors.Is(err, services.ErrAgentNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, grant)
}

func (h *AgentHandler) RevokeGrant(c *gin.Context) {
	orgID, agentID, ok := agentRouteIDs(c)
	if !ok {
		return
	}
	grantID, err := uuid.Parse(c.Param("grantId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid grant ID"})
		return
	}
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	if err := h.agents.RevokeGrant(c.Request.Context(), userID, orgID, agentID, grantID, c.ClientIP()); err != nil {
		if errors.Is(err, services.ErrGrantNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Grant not found or already revoked"})
			return
		}
		respondInternalError(c, "Failed to revoke grant", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *AgentHandler) Me(c *gin.Context) {
	agent, credential, ok := middleware.GetCurrentAgent(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Agent authorization required"})
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{"agent": agent, "credential_id": credential.ID, "credential_name": credential.Name})
}

func (h *AgentHandler) ResolveSecrets(c *gin.Context) {
	agent, credential, ok := middleware.GetCurrentAgent(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Agent authorization required"})
		return
	}
	// Keep the broker request deliberately small; it should contain selectors
	// and key names, never arbitrary prompts or source code.
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 64<<10)
	var req struct {
		Project     string   `json:"project" binding:"required"`
		Environment string   `json:"environment" binding:"required"`
		Keys        []string `json:"keys"`
		Purpose     string   `json:"purpose"`
		SessionID   string   `json:"session_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Project and environment are required"})
		return
	}
	if len(req.Purpose) > 200 || len(req.SessionID) > 200 || len(req.Keys) > 500 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Agent request metadata or key list is too large"})
		return
	}
	access, err := h.agents.AuthorizeResolve(c.Request.Context(), agent, req.Project, req.Environment, req.Keys)
	if errors.Is(err, services.ErrAgentForbidden) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Agent is not authorized for the requested project, environment, or secret keys"})
		return
	}
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	secrets, orgID, err := h.secrets.DecryptEnvironmentSecrets(c.Request.Context(), access.Environment, access.AllowedKeys, access.AllowAll)
	if err != nil {
		respondInternalError(c, "Failed to resolve secrets", err)
		return
	}
	leaseID := uuid.New()
	expiresAt := time.Now().UTC().Add(5 * time.Minute)
	if access.ExpiresAt != nil && access.ExpiresAt.Before(expiresAt) {
		expiresAt = *access.ExpiresAt
	}
	metadataBytes, _ := json.Marshal(gin.H{
		"credential_id": credential.ID,
		"grant_ids":     access.GrantIDs,
		"lease_id":      leaseID,
		"secret_count":  len(secrets),
		"purpose":       strings.TrimSpace(req.Purpose),
		"session_id":    strings.TrimSpace(req.SessionID),
	})
	if h.audit != nil {
		_ = h.audit.LogAgent(c.Request.Context(), agent.ID, orgID, access.Environment, models.ActionSecretRead, "environment", c.ClientIP(), datatypes.JSON(metadataBytes))
	}
	c.Header("Cache-Control", "no-store")
	c.Header("Pragma", "no-cache")
	c.JSON(http.StatusOK, gin.H{
		"agent_id":       agent.ID,
		"environment_id": access.Environment,
		"lease_id":       leaseID,
		"expires_at":     expiresAt,
		"secrets":        secrets,
	})
}
