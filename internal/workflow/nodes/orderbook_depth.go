package nodes

import (
	"context"
	"quantflow/internal/workflow"
)

type OrderbookDepthNode struct {
	id     string
	params map[string]any
}

func NewOrderbookDepthNode(id string, params map[string]any) (workflow.BaseNode, error) {
	return &OrderbookDepthNode{id: id, params: params}, nil
}

func (n *OrderbookDepthNode) ID() string       { return n.id }
func (n *OrderbookDepthNode) NodeType() string { return "orderbook_depth" }
func (n *OrderbookDepthNode) Category() string { return "data" }

func (n *OrderbookDepthNode) InputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "symbol", Type: workflow.PortString, Required: true},
		{Name: "exchange", Type: workflow.PortString, Required: false},
	}
}

func (n *OrderbookDepthNode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "bids", Type: workflow.PortSeries},
		{Name: "asks", Type: workflow.PortSeries},
	}
}

func (n *OrderbookDepthNode) ParamSchema() []workflow.ParamDef {
	return []workflow.ParamDef{{Name: "limit", Type: "number", Default: "20", Description: "Price levels per side"}}
}
func (n *OrderbookDepthNode) Validate() error { return nil }

func (n *OrderbookDepthNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any, nctx *workflow.NodeContext) (map[string]any, error) {
	return map[string]any{"bids": []any{}, "asks": []any{}}, nil
}
