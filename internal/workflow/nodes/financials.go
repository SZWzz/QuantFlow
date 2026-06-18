package nodes

import (
	"context"
	"fmt"
	"log/slog"

	"quantflow/internal/workflow"
)

// FinancialsNode fetches financial data and computes key ratios for a symbol.
// Degrades to mock data when the FinancialsService is not set.
type FinancialsNode struct {
	id     string
	params map[string]any
}

// NewFinancialsNode creates a new FinancialsNode.
func NewFinancialsNode(id string, params map[string]any) (workflow.BaseNode, error) {
	return &FinancialsNode{id: id, params: params}, nil
}

func (n *FinancialsNode) ID() string       { return n.id }
func (n *FinancialsNode) NodeType() string { return "financials" }
func (n *FinancialsNode) Category() string { return "research" }

func (n *FinancialsNode) InputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "symbol", Type: workflow.PortString, Required: true},
	}
}

func (n *FinancialsNode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "financial_data", Type: workflow.PortSeries, Required: false},
		{Name: "ratios", Type: workflow.PortSeries, Required: false},
	}
}

func (n *FinancialsNode) ParamSchema() []workflow.ParamDef {
	return []workflow.ParamDef{
		{Name: "force_refresh", Type: "bool", Default: false, Description: "Force refresh financial data from source"},
	}
}

func (n *FinancialsNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any) (map[string]any, error) {
	symbol, ok := inputs["symbol"].(string)
	if !ok || symbol == "" {
		return nil, fmt.Errorf("financials: missing required input 'symbol'")
	}

	if financialsService != nil {
		fd, err := financialsService.GetFinancials(ctx, symbol)
		if err != nil {
			slog.Warn("financials service returned error, using mock", "symbol", symbol, "error", err)
			return mockFinancialsOutput(symbol), nil
		}
		ratios := financialsService.ComputeRatios(fd)
		return map[string]any{
			"financial_data": fd,
			"ratios":         ratios,
		}, nil
	}

	slog.Warn("financials service not set, using mock", "symbol", symbol)
	return mockFinancialsOutput(symbol), nil
}

func (n *FinancialsNode) Validate() error { return nil }

// mockFinancialsOutput returns mock financial data and ratios when no service is available.
func mockFinancialsOutput(symbol string) map[string]any {
	return map[string]any{
		"financial_data": map[string]any{
			"symbol":        symbol,
			"revenue":       0.0,
			"net_income":    0.0,
			"eps":           0.0,
			"total_assets":  0.0,
			"total_equity":  0.0,
			"total_debt":    0.0,
			"free_cash_flow": 0.0,
			"market_cap":    0.0,
		},
		"ratios": map[string]any{
			"pe_ratio":     0.0,
			"pb_ratio":     0.0,
			"roe":          0.0,
			"roa":          0.0,
			"debt_to_equity": 0.0,
			"net_margin":    0.0,
		},
	}
}
