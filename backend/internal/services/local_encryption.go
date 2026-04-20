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

	"golang.org/x/crypto/hkdf"
)

// LocalEncryptionService provides AES-256-GCM encryption using HKDF-derived
// per-workspace keys. The master key is derived from JWT_SECRET.
// When workspaceID is empty, the legacy (global) key is used for backward compat.
type LocalEncryptionService struct {
	masterKey []byte // 32 bytes for AES-256
	keyID     string
}

// NewLocalEncryptionService creates a local encryption service from a secret string.
func NewLocalEncryptionService(secret string) *LocalEncryptionService {
	hash := sha256.Sum256([]byte(secret))
	return &LocalEncryptionService{
		masterKey: hash[:],
		keyID:     "local",
	}
}

// workspaceKey derives a 32-byte AES key scoped to a specific workspace via HKDF.
// Zero cost, no new services — workspace isolation is mathematical.
func (s *LocalEncryptionService) workspaceKey(workspaceID string) ([]byte, error) {
	r := hkdf.New(sha256.New, s.masterKey, []byte(workspaceID), []byte("envo-secret-v1"))
	key := make([]byte, 32)
	if _, err := io.ReadFull(r, key); err != nil {
		return nil, fmt.Errorf("HKDF key derivation failed: %w", err)
	}
	return key, nil
}

// Encrypt encrypts plaintext using AES-256-GCM with HKDF-derived workspace key.
func (s *LocalEncryptionService) Encrypt(_ context.Context, plaintext string, workspaceID string) (string, error) {
	key, err := s.workspaceKey(workspaceID)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
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

	aad := []byte(workspaceID)
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), aad)
	return "local:" + base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrypts ciphertext. Tries the workspace-scoped key first, then falls
// back to the legacy global key so existing secrets remain readable.
func (s *LocalEncryptionService) Decrypt(_ context.Context, encryptedData string, workspaceID string) (string, error) {
	data := encryptedData
	if strings.HasPrefix(data, "local:") {
		data = data[6:]
	}

	ciphertext, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return "", fmt.Errorf("failed to decode ciphertext: %w", err)
	}

	// Try workspace-scoped key first
	if workspaceID != "" {
		if plain, err := s.decryptWithKey(ciphertext, workspaceID); err == nil {
			return plain, nil
		}
	}

	// Backward-compat fallback for old values that didn't bind workspace as AAD.
	if workspaceID != "" {
		if plain, err := s.decryptWithKeyLegacy(ciphertext, workspaceID); err == nil {
			return plain, nil
		}
	}

	// Fallback: legacy global key (empty workspace ID derives from masterKey directly)
	if plain, err := s.decryptRaw(ciphertext, s.masterKey); err == nil {
		return plain, nil
	}

	return "", fmt.Errorf("failed to decrypt: all key derivations failed")
}

func (s *LocalEncryptionService) decryptWithKey(ciphertext []byte, workspaceID string) (string, error) {
	key, err := s.workspaceKey(workspaceID)
	if err != nil {
		return "", err
	}
	return s.decryptRawWithAAD(ciphertext, key, []byte(workspaceID))
}

func (s *LocalEncryptionService) decryptWithKeyLegacy(ciphertext []byte, workspaceID string) (string, error) {
	key, err := s.workspaceKey(workspaceID)
	if err != nil {
		return "", err
	}
	return s.decryptRaw(ciphertext, key)
}

func (s *LocalEncryptionService) decryptRaw(ciphertext []byte, key []byte) (string, error) {
	return s.decryptRawWithAAD(ciphertext, key, nil)
}

func (s *LocalEncryptionService) decryptRawWithAAD(ciphertext []byte, key []byte, aad []byte) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, ct := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ct, aad)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

// KeyID returns the key identifier.
func (s *LocalEncryptionService) KeyID() string {
	return s.keyID
}
