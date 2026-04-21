package models

import (
	"log"

	"gorm.io/gorm"
)

// AllModels returns all models for migration
func AllModels() []interface{} {
	return []interface{}{
		&User{},
		&Organization{},
		&OrgMember{},
		&OrgInvitation{},
		&Permission{},
		&Role{},
		&Project{},
		&Environment{},
		&Secret{},
		&PlatformConnection{},
		&TierLimit{},
		&AuditLog{},
		&RefreshToken{},
		&CLILoginCode{},
	}
}

// AutoMigrate runs auto migration for all models
func AutoMigrate(db *gorm.DB) error {
	if err := db.AutoMigrate(AllModels()...); err != nil {
		return err
	}
	return RunCustomMigrations(db)
}

// RunCustomMigrations creates indexes and constraints that GORM tags can't express.
func RunCustomMigrations(db *gorm.DB) error {
	indexes := []struct {
		name string
		sql  string
	}{
		{
			name: "idx_secrets_env_key_active",
			sql:  `CREATE UNIQUE INDEX IF NOT EXISTS idx_secrets_env_key_active ON secrets (environment_id, key) WHERE deleted_at IS NULL`,
		},
		{
			name: "idx_secrets_env_created",
			sql:  `CREATE INDEX IF NOT EXISTS idx_secrets_env_created ON secrets (environment_id, created_at ASC) WHERE deleted_at IS NULL`,
		},
		{
			name: "idx_projects_org_active",
			sql:  `CREATE INDEX IF NOT EXISTS idx_projects_org_active ON projects (org_id) WHERE deleted_at IS NULL`,
		},
		{
			name: "idx_org_members_user_org",
			sql:  `CREATE INDEX IF NOT EXISTS idx_org_members_user_org ON org_members (user_id, org_id)`,
		},
		{
			name: "idx_roles_org_name_unique",
			sql:  `CREATE UNIQUE INDEX IF NOT EXISTS idx_roles_org_name_unique ON roles (org_id, name) WHERE deleted_at IS NULL`,
		},
		{
			name: "idx_org_invitations_org_email_pending",
			sql:  `CREATE UNIQUE INDEX IF NOT EXISTS idx_org_invitations_org_email_pending ON org_invitations (org_id, lower(email)) WHERE status = 'pending' AND deleted_at IS NULL`,
		},
		{
			name: "idx_audit_logs_org_created",
			sql:  `CREATE INDEX IF NOT EXISTS idx_audit_logs_org_created ON audit_logs (org_id, created_at DESC)`,
		},
		{
			name: "idx_orgs_owner_personal",
			sql:  `CREATE UNIQUE INDEX IF NOT EXISTS idx_orgs_owner_personal ON organizations (owner_id) WHERE owner_type = 'personal' AND deleted_at IS NULL`,
		},
	}

	for _, idx := range indexes {
		if err := db.Exec(idx.sql).Error; err != nil {
			log.Printf("  ⚠ index %s: %v", idx.name, err)
		} else {
			log.Printf("  ✓ index %s", idx.name)
		}
	}

	if err := backfillPersonalWorkspaces(db); err != nil {
		return err
	}

	return nil
}

// backfillPersonalWorkspaces ensures every existing user has a personal workspace
// with an Owner membership so listing queries pick it up.
// Safe to run repeatedly — skips users who already have one.
func backfillPersonalWorkspaces(db *gorm.DB) error {
	var users []User
	if err := db.Find(&users).Error; err != nil {
		return err
	}

	var ownerRole Role
	if err := db.Where("name = ? AND is_system_role = ?", RoleOwner, true).First(&ownerRole).Error; err != nil {
		log.Printf("  ⚠ backfill: Owner role not found, skipping personal workspace creation")
		return nil
	}

	for _, u := range users {
		var count int64
		db.Model(&Organization{}).
			Where("owner_id = ? AND owner_type = ?", u.ID, OwnerTypePersonal).
			Count(&count)
		if count > 0 {
			continue
		}

		personal := Organization{
			OwnerID:   u.ID,
			Name:      u.Name + "'s workspace",
			OwnerType: OwnerTypePersonal,
		}
		if err := db.Create(&personal).Error; err != nil {
			log.Printf("  ⚠ backfill personal workspace for %s: %v", u.Email, err)
			continue
		}

		member := OrgMember{
			OrgID:  personal.ID,
			UserID: u.ID,
			RoleID: ownerRole.ID,
		}
		if err := db.Create(&member).Error; err != nil {
			log.Printf("  ⚠ backfill membership for %s: %v", u.Email, err)
			continue
		}

		log.Printf("  ✓ backfilled personal workspace for %s", u.Email)
	}

	return nil
}
