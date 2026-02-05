package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Project represents a project within an organization
type Project struct {
	ID          uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	OrgID       uuid.UUID `gorm:"type:uuid;not null;index" json:"org_id"`
	Name        string    `gorm:"type:varchar(255);not null" json:"name"`
	Description *string   `gorm:"type:text" json:"description,omitempty"`
	
	// Timestamps
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	
	// Relationships
	Organization Organization  `gorm:"foreignKey:OrgID" json:"organization,omitempty"`
	Environments []Environment `gorm:"foreignKey:ProjectID" json:"environments,omitempty"`
}

// BeforeCreate hook to generate UUID
func (p *Project) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}

// TableName specifies the table name
func (Project) TableName() string {
	return "projects"
}
