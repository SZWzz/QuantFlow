package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
)

// Capability represents a tool the AI agent can call.
type Capability struct {
	Name        string
	Description string          // LLM function description
	Parameters  json.RawMessage // JSON Schema for parameters
	Handler     CapabilityHandler
}

// CapabilityHandler executes a capability with JSON-encoded arguments.
// Returns a JSON-encoded result string.
type CapabilityHandler func(ctx context.Context, args json.RawMessage) (string, error)

// CapabilityRegistry manages available Agent capabilities (tools).
// Thread-safe.
type CapabilityRegistry struct {
	mu           sync.RWMutex
	capabilities map[string]*Capability
}

// NewCapabilityRegistry creates an empty registry.
func NewCapabilityRegistry() *CapabilityRegistry {
	return &CapabilityRegistry{
		capabilities: make(map[string]*Capability),
	}
}

// Register adds a capability. Returns error if name already exists.
func (r *CapabilityRegistry) Register(c *Capability) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.capabilities[c.Name]; exists {
		return fmt.Errorf("capability %q already registered", c.Name)
	}
	r.capabilities[c.Name] = c
	return nil
}

// Execute runs a capability by name with JSON args. Returns the result string.
func (r *CapabilityRegistry) Execute(ctx context.Context, name string, args json.RawMessage) (string, error) {
	r.mu.RLock()
	c, ok := r.capabilities[name]
	r.mu.RUnlock()
	if !ok {
		return "", fmt.Errorf("unknown capability: %q", name)
	}
	return c.Handler(ctx, args)
}

// GetCapability returns the capability by name, or nil if not found.
func (r *CapabilityRegistry) GetCapability(name string) *Capability {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.capabilities[name]
}

// Has returns true if the capability is registered.
func (r *CapabilityRegistry) Has(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.capabilities[name]
	return ok
}

// LLMFunctionDef is the format LLMs expect for tool/function definitions.
type LLMFunctionDef struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

// ListForLLM returns capabilities formatted as LLM function definitions.
// If names is empty, returns all; otherwise filters to the given names.
func (r *CapabilityRegistry) ListForLLM(names []string) []LLMFunctionDef {
	r.mu.RLock()
	defer r.mu.RUnlock()

	nameSet := make(map[string]bool, len(names))
	filterAll := len(names) == 0
	if !filterAll {
		for _, n := range names {
			nameSet[n] = true
		}
	}

	var result []LLMFunctionDef
	for name, c := range r.capabilities {
		if !filterAll && !nameSet[name] {
			continue
		}
		result = append(result, LLMFunctionDef{
			Name:        c.Name,
			Description: c.Description,
			Parameters:  c.Parameters,
		})
	}
	return result
}

// ListAll returns all registered capability names.
func (r *CapabilityRegistry) ListAll() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var names []string
	for name := range r.capabilities {
		names = append(names, name)
	}
	return names
}
