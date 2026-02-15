package services

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/envo/backend/internal/database"
	"github.com/envo/backend/internal/models"
	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// Encryptor is the interface for encrypting/decrypting secrets.
// Implemented by both KMSService (production) and LocalEncryptionService (dev).
type Encryptor interface {
	Encrypt(ctx context.Context, plaintext string) (string, error)
	Decrypt(ctx context.Context, encryptedData string) (string, error)
	KeyID() string
}

// Ensure both services implement Encryptor
var _ Encryptor = (*KMSService)(nil)
var _ Encryptor = (*LocalEncryptionService)(nil)

// SecretService handles secret CRUD and export
type SecretService struct {
	encryptor     Encryptor
	localEncryptor Encryptor // optional; used to decrypt secrets stored with KMSKeyID "local"
	tierService   *TierService
	auditService  *AuditService
}

// NewSecretService creates a new secret service. Pass localEncryptor so secrets
// stored with local encryption can be decrypted when primary is KMS (or vice versa).
func NewSecretService(encryptor Encryptor, localEncryptor Encryptor, tier *TierService, audit *AuditService) *SecretService {
	return &SecretService{
		encryptor:      encryptor,
		localEncryptor: localEncryptor,
		tierService:    tier,
		auditService:   audit,
	}
}

// CreateSecret creates a new secret in an environment
func (s *SecretService) CreateSecret(ctx context.Context, userID, envID uuid.UUID, key, value string, ip string) (*models.SecretResponse, error) {
	if s.encryptor == nil {
		return nil, fmt.Errorf("secret encryption is not configured")
	}

	db := database.GetDB().WithContext(ctx)

	// Tier limit check
	canCreate, err := s.tierService.CanCreateSecret(envID)
	if err != nil {
		return nil, fmt.Errorf("failed to check tier limits: %w", err)
	}
	if !canCreate {
		return nil, fmt.Errorf("secret limit reached for this environment")
	}

	// Encrypt value
	encrypted, err := s.encryptor.Encrypt(ctx, value)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt secret: %w", err)
	}

	secret := &models.Secret{
		EnvironmentID:  envID,
		Key:            key,
		EncryptedValue: encrypted,
		KMSKeyID:       s.encryptor.KeyID(),
		CreatedBy:      userID,
	}

	if err := db.Create(secret).Error; err != nil {
		return nil, err
	}

	// Load env -> project -> org for audit
	var env models.Environment
	if err := db.Preload("Project.Organization").First(&env, envID).Error; err == nil && s.auditService != nil {
		_ = s.auditService.Log(ctx, userID, env.Project.Organization.ID, secret.ID, models.ActionSecretCreate, "secret", ip,
			datatypes.JSON([]byte(`{"key":"`+key+`"}`)))
	}

	resp := secret.ToResponse()
	return &resp, nil
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
		encrypted, err := s.encryptor.Encrypt(ctx, *newValue)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt secret: %w", err)
		}
		secret.EncryptedValue = encrypted
	}

	if err := db.Save(&secret).Error; err != nil {
		return nil, err
	}

	// Load env -> project -> org for audit
	var env models.Environment
	if err := db.Preload("Project.Organization").First(&env, secret.EnvironmentID).Error; err == nil && s.auditService != nil {
		_ = s.auditService.Log(ctx, userID, env.Project.Organization.ID, secret.ID, models.ActionSecretUpdate, "secret", ip,
			datatypes.JSON([]byte(`{"key":"`+secret.Key+`"}`)))
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
	_ = db.Preload("Project.Organization").First(&env, secret.EnvironmentID).Error

	if err := db.Delete(&models.Secret{}, secretID).Error; err != nil {
		return err
	}

	if s.auditService != nil && env.Project.Organization.ID != uuid.Nil {
		_ = s.auditService.Log(ctx, userID, env.Project.Organization.ID, secretID, models.ActionSecretDelete, "secret", ip,
			datatypes.JSON([]byte(`{"key":"`+secret.Key+`"}`)))
	}

	return nil
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
func (s *SecretService) tryDecrypt(ctx context.Context, sec *models.Secret, dec, alt Encryptor) (string, error) {
	plain, err := dec.Decrypt(ctx, sec.EncryptedValue)
	if err == nil {
		return plain, nil
	}
	if alt != nil && alt != dec {
		plain, err2 := alt.Decrypt(ctx, sec.EncryptedValue)
		if err2 == nil {
			return plain, nil
		}
	}
	return "", err
}

// ExportEnvironmentSecrets returns decrypted secrets for an environment (for CLI).
// Secrets that fail to decrypt are skipped (and logged); decryptor is chosen by KMSKeyID, with fallback to the other if configured.
func (s *SecretService) ExportEnvironmentSecrets(ctx context.Context, userID, envID uuid.UUID, ip string) (map[string]string, uuid.UUID, error) {
	if s.encryptor == nil {
		return nil, uuid.Nil, fmt.Errorf("secret encryption is not configured")
	}

	db := database.GetDB().WithContext(ctx)

	// Load env with project + org
	var env models.Environment
	if err := db.Preload("Project.Organization").First(&env, envID).Error; err != nil {
		return nil, uuid.Nil, err
	}

	// Load secrets
	var secrets []models.Secret
	if err := db.Where("environment_id = ?", envID).
		Find(&secrets).Error; err != nil {
		return nil, uuid.Nil, err
	}

	result := make(map[string]string, len(secrets))
	for _, sec := range secrets {
		dec := s.decryptorForSecret(&sec)
		alt := s.localEncryptor
		if dec == s.localEncryptor {
			alt = s.encryptor
		}
		plaintext, err := s.tryDecrypt(ctx, &sec, dec, alt)
		if err != nil {
			log.Printf("[envo] skip secret %s (%s): decrypt failed: %v", sec.ID, sec.Key, err)
			continue
		}
		result[sec.Key] = plaintext
	}
	if len(secrets) > 0 && len(result) == 0 {
		log.Printf("[envo] export: %d secrets in env but 0 decrypted; check KMS/local config and re-create secrets if needed", len(secrets))
	}

	// Audit read
	if s.auditService != nil {
		_ = s.auditService.Log(ctx, userID, env.Project.Organization.ID, envID, models.ActionSecretRead, "environment", ip, nil)
	}

	return result, env.Project.Organization.ID, nil
}
