package ai

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// AgentProfile defines an AI agent's personality, system prompt, and tool access.
type AgentProfile struct {
	Name         string   `yaml:"name" json:"name"`
	Display      string   `yaml:"display" json:"display"`
	SystemPrompt string   `yaml:"system_prompt" json:"system_prompt"`
	Tools        []string `yaml:"tools" json:"tools"`
	DefaultLLM   string   `yaml:"default_llm" json:"default_llm"`
	MaxSteps     int      `yaml:"max_steps" json:"max_steps"`
}

// ProfileManager loads and caches agent profiles from YAML files.
type ProfileManager struct {
	profiles map[string]*AgentProfile
}

// NewProfileManager creates an empty ProfileManager.
func NewProfileManager() *ProfileManager {
	return &ProfileManager{
		profiles: make(map[string]*AgentProfile),
	}
}

// LoadFile loads a single YAML profile file.
func (pm *ProfileManager) LoadFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read profile %s: %w", path, err)
	}
	var profile AgentProfile
	if err := yaml.Unmarshal(data, &profile); err != nil {
		return fmt.Errorf("parse profile %s: %w", path, err)
	}
	if profile.Name == "" {
		return fmt.Errorf("profile %s has no name field", path)
	}
	if profile.MaxSteps <= 0 {
		profile.MaxSteps = 8
	}
	pm.profiles[profile.Name] = &profile
	return nil
}

// LoadDir loads all .yaml and .yml files from a directory.
func (pm *ProfileManager) LoadDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read profile dir %s: %w", dir, err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := filepath.Ext(entry.Name())
		if ext != ".yaml" && ext != ".yml" {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		if err := pm.LoadFile(path); err != nil {
			return err
		}
	}
	return nil
}

// Get returns a profile by name or an error if not found.
func (pm *ProfileManager) Get(name string) (*AgentProfile, error) {
	p, ok := pm.profiles[name]
	if !ok {
		return nil, fmt.Errorf("profile %q not found", name)
	}
	return p, nil
}

// List returns all loaded profiles.
func (pm *ProfileManager) List() []*AgentProfile {
	var result []*AgentProfile
	for _, p := range pm.profiles {
		result = append(result, p)
	}
	return result
}
