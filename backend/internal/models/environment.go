package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Environment represents an environment within a project
type Environment struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	ProjectID uuid.UUID `gorm:"type:uuid;not null;index" json:"project_id"`
	Name      string    `gorm:"type:varchar(100);not null" json:"name"` // dev, staging, prod
	
	// Timestamps
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	
	// Relationships
	Project Project  `gorm:"foreignKey:ProjectID" json:"project,omitempty"`
	Secrets []Secret `gorm:"foreignKey:EnvironmentID" json:"secrets,omitempty"`
}

// BeforeCreate hook to generate UUID
func (e *Environment) BeforeCreate(tx *gorm.DB) error {
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	return nil
}

// TableName specifies the table name
func (Environment) TableName() string {
	return "environments"
}

// Common environment names
const (
	EnvDevelopment = "development"
	EnvStaging     = "staging"
	EnvProduction  = "production"
)
