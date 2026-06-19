package nodes

import (
	"context"
	"fmt"

	"quantflow/internal/workflow"
)

// ThresholdSignalNode generates buy/sell/hold signals when a value crosses upper/lower thresholds.
type ThresholdSignalNode struct{ id string; params map[string]any }

// NewThresholdSignalNode creates a new ThresholdSignalNode.
func NewThresholdSignalNode(id string, params map[string]any) (workflow.BaseNode, error) {
	return &ThresholdSignalNode{id: id, params: params}, nil
}

func (n *ThresholdSignalNode) ID() string       { return n.id }
func (n *ThresholdSignalNode) NodeType() string { return "threshold_signal" }
func (n *ThresholdSignalNode) Category() string { return "signal" }

func (n *ThresholdSignalNode) InputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{{Name: "values", Type: workflow.PortSeries, Required: true}}
}

func (n *ThresholdSignalNode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{{Name: "signal", Type: workflow.PortSeries, Required: false}}
}

func (n *ThresholdSignalNode) ParamSchema() []workflow.ParamDef {
	return []workflow.ParamDef{
		{Name: "upper", Type: "float", Default: 80, Description: "Upper threshold — value above triggers sell (-1)"},
		{Name: "lower", Type: "float", Default: 20, Description: "Lower threshold — value below triggers buy (1)"},
	}
}

func (n *ThresholdSignalNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any) (map[string]any, error) {
	values := extractFloatSlice(inputs["values"])
	if values == nil {
		return nil, fmt.Errorf("threshold_signal: values input is required")
	}
	upper := getFloatParam(params, "upper", 80)
	lower := getFloatParam(params, "lower", 20)

	signals := make([]float64, len(values))
	for i, v := range values {
		switch {
		case v > upper: signals[i] = -1 // sell
		case v < lower: signals[i] = 1  // buy
		default:        signals[i] = 0  // hold
		}
	}
	return map[string]any{"signal": signals}, nil
}

func (n *ThresholdSignalNode) Validate() error {
	upper := getFloatParam(n.params, "upper", 80)
	lower := getFloatParam(n.params, "lower", 20)
	if lower >= upper {
		return fmt.Errorf("threshold_signal: lower(%v) must be less than upper(%v)", lower, upper)
	}
	return nil
}
