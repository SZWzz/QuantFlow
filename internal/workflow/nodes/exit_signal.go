package nodes

import (
	"context"
	"fmt"
	"quantflow/internal/workflow"
)

// ExitSignalNode detects falling edges (1→0 transitions) in a boolean condition
// series. Outputs -1 at each exit point (first false observation after true) and
// 0 everywhere else.
type ExitSignalNode struct {
	id     string
	params map[string]any
}

// NewExitSignalNode creates a new ExitSignalNode.
func NewExitSignalNode(id string, params map[string]any) (workflow.BaseNode, error) {
	return &ExitSignalNode{id: id, params: params}, nil
}

func (n *ExitSignalNode) ID() string       { return n.id }
func (n *ExitSignalNode) NodeType() string { return "exit_signal" }
func (n *ExitSignalNode) Category() string { return "signal" }

func (n *ExitSignalNode) InputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "condition", Type: workflow.PortSeries, Required: true},
	}
}

func (n *ExitSignalNode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "exit", Type: workflow.PortSeries, Required: false},
	}
}

func (n *ExitSignalNode) ParamSchema() []workflow.ParamDef { return nil }

func (n *ExitSignalNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any, nctx *workflow.NodeContext) (map[string]any, error) {
	condition := extractFloatSlice(inputs["condition"])
	if condition == nil {
		return nil, fmt.Errorf("exit_signal: condition input is required")
	}

	exit := make([]float64, len(condition))
	for i := 0; i < len(condition); i++ {
		if condition[i] == 0 && i > 0 && condition[i-1] == 1 {
			exit[i] = -1
		}
	}

	return map[string]any{"exit": exit}, nil
}

func (n *ExitSignalNode) Validate() error { return nil }
