package services

import (
	"context"
	"fmt"

	"github.com/envo/backend/internal/database"
	"github.com/envo/backend/internal/models"
	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// SecretService handles secret CRUD and export
type SecretService struct {
	kmsService   *KMSService
	tierService  *TierService
	auditService *AuditService
}

// NewSecretService creates a new secret service
func NewSecretService(kms *KMSService, tier *TierService, audit *AuditService) *SecretService {
	return &SecretService{
		kmsService:   kms,
		tierService:  tier,
		auditService: audit,
	}
}

// CreateSecret creates a new secret in an environment
func (s *SecretService) CreateSecret(ctx context.Context, userID, envID uuid.UUID, key, value string, ip string) (*models.SecretResponse, error) {
	if s.kmsService == nil {
		return nil, fmt.Errorf("secret encryption is not configured (KMS not initialized)")
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
	encrypted, err := s.kmsService.Encrypt(ctx, value)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt secret: %w", err)
	}

	secret := &models.Secret{
		EnvironmentID:  envID,
		Key:            key,
		EncryptedValue: encrypted,
		KMSKeyID:       s.kmsService.keyID,
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
	if s.kmsService == nil {
		return nil, fmt.Errorf("secret encryption is not configured (KMS not initialized)")
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
		encrypted, err := s.kmsService.Encrypt(ctx, *newValue)
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

// ExportEnvironmentSecrets returns decrypted secrets for an environment (for CLI)
func (s *SecretService) ExportEnvironmentSecrets(ctx context.Context, userID, envID uuid.UUID, ip string) (map[string]string, uuid.UUID, error) {
	if s.kmsService == nil {
		return nil, uuid.Nil, fmt.Errorf("secret encryption is not configured (KMS not initialized)")
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
		plaintext, err := s.kmsService.Decrypt(ctx, sec.EncryptedValue)
		if err != nil {
			return nil, uuid.Nil, fmt.Errorf("failed to decrypt secret %s: %w", sec.ID, err)
		}
		result[sec.Key] = plaintext
	}

	// Audit read
	if s.auditService != nil {
		_ = s.auditService.Log(ctx, userID, env.Project.Organization.ID, envID, models.ActionSecretRead, "environment", ip, nil)
	}

	return result, env.Project.Organization.ID, nil
}

