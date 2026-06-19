package nodes

import (
	"context"
	"fmt"

	"quantflow/internal/workflow"
)

// RollingMaxMinNode computes rolling maximum or minimum over N periods.
type RollingMaxMinNode struct{ id string; params map[string]any }

func NewRollingMaxMinNode(id string, params map[string]any) (workflow.BaseNode, error) {
	return &RollingMaxMinNode{id: id, params: params}, nil
}

func (n *RollingMaxMinNode) ID() string       { return n.id }
func (n *RollingMaxMinNode) NodeType() string { return "rolling_maxmin" }
func (n *RollingMaxMinNode) Category() string { return "alpha" }

func (n *RollingMaxMinNode) InputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "values", Type: workflow.PortSeries, Required: true},
	}
}

func (n *RollingMaxMinNode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{{Name: "result", Type: workflow.PortSeries, Required: false}}
}

func (n *RollingMaxMinNode) ParamSchema() []workflow.ParamDef {
	return []workflow.ParamDef{
		{Name: "period", Type: "number", Default: "20", Description: "Rolling window size"},
		{Name: "mode", Type: "string", Default: "max", Description: "max or min"},
	}
}

func (n *RollingMaxMinNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any) (map[string]any, error) {
	values := extractFloatSlice(inputs["values"])
	if values == nil {
		return nil, fmt.Errorf("rolling_maxmin: values input is required")
	}

	period := int(getFloatParam(params, "period", 20))
	mode := getStringParam(params, "mode", "max")

	if period <= 0 {
		return nil, fmt.Errorf("rolling_maxmin: period must be > 0, got %d", period)
	}
	if period > len(values) {
		return nil, fmt.Errorf("rolling_maxmin: period %d > data length %d", period, len(values))
	}

	result := make([]float64, len(values))
	for i := range values {
		start := i - period + 1
		if start < 0 {
			start = 0
		}
		window := values[start : i+1]
		if mode == "min" {
			v := window[0]
			for _, x := range window[1:] {
				if x < v {
					v = x
				}
			}
			result[i] = v
		} else {
			v := window[0]
			for _, x := range window[1:] {
				if x > v {
					v = x
				}
			}
			result[i] = v
		}
	}

	return map[string]any{"result": result}, nil
}

func (n *RollingMaxMinNode) Validate() error {
	mode := getStringParam(n.params, "mode", "max")
	switch mode {
	case "max", "min":
		return nil
	default:
		return fmt.Errorf("rolling_maxmin: invalid mode %q, expected max/min", mode)
	}
}
