package nodes

import (
	"context"
	"fmt"

	"quantflow/internal/workflow"
)

// ArithmeticNode performs element-wise arithmetic on two series: a[i] op b[i].
type ArithmeticNode struct{ id string; params map[string]any }

func NewArithmeticNode(id string, params map[string]any) (workflow.BaseNode, error) {
	return &ArithmeticNode{id: id, params: params}, nil
}

func (n *ArithmeticNode) ID() string       { return n.id }
func (n *ArithmeticNode) NodeType() string { return "arithmetic" }
func (n *ArithmeticNode) Category() string { return "alpha" }

func (n *ArithmeticNode) InputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "a", Type: workflow.PortSeries, Required: true},
		{Name: "b", Type: workflow.PortSeries, Required: true},
	}
}

func (n *ArithmeticNode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{{Name: "result", Type: workflow.PortSeries, Required: false}}
}

func (n *ArithmeticNode) ParamSchema() []workflow.ParamDef {
	return []workflow.ParamDef{
		{Name: "op", Type: "string", Default: "add", Description: "Operation: add, sub, mul, div"},
	}
}

func (n *ArithmeticNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any) (map[string]any, error) {
	a := extractFloatSlice(inputs["a"])
	if a == nil {
		return nil, fmt.Errorf("arithmetic: input a is required")
	}
	b := extractFloatSlice(inputs["b"])
	if b == nil {
		return nil, fmt.Errorf("arithmetic: input b is required")
	}
	if len(a) != len(b) {
		return nil, fmt.Errorf("arithmetic: a(%d) and b(%d) must have same length", len(a), len(b))
	}

	op := getStringParam(params, "op", "add")

	result := make([]float64, len(a))
	for i := range a {
		switch op {
		case "add":
			result[i] = a[i] + b[i]
		case "sub":
			result[i] = a[i] - b[i]
		case "mul":
			result[i] = a[i] * b[i]
		case "div":
			if b[i] == 0 {
				result[i] = 0
			} else {
				result[i] = a[i] / b[i]
			}
		default:
			return nil, fmt.Errorf("arithmetic: unknown op %q, expected add/sub/mul/div", op)
		}
	}

	return map[string]any{"result": result}, nil
}

func (n *ArithmeticNode) Validate() error {
	op := getStringParam(n.params, "op", "add")
	switch op {
	case "add", "sub", "mul", "div":
		return nil
	default:
		return fmt.Errorf("arithmetic: invalid op %q, expected add/sub/mul/div", op)
	}
}
