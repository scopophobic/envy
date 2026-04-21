package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// OwnerType distinguishes personal workspaces from team/org workspaces.
type OwnerType string

const (
	OwnerTypePersonal OwnerType = "personal"
	OwnerTypeOrg      OwnerType = "org"
)

// Organization represents a workspace (personal or team).
// Personal workspaces are auto-created on signup and skip all team/invite UI.
type Organization struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	OwnerID   uuid.UUID `gorm:"type:uuid;not null;index" json:"owner_id"`
	Name      string    `gorm:"type:varchar(255);not null" json:"name"`
	OwnerType OwnerType `gorm:"type:varchar(20);not null;default:'org'" json:"owner_type"`

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

// IsPersonal returns true if this is a personal (non-team) workspace.
func (o *Organization) IsPersonal() bool {
	return o.OwnerType == OwnerTypePersonal
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

// OrgInvitationStatus tracks lifecycle of team invitations.
type OrgInvitationStatus string

const (
	InvitationPending  OrgInvitationStatus = "pending"
	InvitationAccepted OrgInvitationStatus = "accepted"
	InvitationRevoked  OrgInvitationStatus = "revoked"
	InvitationExpired  OrgInvitationStatus = "expired"
)

// OrgInvitation represents a pending invite sent to an email.
// Membership is created only after acceptance.
type OrgInvitation struct {
	ID              uuid.UUID           `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	OrgID           uuid.UUID           `gorm:"type:uuid;not null;index" json:"org_id"`
	InvitedByUserID uuid.UUID           `gorm:"type:uuid;not null;index" json:"invited_by_user_id"`
	RoleID          uuid.UUID           `gorm:"type:uuid;not null;index" json:"role_id"`
	Email           string              `gorm:"type:varchar(255);not null;index" json:"email"`
	TokenHash       string              `gorm:"type:varchar(255);not null;index" json:"-"`
	Status          OrgInvitationStatus `gorm:"type:varchar(20);not null;default:'pending';index" json:"status"`
	ExpiresAt       time.Time           `gorm:"not null;index" json:"expires_at"`
	AcceptedAt      *time.Time          `json:"accepted_at,omitempty"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	Organization Organization `gorm:"foreignKey:OrgID" json:"organization,omitempty"`
	InvitedBy    User         `gorm:"foreignKey:InvitedByUserID" json:"invited_by,omitempty"`
	Role         Role         `gorm:"foreignKey:RoleID" json:"role,omitempty"`
}

func (oi *OrgInvitation) BeforeCreate(tx *gorm.DB) error {
	if oi.ID == uuid.Nil {
		oi.ID = uuid.New()
	}
	return nil
}

func (OrgInvitation) TableName() string {
	return "org_invitations"
}
