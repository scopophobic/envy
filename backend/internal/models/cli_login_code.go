package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CLILoginCode is a short-lived one-time code exchanged for tokens.
type CLILoginCode struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	Code      string    `gorm:"type:varchar(128);uniqueIndex;not null" json:"-"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;index" json:"-"`
	ExpiresAt time.Time `gorm:"not null;index" json:"-"`
	UsedAt    *time.Time `json:"-"`

	CreatedAt time.Time `json:"-"`
}

func (c *CLILoginCode) BeforeCreate(tx *gorm.DB) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	return nil
}

func (CLILoginCode) TableName() string {
	return "cli_login_codes"
}

func (c *CLILoginCode) IsValid(now time.Time) bool {
	if c.UsedAt != nil {
		return false
	}
	return now.Before(c.ExpiresAt)
}

