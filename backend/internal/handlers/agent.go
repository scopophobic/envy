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
	c.Header("Cache-Control", "no-store")
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
	c.Header("Cache-Control", "no-store")
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
		ApprovalMode    string     `json:"approval_mode"`
		MaxLeaseSeconds int        `json:"max_lease_seconds"`
		ExpiresAt       *time.Time `json:"expires_at"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "A valid environment and access policy are required"})
		return
	}
	grant, err := h.agents.CreateGrant(c.Request.Context(), userID, orgID, agentID, req.EnvironmentID, req.AllowedKeys, req.AllowAllSecrets, req.ApprovalMode, req.MaxLeaseSeconds, req.ExpiresAt, c.ClientIP())
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
		Project         string     `json:"project" binding:"required"`
		Environment     string     `json:"environment" binding:"required"`
		Keys            []string   `json:"keys"`
		Purpose         string     `json:"purpose"`
		SessionID       string     `json:"session_id"`
		AccessRequestID *uuid.UUID `json:"access_request_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil && req.AccessRequestID == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Project and environment are required"})
		return
	}
	if len(req.Purpose) > 200 || len(req.SessionID) > 200 || len(req.Keys) > 500 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Agent request metadata or key list is too large"})
		return
	}
	if req.AccessRequestID == nil && (strings.TrimSpace(req.Purpose) == "" || strings.TrimSpace(req.SessionID) == "") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "purpose and session_id are required for a new access request"})
		return
	}
	var accessRequest *models.AgentAccessRequest
	if req.AccessRequestID == nil {
		access, err := h.agents.AuthorizeResolve(c.Request.Context(), agent, req.Project, req.Environment, req.Keys)
		if errors.Is(err, services.ErrAgentForbidden) {
			if h.audit != nil {
				meta, _ := json.Marshal(gin.H{"project": req.Project, "environment": req.Environment, "keys": req.Keys, "reason": "policy denied"})
				_ = h.audit.LogAgent(c.Request.Context(), agent.ID, agent.OrgID, agent.ID, models.ActionAgentAccessDenied, "agent", c.ClientIP(), datatypes.JSON(meta))
			}
			c.JSON(http.StatusForbidden, gin.H{"error": "Agent is not authorized for the requested project, environment, or secret keys"})
			return
		}
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		accessRequest, err = h.agents.CreateAccessRequest(c.Request.Context(), agent, credential, access, req.Purpose, req.SessionID, c.ClientIP())
		if err != nil {
			respondInternalError(c, "Failed to persist access request and audit event", err)
			return
		}
	} else {
		var err error
		accessRequest, err = h.agents.GetAccessRequest(c.Request.Context(), agent.ID, credential.ID, *req.AccessRequestID)
		if err != nil {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access request does not belong to this agent credential"})
			return
		}
	}
	if accessRequest.Status == models.AccessRequestPending {
		c.JSON(http.StatusAccepted, gin.H{"status": accessRequest.Status, "access_request_id": accessRequest.ID, "expires_at": accessRequest.ExpiresAt, "message": "Human approval required"})
		return
	}
	if accessRequest.Status != models.AccessRequestApproved {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access request is not approved", "status": accessRequest.Status, "access_request_id": accessRequest.ID})
		return
	}
	accessRequest, err := h.agents.BeginDelivery(c.Request.Context(), agent.ID, credential.ID, accessRequest.ID)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Access approval expired, was revoked, or was already consumed"})
		return
	}
	var approvedKeys []string
	if err := json.Unmarshal(accessRequest.RequestedKeys, &approvedKeys); err != nil {
		h.agents.FailDelivery(c.Request.Context(), accessRequest, "invalid approved key policy", c.ClientIP())
		respondInternalError(c, "Invalid approved access policy", err)
		return
	}
	allowedKeys := make(map[string]struct{}, len(approvedKeys))
	for _, key := range approvedKeys {
		allowedKeys[key] = struct{}{}
	}
	secrets, _, err := h.secrets.DecryptEnvironmentSecrets(c.Request.Context(), accessRequest.EnvironmentID, allowedKeys, accessRequest.AllowAllSecrets)
	if err != nil {
		h.agents.FailDelivery(c.Request.Context(), accessRequest, err.Error(), c.ClientIP())
		respondInternalError(c, "Failed to resolve secrets", err)
		return
	}
	if err := h.agents.CompleteDelivery(c.Request.Context(), accessRequest, len(secrets), c.ClientIP()); err != nil {
		h.agents.FailDelivery(c.Request.Context(), accessRequest, err.Error(), c.ClientIP())
		respondInternalError(c, "Failed to commit the access audit; secrets were not released", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.Header("Pragma", "no-cache")
	c.JSON(http.StatusOK, gin.H{
		"agent_id":                 agent.ID,
		"environment_id":           accessRequest.EnvironmentID,
		"access_request_id":        accessRequest.ID,
		"lease_id":                 accessRequest.ID,
		"expires_at":               accessRequest.ExpiresAt,
		"delivery_mode":            "static_secret_one_time",
		"revocable_after_delivery": false,
		"secrets":                  secrets,
	})
}

func (h *AgentHandler) ListAccessRequests(c *gin.Context) {
	orgID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid organization ID"})
		return
	}
	requests, err := h.agents.ListAccessRequests(c.Request.Context(), orgID, strings.TrimSpace(c.Query("status")), 100)
	if err != nil {
		respondInternalError(c, "Failed to list access requests", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, requests)
}

func (h *AgentHandler) DecideAccessRequest(c *gin.Context) {
	orgID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid organization ID"})
		return
	}
	requestID, err := uuid.Parse(c.Param("requestId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid access request ID"})
		return
	}
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	var req struct {
		Decision string `json:"decision" binding:"required"`
		Reason   string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || (req.Decision != "approve" && req.Decision != "deny" && req.Decision != "revoke") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "decision must be approve, deny, or revoke"})
		return
	}
	if len(strings.TrimSpace(req.Reason)) > 500 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "decision reason cannot exceed 500 characters"})
		return
	}
	var result *models.AgentAccessRequest
	if req.Decision == "revoke" {
		result, err = h.agents.RevokeAccessRequest(c.Request.Context(), userID, orgID, requestID, req.Reason, c.ClientIP())
	} else {
		result, err = h.agents.DecideAccessRequest(c.Request.Context(), userID, orgID, requestID, req.Decision == "approve", req.Reason, c.ClientIP())
	}
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *AgentHandler) SetOrgAgentAccess(c *gin.Context) {
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
		Paused *bool `json:"paused" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Paused == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "paused is required"})
		return
	}
	if err := h.agents.SetOrgAgentAccessPaused(c.Request.Context(), userID, orgID, *req.Paused, c.ClientIP()); err != nil {
		respondInternalError(c, "Failed to update organization agent access", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"agent_access_paused": *req.Paused})
}
