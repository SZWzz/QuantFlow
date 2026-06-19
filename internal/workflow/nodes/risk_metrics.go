package nodes

import (
	"context"
	"fmt"

	"quantflow/internal/workflow"
)

// RiskMetricsNode computes risk metrics for a set of positions.
type RiskMetricsNode struct {
	id     string
	params map[string]any
}

// NewRiskMetricsNode creates a new RiskMetricsNode.
func NewRiskMetricsNode(id string, params map[string]any) (workflow.BaseNode, error) {
	return &RiskMetricsNode{id: id, params: params}, nil
}

func (n *RiskMetricsNode) ID() string        { return n.id }
func (n *RiskMetricsNode) NodeType() string  { return "risk_metrics" }
func (n *RiskMetricsNode) Category() string  { return "risk" }

func (n *RiskMetricsNode) InputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "positions", Type: workflow.PortSeries, Required: false},
	}
}

func (n *RiskMetricsNode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "metrics", Type: workflow.PortSeries, Required: false},
	}
}

func (n *RiskMetricsNode) ParamSchema() []workflow.ParamDef {
	return []workflow.ParamDef{
		{Name: "risk_free_rate", Type: "number", Default: "0.03", Description: "Annual risk-free rate"},
	}
}

func (n *RiskMetricsNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any) (map[string]any, error) {
	// TODO: Compute real risk metrics from portfolio returns (Sharpe, max drawdown,
	// total exposure). Requires wiring to portfolio service + historical returns.
	_ = inputs["equity_curve"]
	_ = inputs["trades"]
	return nil, fmt.Errorf("risk_metrics: not yet implemented — returns placeholder data has been removed; real computation requires portfolio service wiring")
}

func (n *RiskMetricsNode) Validate() error { return nil }
