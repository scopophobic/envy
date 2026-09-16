package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/envo/backend/internal/database"
	"github.com/envo/backend/internal/models"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrAgentUnauthorized = errors.New("invalid or expired agent token")
	ErrAgentForbidden    = errors.New("agent is not authorized for the requested secrets")
	ErrAgentNotFound     = errors.New("agent not found")
	ErrGrantNotFound     = errors.New("agent grant not found")
)

const agentTokenPrefix = "envo_agent_"

type AgentService struct {
	audit              *AuditService
	usageWriteInterval time.Duration
}

func NewAgentService(audit *AuditService, usageWriteInterval time.Duration) *AgentService {
	if usageWriteInterval <= 0 {
		usageWriteInterval = time.Minute
	}
	return &AgentService{audit: audit, usageWriteInterval: usageWriteInterval}
}

func (s *AgentService) ListAgents(ctx context.Context, orgID uuid.UUID) ([]models.AgentIdentity, error) {
	var agents []models.AgentIdentity
	err := database.GetDB().WithContext(ctx).Where("org_id = ?", orgID).Order("created_at DESC").Find(&agents).Error
	return agents, err
}

func (s *AgentService) CreateAgent(ctx context.Context, userID, orgID uuid.UUID, name, description, ip string) (*models.AgentIdentity, error) {
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 120 {
		return nil, fmt.Errorf("agent name must be between 1 and 120 characters")
	}
	agent := &models.AgentIdentity{OrgID: orgID, Name: name, Description: strings.TrimSpace(description), CreatedBy: userID}
	if err := database.GetDB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(agent).Error; err != nil {
			return err
		}
		return tx.Create(&models.AuditLog{UserID: &userID, ActorType: models.AuditActorHuman, OrgID: orgID, Action: models.ActionAgentCreate, ResourceType: "agent", ResourceID: agent.ID, IPAddress: ip}).Error
	}); err != nil {
		return nil, err
	}
	return agent, nil
}

func (s *AgentService) UpdateAgentStatus(ctx context.Context, userID, orgID, agentID uuid.UUID, status, ip string) (*models.AgentIdentity, error) {
	if status != models.AgentStatusActive && status != models.AgentStatusSuspended && status != models.AgentStatusRevoked {
		return nil, fmt.Errorf("status must be active, suspended, or revoked")
	}
	db := database.GetDB().WithContext(ctx)
	var agent models.AgentIdentity
	err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("id = ? AND org_id = ?", agentID, orgID).First(&agent).Error; err != nil {
			return err
		}
		if err := tx.Model(&agent).Update("status", status).Error; err != nil {
			return err
		}
		if status != models.AgentStatusActive {
			if err := tx.Model(&models.AgentAccessRequest{}).Where("agent_id = ? AND status IN ?", agentID, []string{models.AccessRequestPending, models.AccessRequestApproved, models.AccessRequestDelivering}).Updates(map[string]any{"status": models.AccessRequestRevoked, "decision_reason": "agent disabled"}).Error; err != nil {
				return err
			}
		}
		if status == models.AgentStatusRevoked {
			now := time.Now().UTC()
			if err := tx.Model(&models.AgentCredential{}).Where("agent_id = ? AND revoked_at IS NULL", agentID).Update("revoked_at", now).Error; err != nil {
				return err
			}
			if err := tx.Model(&models.AgentGrant{}).Where("agent_id = ? AND revoked_at IS NULL", agentID).Update("revoked_at", now).Error; err != nil {
				return err
			}
		}
		metadata, _ := json.Marshal(map[string]string{"status": status})
		return tx.Create(&models.AuditLog{UserID: &userID, ActorType: models.AuditActorHuman, OrgID: orgID, Action: models.ActionAgentUpdate, ResourceType: "agent", ResourceID: agentID, Metadata: datatypes.JSON(metadata), IPAddress: ip}).Error
	})
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrAgentNotFound
	}
	if err != nil {
		return nil, err
	}
	agent.Status = status
	return &agent, nil
}

// GenerateAgentToken returns a credential identifier, raw token, token hash,
// and safe display prefix. The raw token must only be returned once.
func GenerateAgentToken() (uuid.UUID, string, string, string, error) {
	id := uuid.New()
	random := make([]byte, 32)
	if _, err := rand.Read(random); err != nil {
		return uuid.Nil, "", "", "", err
	}
	raw := agentTokenPrefix + id.String() + "_" + base64.RawURLEncoding.EncodeToString(random)
	hash := sha256.Sum256([]byte(raw))
	display := raw
	if len(display) > 24 {
		display = display[:24]
	}
	return id, raw, hex.EncodeToString(hash[:]), display, nil
}

func credentialIDFromToken(raw string) (uuid.UUID, error) {
	if !strings.HasPrefix(raw, agentTokenPrefix) {
		return uuid.Nil, ErrAgentUnauthorized
	}
	parts := strings.SplitN(strings.TrimPrefix(raw, agentTokenPrefix), "_", 2)
	if len(parts) != 2 || len(parts[1]) < 32 {
		return uuid.Nil, ErrAgentUnauthorized
	}
	id, err := uuid.Parse(parts[0])
	if err != nil {
		return uuid.Nil, ErrAgentUnauthorized
	}
	return id, nil
}

func (s *AgentService) CreateCredential(ctx context.Context, userID, orgID, agentID uuid.UUID, name string, expiresAt *time.Time, ip string) (*models.AgentCredential, string, error) {
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 120 {
		return nil, "", fmt.Errorf("credential name must be between 1 and 120 characters")
	}
	if expiresAt == nil {
		defaultExpiry := time.Now().UTC().Add(24 * time.Hour)
		expiresAt = &defaultExpiry
	}
	if !expiresAt.After(time.Now()) {
		return nil, "", fmt.Errorf("credential expiry must be in the future")
	}
	if expiresAt.After(time.Now().UTC().Add(90 * 24 * time.Hour)) {
		return nil, "", fmt.Errorf("credential expiry cannot exceed 90 days")
	}
	db := database.GetDB().WithContext(ctx)
	var count int64
	if err := db.Model(&models.AgentIdentity{}).Where("id = ? AND org_id = ? AND status <> ?", agentID, orgID, models.AgentStatusRevoked).Count(&count).Error; err != nil {
		return nil, "", err
	}
	if count == 0 {
		return nil, "", ErrAgentNotFound
	}
	id, raw, hash, prefix, err := GenerateAgentToken()
	if err != nil {
		return nil, "", err
	}
	credential := &models.AgentCredential{ID: id, AgentID: agentID, Name: name, TokenHash: hash, TokenPrefix: prefix, ExpiresAt: expiresAt}
	if err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(credential).Error; err != nil {
			return err
		}
		return tx.Create(&models.AuditLog{UserID: &userID, ActorType: models.AuditActorHuman, OrgID: orgID, Action: models.ActionAgentTokenCreate, ResourceType: "agent_credential", ResourceID: credential.ID, IPAddress: ip}).Error
	}); err != nil {
		return nil, "", err
	}
	return credential, raw, nil
}

func (s *AgentService) ListCredentials(ctx context.Context, orgID, agentID uuid.UUID) ([]models.AgentCredential, error) {
	var credentials []models.AgentCredential
	err := database.GetDB().WithContext(ctx).Joins("JOIN agent_identities ON agent_identities.id = agent_credentials.agent_id").
		Where("agent_credentials.agent_id = ? AND agent_identities.org_id = ?", agentID, orgID).
		Order("agent_credentials.created_at DESC").Find(&credentials).Error
	return credentials, err
}

func (s *AgentService) RevokeCredential(ctx context.Context, userID, orgID, agentID, credentialID uuid.UUID, ip string) error {
	now := time.Now().UTC()
	return database.GetDB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&models.AgentCredential{}).
			Where("id = ? AND agent_id = ? AND EXISTS (SELECT 1 FROM agent_identities WHERE id = ? AND org_id = ?)", credentialID, agentID, agentID, orgID).
			Where("revoked_at IS NULL").Update("revoked_at", now)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		if err := tx.Model(&models.AgentAccessRequest{}).Where("credential_id = ? AND status IN ?", credentialID, []string{models.AccessRequestPending, models.AccessRequestApproved, models.AccessRequestDelivering}).Updates(map[string]any{"status": models.AccessRequestRevoked, "decision_reason": "credential revoked"}).Error; err != nil {
			return err
		}
		return tx.Create(&models.AuditLog{UserID: &userID, ActorType: models.AuditActorHuman, OrgID: orgID, Action: models.ActionAgentTokenRevoke, ResourceType: "agent_credential", ResourceID: credentialID, IPAddress: ip}).Error
	})
}

func normalizeKeys(keys []string) []string {
	set := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		key = strings.TrimSpace(key)
		if key != "" {
			set[key] = struct{}{}
		}
	}
	out := make([]string, 0, len(set))
	for key := range set {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}

func (s *AgentService) CreateGrant(ctx context.Context, userID, orgID, agentID, envID uuid.UUID, keys []string, allowAll bool, approvalMode string, maxLeaseSeconds int, expiresAt *time.Time, ip string) (*models.AgentGrant, error) {
	keys = normalizeKeys(keys)
	if !allowAll && len(keys) == 0 {
		return nil, fmt.Errorf("select at least one secret key or explicitly allow all secrets")
	}
	if expiresAt == nil {
		defaultExpiry := time.Now().UTC().Add(7 * 24 * time.Hour)
		expiresAt = &defaultExpiry
	}
	if !expiresAt.After(time.Now()) {
		return nil, fmt.Errorf("grant expiry must be in the future")
	}
	if expiresAt.After(time.Now().UTC().Add(90 * 24 * time.Hour)) {
		return nil, fmt.Errorf("grant expiry cannot exceed 90 days")
	}
	if approvalMode == "" {
		approvalMode = models.AgentApprovalAlways
	}
	if approvalMode != models.AgentApprovalAlways && approvalMode != models.AgentApprovalNone {
		return nil, fmt.Errorf("approval_mode must be always or none")
	}
	if maxLeaseSeconds == 0 {
		maxLeaseSeconds = 300
	}
	if maxLeaseSeconds < 30 || maxLeaseSeconds > 3600 {
		return nil, fmt.Errorf("max_lease_seconds must be between 30 and 3600")
	}
	db := database.GetDB().WithContext(ctx)
	var agent models.AgentIdentity
	if err := db.Where("id = ? AND org_id = ? AND status <> ?", agentID, orgID, models.AgentStatusRevoked).First(&agent).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAgentNotFound
		}
		return nil, err
	}
	var env models.Environment
	if err := db.Joins("JOIN projects ON projects.id = environments.project_id").
		Where("environments.id = ? AND projects.org_id = ?", envID, orgID).First(&env).Error; err != nil {
		return nil, fmt.Errorf("environment does not belong to this organization")
	}
	encoded, err := json.Marshal(keys)
	if err != nil {
		return nil, err
	}
	grant := &models.AgentGrant{AgentID: agentID, EnvironmentID: envID, Capability: models.AgentCapabilitySecretsInject, AllowedKeys: datatypes.JSON(encoded), AllowAllSecrets: allowAll, ApprovalMode: approvalMode, MaxLeaseSeconds: maxLeaseSeconds, ExpiresAt: expiresAt, CreatedBy: userID}
	if err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(grant).Error; err != nil {
			return err
		}
		metadata, _ := json.Marshal(map[string]any{"environment_id": envID, "allowed_keys": keys, "allow_all_secrets": allowAll, "approval_mode": approvalMode, "max_lease_seconds": maxLeaseSeconds})
		return tx.Create(&models.AuditLog{UserID: &userID, ActorType: models.AuditActorHuman, OrgID: orgID, Action: models.ActionAgentGrantCreate, ResourceType: "agent_grant", ResourceID: grant.ID, Metadata: datatypes.JSON(metadata), IPAddress: ip}).Error
	}); err != nil {
		return nil, err
	}
	return grant, nil
}

func (s *AgentService) ListGrants(ctx context.Context, orgID, agentID uuid.UUID) ([]models.AgentGrant, error) {
	var grants []models.AgentGrant
	err := database.GetDB().WithContext(ctx).Preload("Environment.Project").
		Joins("JOIN agent_identities ON agent_identities.id = agent_grants.agent_id").
		Where("agent_grants.agent_id = ? AND agent_identities.org_id = ?", agentID, orgID).
		Order("agent_grants.created_at DESC").Find(&grants).Error
	return grants, err
}

func (s *AgentService) RevokeGrant(ctx context.Context, userID, orgID, agentID, grantID uuid.UUID, ip string) error {
	now := time.Now().UTC()
	return database.GetDB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&models.AgentGrant{}).
			Where("id = ? AND agent_id = ? AND EXISTS (SELECT 1 FROM agent_identities WHERE id = ? AND org_id = ?)", grantID, agentID, agentID, orgID).
			Where("revoked_at IS NULL").Update("revoked_at", now)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return ErrGrantNotFound
		}
		grantRef, _ := json.Marshal([]uuid.UUID{grantID})
		if err := tx.Model(&models.AgentAccessRequest{}).Where("org_id = ? AND grant_ids @> CAST(? AS jsonb) AND status IN ?", orgID, string(grantRef), []string{models.AccessRequestPending, models.AccessRequestApproved, models.AccessRequestDelivering}).Updates(map[string]any{"status": models.AccessRequestRevoked, "decision_reason": "grant revoked"}).Error; err != nil {
			return err
		}
		return tx.Create(&models.AuditLog{UserID: &userID, ActorType: models.AuditActorHuman, OrgID: orgID, Action: models.ActionAgentGrantRevoke, ResourceType: "agent_grant", ResourceID: grantID, IPAddress: ip}).Error
	})
}

// AuthenticateToken validates a credential without ever loading or comparing a
// plaintext token from storage.
func (s *AgentService) AuthenticateToken(ctx context.Context, raw string) (*models.AgentIdentity, *models.AgentCredential, error) {
	id, err := credentialIDFromToken(raw)
	if err != nil {
		return nil, nil, ErrAgentUnauthorized
	}
	db := database.GetDB().WithContext(ctx)
	var credential models.AgentCredential
	if err := db.Preload("Agent.Organization").First(&credential, "id = ?", id).Error; err != nil {
		return nil, nil, ErrAgentUnauthorized
	}
	hash := sha256.Sum256([]byte(raw))
	stored, err := hex.DecodeString(credential.TokenHash)
	if err != nil || len(stored) != sha256.Size || subtle.ConstantTimeCompare(stored, hash[:]) != 1 {
		return nil, nil, ErrAgentUnauthorized
	}
	now := time.Now().UTC()
	if credential.RevokedAt != nil || credential.ExpiresAt == nil || !credential.ExpiresAt.After(now) || credential.Agent.Status != models.AgentStatusActive || credential.Agent.Organization.ID == uuid.Nil || credential.Agent.Organization.AgentAccessPaused {
		return nil, nil, ErrAgentUnauthorized
	}
	// Last-used timestamps are observability metadata, not an authorization
	// input. Throttling these writes removes two database updates from every
	// resolve while token and grant revocation remain live on every request.
	if credential.LastUsedAt == nil || now.Sub(*credential.LastUsedAt) >= s.usageWriteInterval {
		cutoff := now.Add(-s.usageWriteInterval)
		_ = db.Model(&models.AgentCredential{}).Where("id = ? AND (last_used_at IS NULL OR last_used_at < ?)", credential.ID, cutoff).Update("last_used_at", now).Error
		_ = db.Model(&models.AgentIdentity{}).Where("id = ? AND (last_used_at IS NULL OR last_used_at < ?)", credential.Agent.ID, cutoff).Update("last_used_at", now).Error
		credential.LastUsedAt = &now
		credential.Agent.LastUsedAt = &now
	}
	return &credential.Agent, &credential, nil
}

type AgentAccess struct {
	Environment      uuid.UUID
	AllowedKeys      map[string]struct{}
	AllowAll         bool
	ExpiresAt        *time.Time
	GrantIDs         []uuid.UUID
	RequiresApproval bool
	MaxLeaseSeconds  int
}

func resolveSelector(db *gorm.DB, agent *models.AgentIdentity, projectSelector, envSelector string) (*models.Environment, error) {
	var project models.Project
	projectQuery := db.Where("org_id = ?", agent.OrgID)
	if id, err := uuid.Parse(projectSelector); err == nil {
		projectQuery = projectQuery.Where("id = ?", id)
	} else {
		projectQuery = projectQuery.Where("lower(name) = lower(?)", strings.TrimSpace(projectSelector))
	}
	if err := projectQuery.First(&project).Error; err != nil {
		return nil, ErrAgentForbidden
	}
	var env models.Environment
	envQuery := db.Where("project_id = ?", project.ID)
	if id, err := uuid.Parse(envSelector); err == nil {
		envQuery = envQuery.Where("id = ?", id)
	} else {
		envQuery = envQuery.Where("lower(name) = lower(?)", strings.TrimSpace(envSelector))
	}
	if err := envQuery.First(&env).Error; err != nil {
		return nil, ErrAgentForbidden
	}
	return &env, nil
}

// AuthorizeResolve evaluates the current live grants on every request.
func (s *AgentService) AuthorizeResolve(ctx context.Context, agent *models.AgentIdentity, project, environment string, requestedKeys []string) (*AgentAccess, error) {
	if strings.TrimSpace(project) == "" || strings.TrimSpace(environment) == "" {
		return nil, fmt.Errorf("project and environment are required")
	}
	db := database.GetDB().WithContext(ctx)
	env, err := resolveSelector(db, agent, project, environment)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	var grants []models.AgentGrant
	if err := db.Where("agent_id = ? AND environment_id = ? AND capability = ? AND revoked_at IS NULL", agent.ID, env.ID, models.AgentCapabilitySecretsInject).
		Where("expires_at IS NULL OR expires_at > ?", now).Find(&grants).Error; err != nil {
		return nil, err
	}
	if len(grants) == 0 {
		return nil, ErrAgentForbidden
	}
	access := &AgentAccess{Environment: env.ID, AllowedKeys: map[string]struct{}{}, MaxLeaseSeconds: 3600}
	for _, grant := range grants {
		access.GrantIDs = append(access.GrantIDs, grant.ID)
		if grant.AllowAllSecrets {
			access.AllowAll = true
		}
		if grant.ApprovalMode != models.AgentApprovalNone {
			access.RequiresApproval = true
		}
		if grant.MaxLeaseSeconds > 0 && grant.MaxLeaseSeconds < access.MaxLeaseSeconds {
			access.MaxLeaseSeconds = grant.MaxLeaseSeconds
		}
		var keys []string
		if err := json.Unmarshal(grant.AllowedKeys, &keys); err != nil {
			return nil, fmt.Errorf("invalid stored grant policy: %w", err)
		}
		for _, key := range keys {
			access.AllowedKeys[key] = struct{}{}
		}
		if grant.ExpiresAt != nil && (access.ExpiresAt == nil || grant.ExpiresAt.Before(*access.ExpiresAt)) {
			expiry := *grant.ExpiresAt
			access.ExpiresAt = &expiry
		}
	}
	requestedKeys = normalizeKeys(requestedKeys)
	if len(requestedKeys) > 0 {
		for _, key := range requestedKeys {
			if _, ok := access.AllowedKeys[key]; !ok && !access.AllowAll {
				return nil, ErrAgentForbidden
			}
		}
		access.AllowedKeys = make(map[string]struct{}, len(requestedKeys))
		for _, key := range requestedKeys {
			access.AllowedKeys[key] = struct{}{}
		}
		access.AllowAll = false
	}
	return access, nil
}

func accessKeys(access *AgentAccess) []string {
	keys := make([]string, 0, len(access.AllowedKeys))
	for key := range access.AllowedKeys {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// CreateAccessRequest persists the exact authorization before any plaintext is
// released. Policy-approved requests start approved; the rest enter the human
// approval queue.
func (s *AgentService) CreateAccessRequest(ctx context.Context, agent *models.AgentIdentity, credential *models.AgentCredential, access *AgentAccess, purpose, sessionID, ip string) (*models.AgentAccessRequest, error) {
	keysJSON, _ := json.Marshal(accessKeys(access))
	grantsJSON, _ := json.Marshal(access.GrantIDs)
	status := models.AccessRequestApproved
	if access.RequiresApproval {
		status = models.AccessRequestPending
	}
	ttl := access.MaxLeaseSeconds
	if ttl < 30 || ttl > 3600 {
		ttl = 300
	}
	now := time.Now().UTC()
	expiresAt := now.Add(time.Duration(ttl) * time.Second)
	if access.ExpiresAt != nil && access.ExpiresAt.Before(expiresAt) {
		expiresAt = access.ExpiresAt.UTC()
	}
	fingerprintInput, _ := json.Marshal(map[string]any{"environment_id": access.Environment, "keys": json.RawMessage(keysJSON), "allow_all": access.AllowAll, "purpose": strings.TrimSpace(purpose), "session_id": strings.TrimSpace(sessionID)})
	fingerprintSum := sha256.Sum256(fingerprintInput)
	req := &models.AgentAccessRequest{
		OrgID: agent.OrgID, AgentID: agent.ID, CredentialID: credential.ID,
		EnvironmentID: access.Environment, GrantIDs: datatypes.JSON(grantsJSON),
		RequestedKeys: datatypes.JSON(keysJSON), AllowAllSecrets: access.AllowAll,
		Purpose: strings.TrimSpace(purpose), ExternalSessionID: strings.TrimSpace(sessionID),
		RequestFingerprint: hex.EncodeToString(fingerprintSum[:]),
		Status:             status, ExpiresAt: expiresAt,
	}
	metadata, _ := json.Marshal(map[string]any{
		"credential_id": credential.ID, "grant_ids": access.GrantIDs,
		"environment_id": access.Environment, "requested_keys": json.RawMessage(keysJSON),
		"allow_all_secrets": access.AllowAll, "status": status, "expires_at": req.ExpiresAt,
		"purpose": req.Purpose, "session_id": req.ExternalSessionID,
	})
	err := database.GetDB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing models.AgentAccessRequest
		if err := tx.Where("credential_id = ? AND request_fingerprint = ?", credential.ID, req.RequestFingerprint).First(&existing).Error; err == nil {
			*req = existing
			return nil
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if err := tx.Create(req).Error; err != nil {
			return err
		}
		return tx.Create(&models.AuditLog{AgentID: &agent.ID, ActorType: models.AuditActorAgent, OrgID: agent.OrgID, Action: models.ActionAgentAccessRequested, ResourceType: "agent_access_request", ResourceID: req.ID, Metadata: datatypes.JSON(metadata), IPAddress: ip}).Error
	})
	if err != nil {
		var existing models.AgentAccessRequest
		if lookupErr := database.GetDB().WithContext(ctx).Where("credential_id = ? AND request_fingerprint = ?", credential.ID, req.RequestFingerprint).First(&existing).Error; lookupErr == nil {
			return &existing, nil
		}
	}
	return req, err
}

func (s *AgentService) GetAccessRequest(ctx context.Context, agentID, credentialID, requestID uuid.UUID) (*models.AgentAccessRequest, error) {
	var req models.AgentAccessRequest
	err := database.GetDB().WithContext(ctx).Preload("Environment.Project").Where("id = ? AND agent_id = ? AND credential_id = ?", requestID, agentID, credentialID).First(&req).Error
	if err != nil {
		return nil, ErrAgentForbidden
	}
	if time.Now().UTC().After(req.ExpiresAt) && (req.Status == models.AccessRequestPending || req.Status == models.AccessRequestApproved) {
		_ = database.GetDB().WithContext(ctx).Model(&req).Update("status", models.AccessRequestExpired).Error
		req.Status = models.AccessRequestExpired
	}
	return &req, nil
}

// BeginDelivery locks and claims an approved request, preventing concurrent or
// replayed delivery, then confirms every grant is still active.
func (s *AgentService) BeginDelivery(ctx context.Context, agentID, credentialID, requestID uuid.UUID) (*models.AgentAccessRequest, error) {
	var req models.AgentAccessRequest
	now := time.Now().UTC()
	err := database.GetDB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND agent_id = ? AND credential_id = ?", requestID, agentID, credentialID).First(&req).Error; err != nil {
			return ErrAgentForbidden
		}
		if req.Status != models.AccessRequestApproved || !req.ExpiresAt.After(now) {
			return ErrAgentForbidden
		}
		if err := requireLiveAccessEnvironment(tx, req.OrgID, req.EnvironmentID); err != nil {
			return ErrAgentForbidden
		}
		var grantIDs []uuid.UUID
		if err := json.Unmarshal(req.GrantIDs, &grantIDs); err != nil || len(grantIDs) == 0 {
			return ErrAgentForbidden
		}
		var count int64
		if err := tx.Model(&models.AgentGrant{}).Where("id IN ? AND agent_id = ? AND revoked_at IS NULL AND deleted_at IS NULL", grantIDs, agentID).Where("expires_at IS NULL OR expires_at > ?", now).Count(&count).Error; err != nil || count != int64(len(grantIDs)) {
			return ErrAgentForbidden
		}
		return tx.Model(&req).Update("status", models.AccessRequestDelivering).Error
	})
	return &req, err
}

func (s *AgentService) CompleteDelivery(ctx context.Context, req *models.AgentAccessRequest, secretCount int, ip string) error {
	now := time.Now().UTC()
	metadata, _ := json.Marshal(map[string]any{
		"credential_id": req.CredentialID, "access_request_id": req.ID,
		"grant_ids": json.RawMessage(req.GrantIDs), "environment_id": req.EnvironmentID,
		"requested_keys": json.RawMessage(req.RequestedKeys), "allow_all_secrets": req.AllowAllSecrets,
		"secret_count": secretCount, "purpose": req.Purpose, "session_id": req.ExternalSessionID,
	})
	return database.GetDB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var org models.Organization
		if err := tx.Select("id", "agent_access_paused").First(&org, req.OrgID).Error; err != nil || org.AgentAccessPaused {
			return ErrAgentForbidden
		}
		var agentCount int64
		if err := tx.Model(&models.AgentIdentity{}).Where("id = ? AND status = ?", req.AgentID, models.AgentStatusActive).Count(&agentCount).Error; err != nil || agentCount != 1 {
			return ErrAgentForbidden
		}
		if err := requireLiveAccessEnvironment(tx, req.OrgID, req.EnvironmentID); err != nil {
			return ErrAgentForbidden
		}
		var credentialCount int64
		if err := tx.Model(&models.AgentCredential{}).Where("id = ? AND agent_id = ? AND revoked_at IS NULL AND expires_at > ?", req.CredentialID, req.AgentID, now).Count(&credentialCount).Error; err != nil || credentialCount != 1 {
			return ErrAgentForbidden
		}
		var grantIDs []uuid.UUID
		if err := json.Unmarshal(req.GrantIDs, &grantIDs); err != nil || len(grantIDs) == 0 {
			return ErrAgentForbidden
		}
		var grantCount int64
		if err := tx.Model(&models.AgentGrant{}).Where("id IN ? AND agent_id = ? AND revoked_at IS NULL AND deleted_at IS NULL", grantIDs, req.AgentID).Where("expires_at IS NULL OR expires_at > ?", now).Count(&grantCount).Error; err != nil || grantCount != int64(len(grantIDs)) {
			return ErrAgentForbidden
		}
		result := tx.Model(&models.AgentAccessRequest{}).Where("id = ? AND status = ?", req.ID, models.AccessRequestDelivering).Updates(map[string]any{"status": models.AccessRequestConsumed, "used_at": now})
		if result.Error != nil || result.RowsAffected != 1 {
			return ErrAgentForbidden
		}
		return tx.Create(&models.AuditLog{AgentID: &req.AgentID, ActorType: models.AuditActorAgent, OrgID: req.OrgID, Action: models.ActionSecretRead, ResourceType: "environment", ResourceID: req.EnvironmentID, Metadata: datatypes.JSON(metadata), IPAddress: ip}).Error
	})
}

func (s *AgentService) FailDelivery(ctx context.Context, req *models.AgentAccessRequest, reason, ip string) {
	metadata, _ := json.Marshal(map[string]any{"access_request_id": req.ID, "reason": reason})
	_ = database.GetDB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.AgentAccessRequest{}).Where("id = ? AND status = ?", req.ID, models.AccessRequestDelivering).Updates(map[string]any{"status": models.AccessRequestRevoked, "decision_reason": "delivery failed"}).Error; err != nil {
			return err
		}
		return tx.Create(&models.AuditLog{AgentID: &req.AgentID, ActorType: models.AuditActorAgent, OrgID: req.OrgID, Action: models.ActionAgentAccessFailed, ResourceType: "agent_access_request", ResourceID: req.ID, Metadata: datatypes.JSON(metadata), IPAddress: ip}).Error
	})
}

func (s *AgentService) ListAccessRequests(ctx context.Context, orgID uuid.UUID, status string, limit int) ([]models.AgentAccessRequest, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	db := database.GetDB().WithContext(ctx)
	now := time.Now().UTC()
	if err := db.Model(&models.AgentAccessRequest{}).Where("org_id = ? AND status IN ? AND expires_at <= ?", orgID, []string{models.AccessRequestPending, models.AccessRequestApproved}, now).Update("status", models.AccessRequestExpired).Error; err != nil {
		return nil, err
	}
	query := db.Preload("Agent").Preload("Credential").Preload("Environment.Project").Preload("Approver").Where("org_id = ?", orgID)
	if status != "" && status != "all" {
		query = query.Where("status = ?", status)
	}
	var requests []models.AgentAccessRequest
	err := query.Order("created_at DESC").Limit(limit).Find(&requests).Error
	return requests, err
}

func (s *AgentService) DecideAccessRequest(ctx context.Context, userID, orgID, requestID uuid.UUID, approve bool, reason, ip string) (*models.AgentAccessRequest, error) {
	now := time.Now().UTC()
	status, action := models.AccessRequestDenied, models.ActionAgentAccessDenied
	if approve {
		status, action = models.AccessRequestApproved, models.ActionAgentAccessApproved
	}
	var req models.AgentAccessRequest
	err := database.GetDB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND org_id = ?", requestID, orgID).First(&req).Error; err != nil {
			return err
		}
		if req.Status != models.AccessRequestPending || !req.ExpiresAt.After(now) {
			return fmt.Errorf("access request is no longer pending")
		}
		if approve {
			var org models.Organization
			if err := tx.Select("id", "agent_access_paused").First(&org, orgID).Error; err != nil || org.AgentAccessPaused {
				return fmt.Errorf("organization agent access is paused")
			}
			if err := requireLiveAccessEnvironment(tx, orgID, req.EnvironmentID); err != nil {
				return fmt.Errorf("requested environment is no longer active")
			}
			var credentialCount int64
			if err := tx.Model(&models.AgentCredential{}).Joins("JOIN agent_identities ON agent_identities.id = agent_credentials.agent_id").Where("agent_credentials.id = ? AND agent_credentials.revoked_at IS NULL AND agent_credentials.expires_at > ? AND agent_identities.status = ?", req.CredentialID, now, models.AgentStatusActive).Count(&credentialCount).Error; err != nil || credentialCount != 1 {
				return fmt.Errorf("agent credential is no longer active")
			}
			var grantIDs []uuid.UUID
			if err := json.Unmarshal(req.GrantIDs, &grantIDs); err != nil || len(grantIDs) == 0 {
				return fmt.Errorf("access request has no valid grants")
			}
			var grantCount int64
			if err := tx.Model(&models.AgentGrant{}).Where("id IN ? AND agent_id = ? AND revoked_at IS NULL AND deleted_at IS NULL", grantIDs, req.AgentID).Where("expires_at > ?", now).Count(&grantCount).Error; err != nil || grantCount != int64(len(grantIDs)) {
				return fmt.Errorf("one or more grants are no longer active")
			}
		}
		if err := tx.Model(&req).Updates(map[string]any{"status": status, "decided_by": userID, "decided_at": now, "decision_reason": strings.TrimSpace(reason)}).Error; err != nil {
			return err
		}
		metadata, _ := json.Marshal(map[string]any{"agent_id": req.AgentID, "decision": status, "reason": strings.TrimSpace(reason)})
		return tx.Create(&models.AuditLog{UserID: &userID, ActorType: models.AuditActorHuman, OrgID: orgID, Action: action, ResourceType: "agent_access_request", ResourceID: req.ID, Metadata: datatypes.JSON(metadata), IPAddress: ip}).Error
	})
	req.Status, req.DecidedBy, req.DecidedAt, req.DecisionReason = status, &userID, &now, strings.TrimSpace(reason)
	return &req, err
}

func requireLiveAccessEnvironment(tx *gorm.DB, orgID, environmentID uuid.UUID) error {
	var count int64
	err := tx.Model(&models.Environment{}).
		Joins("JOIN projects ON projects.id = environments.project_id AND projects.deleted_at IS NULL").
		Joins("JOIN organizations ON organizations.id = projects.org_id AND organizations.deleted_at IS NULL").
		Where("environments.id = ? AND projects.org_id = ?", environmentID, orgID).
		Count(&count).Error
	if err != nil {
		return err
	}
	if count != 1 {
		return ErrAgentForbidden
	}
	return nil
}

func (s *AgentService) RevokeAccessRequest(ctx context.Context, userID, orgID, requestID uuid.UUID, reason, ip string) (*models.AgentAccessRequest, error) {
	var req models.AgentAccessRequest
	err := database.GetDB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND org_id = ?", requestID, orgID).First(&req).Error; err != nil {
			return err
		}
		if req.Status != models.AccessRequestPending && req.Status != models.AccessRequestApproved && req.Status != models.AccessRequestDelivering {
			return fmt.Errorf("access request can no longer be revoked")
		}
		if err := tx.Model(&req).Updates(map[string]any{"status": models.AccessRequestRevoked, "decided_by": userID, "decided_at": time.Now().UTC(), "decision_reason": strings.TrimSpace(reason)}).Error; err != nil {
			return err
		}
		metadata, _ := json.Marshal(map[string]any{"agent_id": req.AgentID, "reason": strings.TrimSpace(reason)})
		return tx.Create(&models.AuditLog{UserID: &userID, ActorType: models.AuditActorHuman, OrgID: orgID, Action: models.ActionAgentAccessRevoked, ResourceType: "agent_access_request", ResourceID: req.ID, Metadata: datatypes.JSON(metadata), IPAddress: ip}).Error
	})
	req.Status, req.DecisionReason = models.AccessRequestRevoked, strings.TrimSpace(reason)
	return &req, err
}

func (s *AgentService) SetOrgAgentAccessPaused(ctx context.Context, userID, orgID uuid.UUID, paused bool, ip string) error {
	action := "org_agent_access_resumed"
	if paused {
		action = "org_agent_access_paused"
	}
	return database.GetDB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&models.Organization{}).Where("id = ?", orgID).Update("agent_access_paused", paused)
		if result.Error != nil || result.RowsAffected != 1 {
			return gorm.ErrRecordNotFound
		}
		if paused {
			if err := tx.Model(&models.AgentAccessRequest{}).Where("org_id = ? AND status IN ?", orgID, []string{models.AccessRequestPending, models.AccessRequestApproved, models.AccessRequestDelivering}).Updates(map[string]any{"status": models.AccessRequestRevoked, "decision_reason": "organization emergency stop"}).Error; err != nil {
				return err
			}
		}
		meta, _ := json.Marshal(map[string]bool{"paused": paused})
		return tx.Create(&models.AuditLog{UserID: &userID, ActorType: models.AuditActorHuman, OrgID: orgID, Action: action, ResourceType: "organization", ResourceID: orgID, Metadata: datatypes.JSON(meta), IPAddress: ip}).Error
	})
}
