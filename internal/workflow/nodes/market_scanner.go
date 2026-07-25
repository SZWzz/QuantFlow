package nodes

import (
	"context"

	"quantflow/internal/workflow"
)

// MarketScannerNode scans a market for symbols matching filter criteria.
type MarketScannerNode struct {
	id     string
	params map[string]any
}

// NewMarketScannerNode creates a new MarketScannerNode.
func NewMarketScannerNode(id string, params map[string]any) (workflow.BaseNode, error) {
	return &MarketScannerNode{id: id, params: params}, nil
}

func (n *MarketScannerNode) ID() string       { return n.id }
func (n *MarketScannerNode) NodeType() string { return "market_scanner" }
func (n *MarketScannerNode) Category() string { return "data" }

func (n *MarketScannerNode) InputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "filters", Type: workflow.PortAny, Required: false},
	}
}

func (n *MarketScannerNode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "results", Type: workflow.PortSeries, Required: false},
	}
}

func (n *MarketScannerNode) ParamSchema() []workflow.ParamDef {
	return []workflow.ParamDef{
		{Name: "market", Type: "string", Default: "CN", Description: "Market to scan: CN, HK, US, CRYPTO"},
	}
}

func (n *MarketScannerNode) Validate() error { return nil }

func (n *MarketScannerNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any, nctx *workflow.NodeContext) (map[string]any, error) {
	return map[string]any{"results": []any{}}, nil
}
