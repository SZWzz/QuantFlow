package workflow

import (
	"fmt"
	"sync"
)

// NodeConstructor is a factory function that creates a node instance.
type NodeConstructor func(id string, params map[string]any) (BaseNode, error)

// NodePortInfo mirrors PortDefinition for JSON serialization.
type NodePortInfo struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Required bool   `json:"required"`
}

// NodeMeta holds metadata about a registered node type.
type NodeMeta struct {
	NodeType    string         `json:"node_type"`
	Category    string         `json:"category"`
	InputPorts  []NodePortInfo `json:"input_ports"`
	OutputPorts []NodePortInfo `json:"output_ports"`
	Params      []ParamDef     `json:"params"`
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

// ListAll returns metadata for all registered node types, including their
// input/output ports and parameter schemas (read from a temporary instance).
func (r *NodeRegistry) ListAll() []NodeMeta {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []NodeMeta
	for nodeType, ctor := range r.constructors {
		meta := NodeMeta{NodeType: nodeType, Category: r.categories[nodeType]}
		// Create a temporary instance to read ports and params.
		if node, err := ctor("_meta", nil); err == nil {
			for _, p := range node.InputPorts() {
				meta.InputPorts = append(meta.InputPorts, NodePortInfo{Name: p.Name, Type: string(p.Type), Required: p.Required})
			}
			for _, p := range node.OutputPorts() {
				meta.OutputPorts = append(meta.OutputPorts, NodePortInfo{Name: p.Name, Type: string(p.Type), Required: p.Required})
			}
			meta.Params = node.ParamSchema()
		}
		result = append(result, meta)
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
