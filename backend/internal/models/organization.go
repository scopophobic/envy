package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Organization represents an organization
type Organization struct {
	ID      uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	OwnerID uuid.UUID `gorm:"type:uuid;not null;index" json:"owner_id"`
	Name    string    `gorm:"type:varchar(255);not null" json:"name"`
	
	// Timestamps
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	
	// Relationships
	Owner    User        `gorm:"foreignKey:OwnerID" json:"owner,omitempty"`
	Members  []OrgMember `gorm:"foreignKey:OrgID" json:"members,omitempty"`
	Projects []Project   `gorm:"foreignKey:OrgID" json:"projects,omitempty"`
	Roles    []Role      `gorm:"foreignKey:OrgID" json:"roles,omitempty"`
}

// BeforeCreate hook to generate UUID
func (o *Organization) BeforeCreate(tx *gorm.DB) error {
	if o.ID == uuid.Nil {
		o.ID = uuid.New()
	}
	return nil
}

// TableName specifies the table name
func (Organization) TableName() string {
	return "organizations"
}

// OrgMember represents organization membership
type OrgMember struct {
	ID     uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	OrgID  uuid.UUID `gorm:"type:uuid;not null;index" json:"org_id"`
	UserID uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	RoleID uuid.UUID `gorm:"type:uuid;not null;index" json:"role_id"`
	
	// Timestamps
	CreatedAt time.Time `json:"created_at"`
	
	// Relationships
	Organization Organization `gorm:"foreignKey:OrgID" json:"organization,omitempty"`
	User         User         `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Role         Role         `gorm:"foreignKey:RoleID" json:"role,omitempty"`
}

// BeforeCreate hook to generate UUID
func (om *OrgMember) BeforeCreate(tx *gorm.DB) error {
	if om.ID == uuid.Nil {
		om.ID = uuid.New()
	}
	return nil
}

// TableName specifies the table name
func (OrgMember) TableName() string {
	return "org_members"
}
