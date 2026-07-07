// Package auth provides credential encryption and management for API keys.
package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
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
	db        *sql.DB
	masterKey *MasterKey
	oldGCM    cipher.AEAD // for backward compatibility with deriveKey-based encryption
}

// NewCredentialManager creates a credential manager with AES-256-GCM encryption.
// The encryption key is stored in the OS keychain (macOS) or a local key file (Linux),
// replacing the previous hostname-derived key approach.
func NewCredentialManager(db *sql.DB) (*CredentialManager, error) {
	mk, err := LoadOrCreateMasterKey()
	if err != nil {
		return nil, fmt.Errorf("credential: load master key: %w", err)
	}
	return newCredentialManager(mk, db)
}

// newCredentialManager creates a CredentialManager with a given MasterKey.
// It also sets up the old hostname-derived GCM for backward compatibility.
func newCredentialManager(mk *MasterKey, db *sql.DB) (*CredentialManager, error) {
	cm := &CredentialManager{db: db, masterKey: mk}
	oldKey := deriveKey()
	block, err := aes.NewCipher(oldKey)
	if err != nil {
		return nil, fmt.Errorf("credential: create old cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("credential: create old GCM: %w", err)
	}
	cm.oldGCM = gcm
	return cm, nil
}

// deriveKey creates a 256-bit key from machine identity + fixed salt.
// Retained for backward compatibility with data encrypted before the OS keychain migration.
func deriveKey() []byte {
	hostname, _ := os.Hostname()
	material := hostname + ":quantflow-cred-salt-v1"
	h := sha256.Sum256([]byte(material))
	return h[:]
}

// encrypt encrypts plaintext with the master key and returns base64-encoded ciphertext.
func (m *CredentialManager) encrypt(plaintext []byte) (string, error) {
	ct, err := m.masterKey.Encrypt(plaintext)
	if err != nil {
		return "", fmt.Errorf("credential: encrypt: %w", err)
	}
	return base64.StdEncoding.EncodeToString(ct), nil
}

// decryptWithMigrate decrypts a base64-encoded ciphertext. It returns the plaintext
// and a boolean indicating whether the data was decrypted with the old key
// (meaning it should be re-encrypted with the master key for migration).
func (m *CredentialManager) decryptWithMigrate(encoded string) ([]byte, bool, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, false, fmt.Errorf("credential: decode: %w", err)
	}
	plain, err := m.masterKey.Decrypt(ciphertext)
	if err == nil {
		return plain, false, nil
	}
	if m.oldGCM != nil {
		plain, err := m.oldGCMDecrypt(encoded)
		if err == nil {
			return plain, true, nil
		}
	}
	return nil, false, fmt.Errorf("credential: decrypt: %w", err)
}

// oldGCMDecrypt decrypts using the old hostname-derived key (backward compatibility).
func (m *CredentialManager) oldGCMDecrypt(encoded string) ([]byte, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("credential: decode: %w", err)
	}
	nonceSize := m.oldGCM.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("credential: ciphertext too short")
	}
	nonce, ct := ciphertext[:nonceSize], ciphertext[nonceSize:]
	return m.oldGCM.Open(nil, nonce, ct, nil)
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
		plain, migrated, err := m.decryptWithMigrate(row.Data)
		if err != nil {
			return nil, fmt.Errorf("credential: decrypt %q: %w", row.Name, err)
		}
		var keys map[string]string
		if err := json.Unmarshal(plain, &keys); err != nil {
			return nil, fmt.Errorf("credential: unmarshal %q: %w", row.Name, err)
		}
		if migrated {
			newEnc, err := m.encrypt(plain)
			if err == nil {
				_, _ = m.db.Exec(`UPDATE credentials SET data=? WHERE id=?`, newEnc, row.ID)
			}
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
