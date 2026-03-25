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
		&Permission{},
		&Role{},
		&Project{},
		&Environment{},
		&Secret{},
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
			name: "idx_audit_logs_org_created",
			sql:  `CREATE INDEX IF NOT EXISTS idx_audit_logs_org_created ON audit_logs (org_id, created_at DESC)`,
		},
	}

	for _, idx := range indexes {
		if err := db.Exec(idx.sql).Error; err != nil {
			log.Printf("  ⚠ index %s: %v", idx.name, err)
		} else {
			log.Printf("  ✓ index %s", idx.name)
		}
	}

	return nil
}
