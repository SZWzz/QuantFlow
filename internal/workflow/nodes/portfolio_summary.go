package nodes

import (
	"context"

	"quantflow/internal/workflow"
)

// PortfolioSummaryNode computes a summary of the current portfolio state.
type PortfolioSummaryNode struct {
	id     string
	params map[string]any
}

// NewPortfolioSummaryNode creates a new PortfolioSummaryNode.
func NewPortfolioSummaryNode(id string, params map[string]any) (workflow.BaseNode, error) {
	return &PortfolioSummaryNode{id: id, params: params}, nil
}

func (n *PortfolioSummaryNode) ID() string        { return n.id }
func (n *PortfolioSummaryNode) NodeType() string  { return "portfolio_summary" }
func (n *PortfolioSummaryNode) Category() string  { return "portfolio" }

func (n *PortfolioSummaryNode) InputPorts() []workflow.PortDefinition { return nil }

func (n *PortfolioSummaryNode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "summary", Type: workflow.PortSeries, Required: false},
	}
}

func (n *PortfolioSummaryNode) ParamSchema() []workflow.ParamDef { return nil }

func (n *PortfolioSummaryNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any, nctx *workflow.NodeContext) (map[string]any, error) {
	var oms workflow.OMSService
	if nctx != nil {
		oms = nctx.OMS
	}

	if oms == nil {
		return map[string]any{"summary": map[string]any{"total_value": 0}}, nil
	}
	ps := oms.GetAllPositions()
	var pnl, mv float64
	for _, p := range ps {
		mv += p.MarketPrice * p.Quantity
		pnl += p.PnL
	}
	return map[string]any{
		"summary": map[string]any{
			"total_value":    100000 + pnl,
			"market_value":   mv,
			"total_pnl":      pnl,
			"position_count": len(ps),
		},
	}, nil
}

func (n *PortfolioSummaryNode) Validate() error { return nil }
