package nodes

import (
	"context"
	"fmt"

	"quantflow/internal/workflow"
)

// PctChangeNode computes period-over-period percentage change of a series.
type PctChangeNode struct {
	id     string
	params map[string]any
}

func NewPctChangeNode(id string, params map[string]any) (workflow.BaseNode, error) {
	return &PctChangeNode{id: id, params: params}, nil
}

func (n *PctChangeNode) ID() string       { return n.id }
func (n *PctChangeNode) NodeType() string  { return "pct_change" }
func (n *PctChangeNode) Category() string  { return "alpha" }

func (n *PctChangeNode) InputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{{Name: "values", Type: workflow.PortSeries, Required: true}}
}

func (n *PctChangeNode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{{Name: "result", Type: workflow.PortSeries, Required: false}}
}

func (n *PctChangeNode) ParamSchema() []workflow.ParamDef {
	return []workflow.ParamDef{
		{Name: "period", Type: "number", Default: "1", Description: "Lookback period for pct change"},
	}
}

func (n *PctChangeNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any) (map[string]any, error) {
	values := extractFloatSlice(inputs["values"])
	if values == nil {
		return nil, fmt.Errorf("pct_change: values input required")
	}

	period := int(getFloatParam(params, "period", 1))
	if period <= 0 || period >= len(values) {
		return nil, fmt.Errorf("pct_change: period %d must be > 0 and < len(values) %d", period, len(values))
	}

	result := make([]float64, len(values))
	for i := period; i < len(values); i++ {
		if values[i-period] != 0 {
			result[i] = (values[i] - values[i-period]) / values[i-period]
		}
	}
	return map[string]any{"result": result}, nil
}

func (n *PctChangeNode) Validate() error {
	period := int(getFloatParam(n.params, "period", 1))
	if period <= 0 {
		return fmt.Errorf("pct_change: period must be positive, got %d", period)
	}
	return nil
}
