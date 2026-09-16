package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const (
	AgentStatusActive    = "active"
	AgentStatusSuspended = "suspended"
	AgentStatusRevoked   = "revoked"

	AgentCapabilitySecretsInject = "secrets.inject"

	AgentApprovalAlways = "always"
	AgentApprovalNone   = "none"

	AccessRequestPending    = "pending"
	AccessRequestApproved   = "approved"
	AccessRequestDenied     = "denied"
	AccessRequestDelivering = "delivering"
	AccessRequestConsumed   = "consumed"
	AccessRequestExpired    = "expired"
	AccessRequestRevoked    = "revoked"
)

// AgentIdentity is a non-human identity owned by an organization.
type AgentIdentity struct {
	ID          uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	OrgID       uuid.UUID      `gorm:"type:uuid;not null;index" json:"org_id"`
	Name        string         `gorm:"type:varchar(120);not null" json:"name"`
	Description string         `gorm:"type:text" json:"description"`
	Status      string         `gorm:"type:varchar(20);not null;default:active;index" json:"status"`
	CreatedBy   uuid.UUID      `gorm:"type:uuid;not null;index" json:"created_by"`
	LastUsedAt  *time.Time     `json:"last_used_at,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`

	Organization Organization      `gorm:"foreignKey:OrgID" json:"organization,omitempty"`
	Creator      User              `gorm:"foreignKey:CreatedBy" json:"creator,omitempty"`
	Credentials  []AgentCredential `gorm:"foreignKey:AgentID" json:"credentials,omitempty"`
	Grants       []AgentGrant      `gorm:"foreignKey:AgentID" json:"grants,omitempty"`
}

func (a *AgentIdentity) BeforeCreate(_ *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	if a.Status == "" {
		a.Status = AgentStatusActive
	}
	return nil
}

// AgentCredential is an independently revocable API credential. TokenHash is
// the only form of the secret token retained by Envo.
type AgentCredential struct {
	ID          uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	AgentID     uuid.UUID  `gorm:"type:uuid;not null;index" json:"agent_id"`
	Name        string     `gorm:"type:varchar(120);not null" json:"name"`
	TokenHash   string     `gorm:"type:char(64);uniqueIndex;not null" json:"-"`
	TokenPrefix string     `gorm:"type:varchar(32);not null" json:"token_prefix"`
	ExpiresAt   *time.Time `gorm:"index" json:"expires_at,omitempty"`
	LastUsedAt  *time.Time `json:"last_used_at,omitempty"`
	RevokedAt   *time.Time `gorm:"index" json:"revoked_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`

	Agent AgentIdentity `gorm:"foreignKey:AgentID" json:"-"`
}

func (c *AgentCredential) BeforeCreate(_ *gorm.DB) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	return nil
}

// AgentGrant authorizes one capability for one environment. AllowedKeys is a
// JSON string array. AllowAllSecrets must be explicit; an empty key list never
// grants access by itself.
type AgentGrant struct {
	ID              uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	AgentID         uuid.UUID      `gorm:"type:uuid;not null;index" json:"agent_id"`
	EnvironmentID   uuid.UUID      `gorm:"type:uuid;not null;index" json:"environment_id"`
	Capability      string         `gorm:"type:varchar(50);not null;default:secrets.inject" json:"capability"`
	AllowedKeys     datatypes.JSON `gorm:"type:jsonb;not null;default:'[]'" json:"allowed_keys"`
	AllowAllSecrets bool           `gorm:"not null;default:false" json:"allow_all_secrets"`
	ApprovalMode    string         `gorm:"type:varchar(20);not null;default:always" json:"approval_mode"`
	MaxLeaseSeconds int            `gorm:"not null;default:300" json:"max_lease_seconds"`
	ExpiresAt       *time.Time     `gorm:"index" json:"expires_at,omitempty"`
	RevokedAt       *time.Time     `gorm:"index" json:"revoked_at,omitempty"`
	CreatedBy       uuid.UUID      `gorm:"type:uuid;not null;index" json:"created_by"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`

	Agent       AgentIdentity `gorm:"foreignKey:AgentID" json:"-"`
	Environment Environment   `gorm:"foreignKey:EnvironmentID" json:"environment,omitempty"`
	Creator     User          `gorm:"foreignKey:CreatedBy" json:"-"`
}

// AgentAccessRequest is a one-time authorization to deliver an exact set of
// secret values. It binds approval to an agent credential and request context
// so approval cannot be replayed with broader parameters.
type AgentAccessRequest struct {
	ID                 uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	OrgID              uuid.UUID      `gorm:"type:uuid;not null;index" json:"org_id"`
	AgentID            uuid.UUID      `gorm:"type:uuid;not null;index" json:"agent_id"`
	CredentialID       uuid.UUID      `gorm:"type:uuid;not null;index" json:"credential_id"`
	EnvironmentID      uuid.UUID      `gorm:"type:uuid;not null;index" json:"environment_id"`
	GrantIDs           datatypes.JSON `gorm:"type:jsonb;not null;default:'[]'" json:"grant_ids"`
	RequestedKeys      datatypes.JSON `gorm:"type:jsonb;not null;default:'[]'" json:"requested_keys"`
	AllowAllSecrets    bool           `gorm:"not null;default:false" json:"allow_all_secrets"`
	Purpose            string         `gorm:"type:varchar(200);not null" json:"purpose"`
	ExternalSessionID  string         `gorm:"type:varchar(200);not null" json:"session_id"`
	RequestFingerprint string         `gorm:"type:char(64);not null;default:'';index" json:"-"`
	Status             string         `gorm:"type:varchar(20);not null;index" json:"status"`
	DecisionReason     string         `gorm:"type:varchar(500)" json:"decision_reason,omitempty"`
	DecidedBy          *uuid.UUID     `gorm:"type:uuid;index" json:"decided_by,omitempty"`
	DecidedAt          *time.Time     `json:"decided_at,omitempty"`
	ExpiresAt          time.Time      `gorm:"not null;index" json:"expires_at"`
	UsedAt             *time.Time     `json:"used_at,omitempty"`
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`

	Agent       AgentIdentity   `gorm:"foreignKey:AgentID" json:"agent,omitempty"`
	Credential  AgentCredential `gorm:"foreignKey:CredentialID" json:"credential,omitempty"`
	Environment Environment     `gorm:"foreignKey:EnvironmentID" json:"environment,omitempty"`
	Approver    *User           `gorm:"foreignKey:DecidedBy" json:"approver,omitempty"`
}

func (r *AgentAccessRequest) BeforeCreate(_ *gorm.DB) error {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return nil
}

func (g *AgentGrant) BeforeCreate(_ *gorm.DB) error {
	if g.ID == uuid.Nil {
		g.ID = uuid.New()
	}
	if g.Capability == "" {
		g.Capability = AgentCapabilitySecretsInject
	}
	if g.ApprovalMode == "" {
		g.ApprovalMode = AgentApprovalAlways
	}
	if g.MaxLeaseSeconds <= 0 {
		g.MaxLeaseSeconds = 300
	}
	if len(g.AllowedKeys) == 0 {
		g.AllowedKeys = datatypes.JSON([]byte("[]"))
	}
	return nil
}
