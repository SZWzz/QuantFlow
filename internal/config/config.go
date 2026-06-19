package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config holds all application configuration.
type Config struct {
	Version  string `yaml:"version"`
	LogLevel string `yaml:"log_level"`
	DBPath   string `yaml:"db_path"`
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		Version:  "0.0.1",
		LogLevel: "info",
		DBPath:   "data/quantflow.db",
	}
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

// Load reads config from default locations, falling back to defaults.
func Load() (*Config, error) {
	paths := []string{"config.yaml", "config/config.yaml"}
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return loadFile(p)
		}
	}
	return DefaultConfig(), nil
}

func loadFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}
	cfg := DefaultConfig()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err)
	}
	return cfg, nil
}
