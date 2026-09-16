package models

import (
	"crypto/sha256"
	"encoding/hex"
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
		&AgentIdentity{},
		&AgentCredential{},
		&AgentGrant{},
		&AgentAccessRequest{},
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
	// Audit entries may be produced by a human or an agent. Existing installs
	// created user_id as NOT NULL, so relax it before agent audit writes begin.
	if db.Migrator().HasTable(&AuditLog{}) {
		if err := db.Exec(`ALTER TABLE audit_logs ALTER COLUMN user_id DROP NOT NULL`).Error; err != nil {
			return err
		}
	}
	// Agent credentials are intentionally time-bounded. Credentials issued by
	// older versions without an expiry are made unusable during migration and
	// can be replaced from the dashboard with an explicitly expiring token.
	if db.Migrator().HasTable(&AgentCredential{}) {
		if err := db.Exec(`UPDATE agent_credentials SET revoked_at = NOW() WHERE expires_at IS NULL AND revoked_at IS NULL`).Error; err != nil {
			return err
		}
	}
	if db.Migrator().HasTable(&AgentGrant{}) {
		if err := db.Exec(`UPDATE agent_grants SET expires_at = NOW() + INTERVAL '7 days' WHERE expires_at IS NULL AND revoked_at IS NULL`).Error; err != nil {
			return err
		}
	}

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
			name: "idx_agent_grants_live_lookup",
			sql:  `CREATE INDEX IF NOT EXISTS idx_agent_grants_live_lookup ON agent_grants (agent_id, environment_id, capability) WHERE revoked_at IS NULL AND deleted_at IS NULL`,
		},
		{
			name: "idx_agent_credentials_auth_lookup",
			sql:  `CREATE INDEX IF NOT EXISTS idx_agent_credentials_auth_lookup ON agent_credentials (id, expires_at) WHERE revoked_at IS NULL`,
		},
		{
			name: "idx_agent_access_requests_queue",
			sql:  `CREATE INDEX IF NOT EXISTS idx_agent_access_requests_queue ON agent_access_requests (org_id, status, created_at DESC)`,
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
	// A session-scoped request is idempotent even after it reaches a terminal
	// state. A network retry must never mint a second one-time delivery.
	if err := db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_agent_access_requests_fingerprint_once ON agent_access_requests (credential_id, request_fingerprint) WHERE request_fingerprint <> ''`).Error; err != nil {
		return err
	}
	constraints := []string{
		`DO $$ BEGIN IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_audit_actor_identity') THEN ALTER TABLE audit_logs ADD CONSTRAINT chk_audit_actor_identity CHECK ((actor_type = 'human' AND user_id IS NOT NULL AND agent_id IS NULL) OR (actor_type = 'agent' AND agent_id IS NOT NULL AND user_id IS NULL)); END IF; END $$`,
		`DO $$ BEGIN IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_agent_grant_approval_mode') THEN ALTER TABLE agent_grants ADD CONSTRAINT chk_agent_grant_approval_mode CHECK (approval_mode IN ('always', 'none')); END IF; END $$`,
		`DO $$ BEGIN IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_agent_grant_lease_range') THEN ALTER TABLE agent_grants ADD CONSTRAINT chk_agent_grant_lease_range CHECK (max_lease_seconds BETWEEN 30 AND 3600); END IF; END $$`,
		`DO $$ BEGIN IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_agent_access_request_status') THEN ALTER TABLE agent_access_requests ADD CONSTRAINT chk_agent_access_request_status CHECK (status IN ('pending', 'approved', 'denied', 'delivering', 'consumed', 'expired', 'revoked')); END IF; END $$`,
	}
	for _, statement := range constraints {
		if err := db.Exec(statement).Error; err != nil {
			return err
		}
	}
	if err := ensureAgentManagementPermission(db); err != nil {
		return err
	}

	if err := backfillPersonalWorkspaces(db); err != nil {
		return err
	}
	if err := HashLegacyRefreshTokens(db); err != nil {
		return err
	}

	return nil
}

// ensureAgentManagementPermission makes the feature usable immediately after
// a schema migration on existing installations. A later full seed remains
// authoritative for all system-role permission sets.
func ensureAgentManagementPermission(db *gorm.DB) error {
	permissions := []Permission{
		{Name: PermissionAgentsManage, Description: "Create agents and credentials"},
		{Name: PermissionAgentGrantsManage, Description: "Delegate scoped secret access to agents"},
		{Name: PermissionAgentApprovalsManage, Description: "Approve or deny agent access requests"},
	}
	for i := range permissions {
		if err := db.Where("name = ?", permissions[i].Name).FirstOrCreate(&permissions[i]).Error; err != nil {
			return err
		}
	}
	var roles []Role
	if err := db.Where("is_system_role = ? AND name IN ?", true, []string{RoleOwner, RoleAdmin}).Find(&roles).Error; err != nil {
		return err
	}
	for i := range roles {
		for j := range permissions {
			if err := db.Model(&roles[i]).Association("Permissions").Append(&permissions[j]); err != nil {
				return err
			}
		}
	}
	return nil
}

// HashLegacyRefreshTokens upgrades refresh tokens written by releases that
// stored the raw JWT. It is safe to run repeatedly and includes revoked and
// soft-deleted rows so database snapshots do not retain usable credentials.
func HashLegacyRefreshTokens(db *gorm.DB) error {
	var tokens []RefreshToken
	if err := db.Unscoped().Select("id", "token").Find(&tokens).Error; err != nil {
		return err
	}

	upgraded := 0
	for _, token := range tokens {
		decoded, decodeErr := hex.DecodeString(token.Token)
		if decodeErr == nil && len(decoded) == sha256.Size {
			continue
		}
		hash := sha256.Sum256([]byte(token.Token))
		if err := db.Unscoped().Model(&RefreshToken{}).
			Where("id = ?", token.ID).
			Update("token", hex.EncodeToString(hash[:])).Error; err != nil {
			return err
		}
		upgraded++
	}
	if upgraded > 0 {
		log.Printf("  ✓ hashed %d legacy refresh tokens", upgraded)
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
