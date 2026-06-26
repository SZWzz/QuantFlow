package nodes

import (
	"context"
	"fmt"

	"quantflow/internal/workflow"
)

// IfElseNode selects values from a or b based on a condition series.
// If condition[i] > 0 then a[i] else b[i].
type IfElseNode struct{ id string; params map[string]any }

func NewIfElseNode(id string, params map[string]any) (workflow.BaseNode, error) {
	return &IfElseNode{id: id, params: params}, nil
}

func (n *IfElseNode) ID() string       { return n.id }
func (n *IfElseNode) NodeType() string { return "if_else" }
func (n *IfElseNode) Category() string { return "alpha" }

func (n *IfElseNode) InputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "condition", Type: workflow.PortSeries, Required: true},
		{Name: "a", Type: workflow.PortSeries, Required: true},
		{Name: "b", Type: workflow.PortSeries, Required: true},
	}
}

func (n *IfElseNode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{{Name: "result", Type: workflow.PortSeries, Required: false}}
}

func (n *IfElseNode) ParamSchema() []workflow.ParamDef { return nil }

func (n *IfElseNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any, nctx *workflow.NodeContext) (map[string]any, error) {
	condition := extractFloatSlice(inputs["condition"])
	if condition == nil {
		return nil, fmt.Errorf("if_else: condition input is required")
	}
	a := extractFloatSlice(inputs["a"])
	if a == nil {
		return nil, fmt.Errorf("if_else: a input is required")
	}
	b := extractFloatSlice(inputs["b"])
	if b == nil {
		return nil, fmt.Errorf("if_else: b input is required")
	}
	if len(condition) != len(a) || len(condition) != len(b) {
		return nil, fmt.Errorf("if_else: all inputs must have same length (condition=%d, a=%d, b=%d)",
			len(condition), len(a), len(b))
	}

	result := make([]float64, len(condition))
	for i := range condition {
		if condition[i] > 0 {
			result[i] = a[i]
		} else {
			result[i] = b[i]
		}
	}

	return map[string]any{"result": result}, nil
}

func (n *IfElseNode) Validate() error { return nil }
