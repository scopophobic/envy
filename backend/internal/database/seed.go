package database

import (
	"log"

	"github.com/envo/backend/internal/models"
	"gorm.io/gorm"
)

// SeedInitialData seeds permissions, roles, and tier limits
func SeedInitialData(db *gorm.DB) error {
	log.Println("🌱 Seeding initial data...")

	// Seed permissions
	if err := seedPermissions(db); err != nil {
		return err
	}

	// Seed system roles
	if err := seedSystemRoles(db); err != nil {
		return err
	}

	// Seed tier limits
	if err := seedTierLimits(db); err != nil {
		return err
	}

	log.Println("✅ Initial data seeded successfully!")
	return nil
}

func seedPermissions(db *gorm.DB) error {
	permissions := []models.Permission{
		{Name: models.PermissionSecretsRead, Description: "View secrets"},
		{Name: models.PermissionSecretsCreate, Description: "Create new secrets"},
		{Name: models.PermissionSecretsUpdate, Description: "Update existing secrets"},
		{Name: models.PermissionSecretsDelete, Description: "Delete secrets"},
		{Name: models.PermissionProjectsManage, Description: "Create, edit, and delete projects"},
		{Name: models.PermissionEnvironmentsManage, Description: "Create, edit, and delete environments"},
		{Name: models.PermissionMembersInvite, Description: "Invite team members"},
		{Name: models.PermissionMembersManage, Description: "Change roles and remove members"},
		{Name: models.PermissionAuditView, Description: "View audit logs"},
		{Name: models.PermissionOrgManage, Description: "Edit organization settings and billing"},
	}

	for _, perm := range permissions {
		var existing models.Permission
		if err := db.Where("name = ?", perm.Name).First(&existing).Error; err == gorm.ErrRecordNotFound {
			if err := db.Create(&perm).Error; err != nil {
				return err
			}
			log.Printf("  ✓ Created permission: %s", perm.Name)
		}
	}

	return nil
}

func seedSystemRoles(db *gorm.DB) error {
	// Get all permissions
	var allPermissions []models.Permission
	if err := db.Find(&allPermissions).Error; err != nil {
		return err
	}

	// Create permission map for easy lookup
	permMap := make(map[string]models.Permission)
	for _, p := range allPermissions {
		permMap[p.Name] = p
	}

	// Define roles with their permissions
	roleDefinitions := map[string][]string{
		models.RoleOwner: {
			models.PermissionSecretsRead,
			models.PermissionSecretsCreate,
			models.PermissionSecretsUpdate,
			models.PermissionSecretsDelete,
			models.PermissionProjectsManage,
			models.PermissionEnvironmentsManage,
			models.PermissionMembersInvite,
			models.PermissionMembersManage,
			models.PermissionAuditView,
			models.PermissionOrgManage,
		},
		models.RoleAdmin: {
			models.PermissionSecretsRead,
			models.PermissionSecretsCreate,
			models.PermissionSecretsUpdate,
			models.PermissionSecretsDelete,
			models.PermissionProjectsManage,
			models.PermissionEnvironmentsManage,
			models.PermissionMembersInvite,
			models.PermissionMembersManage,
			models.PermissionAuditView,
		},
		models.RoleSecretManager: {
			models.PermissionSecretsRead,
			models.PermissionSecretsCreate,
			models.PermissionSecretsUpdate,
			models.PermissionSecretsDelete,
			models.PermissionAuditView,
		},
		models.RoleDeveloper: {
			models.PermissionSecretsRead,
		},
		models.RoleViewer: {
			models.PermissionSecretsRead,
			models.PermissionAuditView,
		},
	}

	// Create system roles
	for roleName, permNames := range roleDefinitions {
		var existing models.Role
		if err := db.Where("name = ? AND is_system_role = ?", roleName, true).First(&existing).Error; err == gorm.ErrRecordNotFound {
			// Create role
			role := models.Role{
				Name:         roleName,
				IsSystemRole: true,
				OrgID:        nil, // System role
			}

			if err := db.Create(&role).Error; err != nil {
				return err
			}

			// Assign permissions
			var permissions []models.Permission
			for _, permName := range permNames {
				if perm, ok := permMap[permName]; ok {
					permissions = append(permissions, perm)
				}
			}

			if err := db.Model(&role).Association("Permissions").Append(permissions); err != nil {
				return err
			}

			log.Printf("  ✓ Created system role: %s with %d permissions", roleName, len(permissions))
		}
	}

	return nil
}

func seedTierLimits(db *gorm.DB) error {
	limits := []models.TierLimit{
		// Free tier
		{Tier: models.TierFree, LimitType: models.LimitTypeMaxDevs, LimitValue: 2},
		{Tier: models.TierFree, LimitType: models.LimitTypeMaxProjects, LimitValue: 1},
		{Tier: models.TierFree, LimitType: models.LimitTypeMaxOrgs, LimitValue: 1},
		{Tier: models.TierFree, LimitType: models.LimitTypeMaxSecretsPerEnv, LimitValue: 50},
		{Tier: models.TierFree, LimitType: models.LimitTypeAPIRateLimitPerHour, LimitValue: 100},
		{Tier: models.TierFree, LimitType: models.LimitTypeAuditRetentionDays, LimitValue: 7},

		// Starter tier
		{Tier: models.TierStarter, LimitType: models.LimitTypeMaxDevs, LimitValue: 8},
		{Tier: models.TierStarter, LimitType: models.LimitTypeMaxProjects, LimitValue: 5},
		{Tier: models.TierStarter, LimitType: models.LimitTypeMaxOrgs, LimitValue: 1},
		{Tier: models.TierStarter, LimitType: models.LimitTypeMaxSecretsPerEnv, LimitValue: 200},
		{Tier: models.TierStarter, LimitType: models.LimitTypeAPIRateLimitPerHour, LimitValue: 500},
		{Tier: models.TierStarter, LimitType: models.LimitTypeAuditRetentionDays, LimitValue: 30},

		// Team tier
		{Tier: models.TierTeam, LimitType: models.LimitTypeMaxDevs, LimitValue: models.UnlimitedValue},
		{Tier: models.TierTeam, LimitType: models.LimitTypeMaxProjects, LimitValue: models.UnlimitedValue},
		{Tier: models.TierTeam, LimitType: models.LimitTypeMaxOrgs, LimitValue: models.UnlimitedValue},
		{Tier: models.TierTeam, LimitType: models.LimitTypeMaxSecretsPerEnv, LimitValue: models.UnlimitedValue},
		{Tier: models.TierTeam, LimitType: models.LimitTypeAPIRateLimitPerHour, LimitValue: 2000},
		{Tier: models.TierTeam, LimitType: models.LimitTypeAuditRetentionDays, LimitValue: 365},
	}

	for _, limit := range limits {
		var existing models.TierLimit
		if err := db.Where("tier = ? AND limit_type = ?", limit.Tier, limit.LimitType).First(&existing).Error; err == gorm.ErrRecordNotFound {
			if err := db.Create(&limit).Error; err != nil {
				return err
			}
		} else if err == nil {
			// Update existing limit
			existing.LimitValue = limit.LimitValue
			if err := db.Save(&existing).Error; err != nil {
				return err
			}
		}
	}

	log.Printf("  ✓ Created/updated tier limits for all tiers")
	return nil
}
