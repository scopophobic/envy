package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// AuditLog represents an audit log entry
type AuditLog struct {
	ID           uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID       *uuid.UUID     `gorm:"type:uuid;index" json:"user_id,omitempty"`
	AgentID      *uuid.UUID     `gorm:"type:uuid;index" json:"agent_id,omitempty"`
	ActorType    string         `gorm:"type:varchar(20);not null;default:human;index" json:"actor_type"`
	OrgID        uuid.UUID      `gorm:"type:uuid;not null;index" json:"org_id"`
	Action       string         `gorm:"type:varchar(100);not null;index" json:"action"` // secret_read, secret_write, etc.
	ResourceType string         `gorm:"type:varchar(50);not null" json:"resource_type"` // secret, project, etc.
	ResourceID   uuid.UUID      `gorm:"type:uuid;not null;index" json:"resource_id"`
	Metadata     datatypes.JSON `gorm:"type:jsonb" json:"metadata,omitempty"` // Additional context
	IPAddress    string         `gorm:"type:varchar(45)" json:"ip_address"`   // IPv4 or IPv6

	// Timestamps
	CreatedAt time.Time `json:"created_at"`

	// Relationships
	User         *User          `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Agent        *AgentIdentity `gorm:"foreignKey:AgentID" json:"agent,omitempty"`
	Organization Organization   `gorm:"foreignKey:OrgID" json:"organization,omitempty"`
}

// BeforeCreate hook to generate UUID
func (al *AuditLog) BeforeCreate(tx *gorm.DB) error {
	if al.ID == uuid.Nil {
		al.ID = uuid.New()
	}
	return nil
}

// TableName specifies the table name
func (AuditLog) TableName() string {
	return "audit_logs"
}

// Audit actions
const (
	AuditActorHuman = "human"
	AuditActorAgent = "agent"

	ActionSecretRead       = "secret_read"
	ActionSecretCreate     = "secret_create"
	ActionSecretUpdate     = "secret_update"
	ActionSecretDelete     = "secret_delete"
	ActionProjectCreate    = "project_create"
	ActionProjectUpdate    = "project_update"
	ActionProjectDelete    = "project_delete"
	ActionMemberInvite     = "member_invite"
	ActionMemberRemove     = "member_remove"
	ActionRoleChange       = "role_change"
	ActionOrgCreate        = "org_create"
	ActionOrgUpdate        = "org_update"
	ActionOrgDelete        = "org_delete"
	ActionAgentCreate      = "agent_create"
	ActionAgentUpdate      = "agent_update"
	ActionAgentTokenCreate = "agent_token_create"
	ActionAgentTokenRevoke = "agent_token_revoke"
	ActionAgentGrantCreate = "agent_grant_create"
	ActionAgentGrantRevoke = "agent_grant_revoke"
)
