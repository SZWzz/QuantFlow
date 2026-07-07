//go:build darwin

package auth

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os/exec"
	"strings"
)

// LoadOrCreateMasterKey loads an existing master key from the macOS Keychain,
// or generates a new 256-bit key and stores it in the Keychain.
//
// It uses the security(1) CLI to interact with the system Keychain.
// The key is stored as a generic password with account "quantflow"
// and service "quantflow-master-key".
func LoadOrCreateMasterKey() (*MasterKey, error) {
	key, err := loadMasterKey()
	if err == nil {
		return NewMasterKey(key), nil
	}
	key, err = createMasterKey()
	if err != nil {
		return nil, fmt.Errorf("master key: %w", err)
	}
	return NewMasterKey(key), nil
}

func loadMasterKey() ([32]byte, error) {
	cmd := exec.Command("security", "find-generic-password",
		"-a", "quantflow", "-s", "quantflow-master-key", "-w")
	out, err := cmd.Output()
	if err != nil {
		return [32]byte{}, fmt.Errorf("keychain read: %w", err)
	}
	decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(out)))
	if err != nil {
		return [32]byte{}, fmt.Errorf("decode key: %w", err)
	}
	if len(decoded) != 32 {
		return [32]byte{}, fmt.Errorf("unexpected key length: %d", len(decoded))
	}
	var key [32]byte
	copy(key[:], decoded)
	return key, nil
}

func createMasterKey() ([32]byte, error) {
	var key [32]byte
	if _, err := rand.Read(key[:]); err != nil {
		return [32]byte{}, fmt.Errorf("generate key: %w", err)
	}
	encoded := base64.StdEncoding.EncodeToString(key[:])
	cmd := exec.Command("security", "add-generic-password",
		"-a", "quantflow", "-s", "quantflow-master-key", "-w", encoded, "-U")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return [32]byte{}, fmt.Errorf("keychain write: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	return key, nil
}
