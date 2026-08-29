package nodes

import (
	"context"
	"fmt"
	"math"
	"quantflow/internal/workflow"
)

// MathOpNode performs a binary math operation on inputs a and b.
// For sqrt and log operations only input a is used.
type MathOpNode struct {
	id     string
	params map[string]any
}

// NewMathOpNode creates a new MathOpNode.
func NewMathOpNode(id string, params map[string]any) (workflow.BaseNode, error) {
	return &MathOpNode{id: id, params: params}, nil
}

func (n *MathOpNode) ID() string       { return n.id }
func (n *MathOpNode) NodeType() string { return "math_op" }
func (n *MathOpNode) Category() string { return "utility" }

func (n *MathOpNode) InputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "a", Type: workflow.PortNumber, Required: true},
		{Name: "b", Type: workflow.PortNumber, Required: false},
	}
}

func (n *MathOpNode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "result", Type: workflow.PortNumber, Required: false},
	}
}

func (n *MathOpNode) ParamSchema() []workflow.ParamDef {
	return []workflow.ParamDef{
		{Name: "operation", Type: "string", Default: "add", Description: "Operation: add, sub, mul, div, pow, sqrt, log"},
	}
}

func (n *MathOpNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any, nctx *workflow.NodeContext) (map[string]any, error) {
	a := extractFloat(inputs["a"])
	if a == nil {
		return nil, fmt.Errorf("math_op: input a is required")
	}

	op := getStringParam(params, "operation", "add")
	var b *float64
	if op != "sqrt" && op != "log" {
		b = extractFloat(inputs["b"])
		if b == nil {
			return nil, fmt.Errorf("math_op: input b is required for operation %q", op)
		}
	}

	var result float64

	switch op {
	case "add":
		result = *a + *b
	case "sub":
		result = *a - *b
	case "mul":
		result = *a * *b
	case "div":
		if *b == 0 {
			return nil, fmt.Errorf("math_op: division by zero")
		}
		result = *a / *b
	case "pow":
		result = math.Pow(*a, *b)
	case "sqrt":
		if *a < 0 {
			return nil, fmt.Errorf("math_op: cannot take square root of negative number %v", *a)
		}
		result = math.Sqrt(*a)
	case "log":
		if *a <= 0 {
			return nil, fmt.Errorf("math_op: cannot take log of non-positive number %v", *a)
		}
		result = math.Log(*a)
	default:
		return nil, fmt.Errorf("math_op: unknown operation %q, expected add/sub/mul/div/pow/sqrt/log", op)
	}

	return map[string]any{"result": result}, nil
}

func (n *MathOpNode) Validate() error {
	op := getStringParam(n.params, "operation", "add")
	switch op {
	case "add", "sub", "mul", "div", "pow", "sqrt", "log":
		return nil
	default:
		return fmt.Errorf("math_op: invalid operation %q, expected add/sub/mul/div/pow/sqrt/log", op)
	}
}
