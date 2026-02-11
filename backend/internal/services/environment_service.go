package services

import (
	"github.com/envo/backend/internal/database"
	"github.com/envo/backend/internal/models"
	"github.com/google/uuid"
)

// EnvironmentService handles environment business logic
type EnvironmentService struct{}

// NewEnvironmentService creates a new environment service
func NewEnvironmentService() *EnvironmentService {
	return &EnvironmentService{}
}

// CreateEnvironment creates a new environment within a project
func (s *EnvironmentService) CreateEnvironment(projectID uuid.UUID, name string) (*models.Environment, error) {
	db := database.GetDB()

	env := &models.Environment{
		ProjectID: projectID,
		Name:      name,
	}

	if err := db.Create(env).Error; err != nil {
		return nil, err
	}

	return env, nil
}

// ListProjectEnvironments lists all environments for a project
func (s *EnvironmentService) ListProjectEnvironments(projectID uuid.UUID) ([]models.Environment, error) {
	db := database.GetDB()

	var envs []models.Environment
	if err := db.Where("project_id = ?", projectID).
		Order("created_at ASC").
		Find(&envs).Error; err != nil {
		return nil, err
	}

	return envs, nil
}

// UpdateEnvironment updates an environment's name
func (s *EnvironmentService) UpdateEnvironment(envID uuid.UUID, name string) (*models.Environment, error) {
	db := database.GetDB()

	var env models.Environment
	if err := db.First(&env, envID).Error; err != nil {
		return nil, err
	}

	env.Name = name
	if err := db.Save(&env).Error; err != nil {
		return nil, err
	}

	return &env, nil
}

// DeleteEnvironment deletes an environment and its secrets
func (s *EnvironmentService) DeleteEnvironment(envID uuid.UUID) error {
	db := database.GetDB()

	// Delete secrets first, then the environment
	if err := db.Where("environment_id = ?", envID).Delete(&models.Secret{}).Error; err != nil {
		return err
	}

	if err := db.Delete(&models.Environment{}, envID).Error; err != nil {
		return err
	}

	return nil
}

