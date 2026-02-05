package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Permission represents a granular permission
type Permission struct {
	ID          uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	Name        string    `gorm:"type:varchar(100);uniqueIndex;not null" json:"name"` // e.g., secrets.read, secrets.create
	Description string    `gorm:"type:text" json:"description"`
	
	// Timestamps
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	
	// Relationships
	Roles []Role `gorm:"many2many:role_permissions;" json:"roles,omitempty"`
}

// BeforeCreate hook to generate UUID
func (p *Permission) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}

// TableName specifies the table name
func (Permission) TableName() string {
	return "permissions"
}

// Predefined permissions
const (
	PermissionSecretsRead         = "secrets.read"
	PermissionSecretsCreate       = "secrets.create"
	PermissionSecretsUpdate       = "secrets.update"
	PermissionSecretsDelete       = "secrets.delete"
	PermissionProjectsManage      = "projects.manage"
	PermissionEnvironmentsManage  = "environments.manage"
	PermissionMembersInvite       = "members.invite"
	PermissionMembersManage       = "members.manage"
	PermissionAuditView           = "audit.view"
	PermissionOrgManage           = "org.manage"
)

// Role represents a role with permissions
type Role struct {
	ID           uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	OrgID        *uuid.UUID `gorm:"type:uuid;index" json:"org_id,omitempty"` // null for system roles
	Name         string     `gorm:"type:varchar(100);not null" json:"name"`
	IsSystemRole bool       `gorm:"default:false" json:"is_system_role"` // true for predefined roles
	
	// Timestamps
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	
	// Relationships
	Organization *Organization `gorm:"foreignKey:OrgID" json:"organization,omitempty"`
	Permissions  []Permission  `gorm:"many2many:role_permissions;" json:"permissions,omitempty"`
	Members      []OrgMember   `gorm:"foreignKey:RoleID" json:"members,omitempty"`
}

// BeforeCreate hook to generate UUID
func (r *Role) BeforeCreate(tx *gorm.DB) error {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return nil
}

// TableName specifies the table name
func (Role) TableName() string {
	return "roles"
}

// Predefined system roles
const (
	RoleOwner         = "Owner"
	RoleAdmin         = "Admin"
	RoleSecretManager = "Secret Manager"
	RoleDeveloper     = "Developer"
	RoleViewer        = "Viewer"
)
