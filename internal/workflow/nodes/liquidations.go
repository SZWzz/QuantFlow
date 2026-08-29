package nodes

import (
	"context"
	"quantflow/internal/workflow"
)

type LiquidationsNode struct {
	id     string
	params map[string]any
}

func NewLiquidationsNode(id string, params map[string]any) (workflow.BaseNode, error) {
	return &LiquidationsNode{id: id, params: params}, nil
}

func (n *LiquidationsNode) ID() string       { return n.id }
func (n *LiquidationsNode) NodeType() string { return "liquidations" }
func (n *LiquidationsNode) Category() string { return "data" }

func (n *LiquidationsNode) InputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{{Name: "symbol", Type: workflow.PortString, Required: false}}
}

func (n *LiquidationsNode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{{Name: "liquidations", Type: workflow.PortSeries}}
}

func (n *LiquidationsNode) ParamSchema() []workflow.ParamDef {
	return []workflow.ParamDef{{Name: "limit", Type: "number", Default: "50", Description: "Max liquidation events"}}
}
func (n *LiquidationsNode) Validate() error { return nil }

func (n *LiquidationsNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any, nctx *workflow.NodeContext) (map[string]any, error) {
	return map[string]any{"liquidations": []any{}}, nil
}
