package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TierLimit represents dynamic tier limits
type TierLimit struct {
	ID         uuid.UUID        `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	Tier       SubscriptionTier `gorm:"type:varchar(20);not null;index" json:"tier"`
	LimitType  string           `gorm:"type:varchar(100);not null" json:"limit_type"` // max_devs, max_projects, etc.
	LimitValue int              `gorm:"not null" json:"limit_value"`                  // -1 for unlimited
	
	// Timestamps
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// BeforeCreate hook to generate UUID
func (tl *TierLimit) BeforeCreate(tx *gorm.DB) error {
	if tl.ID == uuid.Nil {
		tl.ID = uuid.New()
	}
	return nil
}

// TableName specifies the table name
func (TierLimit) TableName() string {
	return "tier_limits"
}

// Limit types
const (
	LimitTypeMaxDevs              = "max_devs"
	LimitTypeMaxProjects          = "max_projects"
	LimitTypeMaxOrgs              = "max_orgs"
	LimitTypeMaxSecretsPerEnv     = "max_secrets_per_env"
	LimitTypeAPIRateLimitPerHour  = "api_rate_limit_per_hour"
	LimitTypeAuditRetentionDays   = "audit_retention_days"
)

// Unlimited value
const UnlimitedValue = -1
