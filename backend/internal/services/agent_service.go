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
	if err := database.GetDB().WithContext(ctx).Create(agent).Error; err != nil {
		return nil, err
	}
	if s.audit != nil {
		_ = s.audit.Log(ctx, userID, orgID, agent.ID, models.ActionAgentCreate, "agent", ip, nil)
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
		if status == models.AgentStatusRevoked {
			now := time.Now().UTC()
			if err := tx.Model(&models.AgentCredential{}).Where("agent_id = ? AND revoked_at IS NULL", agentID).Update("revoked_at", now).Error; err != nil {
				return err
			}
			if err := tx.Model(&models.AgentGrant{}).Where("agent_id = ? AND revoked_at IS NULL", agentID).Update("revoked_at", now).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrAgentNotFound
	}
	if err != nil {
		return nil, err
	}
	agent.Status = status
	if s.audit != nil {
		metadata, _ := json.Marshal(map[string]string{"status": status})
		_ = s.audit.Log(ctx, userID, orgID, agentID, models.ActionAgentUpdate, "agent", ip, datatypes.JSON(metadata))
	}
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
	if expiresAt != nil && !expiresAt.After(time.Now()) {
		return nil, "", fmt.Errorf("credential expiry must be in the future")
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
	if err := db.Create(credential).Error; err != nil {
		return nil, "", err
	}
	if s.audit != nil {
		_ = s.audit.Log(ctx, userID, orgID, credential.ID, models.ActionAgentTokenCreate, "agent_credential", ip, nil)
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
	result := database.GetDB().WithContext(ctx).Model(&models.AgentCredential{}).
		Where("id = ? AND agent_id = ? AND EXISTS (SELECT 1 FROM agent_identities WHERE id = ? AND org_id = ?)", credentialID, agentID, agentID, orgID).
		Where("revoked_at IS NULL").Update("revoked_at", now)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	if s.audit != nil {
		_ = s.audit.Log(ctx, userID, orgID, credentialID, models.ActionAgentTokenRevoke, "agent_credential", ip, nil)
	}
	return nil
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

func (s *AgentService) CreateGrant(ctx context.Context, userID, orgID, agentID, envID uuid.UUID, keys []string, allowAll bool, expiresAt *time.Time, ip string) (*models.AgentGrant, error) {
	keys = normalizeKeys(keys)
	if !allowAll && len(keys) == 0 {
		return nil, fmt.Errorf("select at least one secret key or explicitly allow all secrets")
	}
	if expiresAt != nil && !expiresAt.After(time.Now()) {
		return nil, fmt.Errorf("grant expiry must be in the future")
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
	grant := &models.AgentGrant{AgentID: agentID, EnvironmentID: envID, Capability: models.AgentCapabilitySecretsInject, AllowedKeys: datatypes.JSON(encoded), AllowAllSecrets: allowAll, ExpiresAt: expiresAt, CreatedBy: userID}
	if err := db.Create(grant).Error; err != nil {
		return nil, err
	}
	if s.audit != nil {
		_ = s.audit.Log(ctx, userID, orgID, grant.ID, models.ActionAgentGrantCreate, "agent_grant", ip, nil)
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
	result := database.GetDB().WithContext(ctx).Model(&models.AgentGrant{}).
		Where("id = ? AND agent_id = ? AND EXISTS (SELECT 1 FROM agent_identities WHERE id = ? AND org_id = ?)", grantID, agentID, agentID, orgID).
		Where("revoked_at IS NULL").Update("revoked_at", now)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrGrantNotFound
	}
	if s.audit != nil {
		_ = s.audit.Log(ctx, userID, orgID, grantID, models.ActionAgentGrantRevoke, "agent_grant", ip, nil)
	}
	return nil
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
	if err := db.Preload("Agent").First(&credential, "id = ?", id).Error; err != nil {
		return nil, nil, ErrAgentUnauthorized
	}
	hash := sha256.Sum256([]byte(raw))
	stored, err := hex.DecodeString(credential.TokenHash)
	if err != nil || len(stored) != sha256.Size || subtle.ConstantTimeCompare(stored, hash[:]) != 1 {
		return nil, nil, ErrAgentUnauthorized
	}
	now := time.Now().UTC()
	if credential.RevokedAt != nil || (credential.ExpiresAt != nil && !credential.ExpiresAt.After(now)) || credential.Agent.Status != models.AgentStatusActive {
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
	Environment uuid.UUID
	AllowedKeys map[string]struct{}
	AllowAll    bool
	ExpiresAt   *time.Time
	GrantIDs    []uuid.UUID
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
	access := &AgentAccess{Environment: env.ID, AllowedKeys: map[string]struct{}{}}
	for _, grant := range grants {
		access.GrantIDs = append(access.GrantIDs, grant.ID)
		if grant.AllowAllSecrets {
			access.AllowAll = true
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
