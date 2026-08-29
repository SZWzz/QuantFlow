package nodes

import (
	"context"
	"fmt"
	"quantflow/internal/workflow"
)

// RebalanceNode forward-fills target weights and only applies new weights at
// rebalance periods (every N periods). Between rebalance points the previous
// weights are carried forward unchanged.
type RebalanceNode struct {
	id     string
	params map[string]any
}

// NewRebalanceNode creates a new RebalanceNode.
func NewRebalanceNode(id string, params map[string]any) (workflow.BaseNode, error) {
	return &RebalanceNode{id: id, params: params}, nil
}

func (n *RebalanceNode) ID() string       { return n.id }
func (n *RebalanceNode) NodeType() string { return "rebalance" }
func (n *RebalanceNode) Category() string { return "signal" }

func (n *RebalanceNode) InputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "weights", Type: workflow.PortSeries, Required: true},
		{Name: "schedule", Type: workflow.PortSeries, Required: false},
	}
}

func (n *RebalanceNode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "rebalanced", Type: workflow.PortSeries, Required: false},
	}
}

func (n *RebalanceNode) ParamSchema() []workflow.ParamDef {
	return []workflow.ParamDef{
		{Name: "frequency", Type: "number", Default: 1, Description: "Rebalance every N periods"},
	}
}

func (n *RebalanceNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any, nctx *workflow.NodeContext) (map[string]any, error) {
	weights := extractFloatSlice(inputs["weights"])
	if weights == nil {
		return nil, fmt.Errorf("rebalance: weights input is required")
	}

	freq := int(getFloatParam(params, "frequency", 1))
	if freq <= 0 {
		freq = 1
	}

	// If schedule is provided, use it; otherwise rebalance every freq periods.
	var schedule []float64
	if raw, ok := inputs["schedule"]; ok && raw != nil {
		schedule = extractFloatSlice(raw)
	}

	rebalanced := make([]float64, len(weights))
	carry := 0.0
	for i, w := range weights {
		shouldRebalance := i%freq == 0
		if schedule != nil && i < len(schedule) && schedule[i] != 0 {
			shouldRebalance = true
		}
		if shouldRebalance {
			carry = w
		}
		rebalanced[i] = carry
	}

	return map[string]any{"rebalanced": rebalanced}, nil
}

func (n *RebalanceNode) Validate() error {
	freq := int(getFloatParam(n.params, "frequency", 1))
	if freq <= 0 {
		return fmt.Errorf("rebalance: frequency must be > 0, got %d", freq)
	}
	return nil
}
