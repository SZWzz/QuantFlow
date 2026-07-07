package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_Defaults(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, "config.yaml")
	os.WriteFile(cfgPath, []byte(`version: "0.0.1"
log_level: "info"
db_path: "data/quantflow.db"`), 0644)

	cfg, err := loadFile(cfgPath)
	if err != nil {
		t.Fatalf("loadFile() error = %v", err)
	}
	if cfg.Version != "0.0.1" {
		t.Errorf("version = %q, want %q", cfg.Version, "0.0.1")
	}
	if cfg.LogLevel != "info" {
		t.Errorf("log_level = %q, want %q", cfg.LogLevel, "info")
	}
	if cfg.DBPath != "data/quantflow.db" {
		t.Errorf("db_path = %q, want %q", cfg.DBPath, "data/quantflow.db")
	}
}

func TestLoad_MissingFile(t *testing.T) {
	_, err := loadFile("/nonexistent/path/config.yaml")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestLoadConfig_WithFile(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, "config.yaml")
	// Only override log_level — version and db_path should stay as defaults
	os.WriteFile(cfgPath, []byte(`log_level: "debug"`), 0644)

	cfg, err := loadFile(cfgPath)
	if err != nil {
		t.Fatalf("loadFile() error = %v", err)
	}
	// These should come from the file
	if cfg.LogLevel != "debug" {
		t.Errorf("log_level = %q, want %q", cfg.LogLevel, "debug")
	}
	// These should remain as defaults since not set in the file
	if cfg.Version != "0.0.1" {
		t.Errorf("version = %q, want %q (default)", cfg.Version, "0.0.1")
	}
	if cfg.DBPath != "data/quantflow.db" {
		t.Errorf("db_path = %q, want %q (default)", cfg.DBPath, "data/quantflow.db")
	}
}

func TestLoadConfig_MissingFile(t *testing.T) {
	// Load() with nonexistent path should return defaults, not error
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, "nonexistent.yaml")

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Version != "0.0.1" {
		t.Errorf("version = %q, want default %q", cfg.Version, "0.0.1")
	}
	if cfg.LogLevel != "info" {
		t.Errorf("log_level = %q, want default %q", cfg.LogLevel, "info")
	}
	if cfg.DBPath != "data/quantflow.db" {
		t.Errorf("db_path = %q, want default %q", cfg.DBPath, "data/quantflow.db")
	}
}

func TestLoadConfig_FromConfigDir(t *testing.T) {
	tmp := t.TempDir()
	configDir := filepath.Join(tmp, "config")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	cfgPath := filepath.Join(configDir, "config.yaml")
	os.WriteFile(cfgPath, []byte(`log_level: "warn"`), 0644)

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.LogLevel != "warn" {
		t.Errorf("log_level = %q, want %q", cfg.LogLevel, "warn")
	}
}
