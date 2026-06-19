package nodes

import (
	"context"
	"fmt"

	"quantflow/internal/workflow"
)

// EntrySignalNode detects rising edges (0→1 transitions) in a boolean condition
// series. Outputs 1 at each entry point (first true observation) and 0 everywhere else.
type EntrySignalNode struct {
	id     string
	params map[string]any
}

// NewEntrySignalNode creates a new EntrySignalNode.
func NewEntrySignalNode(id string, params map[string]any) (workflow.BaseNode, error) {
	return &EntrySignalNode{id: id, params: params}, nil
}

func (n *EntrySignalNode) ID() string       { return n.id }
func (n *EntrySignalNode) NodeType() string { return "entry_signal" }
func (n *EntrySignalNode) Category() string { return "signal" }

func (n *EntrySignalNode) InputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "condition", Type: workflow.PortSeries, Required: true},
	}
}

func (n *EntrySignalNode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "entry", Type: workflow.PortSeries, Required: false},
	}
}

func (n *EntrySignalNode) ParamSchema() []workflow.ParamDef { return nil }

func (n *EntrySignalNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any) (map[string]any, error) {
	condition := extractFloatSlice(inputs["condition"])
	if condition == nil {
		return nil, fmt.Errorf("entry_signal: condition input is required")
	}

	entry := make([]float64, len(condition))
	for i := 0; i < len(condition); i++ {
		if condition[i] == 1 && (i == 0 || condition[i-1] == 0) {
			entry[i] = 1
		}
	}

	return map[string]any{"entry": entry}, nil
}

func (n *EntrySignalNode) Validate() error { return nil }
