package services

import (
	"fmt"

	"github.com/envo/backend/internal/database"
	"github.com/envo/backend/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// OrgService handles organization business logic
type OrgService struct {
	tierService *TierService
}

// NewOrgService creates a new organization service
func NewOrgService(tierService *TierService) *OrgService {
	return &OrgService{
		tierService: tierService,
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

	// Create organization
	org := &models.Organization{
		Name:    name,
		OwnerID: ownerID,
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

// ListUserOrganizations lists all organizations a user has access to
func (s *OrgService) ListUserOrganizations(userID uuid.UUID) ([]models.Organization, error) {
	db := database.GetDB()

	var orgs []models.Organization
	
	// Get organizations where user is a member
	err := db.Joins("JOIN org_members ON org_members.org_id = organizations.id").
		Where("org_members.user_id = ?", userID).
		Preload("Owner").
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

// DeleteOrganization deletes an organization
func (s *OrgService) DeleteOrganization(orgID uuid.UUID) error {
	db := database.GetDB()

	return db.Transaction(func(tx *gorm.DB) error {
		// Delete all members
		if err := tx.Where("org_id = ?", orgID).Delete(&models.OrgMember{}).Error; err != nil {
			return err
		}

		// Delete organization
		if err := tx.Delete(&models.Organization{}, orgID).Error; err != nil {
			return err
		}

		return nil
	})
}

// InviteMember adds a user to an organization
func (s *OrgService) InviteMember(orgID uuid.UUID, email string, roleName string) (*models.OrgMember, error) {
	db := database.GetDB()

	// Check tier limits
	canInvite, err := s.tierService.CanInviteMember(orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to check tier limits: %w", err)
	}
	if !canInvite {
		return nil, fmt.Errorf("member limit reached for this organization")
	}

	// Find user by email
	var user models.User
	if err := db.Where("email = ?", email).First(&user).Error; err != nil {
		return nil, fmt.Errorf("user not found with email: %s", email)
	}

	// Check if already a member
	var existing models.OrgMember
	err = db.Where("org_id = ? AND user_id = ?", orgID, user.ID).First(&existing).Error
	if err == nil {
		return nil, fmt.Errorf("user is already a member of this organization")
	}

	// Get role
	var role models.Role
	if err := db.Where("name = ? AND (is_system_role = ? OR org_id = ?)", roleName, true, orgID).First(&role).Error; err != nil {
		return nil, fmt.Errorf("role not found: %s", roleName)
	}

	// Create membership
	member := &models.OrgMember{
		OrgID:  orgID,
		UserID: user.ID,
		RoleID: role.ID,
	}

	if err := db.Create(member).Error; err != nil {
		return nil, err
	}

	// Load relationships
	db.Preload("User").Preload("Role").First(member, member.ID)

	return member, nil
}

// UpdateMemberRole updates a member's role
func (s *OrgService) UpdateMemberRole(memberID uuid.UUID, roleName string) (*models.OrgMember, error) {
	db := database.GetDB()

	var member models.OrgMember
	if err := db.First(&member, memberID).Error; err != nil {
		return nil, err
	}

	// Get new role
	var role models.Role
	if err := db.Where("name = ? AND (is_system_role = ? OR org_id = ?)", roleName, true, member.OrgID).First(&role).Error; err != nil {
		return nil, fmt.Errorf("role not found: %s", roleName)
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
