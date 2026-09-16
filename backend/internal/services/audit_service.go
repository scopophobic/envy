package services

import (
	"context"
	"strings"
	"time"

	"github.com/envo/backend/internal/database"
	"github.com/envo/backend/internal/models"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// AuditService handles writing and querying audit logs
type AuditService struct{}

// NewAuditService creates a new audit service
func NewAuditService() *AuditService {
	return &AuditService{}
}

// Log writes an audit log entry
func (s *AuditService) Log(ctx context.Context, userID, orgID, resourceID uuid.UUID, action, resourceType, ip string, metadata datatypes.JSON) error {
	return s.LogWithDB(database.GetDB().WithContext(ctx), userID, orgID, resourceID, action, resourceType, ip, metadata)
}

// LogWithDB lets security-sensitive mutations and their audit record share one
// transaction. A failed audit write therefore rolls back the protected action.
func (s *AuditService) LogWithDB(db *gorm.DB, userID, orgID, resourceID uuid.UUID, action, resourceType, ip string, metadata datatypes.JSON) error {

	logEntry := &models.AuditLog{
		UserID:       &userID,
		ActorType:    models.AuditActorHuman,
		OrgID:        orgID,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Metadata:     metadata,
		IPAddress:    ip,
	}

	return db.Create(logEntry).Error
}

// LogAgent writes an audit entry attributed to a non-human identity.
func (s *AuditService) LogAgent(ctx context.Context, agentID, orgID, resourceID uuid.UUID, action, resourceType, ip string, metadata datatypes.JSON) error {
	return database.GetDB().WithContext(ctx).Create(&models.AuditLog{
		AgentID:      &agentID,
		ActorType:    models.AuditActorAgent,
		OrgID:        orgID,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Metadata:     metadata,
		IPAddress:    ip,
	}).Error
}

// ListOrgLogs lists audit logs for an organization (most recent first)
func (s *AuditService) ListOrgLogs(orgID uuid.UUID, limit int) ([]models.AuditLog, error) {
	return s.QueryOrgLogs(context.Background(), orgID, AuditQuery{Limit: limit})
}

type AuditQuery struct {
	Limit        int
	ActorType    string
	Action       string
	ResourceType string
	Before       *time.Time
}

func (s *AuditService) QueryOrgLogs(ctx context.Context, orgID uuid.UUID, filter AuditQuery) ([]models.AuditLog, error) {
	db := database.GetDB().WithContext(ctx)

	if filter.Limit <= 0 || filter.Limit > 500 {
		filter.Limit = 100
	}
	query := db.Preload("User").Preload("Agent").Where("org_id = ?", orgID)
	if actor := strings.TrimSpace(filter.ActorType); actor != "" {
		query = query.Where("actor_type = ?", actor)
	}
	if action := strings.TrimSpace(filter.Action); action != "" {
		query = query.Where("action = ?", action)
	}
	if resourceType := strings.TrimSpace(filter.ResourceType); resourceType != "" {
		query = query.Where("resource_type = ?", resourceType)
	}
	if filter.Before != nil {
		query = query.Where("created_at < ?", filter.Before.UTC())
	}
	var logs []models.AuditLog
	if err := query.Order("created_at DESC").Limit(filter.Limit).Find(&logs).Error; err != nil {
		return nil, err
	}

	return logs, nil
}
