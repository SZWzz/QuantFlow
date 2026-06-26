package nodes

import (
	"context"
	"fmt"

	"quantflow/internal/workflow"
)

// BoolCombineNode performs element-wise boolean combination of two series.
// Treats values > 0 as true.
type BoolCombineNode struct{ id string; params map[string]any }

func NewBoolCombineNode(id string, params map[string]any) (workflow.BaseNode, error) {
	return &BoolCombineNode{id: id, params: params}, nil
}

func (n *BoolCombineNode) ID() string       { return n.id }
func (n *BoolCombineNode) NodeType() string { return "bool_combine" }
func (n *BoolCombineNode) Category() string { return "alpha" }

func (n *BoolCombineNode) InputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "a", Type: workflow.PortSeries, Required: true},
		{Name: "b", Type: workflow.PortSeries, Required: true},
	}
}

func (n *BoolCombineNode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{{Name: "result", Type: workflow.PortSeries, Required: false}}
}

func (n *BoolCombineNode) ParamSchema() []workflow.ParamDef {
	return []workflow.ParamDef{
		{Name: "op", Type: "string", Default: "and", Description: "Boolean operation: and, or, xor"},
	}
}

func (n *BoolCombineNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any, nctx *workflow.NodeContext) (map[string]any, error) {
	a := extractFloatSlice(inputs["a"])
	if a == nil {
		return nil, fmt.Errorf("bool_combine: input a is required")
	}
	b := extractFloatSlice(inputs["b"])
	if b == nil {
		return nil, fmt.Errorf("bool_combine: input b is required")
	}
	if len(a) != len(b) {
		return nil, fmt.Errorf("bool_combine: a(%d) and b(%d) must have same length", len(a), len(b))
	}

	op := getStringParam(params, "op", "and")

	result := make([]float64, len(a))
	for i := range a {
		av := a[i] > 0
		bv := b[i] > 0
		var ok bool
		switch op {
		case "and":
			ok = av && bv
		case "or":
			ok = av || bv
		case "xor":
			ok = av != bv
		default:
			return nil, fmt.Errorf("bool_combine: unknown op %q, expected and/or/xor", op)
		}
		if ok {
			result[i] = 1
		}
	}

	return map[string]any{"result": result}, nil
}

func (n *BoolCombineNode) Validate() error {
	op := getStringParam(n.params, "op", "and")
	switch op {
	case "and", "or", "xor":
		return nil
	default:
		return fmt.Errorf("bool_combine: invalid op %q, expected and/or/xor", op)
	}
}
