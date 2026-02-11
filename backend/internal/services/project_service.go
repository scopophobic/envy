package services

import (
	"fmt"

	"github.com/envo/backend/internal/database"
	"github.com/envo/backend/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ProjectService handles project business logic
type ProjectService struct {
	tierService *TierService
}

// NewProjectService creates a new project service
func NewProjectService(tierService *TierService) *ProjectService {
	return &ProjectService{
		tierService: tierService,
	}
}

// CreateProject creates a new project within an organization
func (s *ProjectService) CreateProject(orgID uuid.UUID, name string, description *string) (*models.Project, error) {
	db := database.GetDB()

	// Check tier limits
	canCreate, err := s.tierService.CanCreateProject(orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to check tier limits: %w", err)
	}
	if !canCreate {
		return nil, fmt.Errorf("project limit reached for this organization")
	}

	project := &models.Project{
		OrgID:       orgID,
		Name:        name,
		Description: description,
	}

	if err := db.Create(project).Error; err != nil {
		return nil, fmt.Errorf("failed to create project: %w", err)
	}

	// Preload relations for response
	if err := db.Preload("Organization").First(project, project.ID).Error; err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}

	return project, nil
}

// ListOrgProjects lists all projects for an organization
func (s *ProjectService) ListOrgProjects(orgID uuid.UUID) ([]models.Project, error) {
	db := database.GetDB()

	var projects []models.Project
	if err := db.Where("org_id = ?", orgID).
		Order("created_at ASC").
		Find(&projects).Error; err != nil {
		return nil, err
	}

	return projects, nil
}

// GetProject retrieves a project by ID
func (s *ProjectService) GetProject(projectID uuid.UUID) (*models.Project, error) {
	db := database.GetDB()

	var project models.Project
	if err := db.Preload("Organization").
		First(&project, projectID).Error; err != nil {
		return nil, err
	}

	return &project, nil
}

// UpdateProject updates a project's basic fields
func (s *ProjectService) UpdateProject(projectID uuid.UUID, name string, description *string) (*models.Project, error) {
	db := database.GetDB()

	var project models.Project
	if err := db.First(&project, projectID).Error; err != nil {
		return nil, err
	}

	project.Name = name
	project.Description = description

	if err := db.Save(&project).Error; err != nil {
		return nil, err
	}

	return &project, nil
}

// DeleteProject deletes a project and its environments and secrets (via cascading)
func (s *ProjectService) DeleteProject(projectID uuid.UUID) error {
	db := database.GetDB()

	return db.Transaction(func(tx *gorm.DB) error {
		// Delete environments and their secrets via explicit deletes to keep behavior clear
		var envs []models.Environment
		if err := tx.Where("project_id = ?", projectID).Find(&envs).Error; err != nil {
			return err
		}

		for _, env := range envs {
			if err := tx.Where("environment_id = ?", env.ID).Delete(&models.Secret{}).Error; err != nil {
				return err
			}
		}

		if err := tx.Where("project_id = ?", projectID).Delete(&models.Environment{}).Error; err != nil {
			return err
		}

		// Finally delete the project
		if err := tx.Delete(&models.Project{}, projectID).Error; err != nil {
			return err
		}

		return nil
	})
}

