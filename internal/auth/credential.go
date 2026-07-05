// Package auth provides credential encryption and management for API keys.
package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"
)

// Credential represents a stored API credential.
type Credential struct {
	ID        int               `json:"id"`
	Name      string            `json:"name"`
	Type      string            `json:"type"`
	Keys      map[string]string `json:"keys"` // decrypted key-value pairs
	CreatedAt string            `json:"created_at"`
	UpdatedAt string            `json:"updated_at"`
}

// credentialRow is the database representation with encrypted JSON blob.
type credentialRow struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	Data      string `json:"data"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// CredentialManager handles encrypted credential storage.
type CredentialManager struct {
	db  *sql.DB
	gcm cipher.AEAD
}

// NewCredentialManager creates a credential manager with AES-256-GCM encryption.
// The encryption key is derived from the machine's hostname + a salt, providing
// basic protection for API keys stored on disk.
func NewCredentialManager(db *sql.DB) (*CredentialManager, error) {
	key := deriveKey()
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("credential: create cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("credential: create GCM: %w", err)
	}
	return &CredentialManager{db: db, gcm: gcm}, nil
}

// deriveKey creates a 256-bit key from machine identity + fixed salt.
func deriveKey() []byte {
	hostname, _ := os.Hostname()
	material := hostname + ":quantflow-cred-salt-v1"
	h := sha256.Sum256([]byte(material))
	return h[:]
}

// encrypt encrypts plaintext with AES-GCM and returns base64-encoded ciphertext.
func (m *CredentialManager) encrypt(plaintext []byte) (string, error) {
	nonce := make([]byte, m.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("credential: generate nonce: %w", err)
	}
	ciphertext := m.gcm.Seal(nonce, nonce, plaintext, nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// decrypt decrypts a base64-encoded ciphertext back to plaintext.
func (m *CredentialManager) decrypt(encoded string) ([]byte, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("credential: decode: %w", err)
	}
	nonceSize := m.gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("credential: ciphertext too short")
	}
	nonce, ct := ciphertext[:nonceSize], ciphertext[nonceSize:]
	return m.gcm.Open(nil, nonce, ct, nil)
}

// List returns all stored credentials (with encrypted values decrypted).
func (m *CredentialManager) List() ([]Credential, error) {
	rows, err := m.db.Query(`SELECT id, name, type, data, created_at, updated_at FROM credentials ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("credential: list: %w", err)
	}
	defer rows.Close()

	var creds []Credential
	for rows.Next() {
		var row credentialRow
		if err := rows.Scan(&row.ID, &row.Name, &row.Type, &row.Data, &row.CreatedAt, &row.UpdatedAt); err != nil {
			return nil, fmt.Errorf("credential: scan: %w", err)
		}
		plain, err := m.decrypt(row.Data)
		if err != nil {
			return nil, fmt.Errorf("credential: decrypt %q: %w", row.Name, err)
		}
		var keys map[string]string
		if err := json.Unmarshal(plain, &keys); err != nil {
			return nil, fmt.Errorf("credential: unmarshal %q: %w", row.Name, err)
		}
		creds = append(creds, Credential{
			ID: row.ID, Name: row.Name, Type: row.Type, Keys: keys,
			CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
		})
	}
	return creds, rows.Err()
}

// Save creates or updates a credential. Keys are encrypted before storage.
func (m *CredentialManager) Save(name, credType string, keys map[string]string) error {
	plain, err := json.Marshal(keys)
	if err != nil {
		return fmt.Errorf("credential: marshal keys: %w", err)
	}
	encrypted, err := m.encrypt(plain)

	if err != nil {
		return err
	}
	now := time.Now().UTC().Format(time.RFC3339)

	_, err = m.db.Exec(
		`INSERT INTO credentials (name, type, data, updated_at) VALUES (?, ?, ?, ?)
		 ON CONFLICT(name) DO UPDATE SET type=excluded.type, data=excluded.data, updated_at=excluded.updated_at`,
		name, credType, encrypted, now,
	)
	if err != nil {
		return fmt.Errorf("credential: save %q: %w", name, err)
	}
	return nil
}

// Delete removes a credential by name.
func (m *CredentialManager) Delete(name string) error {
	_, err := m.db.Exec(`DELETE FROM credentials WHERE name=?`, name)
	if err != nil {
		return fmt.Errorf("credential: delete %q: %w", name, err)
	}
	return nil
}

// Names returns a list of credential names for dropdown use.
func (m *CredentialManager) Names() ([]string, error) {
	creds, err := m.List()
	if err != nil {
		return nil, err
	}
	names := make([]string, len(creds))
	for i, c := range creds {
		names[i] = c.Name
	}
	return names, nil
}
