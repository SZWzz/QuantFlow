package nodes

import (
	"context"
	"quantflow/internal/workflow"
	"time"
)

// WaitNode pauses workflow execution for a configurable duration.
type WaitNode struct {
	id     string
	params map[string]any
}

// NewWaitNode creates a new WaitNode.
func NewWaitNode(id string, params map[string]any) (workflow.BaseNode, error) {
	return &WaitNode{id: id, params: params}, nil
}

func (n *WaitNode) ID() string       { return n.id }
func (n *WaitNode) NodeType() string { return "wait" }
func (n *WaitNode) Category() string { return "schedule" }

func (n *WaitNode) InputPorts() []workflow.PortDefinition  { return nil }
func (n *WaitNode) OutputPorts() []workflow.PortDefinition { return nil }

func (n *WaitNode) ParamSchema() []workflow.ParamDef {
	return []workflow.ParamDef{
		{Name: "duration_sec", Type: "number", Default: "1", Description: "Seconds to wait (max 3600)"},
	}
}

func (n *WaitNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any, nctx *workflow.NodeContext) (map[string]any, error) {
	d := getFloatParam(params, "duration_sec", 1)
	if d < 0 {
		d = 0
	}
	if d > 3600 {
		d = 3600
	}
	select {
	case <-time.After(time.Duration(d * float64(time.Second))):
		return nil, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (n *WaitNode) Validate() error { return nil }
