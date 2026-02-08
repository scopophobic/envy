package services

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"

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

	// Create AWS config
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion(cfg.AWSRegion),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.AWSAccessKeyID,
			cfg.AWSSecretAccessKey,
			"",
		)),
	)
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

// Encrypt encrypts plaintext using envelope encryption
// Returns base64-encoded encrypted data in format: encryptedDataKey:encryptedValue
func (s *KMSService) Encrypt(ctx context.Context, plaintext string) (string, error) {
	// Step 1: Generate a data key from KMS
	dataKeyOutput, err := s.client.GenerateDataKey(ctx, &kms.GenerateDataKeyInput{
		KeyId:   aws.String(s.keyID),
		KeySpec: "AES_256",
	})
	if err != nil {
		return "", fmt.Errorf("failed to generate data key: %w", err)
	}

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

	// Encrypt the plaintext
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)

	// Step 3: Encode encrypted data key and ciphertext
	encryptedDataKey := base64.StdEncoding.EncodeToString(dataKeyOutput.CiphertextBlob)
	encryptedValue := base64.StdEncoding.EncodeToString(ciphertext)

	// Return in format: encryptedDataKey:encryptedValue
	return fmt.Sprintf("%s:%s", encryptedDataKey, encryptedValue), nil
}

// Decrypt decrypts ciphertext using envelope encryption
// Expects input in format: encryptedDataKey:encryptedValue
func (s *KMSService) Decrypt(ctx context.Context, encryptedData string) (string, error) {
	// Parse the encrypted data
	var encryptedDataKey, encryptedValue string
	if _, err := fmt.Sscanf(encryptedData, "%s:%s", &encryptedDataKey, &encryptedValue); err != nil {
		return "", fmt.Errorf("invalid encrypted data format: %w", err)
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
		CiphertextBlob: encryptedKeyBytes,
	})
	if err != nil {
		return "", fmt.Errorf("failed to decrypt data key: %w", err)
	}

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

	// Decrypt
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}

	return string(plaintext), nil
}

// TestConnection tests the KMS connection by describing the key
func (s *KMSService) TestConnection(ctx context.Context) error {
	_, err := s.client.DescribeKey(ctx, &kms.DescribeKeyInput{
		KeyId: aws.String(s.keyID),
	})
	if err != nil {
		return fmt.Errorf("failed to describe KMS key: %w", err)
	}
	return nil
}
