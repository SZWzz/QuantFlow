//go:build linux

package auth

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// LoadOrCreateMasterKey loads an existing master key from secret-tool (libsecret)
// or falls back to a file at ~/.config/quantflow/master.key.
// If no key exists, it generates a new 256-bit key and stores it.
func LoadOrCreateMasterKey() (*MasterKey, error) {
	key, err := loadSecretTool()
	if err == nil {
		return NewMasterKey(key), nil
	}
	key, err = loadKeyFile()
	if err == nil {
		return NewMasterKey(key), nil
	}
	key, err = createMasterKey()
	if err != nil {
		return nil, fmt.Errorf("master key: %w", err)
	}
	return NewMasterKey(key), nil
}

func loadSecretTool() ([32]byte, error) {
	cmd := exec.Command("secret-tool", "lookup", "quantflow", "master-key")
	out, err := cmd.Output()
	if err != nil {
		return [32]byte{}, fmt.Errorf("secret-tool lookup: %w", err)
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

func loadKeyFile() ([32]byte, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return [32]byte{}, fmt.Errorf("home dir: %w", err)
	}
	data, err := os.ReadFile(filepath.Join(home, ".config", "quantflow", "master.key"))
	if err != nil {
		return [32]byte{}, fmt.Errorf("key file read: %w", err)
	}
	decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(data)))
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

	if cmd := exec.Command("secret-tool", "store",
		"--label=QuantFlow Master Key", "quantflow", "master-key"); true {
		cmd := exec.Command("secret-tool", "store",
			"--label=QuantFlow Master Key", "quantflow", "master-key")
		cmd.Stdin = strings.NewReader(encoded)
		if err := cmd.Run(); err == nil {
			return key, nil
		}
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return [32]byte{}, fmt.Errorf("home dir: %w", err)
	}
	dir := filepath.Join(home, ".config", "quantflow")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return [32]byte{}, fmt.Errorf("mkdir: %w", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "master.key"), []byte(encoded), 0600); err != nil {
		return [32]byte{}, fmt.Errorf("key file write: %w", err)
	}
	return key, nil
}
