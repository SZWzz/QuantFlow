package config

import (
	"fmt"
	"os"
	"path/filepath"

	"quantflow/internal/auth"

	"gopkg.in/yaml.v3"
)

// Config holds all application configuration.
type Config struct {
	path string // resolved absolute path, set by Load
	cm   *auth.CredentialManager // optional, set at startup

	Version              string            `yaml:"version"`
	UpdateCheckInterval  string            `yaml:"update_check_interval"`
	LogLevel             string            `yaml:"log_level"`
	DBPath         string            `yaml:"db_path"`
	// APIKeys stores optional API keys loaded from config YAML.
	// Deprecated: API keys should be stored in the CredentialManager (AES-256-GCM
	// encrypted). This field is populated during Config.Load() for backward
	// compatibility and is migrated to CredentialManager at app startup.
	// After migration, the map is emptied. New installations should use
	// CredentialManager directly via the frontend's ModelRegistryPanel.
	APIKeys map[string]string `yaml:"api_keys"`
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		Version:              "0.0.1",
		UpdateCheckInterval:  "daily",
		LogLevel:             "info",
		DBPath:   "data/quantflow.db",
	}
}

// Save writes the configuration back to its loaded path.
func (c *Config) Save() error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	savePath := c.path
	if savePath == "" {
		savePath = "config.yaml"
	}
	if err := os.WriteFile(savePath, data, 0644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

// SetCredentialManager injects a CredentialManager for CM-first key lookup.
func (c *Config) SetCredentialManager(cm *auth.CredentialManager) {
	c.cm = cm
}

// GetAPIKey retrieves an API key for the given provider name.
// Checks CredentialManager first, then falls back to environment variables.
// Does NOT check the deprecated Config.APIKeys map — API keys in YAML are
// migrated to CredentialManager on first startup and should not be read from
// config file after that point.
func (c *Config) GetAPIKey(name string) string {
	if c.cm != nil {
		creds, err := c.cm.List()
		if err == nil {
			for _, cred := range creds {
				if cred.Name == name+"_api_key" {
					if key, ok := cred.Keys["api_key"]; ok && key != "" {
						return key
					}
				}
			}
		}
	}
	envKey := fmt.Sprintf("%s_API_KEY", toEnvName(name))
	return os.Getenv(envKey)
}

func toEnvName(name string) string {
	var result []byte
	for i, ch := range name {
		if ch >= 'a' && ch <= 'z' && (i == 0 || name[i-1] == '_') {
			result = append(result, byte(ch-32)) // uppercase
		} else {
			result = append(result, byte(ch))
		}
	}
	return string(result)
}

// ResolveDBPath resolves a relative DB path against the executable directory
// so that wails dev (CWD=project root) and the built .app bundle (CWD varies)
// always land on the same database file. Absolute paths are returned unchanged.
func ResolveDBPath(dbPath string) string {
	if filepath.IsAbs(dbPath) {
		return dbPath
	}
	execPath, err := os.Executable()
	if err != nil {
		return dbPath
	}
	return filepath.Join(filepath.Dir(execPath), dbPath)
}

// Load reads config from the given path.
// If the file does not exist, returns DefaultConfig with the path set.
func Load(path string) (*Config, error) {
	if _, err := os.Stat(path); err != nil {
		cfg := DefaultConfig()
		cfg.path = path
		return cfg, nil
	}
	return loadFile(path)
}

func loadFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}
	cfg := DefaultConfig()
	cfg.path = path
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err)
	}
	return cfg, nil
}
