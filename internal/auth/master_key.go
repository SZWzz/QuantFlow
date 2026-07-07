// Package auth provides credential encryption and management for API keys.
package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
)

// MasterKey is a 256-bit AES key used for credential encryption.
// It is stored in the OS keychain (macOS Keychain, Linux secret-tool)
// or in a file at ~/.config/quantflow/master.key.
type MasterKey struct {
	key [32]byte
}

// NewMasterKey creates a MasterKey from a 32-byte key.
func NewMasterKey(key [32]byte) *MasterKey {
	return &MasterKey{key: key}
}

// Encrypt encrypts plaintext using AES-256-GCM.
// Returns ciphertext with nonce prepended (nonce || ciphertext || tag).
func (k *MasterKey) Encrypt(plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(k.key[:])
	if err != nil {
		return nil, fmt.Errorf("master key: create cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("master key: create GCM: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("master key: generate nonce: %w", err)
	}
	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

// Decrypt decrypts ciphertext that was previously encrypted with Encrypt.
func (k *MasterKey) Decrypt(ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(k.key[:])
	if err != nil {
		return nil, fmt.Errorf("master key: create cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("master key: create GCM: %w", err)
	}
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("master key: ciphertext too short")
	}
	nonce, ct := ciphertext[:nonceSize], ciphertext[nonceSize:]
	return gcm.Open(nil, nonce, ct, nil)
}
