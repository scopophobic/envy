package models

import "gorm.io/gorm"

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
	}
}

// AutoMigrate runs auto migration for all models
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(AllModels()...)
}
