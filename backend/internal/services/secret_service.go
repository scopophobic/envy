package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/envo/backend/internal/database"
	"github.com/envo/backend/internal/models"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// Encryptor is the interface for encrypting/decrypting secrets.
// workspaceID scopes the derived key via HKDF so each workspace is cryptographically isolated.
type Encryptor interface {
	Encrypt(ctx context.Context, plaintext string, workspaceID string) (string, error)
	Decrypt(ctx context.Context, encryptedData string, workspaceID string) (string, error)
	KeyID() string
}

// Ensure both services implement Encryptor
var _ Encryptor = (*KMSService)(nil)
var _ Encryptor = (*LocalEncryptionService)(nil)

// SecretService handles secret CRUD and export
type SecretService struct {
	encryptor          Encryptor
	localEncryptor     Encryptor // optional; used to decrypt secrets stored with KMSKeyID "local"
	tierService        *TierService
	auditService       *AuditService
	decryptConcurrency int
}

func secretAuditMetadata(key string, extra map[string]any) datatypes.JSON {
	metadata := map[string]any{"key": key}
	for name, value := range extra {
		metadata[name] = value
	}
	encoded, _ := json.Marshal(metadata)
	return datatypes.JSON(encoded)
}

// NewSecretService creates a new secret service. Pass localEncryptor so secrets
// stored with local encryption can be decrypted when primary is KMS (or vice versa).
func NewSecretService(encryptor Encryptor, localEncryptor Encryptor, tier *TierService, audit *AuditService, decryptConcurrency int) *SecretService {
	if decryptConcurrency <= 0 {
		decryptConcurrency = 8
	}
	return &SecretService{
		encryptor:          encryptor,
		localEncryptor:     localEncryptor,
		tierService:        tier,
		auditService:       audit,
		decryptConcurrency: decryptConcurrency,
	}
}

// resolveWorkspaceID loads the workspace (org) ID for an environment, needed for HKDF key scoping.
func (s *SecretService) resolveWorkspaceID(db *gorm.DB, envID uuid.UUID) (uuid.UUID, error) {
	var env models.Environment
	if err := db.Preload("Project").First(&env, envID).Error; err != nil {
		return uuid.Nil, err
	}
	return env.Project.OrgID, nil
}

// CreateSecret creates a new secret or updates an existing one with the same key (upsert).
func (s *SecretService) CreateSecret(ctx context.Context, userID, envID uuid.UUID, key, value string, ip string) (*models.SecretResponse, bool, error) {
	if s.encryptor == nil {
		return nil, false, fmt.Errorf("secret encryption is not configured")
	}

	db := database.GetDB().WithContext(ctx)

	wsID, err := s.resolveWorkspaceID(db, envID)
	if err != nil {
		return nil, false, fmt.Errorf("failed to resolve workspace: %w", err)
	}
	wsKey := wsID.String()

	// Check if a secret with this key already exists in the environment
	var existing models.Secret
	err = db.Where("environment_id = ? AND key = ?", envID, key).First(&existing).Error

	if err == nil {
		// Key exists — update its value
		encrypted, encErr := s.encryptor.Encrypt(ctx, value, wsKey)
		if encErr != nil {
			return nil, false, fmt.Errorf("failed to encrypt secret: %w", encErr)
		}
		existing.EncryptedValue = encrypted
		existing.KMSKeyID = s.encryptor.KeyID()
		var env models.Environment
		if err := db.Preload("Project.Organization").First(&env, envID).Error; err != nil {
			return nil, false, err
		}
		if err := db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Save(&existing).Error; err != nil {
				return err
			}
			if s.auditService == nil {
				return fmt.Errorf("audit service is not configured")
			}
			return s.auditService.LogWithDB(tx, userID, env.Project.Organization.ID, existing.ID, models.ActionSecretUpdate, "secret", ip, secretAuditMetadata(key, map[string]any{"via": "upsert"}))
		}); err != nil {
			return nil, false, err
		}

		resp := existing.ToResponse()
		return &resp, true, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, err
	}

	// New secret — check tier limits
	canCreate, err := s.tierService.CanCreateSecret(envID)
	if err != nil {
		return nil, false, fmt.Errorf("failed to check tier limits: %w", err)
	}
	if !canCreate {
		return nil, false, fmt.Errorf("secret limit reached for this environment")
	}

	encrypted, err := s.encryptor.Encrypt(ctx, value, wsKey)
	if err != nil {
		return nil, false, fmt.Errorf("failed to encrypt secret: %w", err)
	}

	secret := &models.Secret{
		EnvironmentID:  envID,
		Key:            key,
		EncryptedValue: encrypted,
		KMSKeyID:       s.encryptor.KeyID(),
		CreatedBy:      userID,
	}

	var env models.Environment
	if err := db.Preload("Project.Organization").First(&env, envID).Error; err != nil {
		return nil, false, err
	}
	if err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(secret).Error; err != nil {
			return err
		}
		if s.auditService == nil {
			return fmt.Errorf("audit service is not configured")
		}
		return s.auditService.LogWithDB(tx, userID, env.Project.Organization.ID, secret.ID, models.ActionSecretCreate, "secret", ip, secretAuditMetadata(key, nil))
	}); err != nil {
		return nil, false, err
	}

	resp := secret.ToResponse()
	return &resp, false, nil
}

// ListSecrets lists secrets for an environment (metadata only)
func (s *SecretService) ListSecrets(ctx context.Context, envID uuid.UUID) ([]models.SecretResponse, error) {
	db := database.GetDB().WithContext(ctx)

	var secrets []models.Secret
	if err := db.Where("environment_id = ?", envID).
		Order("created_at ASC").
		Find(&secrets).Error; err != nil {
		return nil, err
	}

	responses := make([]models.SecretResponse, 0, len(secrets))
	for _, sec := range secrets {
		resp := sec.ToResponse()
		responses = append(responses, resp)
	}

	return responses, nil
}

// UpdateSecret updates a secret's key and/or value
func (s *SecretService) UpdateSecret(ctx context.Context, userID, secretID uuid.UUID, newKey *string, newValue *string, ip string) (*models.SecretResponse, error) {
	if s.encryptor == nil {
		return nil, fmt.Errorf("secret encryption is not configured")
	}

	db := database.GetDB().WithContext(ctx)

	var secret models.Secret
	if err := db.First(&secret, secretID).Error; err != nil {
		return nil, err
	}

	if newKey != nil {
		secret.Key = *newKey
	}

	if newValue != nil {
		wsID, err := s.resolveWorkspaceID(db, secret.EnvironmentID)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve workspace: %w", err)
		}
		encrypted, err := s.encryptor.Encrypt(ctx, *newValue, wsID.String())
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt secret: %w", err)
		}
		secret.EncryptedValue = encrypted
		secret.KMSKeyID = s.encryptor.KeyID()
	}

	var env models.Environment
	if err := db.Preload("Project.Organization").First(&env, secret.EnvironmentID).Error; err != nil {
		return nil, err
	}
	if err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&secret).Error; err != nil {
			return err
		}
		if s.auditService == nil {
			return fmt.Errorf("audit service is not configured")
		}
		return s.auditService.LogWithDB(tx, userID, env.Project.Organization.ID, secret.ID, models.ActionSecretUpdate, "secret", ip, secretAuditMetadata(secret.Key, nil))
	}); err != nil {
		return nil, err
	}

	resp := secret.ToResponse()
	return &resp, nil
}

// DeleteSecret deletes a secret
func (s *SecretService) DeleteSecret(ctx context.Context, userID, secretID uuid.UUID, ip string) error {
	db := database.GetDB().WithContext(ctx)

	var secret models.Secret
	if err := db.First(&secret, secretID).Error; err != nil {
		return err
	}

	// Load env -> project -> org before deletion
	var env models.Environment
	if err := db.Preload("Project.Organization").First(&env, secret.EnvironmentID).Error; err != nil {
		return err
	}

	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(&models.Secret{}, secretID).Error; err != nil {
			return err
		}
		if s.auditService == nil || env.Project.Organization.ID == uuid.Nil {
			return fmt.Errorf("audit context is unavailable")
		}
		return s.auditService.LogWithDB(tx, userID, env.Project.Organization.ID, secretID, models.ActionSecretDelete, "secret", ip, secretAuditMetadata(secret.Key, nil))
	})
}

// PurgeSecret permanently removes a secret from the database (hard delete).
func (s *SecretService) PurgeSecret(ctx context.Context, userID, secretID uuid.UUID, ip string) error {
	db := database.GetDB().WithContext(ctx)

	// Use Unscoped to include soft-deleted records
	var secret models.Secret
	if err := db.Unscoped().First(&secret, secretID).Error; err != nil {
		return err
	}

	var env models.Environment
	if err := db.Preload("Project.Organization").First(&env, secret.EnvironmentID).Error; err != nil {
		return err
	}

	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Unscoped().Delete(&models.Secret{}, secretID).Error; err != nil {
			return err
		}
		if s.auditService == nil || env.Project.Organization.ID == uuid.Nil {
			return fmt.Errorf("audit context is unavailable")
		}
		return s.auditService.LogWithDB(tx, userID, env.Project.Organization.ID, secretID, "secret.purge", "secret", ip, secretAuditMetadata(secret.Key, map[string]any{"permanent": true}))
	})
}

// decryptorForSecret returns the primary encryptor to use for this secret (by KMSKeyID and value format).
func (s *SecretService) decryptorForSecret(sec *models.Secret) Encryptor {
	// Stored value starts with "local:" => was encrypted with local
	if s.localEncryptor != nil && strings.HasPrefix(sec.EncryptedValue, "local:") {
		return s.localEncryptor
	}
	if sec.KMSKeyID == "local" && s.localEncryptor != nil {
		return s.localEncryptor
	}
	return s.encryptor
}

// tryDecrypt tries dec with the given encryptor; if it fails and alt is different, tries alt.
func (s *SecretService) tryDecrypt(ctx context.Context, sec *models.Secret, dec, alt Encryptor, wsID string) (string, error) {
	plain, err := dec.Decrypt(ctx, sec.EncryptedValue, wsID)
	if err == nil {
		return plain, nil
	}
	if alt != nil && alt != dec {
		plain, err2 := alt.Decrypt(ctx, sec.EncryptedValue, wsID)
		if err2 == nil {
			return plain, nil
		}
	}
	return "", err
}

// ExportEnvironmentSecrets returns decrypted secrets for an environment (for CLI).
// Any decryption failure rejects the full export; decryptor is chosen by KMSKeyID,
// with fallback to the other configured encryptor.
func (s *SecretService) ExportEnvironmentSecrets(ctx context.Context, userID, envID uuid.UUID, ip string) (map[string]string, uuid.UUID, error) {
	result, orgID, err := s.DecryptEnvironmentSecrets(ctx, envID, nil, true)
	if err != nil {
		return nil, uuid.Nil, err
	}
	if s.auditService != nil {
		if err := s.auditService.Log(ctx, userID, orgID, envID, models.ActionSecretRead, "environment", ip, nil); err != nil {
			return nil, uuid.Nil, fmt.Errorf("failed to persist secret access audit: %w", err)
		}
	} else {
		return nil, uuid.Nil, fmt.Errorf("audit service is not configured")
	}
	return result, orgID, nil
}

// DecryptEnvironmentSecrets decrypts only the approved keys. It intentionally
// does not audit by itself so callers can attribute the read to a human or an
// agent correctly.
func (s *SecretService) DecryptEnvironmentSecrets(ctx context.Context, envID uuid.UUID, allowedKeys map[string]struct{}, allowAll bool) (map[string]string, uuid.UUID, error) {
	if s.encryptor == nil {
		return nil, uuid.Nil, fmt.Errorf("secret encryption is not configured")
	}

	db := database.GetDB().WithContext(ctx)

	// Load env with project + org
	var env models.Environment
	if err := db.Preload("Project.Organization").First(&env, envID).Error; err != nil {
		return nil, uuid.Nil, err
	}
	if env.Project.ID == uuid.Nil || env.Project.Organization.ID == uuid.Nil {
		return nil, uuid.Nil, fmt.Errorf("environment project or organization is no longer active")
	}

	// Load secrets
	var secrets []models.Secret
	query := db.Where("environment_id = ?", envID)
	if !allowAll {
		keys := make([]string, 0, len(allowedKeys))
		for key := range allowedKeys {
			keys = append(keys, key)
		}
		if len(keys) == 0 {
			return map[string]string{}, env.Project.OrgID, nil
		}
		query = query.Where("key IN ?", keys)
	}
	if err := query.Find(&secrets).Error; err != nil {
		return nil, uuid.Nil, err
	}
	if !allowAll && len(secrets) != len(allowedKeys) {
		return nil, env.Project.OrgID, fmt.Errorf("one or more approved secret keys no longer exist")
	}

	wsID := env.Project.OrgID.String()
	result := make(map[string]string, len(secrets))
	workers := s.decryptConcurrency
	if workers > len(secrets) {
		workers = len(secrets)
	}
	if workers > 0 {
		jobs := make(chan models.Secret)
		decryptErrors := make(chan error, len(secrets))
		var wg sync.WaitGroup
		var resultMu sync.Mutex
		wg.Add(workers)
		for range workers {
			go func() {
				defer wg.Done()
				for sec := range jobs {
					if ctx.Err() != nil {
						continue
					}
					dec := s.decryptorForSecret(&sec)
					alt := s.localEncryptor
					if dec == s.localEncryptor {
						alt = s.encryptor
					}
					plaintext, err := s.tryDecrypt(ctx, &sec, dec, alt, wsID)
					if err != nil {
						decryptErrors <- fmt.Errorf("secret %s could not be decrypted: %w", sec.ID, err)
						continue
					}
					resultMu.Lock()
					result[sec.Key] = plaintext
					resultMu.Unlock()
				}
			}()
		}
		for _, sec := range secrets {
			jobs <- sec
		}
		close(jobs)
		wg.Wait()
		close(decryptErrors)
		if err, ok := <-decryptErrors; ok {
			return nil, env.Project.OrgID, err
		}
	}

	return result, env.Project.Organization.ID, nil
}
