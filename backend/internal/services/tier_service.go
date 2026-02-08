package services

import (
	"fmt"

	"github.com/envo/backend/internal/database"
	"github.com/envo/backend/internal/models"
	"github.com/google/uuid"
)

// TierService handles tier limit enforcement
type TierService struct{}

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

	// Count user's owned organizations
	var count int64
	if err := db.Model(&models.Organization{}).Where("owner_id = ?", userID).Count(&count).Error; err != nil {
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

	// Get max devs limit for owner's tier
	maxDevs, err := s.GetLimit(org.Owner.SubscriptionTier, models.LimitTypeMaxDevs)
	if err != nil {
		return false, err
	}

	// Unlimited
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

	// Get max projects limit for owner's tier
	maxProjects, err := s.GetLimit(org.Owner.SubscriptionTier, models.LimitTypeMaxProjects)
	if err != nil {
		return false, err
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
	db := database.GetDB()

	// Get environment with project and organization
	var env models.Environment
	if err := db.Preload("Project.Organization.Owner").First(&env, envID).Error; err != nil {
		return false, err
	}

	// Get max secrets limit for owner's tier
	maxSecrets, err := s.GetLimit(env.Project.Organization.Owner.SubscriptionTier, models.LimitTypeMaxSecretsPerEnv)
	if err != nil {
		return false, err
	}

	// Unlimited
	if maxSecrets == models.UnlimitedValue {
		return true, nil
	}

	// Count environment secrets
	var count int64
	if err := db.Model(&models.Secret{}).Where("environment_id = ?", envID).Count(&count).Error; err != nil {
		return false, err
	}

	return int(count) < maxSecrets, nil
}
