package services

import (
	"context"

	"github.com/envo/backend/internal/database"
	"github.com/envo/backend/internal/models"
	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// AuditService handles writing and querying audit logs
type AuditService struct{}

// NewAuditService creates a new audit service
func NewAuditService() *AuditService {
	return &AuditService{}
}

// Log writes an audit log entry
func (s *AuditService) Log(ctx context.Context, userID, orgID, resourceID uuid.UUID, action, resourceType, ip string, metadata datatypes.JSON) error {
	db := database.GetDB().WithContext(ctx)

	logEntry := &models.AuditLog{
		UserID:       userID,
		OrgID:        orgID,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Metadata:     metadata,
		IPAddress:    ip,
	}

	return db.Create(logEntry).Error
}

// ListOrgLogs lists audit logs for an organization (most recent first)
func (s *AuditService) ListOrgLogs(orgID uuid.UUID, limit int) ([]models.AuditLog, error) {
	db := database.GetDB()

	if limit <= 0 || limit > 500 {
		limit = 100
	}

	var logs []models.AuditLog
	if err := db.Where("org_id = ?", orgID).
		Order("created_at DESC").
		Limit(limit).
		Find(&logs).Error; err != nil {
		return nil, err
	}

	return logs, nil
}

