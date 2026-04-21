package services

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/envo/backend/internal/database"
	"github.com/envo/backend/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// OrgService handles organization business logic
type OrgService struct {
	tierService *TierService
	emailSender EmailSender
	frontendURL string
	inviteTTL   time.Duration
}

// NewOrgService creates a new organization service
func NewOrgService(tierService *TierService, emailSender EmailSender, frontendURL string, inviteTTLHours int) *OrgService {
	if inviteTTLHours <= 0 {
		inviteTTLHours = 168
	}
	if emailSender == nil {
		emailSender = &LogEmailSender{}
	}
	return &OrgService{
		tierService: tierService,
		emailSender: emailSender,
		frontendURL: strings.TrimRight(frontendURL, "/"),
		inviteTTL:   time.Duration(inviteTTLHours) * time.Hour,
	}
}

// CreateOrganization creates a new organization
func (s *OrgService) CreateOrganization(ownerID uuid.UUID, name string) (*models.Organization, error) {
	db := database.GetDB()

	// Check tier limits
	canCreate, err := s.tierService.CanCreateOrganization(ownerID)
	if err != nil {
		return nil, fmt.Errorf("failed to check tier limits: %w", err)
	}
	if !canCreate {
		return nil, fmt.Errorf("organization limit reached for your tier")
	}

	// Get Owner system role
	var ownerRole models.Role
	if err := db.Where("name = ? AND is_system_role = ?", models.RoleOwner, true).First(&ownerRole).Error; err != nil {
		return nil, fmt.Errorf("failed to find owner role: %w", err)
	}

	// Create organization (explicit creation is always an org workspace, not personal)
	org := &models.Organization{
		Name:      name,
		OwnerID:   ownerID,
		OwnerType: models.OwnerTypeOrg,
	}

	err = db.Transaction(func(tx *gorm.DB) error {
		// Create organization
		if err := tx.Create(org).Error; err != nil {
			return err
		}

		// Add owner as member with Owner role
		member := &models.OrgMember{
			OrgID:  org.ID,
			UserID: ownerID,
			RoleID: ownerRole.ID,
		}
		if err := tx.Create(member).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create organization: %w", err)
	}

	// Load owner for response
	db.Preload("Owner").First(org, org.ID)

	return org, nil
}

// GetOrganization retrieves an organization by ID
func (s *OrgService) GetOrganization(orgID uuid.UUID) (*models.Organization, error) {
	db := database.GetDB()

	var org models.Organization
	if err := db.Preload("Owner").Preload("Members.User").Preload("Members.Role").First(&org, orgID).Error; err != nil {
		return nil, err
	}

	return &org, nil
}

// ListUserOrganizations lists all workspaces a user has access to.
// Personal workspaces appear first, then org workspaces sorted by name.
func (s *OrgService) ListUserOrganizations(userID uuid.UUID) ([]models.Organization, error) {
	db := database.GetDB()

	var orgs []models.Organization

	err := db.Joins("JOIN org_members ON org_members.org_id = organizations.id").
		Where("org_members.user_id = ?", userID).
		Preload("Owner").
		Order("CASE WHEN organizations.owner_type = 'personal' THEN 0 ELSE 1 END, organizations.name ASC").
		Find(&orgs).Error

	if err != nil {
		return nil, err
	}

	return orgs, nil
}

// UpdateOrganization updates an organization
func (s *OrgService) UpdateOrganization(orgID uuid.UUID, name string) (*models.Organization, error) {
	db := database.GetDB()

	var org models.Organization
	if err := db.First(&org, orgID).Error; err != nil {
		return nil, err
	}

	org.Name = name
	if err := db.Save(&org).Error; err != nil {
		return nil, err
	}

	return &org, nil
}

// DeleteOrganization deletes an organization. Personal workspaces cannot be deleted.
func (s *OrgService) DeleteOrganization(orgID uuid.UUID) error {
	db := database.GetDB()

	var org models.Organization
	if err := db.First(&org, orgID).Error; err != nil {
		return err
	}
	if org.IsPersonal() {
		return fmt.Errorf("personal workspaces cannot be deleted")
	}

	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("org_id = ?", orgID).Delete(&models.OrgMember{}).Error; err != nil {
			return err
		}

		if err := tx.Delete(&models.Organization{}, orgID).Error; err != nil {
			return err
		}

		return nil
	})
}

// InviteMember creates a pending invitation and emails the recipient.
func (s *OrgService) InviteMember(orgID uuid.UUID, invitedBy uuid.UUID, email string, roleID *uuid.UUID, roleName string) (*models.OrgInvitation, string, string, error) {
	db := database.GetDB()
	normalizedEmail := strings.ToLower(strings.TrimSpace(email))
	if normalizedEmail == "" {
		return nil, "", "", fmt.Errorf("email is required")
	}

	// Check tier limits
	canInvite, err := s.tierService.CanInviteMember(orgID)
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to check tier limits: %w", err)
	}
	if !canInvite {
		return nil, "", "", fmt.Errorf("member limit reached for this organization")
	}

	var org models.Organization
	if err := db.First(&org, orgID).Error; err != nil {
		return nil, "", "", fmt.Errorf("organization not found")
	}

	// Role can be provided by id or by name
	var role models.Role
	if roleID != nil && *roleID != uuid.Nil {
		if err := db.Where("id = ? AND (is_system_role = ? OR org_id = ?)", *roleID, true, orgID).First(&role).Error; err != nil {
			return nil, "", "", fmt.Errorf("role not found")
		}
	} else {
		if err := db.Where("name = ? AND (is_system_role = ? OR org_id = ?)", roleName, true, orgID).First(&role).Error; err != nil {
			return nil, "", "", fmt.Errorf("role not found: %s", roleName)
		}
	}

	// Existing member check (if account exists)
	var existingUser models.User
	if err := db.Where("LOWER(email) = ?", normalizedEmail).First(&existingUser).Error; err == nil {
		var existing models.OrgMember
		if err := db.Where("org_id = ? AND user_id = ?", orgID, existingUser.ID).First(&existing).Error; err == nil {
			return nil, "", "", fmt.Errorf("user is already a member of this organization")
		}
	}

	rawToken, tokenHash, err := generateInviteToken()
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to generate invite token")
	}

	invitation := &models.OrgInvitation{
		OrgID:           orgID,
		InvitedByUserID: invitedBy,
		RoleID:          role.ID,
		Email:           normalizedEmail,
		TokenHash:       tokenHash,
		Status:          models.InvitationPending,
		ExpiresAt:       time.Now().Add(s.inviteTTL),
	}

	// Replace any pending invite for same org/email
	if err := db.Where("org_id = ? AND LOWER(email) = ? AND status = ?", orgID, normalizedEmail, models.InvitationPending).
		Delete(&models.OrgInvitation{}).Error; err != nil {
		return nil, "", "", err
	}
	if err := db.Create(invitation).Error; err != nil {
		return nil, "", "", err
	}

	var inviter models.User
	_ = db.First(&inviter, invitedBy).Error
	inviteURL := fmt.Sprintf("%s/invite/accept?token=%s", s.frontendURL, rawToken)
	emailWarning := ""
	if err := s.emailSender.SendInvite(normalizedEmail, org.Name, inviter.Name, role.Name, inviteURL); err != nil {
		// keep invitation; allow resend from UI
		emailWarning = fmt.Sprintf("invitation created, but email delivery failed: %v", err)
	}
	db.Preload("Role").Preload("InvitedBy").First(invitation, invitation.ID)

	return invitation, inviteURL, emailWarning, nil
}

// UpdateMemberRole updates a member's role
func (s *OrgService) UpdateMemberRole(memberID uuid.UUID, roleID *uuid.UUID, roleName string) (*models.OrgMember, error) {
	db := database.GetDB()

	var member models.OrgMember
	if err := db.First(&member, memberID).Error; err != nil {
		return nil, err
	}

	role, err := s.resolveRole(member.OrgID, roleID, roleName)
	if err != nil {
		return nil, err
	}

	member.RoleID = role.ID
	if err := db.Save(&member).Error; err != nil {
		return nil, err
	}

	// Load relationships
	db.Preload("User").Preload("Role").First(&member, member.ID)

	return &member, nil
}

// RemoveMember removes a user from an organization
func (s *OrgService) RemoveMember(memberID uuid.UUID) error {
	db := database.GetDB()

	return db.Delete(&models.OrgMember{}, memberID).Error
}

// CheckUserAccess checks if a user has access to an organization
func (s *OrgService) CheckUserAccess(userID uuid.UUID, orgID uuid.UUID) (bool, error) {
	db := database.GetDB()

	var count int64
	err := db.Model(&models.OrgMember{}).
		Where("org_id = ? AND user_id = ?", orgID, userID).
		Count(&count).Error

	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func (s *OrgService) resolveRole(orgID uuid.UUID, roleID *uuid.UUID, roleName string) (*models.Role, error) {
	db := database.GetDB()
	var role models.Role
	if roleID != nil && *roleID != uuid.Nil {
		if err := db.Where("id = ? AND (is_system_role = ? OR org_id = ?)", *roleID, true, orgID).First(&role).Error; err != nil {
			return nil, fmt.Errorf("role not found")
		}
		return &role, nil
	}
	if strings.TrimSpace(roleName) == "" {
		return nil, fmt.Errorf("role is required")
	}
	if err := db.Where("name = ? AND (is_system_role = ? OR org_id = ?)", roleName, true, orgID).First(&role).Error; err != nil {
		return nil, fmt.Errorf("role not found: %s", roleName)
	}
	return &role, nil
}

func generateInviteToken() (raw string, hash string, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", err
	}
	raw = hex.EncodeToString(b)
	sum := sha256.Sum256([]byte(raw))
	hash = hex.EncodeToString(sum[:])
	return raw, hash, nil
}

func invitationTokenHash(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

// ListRoles returns system roles and org custom roles.
func (s *OrgService) ListRoles(orgID uuid.UUID) ([]models.Role, error) {
	db := database.GetDB()
	var roles []models.Role
	if err := db.
		Where("is_system_role = ? OR org_id = ?", true, orgID).
		Preload("Permissions").
		Order("is_system_role DESC, name ASC").
		Find(&roles).Error; err != nil {
		return nil, err
	}
	return roles, nil
}

// CreateRole creates a custom role for the org and attaches permissions.
func (s *OrgService) CreateRole(orgID uuid.UUID, name string, permissionNames []string) (*models.Role, error) {
	db := database.GetDB()
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("role name is required")
	}
	if strings.EqualFold(name, models.RoleOwner) || strings.EqualFold(name, models.RoleAdmin) ||
		strings.EqualFold(name, models.RoleSecretManager) || strings.EqualFold(name, models.RoleDeveloper) ||
		strings.EqualFold(name, models.RoleViewer) {
		return nil, fmt.Errorf("role name conflicts with system role")
	}
	var existing models.Role
	if err := db.Where("org_id = ? AND LOWER(name) = LOWER(?)", orgID, name).First(&existing).Error; err == nil {
		return nil, fmt.Errorf("role already exists")
	}
	role := &models.Role{Name: name, OrgID: &orgID, IsSystemRole: false}
	if err := db.Create(role).Error; err != nil {
		return nil, err
	}
	if len(permissionNames) > 0 {
		var perms []models.Permission
		if err := db.Where("name IN ?", permissionNames).Find(&perms).Error; err != nil {
			return nil, err
		}
		if err := db.Model(role).Association("Permissions").Replace(perms); err != nil {
			return nil, err
		}
	}
	if err := db.Preload("Permissions").First(role, role.ID).Error; err != nil {
		return nil, err
	}
	return role, nil
}

func (s *OrgService) UpdateRole(orgID uuid.UUID, roleID uuid.UUID, name string, permissionNames []string) (*models.Role, error) {
	db := database.GetDB()
	var role models.Role
	if err := db.Where("id = ? AND org_id = ?", roleID, orgID).First(&role).Error; err != nil {
		return nil, fmt.Errorf("role not found")
	}
	if role.IsSystemRole {
		return nil, fmt.Errorf("system roles cannot be modified")
	}
	if strings.TrimSpace(name) != "" {
		role.Name = strings.TrimSpace(name)
		if err := db.Save(&role).Error; err != nil {
			return nil, err
		}
	}
	var perms []models.Permission
	if len(permissionNames) > 0 {
		if err := db.Where("name IN ?", permissionNames).Find(&perms).Error; err != nil {
			return nil, err
		}
	}
	if err := db.Model(&role).Association("Permissions").Replace(perms); err != nil {
		return nil, err
	}
	if err := db.Preload("Permissions").First(&role, role.ID).Error; err != nil {
		return nil, err
	}
	return &role, nil
}

func (s *OrgService) DeleteRole(orgID uuid.UUID, roleID uuid.UUID, replacementRoleID *uuid.UUID) error {
	db := database.GetDB()
	var role models.Role
	if err := db.Where("id = ? AND org_id = ?", roleID, orgID).First(&role).Error; err != nil {
		return fmt.Errorf("role not found")
	}
	if role.IsSystemRole {
		return fmt.Errorf("system roles cannot be deleted")
	}
	return db.Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Model(&models.OrgMember{}).Where("role_id = ?", roleID).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			if replacementRoleID == nil || *replacementRoleID == uuid.Nil {
				return fmt.Errorf("role is assigned to members; provide replacement_role_id")
			}
			if err := tx.Model(&models.OrgMember{}).Where("role_id = ?", roleID).Update("role_id", *replacementRoleID).Error; err != nil {
				return err
			}
		}
		return tx.Delete(&models.Role{}, roleID).Error
	})
}

func (s *OrgService) ListInvitations(orgID uuid.UUID) ([]models.OrgInvitation, error) {
	var invitations []models.OrgInvitation
	db := database.GetDB()
	if err := db.Where("org_id = ?", orgID).
		Preload("Role").
		Preload("InvitedBy").
		Order("created_at DESC").
		Find(&invitations).Error; err != nil {
		return nil, err
	}
	return invitations, nil
}

func (s *OrgService) ResendInvitation(orgID, inviteID, actorID uuid.UUID) (string, error) {
	db := database.GetDB()
	var invite models.OrgInvitation
	if err := db.Where("id = ? AND org_id = ?", inviteID, orgID).Preload("Role").First(&invite).Error; err != nil {
		return "", fmt.Errorf("invitation not found")
	}
	if invite.Status != models.InvitationPending {
		return "", fmt.Errorf("only pending invitations can be resent")
	}
	raw, tokenHash, err := generateInviteToken()
	if err != nil {
		return "", err
	}
	invite.TokenHash = tokenHash
	invite.ExpiresAt = time.Now().Add(s.inviteTTL)
	invite.InvitedByUserID = actorID
	if err := db.Save(&invite).Error; err != nil {
		return "", err
	}
	var org models.Organization
	var inviter models.User
	_ = db.First(&org, orgID).Error
	_ = db.First(&inviter, actorID).Error
	url := fmt.Sprintf("%s/invite/accept?token=%s", s.frontendURL, raw)
	if err := s.emailSender.SendInvite(invite.Email, org.Name, inviter.Name, invite.Role.Name, url); err != nil {
		return "", err
	}
	return url, nil
}

func (s *OrgService) RevokeInvitation(orgID, inviteID uuid.UUID) error {
	db := database.GetDB()
	return db.Model(&models.OrgInvitation{}).
		Where("id = ? AND org_id = ?", inviteID, orgID).
		Updates(map[string]interface{}{"status": models.InvitationRevoked}).Error
}

func (s *OrgService) AcceptInvitation(userID uuid.UUID, rawToken string) (*models.OrgMember, error) {
	db := database.GetDB()
	tokenHash := invitationTokenHash(strings.TrimSpace(rawToken))
	var invite models.OrgInvitation
	if err := db.Where("token_hash = ?", tokenHash).Preload("Role").First(&invite).Error; err != nil {
		return nil, fmt.Errorf("invitation not found")
	}
	if invite.Status != models.InvitationPending {
		return nil, fmt.Errorf("invitation is no longer active")
	}
	if time.Now().After(invite.ExpiresAt) {
		_ = db.Model(&invite).Update("status", models.InvitationExpired).Error
		return nil, fmt.Errorf("invitation expired")
	}
	var user models.User
	if err := db.First(&user, userID).Error; err != nil {
		return nil, err
	}
	if !strings.EqualFold(strings.TrimSpace(user.Email), strings.TrimSpace(invite.Email)) {
		return nil, fmt.Errorf("invitation email does not match signed-in user")
	}
	member := &models.OrgMember{}
	err := db.Transaction(func(tx *gorm.DB) error {
		var existing models.OrgMember
		if err := tx.Where("org_id = ? AND user_id = ?", invite.OrgID, user.ID).First(&existing).Error; err == nil {
			member = &existing
		} else {
			member = &models.OrgMember{
				OrgID:  invite.OrgID,
				UserID: user.ID,
				RoleID: invite.RoleID,
			}
			if err := tx.Create(member).Error; err != nil {
				return err
			}
		}
		now := time.Now()
		return tx.Model(&models.OrgInvitation{}).Where("id = ?", invite.ID).Updates(map[string]interface{}{
			"status":      models.InvitationAccepted,
			"accepted_at": &now,
		}).Error
	})
	if err != nil {
		return nil, err
	}
	if err := db.Preload("Role").Preload("User").First(member, member.ID).Error; err != nil {
		return nil, err
	}
	return member, nil
}

// AcceptInvitationByID accepts a pending invitation for the signed-in user by invite id.
func (s *OrgService) AcceptInvitationByID(userID, inviteID uuid.UUID) (*models.OrgMember, error) {
	db := database.GetDB()
	var user models.User
	if err := db.First(&user, userID).Error; err != nil {
		return nil, err
	}
	var invite models.OrgInvitation
	if err := db.Where("id = ?", inviteID).Preload("Role").First(&invite).Error; err != nil {
		return nil, fmt.Errorf("invitation not found")
	}
	if invite.Status != models.InvitationPending {
		return nil, fmt.Errorf("invitation is no longer pending")
	}
	if time.Now().After(invite.ExpiresAt) {
		_ = db.Model(&invite).Update("status", models.InvitationExpired).Error
		return nil, fmt.Errorf("invitation expired")
	}
	if !strings.EqualFold(strings.TrimSpace(user.Email), strings.TrimSpace(invite.Email)) {
		return nil, fmt.Errorf("invitation email does not match signed-in user")
	}

	var created models.OrgMember
	err := db.Transaction(func(tx *gorm.DB) error {
		var existing models.OrgMember
		if err := tx.Where("org_id = ? AND user_id = ?", invite.OrgID, user.ID).First(&existing).Error; err == nil {
			created = existing
		} else {
			created = models.OrgMember{
				OrgID:  invite.OrgID,
				UserID: user.ID,
				RoleID: invite.RoleID,
			}
			if err := tx.Create(&created).Error; err != nil {
				return err
			}
		}
		return tx.Model(&models.OrgInvitation{}).Where("id = ?", invite.ID).Updates(map[string]interface{}{
			"status":      models.InvitationAccepted,
			"accepted_at": time.Now(),
		}).Error
	})
	if err != nil {
		return nil, err
	}
	if err := db.Preload("User").Preload("Role").First(&created, created.ID).Error; err != nil {
		return nil, err
	}
	return &created, nil
}

// ListInvitationsForUser returns pending invites that match the user's email.
func (s *OrgService) ListInvitationsForUser(userID uuid.UUID) ([]models.OrgInvitation, error) {
	db := database.GetDB()
	var user models.User
	if err := db.First(&user, userID).Error; err != nil {
		return nil, err
	}
	var invites []models.OrgInvitation
	if err := db.
		Where("LOWER(email) = LOWER(?) AND status = ?", user.Email, models.InvitationPending).
		Where("expires_at > ?", time.Now()).
		Preload("Role").
		Preload("Organization").
		Preload("InvitedBy").
		Order("created_at DESC").
		Find(&invites).Error; err != nil {
		return nil, err
	}
	return invites, nil
}
