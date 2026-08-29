package nodes

import (
	"context"
	"quantflow/internal/workflow"
)

type TradeHistoryNode struct {
	id     string
	params map[string]any
}

func NewTradeHistoryNode(id string, params map[string]any) (workflow.BaseNode, error) {
	return &TradeHistoryNode{id: id, params: params}, nil
}

func (n *TradeHistoryNode) ID() string       { return n.id }
func (n *TradeHistoryNode) NodeType() string { return "trade_history" }
func (n *TradeHistoryNode) Category() string { return "data" }

func (n *TradeHistoryNode) InputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{{Name: "symbol", Type: workflow.PortString, Required: false}}
}

func (n *TradeHistoryNode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{{Name: "trades", Type: workflow.PortSeries}}
}

func (n *TradeHistoryNode) ParamSchema() []workflow.ParamDef {
	return []workflow.ParamDef{{Name: "limit", Type: "number", Default: "100", Description: "Max trade records"}}
}
func (n *TradeHistoryNode) Validate() error { return nil }

func (n *TradeHistoryNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any, nctx *workflow.NodeContext) (map[string]any, error) {
	return map[string]any{"trades": []any{}}, nil
}
