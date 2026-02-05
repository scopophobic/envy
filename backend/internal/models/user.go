package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SubscriptionTier represents the subscription tier
type SubscriptionTier string

const (
	TierFree    SubscriptionTier = "free"
	TierStarter SubscriptionTier = "starter"
	TierTeam    SubscriptionTier = "team"
)

// SubscriptionStatus represents the subscription status
type SubscriptionStatus string

const (
	StatusActive    SubscriptionStatus = "active"
	StatusCancelled SubscriptionStatus = "cancelled"
	StatusExpired   SubscriptionStatus = "expired"
)

// User represents a user in the system
type User struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	Email     string    `gorm:"type:varchar(255);uniqueIndex;not null" json:"email"`
	Name      string    `gorm:"type:varchar(255);not null" json:"name"`
	
	// OAuth fields
	OAuthProvider string `gorm:"type:varchar(50);not null" json:"oauth_provider"` // google, github
	OAuthID       string `gorm:"type:varchar(255);not null" json:"oauth_id"`
	
	// Subscription fields
	SubscriptionTier      SubscriptionTier   `gorm:"type:varchar(20);not null;default:'free'" json:"subscription_tier"`
	SubscriptionStatus    SubscriptionStatus `gorm:"type:varchar(20);not null;default:'active'" json:"subscription_status"`
	SubscriptionExpiresAt *time.Time         `json:"subscription_expires_at,omitempty"`
	
	// Timestamps
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	
	// Relationships
	OwnedOrganizations []Organization `gorm:"foreignKey:OwnerID" json:"owned_organizations,omitempty"`
	OrgMemberships     []OrgMember    `gorm:"foreignKey:UserID" json:"org_memberships,omitempty"`
	CreatedSecrets     []Secret       `gorm:"foreignKey:CreatedBy" json:"-"`
	AuditLogs          []AuditLog     `gorm:"foreignKey:UserID" json:"-"`
}

// BeforeCreate hook to generate UUID
func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return nil
}

// TableName specifies the table name
func (User) TableName() string {
	return "users"
}
