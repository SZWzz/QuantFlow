package nodes

import (
	"context"
	"fmt"
	"quantflow/internal/workflow"
)

// CompareNode performs element-wise comparison of two series: a[i] op b[i] -> 1 else 0.
type CompareNode struct {
	id     string
	params map[string]any
}

func NewCompareNode(id string, params map[string]any) (workflow.BaseNode, error) {
	return &CompareNode{id: id, params: params}, nil
}

func (n *CompareNode) ID() string       { return n.id }
func (n *CompareNode) NodeType() string { return "compare" }
func (n *CompareNode) Category() string { return "alpha" }

func (n *CompareNode) InputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "a", Type: workflow.PortSeries, Required: true},
		{Name: "b", Type: workflow.PortSeries, Required: true},
	}
}

func (n *CompareNode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{{Name: "result", Type: workflow.PortSeries, Required: false}}
}

func (n *CompareNode) ParamSchema() []workflow.ParamDef {
	return []workflow.ParamDef{
		{Name: "op", Type: "string", Default: "gt", Description: "Comparison: gt, lt, gte, lte, eq, neq"},
	}
}

func (n *CompareNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any, nctx *workflow.NodeContext) (map[string]any, error) {
	a := extractFloatSlice(inputs["a"])
	if a == nil {
		return nil, fmt.Errorf("compare: input a is required")
	}
	b := extractFloatSlice(inputs["b"])
	if b == nil {
		return nil, fmt.Errorf("compare: input b is required")
	}
	if len(a) != len(b) {
		return nil, fmt.Errorf("compare: a(%d) and b(%d) must have same length", len(a), len(b))
	}

	op := getStringParam(params, "op", "gt")

	result := make([]float64, len(a))
	for i := range a {
		var ok bool
		switch op {
		case "gt":
			ok = a[i] > b[i]
		case "lt":
			ok = a[i] < b[i]
		case "gte":
			ok = a[i] >= b[i]
		case "lte":
			ok = a[i] <= b[i]
		case "eq":
			ok = a[i] == b[i]
		case "neq":
			ok = a[i] != b[i]
		default:
			return nil, fmt.Errorf("compare: unknown op %q, expected gt/lt/gte/lte/eq/neq", op)
		}
		if ok {
			result[i] = 1
		}
	}

	return map[string]any{"result": result}, nil
}

func (n *CompareNode) Validate() error {
	op := getStringParam(n.params, "op", "gt")
	switch op {
	case "gt", "lt", "gte", "lte", "eq", "neq":
		return nil
	default:
		return fmt.Errorf("compare: invalid op %q, expected gt/lt/gte/lte/eq/neq", op)
	}
}
