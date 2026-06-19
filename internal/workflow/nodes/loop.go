package nodes

import (
	"context"
	"fmt"

	"quantflow/internal/workflow"
)

// LoopNode batches input items for downstream processing.
type LoopNode struct {
	id     string
	params map[string]any
}

// NewLoopNode creates a new LoopNode.
func NewLoopNode(id string, params map[string]any) (workflow.BaseNode, error) {
	return &LoopNode{id: id, params: params}, nil
}

func (n *LoopNode) ID() string       { return n.id }
func (n *LoopNode) NodeType() string { return "loop" }
func (n *LoopNode) Category() string { return "control" }

func (n *LoopNode) InputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "items", Type: workflow.PortAny, Required: true},
	}
}

func (n *LoopNode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "batched", Type: workflow.PortAny, Required: false},
	}
}

func (n *LoopNode) ParamSchema() []workflow.ParamDef { return nil }

func (n *LoopNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any) (map[string]any, error) {
	raw, ok := inputs["items"]
	if !ok {
		return nil, fmt.Errorf("loop: missing required input 'items'")
	}
	switch items := raw.(type) {
	case []any:
		return map[string]any{"batched": items}, nil
	case []string:
		result := make([]any, len(items))
		for i, s := range items {
			result[i] = s
		}
		return map[string]any{"batched": result}, nil
	default:
		return nil, fmt.Errorf("loop: items must be an array, got %T", raw)
	}
}

func (n *LoopNode) Validate() error { return nil }
