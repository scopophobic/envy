package services

import (
	"fmt"
	"strings"

	"github.com/envo/backend/internal/database"
	"github.com/envo/backend/internal/models"
	"github.com/google/uuid"
)

type AdminService struct{}

func NewAdminService() *AdminService { return &AdminService{} }

func (s *AdminService) ListUsers(query string, limit int) ([]models.User, error) {
	db := database.GetDB()
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	var users []models.User
	q := db.Model(&models.User{}).Order("created_at DESC").Limit(limit)
	query = strings.TrimSpace(query)
	if query != "" {
		like := "%" + strings.ToLower(query) + "%"
		q = q.Where("LOWER(email) LIKE ? OR LOWER(name) LIKE ?", like, like)
	}
	if err := q.Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

func (s *AdminService) UpdateUserTier(userID uuid.UUID, tier string) (*models.User, error) {
	tier = strings.TrimSpace(strings.ToLower(tier))
	if tier != "free" && tier != "starter" && tier != "team" {
		return nil, fmt.Errorf("invalid tier")
	}
	db := database.GetDB()
	var user models.User
	if err := db.First(&user, userID).Error; err != nil {
		return nil, err
	}
	user.SubscriptionTier = tier
	if tier == "free" {
		user.SubscriptionStatus = string(models.StatusActive)
	}
	if err := db.Save(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

