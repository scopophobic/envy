package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Secret represents an encrypted secret
type Secret struct {
	ID             uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	EnvironmentID  uuid.UUID `gorm:"type:uuid;not null;index" json:"environment_id"`
	Key            string    `gorm:"type:varchar(255);not null" json:"key"`
	EncryptedValue string    `gorm:"type:text;not null" json:"-"` // Never expose in JSON
	KMSKeyID       string    `gorm:"type:varchar(255);not null" json:"-"`
	CreatedBy      uuid.UUID `gorm:"type:uuid;not null;index" json:"created_by"`
	
	// Timestamps
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"` // Soft delete for audit trail
	
	// Relationships
	Environment Environment `gorm:"foreignKey:EnvironmentID" json:"environment,omitempty"`
	Creator     User        `gorm:"foreignKey:CreatedBy" json:"creator,omitempty"`
}

// BeforeCreate hook to generate UUID
func (s *Secret) BeforeCreate(tx *gorm.DB) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return nil
}

// TableName specifies the table name
func (Secret) TableName() string {
	return "secrets"
}

// SecretResponse is used for API responses (without encrypted value)
type SecretResponse struct {
	ID            uuid.UUID `json:"id"`
	EnvironmentID uuid.UUID `json:"environment_id"`
	Key           string    `json:"key"`
	CreatedBy     uuid.UUID `json:"created_by"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// ToResponse converts Secret to SecretResponse
func (s *Secret) ToResponse() SecretResponse {
	return SecretResponse{
		ID:            s.ID,
		EnvironmentID: s.EnvironmentID,
		Key:           s.Key,
		CreatedBy:     s.CreatedBy,
		CreatedAt:     s.CreatedAt,
		UpdatedAt:     s.UpdatedAt,
	}
}
