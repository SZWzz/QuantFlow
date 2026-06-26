package nodes

import (
	"context"
	"fmt"

	"quantflow/internal/workflow"
)

// HoldSignalNode converts discrete entry/exit signals (-1/0/1) into a continuous
// position signal by holding the last non-zero state. Signal=1 opens the position
// (or stays in), signal=-1 closes it, and 0 means hold the previous state.
type HoldSignalNode struct {
	id     string
	params map[string]any
}

// NewHoldSignalNode creates a new HoldSignalNode.
func NewHoldSignalNode(id string, params map[string]any) (workflow.BaseNode, error) {
	return &HoldSignalNode{id: id, params: params}, nil
}

func (n *HoldSignalNode) ID() string       { return n.id }
func (n *HoldSignalNode) NodeType() string { return "hold_signal" }
func (n *HoldSignalNode) Category() string { return "signal" }

func (n *HoldSignalNode) InputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "signal", Type: workflow.PortSeries, Required: true},
	}
}

func (n *HoldSignalNode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "position", Type: workflow.PortSeries, Required: false},
	}
}

func (n *HoldSignalNode) ParamSchema() []workflow.ParamDef { return nil }

func (n *HoldSignalNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any, nctx *workflow.NodeContext) (map[string]any, error) {
	signal := extractFloatSlice(inputs["signal"])
	if signal == nil {
		return nil, fmt.Errorf("hold_signal: signal input is required")
	}

	position := make([]float64, len(signal))
	prev := 0.0
	for i, s := range signal {
		switch {
		case s == 1:
			prev = 1
		case s == -1:
			prev = 0
		}
		position[i] = prev
	}

	return map[string]any{"position": position}, nil
}

func (n *HoldSignalNode) Validate() error { return nil }
