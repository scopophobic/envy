package services

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"strings"
)

// LocalEncryptionService provides AES-256-GCM encryption using a local key derived from JWT_SECRET.
// This is a development-only fallback when AWS KMS is not configured.
type LocalEncryptionService struct {
	key   []byte // 32 bytes for AES-256
	keyID string
}

// NewLocalEncryptionService creates a local encryption service from a secret string.
func NewLocalEncryptionService(secret string) *LocalEncryptionService {
	// Derive a 32-byte key from the secret using SHA-256
	hash := sha256.Sum256([]byte(secret))
	return &LocalEncryptionService{
		key:   hash[:],
		keyID: "local",
	}
}

// Encrypt encrypts plaintext using AES-256-GCM (local key, no KMS).
func (s *LocalEncryptionService) Encrypt(_ context.Context, plaintext string) (string, error) {
	block, err := aes.NewCipher(s.key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return "local:" + base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrypts ciphertext encrypted by Encrypt.
func (s *LocalEncryptionService) Decrypt(_ context.Context, encryptedData string) (string, error) {
	// Strip "local:" prefix
	data := encryptedData
	if strings.HasPrefix(data, "local:") {
		data = data[6:]
	}

	ciphertext, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return "", fmt.Errorf("failed to decode ciphertext: %w", err)
	}

	block, err := aes.NewCipher(s.key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}

	return string(plaintext), nil
}

// KeyID returns the key identifier.
func (s *LocalEncryptionService) KeyID() string {
	return s.keyID
}
