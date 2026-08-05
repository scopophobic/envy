package services

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/envo/backend/internal/config"
)

// KMSService handles encryption and decryption using AWS KMS
type KMSService struct {
	client *kms.Client
	keyID  string
}

// NewKMSService creates a new KMS service
func NewKMSService(cfg *config.Config) (*KMSService, error) {
	ctx := context.Background()

	// Use explicit credentials only when configured. Otherwise preserve the AWS
	// default chain so EC2/ECS workload roles can be used without stored keys.
	options := []func(*awsconfig.LoadOptions) error{awsconfig.WithRegion(cfg.AWSRegion)}
	if cfg.AWSAccessKeyID != "" && cfg.AWSSecretAccessKey != "" {
		options = append(options, awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.AWSAccessKeyID,
			cfg.AWSSecretAccessKey,
			"",
		)))
	}
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, options...)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create KMS client
	client := kms.NewFromConfig(awsCfg)

	return &KMSService{
		client: client,
		keyID:  cfg.AWSKMSKeyID,
	}, nil
}

// Encrypt encrypts plaintext using envelope encryption.
// workspaceID is accepted for interface conformance; KMS uses its own key hierarchy.
func (s *KMSService) Encrypt(ctx context.Context, plaintext string, workspaceID string) (string, error) {
	encCtx := kmsContext(workspaceID)

	// Step 1: Generate a data key from KMS
	dataKeyOutput, err := s.client.GenerateDataKey(ctx, &kms.GenerateDataKeyInput{
		KeyId:             aws.String(s.keyID),
		KeySpec:           "AES_256",
		EncryptionContext: encCtx,
	})
	if err != nil {
		return "", fmt.Errorf("failed to generate data key: %w", err)
	}
	defer zeroBytes(dataKeyOutput.Plaintext)

	// Step 2: Encrypt the plaintext with the data key using AES-GCM
	block, err := aes.NewCipher(dataKeyOutput.Plaintext)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	// Create nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt the plaintext (workspace-bound associated data).
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), []byte(workspaceID))

	// Step 3: Encode encrypted data key and ciphertext
	encryptedDataKey := base64.StdEncoding.EncodeToString(dataKeyOutput.CiphertextBlob)
	encryptedValue := base64.StdEncoding.EncodeToString(ciphertext)

	// Return in format: encryptedDataKey:encryptedValue
	return fmt.Sprintf("%s:%s", encryptedDataKey, encryptedValue), nil
}

// Decrypt decrypts ciphertext using envelope encryption.
// workspaceID is accepted for interface conformance; KMS uses its own key hierarchy.
func (s *KMSService) Decrypt(ctx context.Context, encryptedData string, workspaceID string) (string, error) {
	// Reject values encrypted by local encryptor (they start with "local:")
	if strings.HasPrefix(encryptedData, "local:") {
		return "", fmt.Errorf("value was encrypted with local encryptor, not KMS")
	}

	// Parse the encrypted data — split on the FIRST colon only
	encryptedDataKey, encryptedValue, found := strings.Cut(encryptedData, ":")
	if !found || encryptedDataKey == "" || encryptedValue == "" {
		return "", fmt.Errorf("invalid encrypted data format: expected 'datakey:ciphertext'")
	}

	// Decode base64
	encryptedKeyBytes, err := base64.StdEncoding.DecodeString(encryptedDataKey)
	if err != nil {
		return "", fmt.Errorf("failed to decode encrypted key: %w", err)
	}

	ciphertext, err := base64.StdEncoding.DecodeString(encryptedValue)
	if err != nil {
		return "", fmt.Errorf("failed to decode ciphertext: %w", err)
	}

	// Step 1: Decrypt the data key using KMS
	decryptOutput, err := s.client.Decrypt(ctx, &kms.DecryptInput{
		CiphertextBlob:    encryptedKeyBytes,
		EncryptionContext: kmsContext(workspaceID),
	})
	if err != nil {
		// Backward-compat for old data keys encrypted without KMS encryption context.
		decryptOutput, err = s.client.Decrypt(ctx, &kms.DecryptInput{
			CiphertextBlob: encryptedKeyBytes,
		})
	}
	if err != nil {
		return "", fmt.Errorf("failed to decrypt data key: %w", err)
	}
	defer zeroBytes(decryptOutput.Plaintext)

	// Step 2: Decrypt the ciphertext with the data key
	block, err := aes.NewCipher(decryptOutput.Plaintext)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	// Extract nonce
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

	// Decrypt (first with workspace AAD, then legacy without AAD for compatibility).
	plaintext, err := gcm.Open(nil, nonce, ciphertext, []byte(workspaceID))
	if err != nil {
		plaintext, err = gcm.Open(nil, nonce, ciphertext, nil)
	}
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}

	return string(plaintext), nil
}

// KeyID returns the KMS key ID.
func (s *KMSService) KeyID() string {
	return s.keyID
}

// TestConnection verifies the exact KMS operations Envo needs at runtime.
func (s *KMSService) TestConnection(ctx context.Context) error {
	encryptionContext := kmsContext("startup-check")
	generated, err := s.client.GenerateDataKey(ctx, &kms.GenerateDataKeyInput{
		KeyId:             aws.String(s.keyID),
		KeySpec:           "AES_256",
		EncryptionContext: encryptionContext,
	})
	if err != nil {
		return fmt.Errorf("failed to generate KMS data key: %w", err)
	}
	defer zeroBytes(generated.Plaintext)

	decrypted, err := s.client.Decrypt(ctx, &kms.DecryptInput{
		CiphertextBlob:    generated.CiphertextBlob,
		EncryptionContext: encryptionContext,
	})
	if err != nil {
		return fmt.Errorf("failed to decrypt KMS data key: %w", err)
	}
	defer zeroBytes(decrypted.Plaintext)
	if subtle.ConstantTimeCompare(generated.Plaintext, decrypted.Plaintext) != 1 {
		return fmt.Errorf("KMS data key round-trip mismatch")
	}
	return nil
}

func zeroBytes(value []byte) {
	for i := range value {
		value[i] = 0
	}
}

func kmsContext(workspaceID string) map[string]string {
	ctx := map[string]string{
		"service": "envo",
	}
	if ws := strings.TrimSpace(workspaceID); ws != "" {
		ctx["workspace_id"] = ws
	}
	return ctx
}
