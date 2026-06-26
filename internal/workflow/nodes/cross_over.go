package nodes

import (
	"context"
	"fmt"

	"quantflow/internal/workflow"
)

// CrossOverNode detects golden cross (+1) and death cross (-1) events
// between a fast and a slow input series.
type CrossOverNode struct {
	id     string
	params map[string]any
}

func NewCrossOverNode(id string, params map[string]any) (workflow.BaseNode, error) {
	return &CrossOverNode{id: id, params: params}, nil
}

func (n *CrossOverNode) ID() string       { return n.id }
func (n *CrossOverNode) NodeType() string  { return "cross_over" }
func (n *CrossOverNode) Category() string  { return "alpha" }

func (n *CrossOverNode) InputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "fast", Type: workflow.PortSeries, Required: true},
		{Name: "slow", Type: workflow.PortSeries, Required: true},
	}
}

func (n *CrossOverNode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{{Name: "cross", Type: workflow.PortSeries, Required: false}}
}

func (n *CrossOverNode) ParamSchema() []workflow.ParamDef { return nil }

func (n *CrossOverNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any, nctx *workflow.NodeContext) (map[string]any, error) {
	fast := extractFloatSlice(inputs["fast"])
	if fast == nil {
		return nil, fmt.Errorf("cross_over: fast input required")
	}
	slow := extractFloatSlice(inputs["slow"])
	if slow == nil {
		return nil, fmt.Errorf("cross_over: slow input required")
	}
	if len(fast) != len(slow) {
		return nil, fmt.Errorf("cross_over: fast(%d) and slow(%d) must have same length", len(fast), len(slow))
	}
	if len(fast) == 0 {
		return nil, fmt.Errorf("cross_over: inputs must not be empty")
	}

	cross := make([]float64, len(fast))
	for i := 1; i < len(fast); i++ {
		// Golden cross: fast crosses above slow
		if fast[i-1] <= slow[i-1] && fast[i] > slow[i] {
			cross[i] = 1
		} else if fast[i-1] >= slow[i-1] && fast[i] < slow[i] {
			// Death cross: fast crosses below slow
			cross[i] = -1
		}
	}
	return map[string]any{"cross": cross}, nil
}

func (n *CrossOverNode) Validate() error { return nil }
