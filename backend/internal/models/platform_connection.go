package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// PlatformConnection stores encrypted deploy-platform credentials for manual sync.
type PlatformConnection struct {
	ID             uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID         uuid.UUID      `gorm:"type:uuid;not null;index" json:"user_id"`
	Platform       string         `gorm:"size:32;not null;index" json:"platform"`
	Name           string         `gorm:"size:120;not null" json:"name"`
	EncryptedToken string         `gorm:"type:text;not null" json:"-"`
	KeyID          string         `gorm:"size:120;not null" json:"key_id"`
	TokenPrefix    string         `gorm:"size:12;not null" json:"token_prefix"`
	Metadata       datatypes.JSON `gorm:"type:jsonb" json:"metadata"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
}
