package nodes

import (
	"context"
	"fmt"

	"quantflow/internal/workflow"
)

// IfConditionNode evaluates a single condition: condition_value op threshold.
// Outputs a boolean true_branch — true if the condition is met, false otherwise.
type IfConditionNode struct {
	id     string
	params map[string]any
}

// NewIfConditionNode creates a new IfConditionNode.
func NewIfConditionNode(id string, params map[string]any) (workflow.BaseNode, error) {
	return &IfConditionNode{id: id, params: params}, nil
}

func (n *IfConditionNode) ID() string       { return n.id }
func (n *IfConditionNode) NodeType() string { return "if_condition" }
func (n *IfConditionNode) Category() string { return "control" }

func (n *IfConditionNode) InputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "condition_value", Type: workflow.PortNumber, Required: true},
	}
}

func (n *IfConditionNode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "true_branch", Type: workflow.PortBoolean, Required: false},
	}
}

func (n *IfConditionNode) ParamSchema() []workflow.ParamDef {
	return []workflow.ParamDef{
		{Name: "op", Type: "string", Default: "gt",
			Description: "Comparison operator: gt, lt, gte, lte, eq"},
		{Name: "threshold", Type: "float", Default: float64(0),
			Description: "Threshold value to compare against"},
	}
}

func (n *IfConditionNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any, nctx *workflow.NodeContext) (map[string]any, error) {
	op := getStringParam(params, "op", "gt")
	threshold := getFloatParam(params, "threshold", 0)

	raw, ok := inputs["condition_value"]
	if !ok || raw == nil {
		return nil, fmt.Errorf("if_condition: condition_value input is required")
	}

	var value float64
	switch v := raw.(type) {
	case float64:
		value = v
	case int:
		value = float64(v)
	default:
		return nil, fmt.Errorf("if_condition: condition_value must be a number, got %T", raw)
	}

	var result bool
	switch op {
	case "gt":
		result = value > threshold
	case "lt":
		result = value < threshold
	case "gte":
		result = value >= threshold
	case "lte":
		result = value <= threshold
	case "eq":
		result = value == threshold
	default:
		return nil, fmt.Errorf("if_condition: unknown op %q, expected gt/lt/gte/lte/eq", op)
	}

	return map[string]any{"true_branch": result}, nil
}

func (n *IfConditionNode) Validate() error {
	op := getStringParam(n.params, "op", "gt")
	switch op {
	case "gt", "lt", "gte", "lte", "eq":
		return nil
	default:
		return fmt.Errorf("if_condition: invalid op %q, expected gt/lt/gte/lte/eq", op)
	}
}
