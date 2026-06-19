package workflow

import (
	"fmt"
	"sync"
)

// NodeConstructor is a factory function that creates a node instance.
type NodeConstructor func(id string, params map[string]any) (BaseNode, error)

// NodeMeta holds metadata about a registered node type.
type NodeMeta struct {
	NodeType string `json:"node_type"`
	Category string `json:"category"`
}

// NodeRegistry manages node type registration and instantiation.
// It is safe for concurrent use.
type NodeRegistry struct {
	mu           sync.RWMutex
	constructors map[string]NodeConstructor
	categories   map[string]string
}

// NewRegistry creates an empty NodeRegistry.
func NewRegistry() *NodeRegistry {
	return &NodeRegistry{
		constructors: make(map[string]NodeConstructor),
		categories:   make(map[string]string),
	}
}

// Register adds a node type constructor.
func (r *NodeRegistry) Register(nodeType string, ctor NodeConstructor) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.constructors[nodeType] = ctor
}

// RegisterWithCategory registers a node type with its category metadata.
func (r *NodeRegistry) RegisterWithCategory(nodeType string, ctor NodeConstructor, category string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.constructors[nodeType] = ctor
	r.categories[nodeType] = category
}

// Create instantiates a node of the given type.
func (r *NodeRegistry) Create(nodeType string, id string, params map[string]any) (BaseNode, error) {
	r.mu.RLock()
	ctor, ok := r.constructors[nodeType]
	r.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("unknown node type: %q", nodeType)
	}
	return ctor(id, params)
}

// ListAll returns metadata for all registered node types.
func (r *NodeRegistry) ListAll() []NodeMeta {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []NodeMeta
	for nodeType := range r.constructors {
		result = append(result, NodeMeta{NodeType: nodeType, Category: r.categories[nodeType]})
	}
	return result
}

// Has returns true if the node type is registered.
func (r *NodeRegistry) Has(nodeType string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.constructors[nodeType]
	return ok
}
