package nodes

import (
	"context"
	"fmt"

	"quantflow/internal/workflow"
)

// DeltaNode computes the difference v[i] - v[i-period] over a series.
type DeltaNode struct {
	id     string
	params map[string]any
}

func NewDeltaNode(id string, params map[string]any) (workflow.BaseNode, error) {
	return &DeltaNode{id: id, params: params}, nil
}

func (n *DeltaNode) ID() string       { return n.id }
func (n *DeltaNode) NodeType() string  { return "delta" }
func (n *DeltaNode) Category() string  { return "alpha" }

func (n *DeltaNode) InputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{{Name: "values", Type: workflow.PortSeries, Required: true}}
}

func (n *DeltaNode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{{Name: "result", Type: workflow.PortSeries, Required: false}}
}

func (n *DeltaNode) ParamSchema() []workflow.ParamDef {
	return []workflow.ParamDef{
		{Name: "period", Type: "number", Default: "1", Description: "Lookback period for delta"},
	}
}

func (n *DeltaNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any, nctx *workflow.NodeContext) (map[string]any, error) {
	values := extractFloatSlice(inputs["values"])
	if values == nil {
		return nil, fmt.Errorf("delta: values input required")
	}

	period := int(getFloatParam(params, "period", 1))
	if period <= 0 || period >= len(values) {
		return nil, fmt.Errorf("delta: period %d must be > 0 and < len(values) %d", period, len(values))
	}

	result := make([]float64, len(values))
	for i := period; i < len(values); i++ {
		result[i] = values[i] - values[i-period]
	}
	return map[string]any{"result": result}, nil
}

func (n *DeltaNode) Validate() error {
	period := int(getFloatParam(n.params, "period", 1))
	if period <= 0 {
		return fmt.Errorf("delta: period must be positive, got %d", period)
	}
	return nil
}
