package services

import (
	"fmt"

	"github.com/envo/backend/internal/database"
	"github.com/envo/backend/internal/models"
	"github.com/google/uuid"
)

// TierService handles tier limit enforcement
type TierService struct{}

const (
	personalMaxProjects = 10
	personalMaxEnvs     = 20

	orgMaxProjects = 2
	orgMaxMembers  = 2
	orgMaxEnvs     = 10
)

// NewTierService creates a new tier service
func NewTierService() *TierService {
	return &TierService{}
}

// GetLimit retrieves a tier limit value
func (s *TierService) GetLimit(tier string, limitType string) (int, error) {
	db := database.GetDB()

	var limit models.TierLimit
	err := db.Where("tier = ? AND limit_type = ?", tier, limitType).First(&limit).Error
	if err != nil {
		return 0, fmt.Errorf("failed to get tier limit: %w", err)
	}

	return limit.LimitValue, nil
}

// CanCreateOrganization checks if user can create a new organization
func (s *TierService) CanCreateOrganization(userID uuid.UUID) (bool, error) {
	db := database.GetDB()

	// Get user with subscription tier
	var user models.User
	if err := db.First(&user, userID).Error; err != nil {
		return false, err
	}

	// Get max orgs limit for user's tier
	maxOrgs, err := s.GetLimit(user.SubscriptionTier, models.LimitTypeMaxOrgs)
	if err != nil {
		return false, err
	}

	// Unlimited
	if maxOrgs == models.UnlimitedValue {
		return true, nil
	}

	// Count user's owned org-type organizations (personal workspaces don't count)
	var count int64
	if err := db.Model(&models.Organization{}).Where("owner_id = ? AND owner_type = ?", userID, models.OwnerTypeOrg).Count(&count).Error; err != nil {
		return false, err
	}

	return int(count) < maxOrgs, nil
}

// CanInviteMember checks if organization can invite more members
func (s *TierService) CanInviteMember(orgID uuid.UUID) (bool, error) {
	db := database.GetDB()

	// Get organization with owner
	var org models.Organization
	if err := db.Preload("Owner").First(&org, orgID).Error; err != nil {
		return false, err
	}

	// Personal vaults don't support team member invites.
	if org.OwnerType == models.OwnerTypePersonal {
		return false, nil
	}
	// Team org policy: 2 members max.
	maxDevs := orgMaxMembers
	if maxDevs == models.UnlimitedValue {
		return true, nil
	}

	// Count organization members
	var count int64
	if err := db.Model(&models.OrgMember{}).Where("org_id = ?", orgID).Count(&count).Error; err != nil {
		return false, err
	}

	return int(count) < maxDevs, nil
}

// CanCreateProject checks if organization can create more projects
func (s *TierService) CanCreateProject(orgID uuid.UUID) (bool, error) {
	db := database.GetDB()

	// Get organization with owner
	var org models.Organization
	if err := db.Preload("Owner").First(&org, orgID).Error; err != nil {
		return false, err
	}

	maxProjects := orgMaxProjects
	if org.OwnerType == models.OwnerTypePersonal {
		maxProjects = personalMaxProjects
	}

	// Unlimited
	if maxProjects == models.UnlimitedValue {
		return true, nil
	}

	// Count organization projects
	var count int64
	if err := db.Model(&models.Project{}).Where("org_id = ?", orgID).Count(&count).Error; err != nil {
		return false, err
	}

	return int(count) < maxProjects, nil
}

// CanCreateSecret checks if environment can have more secrets
func (s *TierService) CanCreateSecret(envID uuid.UUID) (bool, error) {
	// Current workspace policy: unlimited secrets for personal and org workspaces.
	return true, nil
}

// CanCreateEnvironment checks if a workspace can have more environments.
func (s *TierService) CanCreateEnvironment(projectID uuid.UUID) (bool, error) {
	db := database.GetDB()

	var project models.Project
	if err := db.Preload("Organization").First(&project, projectID).Error; err != nil {
		return false, err
	}

	maxEnvs := orgMaxEnvs
	if project.Organization.OwnerType == models.OwnerTypePersonal {
		maxEnvs = personalMaxEnvs
	}
	if maxEnvs == models.UnlimitedValue {
		return true, nil
	}

	var count int64
	err := db.Model(&models.Environment{}).
		Joins("JOIN projects ON projects.id = environments.project_id").
		Where("projects.org_id = ?", project.OrgID).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return int(count) < maxEnvs, nil
}
