package nodes

import (
	"context"
	"fmt"
	"math"

	"quantflow/internal/workflow"
)

// PositionSizerNode computes position size from portfolio value and risk
// parameters using one of three methods: fixed, percent, or kelly.
type PositionSizerNode struct {
	id     string
	params map[string]any
}

// NewPositionSizerNode creates a new PositionSizerNode.
func NewPositionSizerNode(id string, params map[string]any) (workflow.BaseNode, error) {
	return &PositionSizerNode{id: id, params: params}, nil
}

func (n *PositionSizerNode) ID() string       { return n.id }
func (n *PositionSizerNode) NodeType() string { return "position_sizer" }
func (n *PositionSizerNode) Category() string { return "risk" }

func (n *PositionSizerNode) InputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "portfolio_value", Type: workflow.PortNumber, Required: true},
		{Name: "risk_per_trade", Type: workflow.PortNumber, Required: false},
	}
}

func (n *PositionSizerNode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "position_size", Type: workflow.PortNumber, Required: false},
		{Name: "risk_amount", Type: workflow.PortNumber, Required: false},
	}
}

func (n *PositionSizerNode) ParamSchema() []workflow.ParamDef {
	return []workflow.ParamDef{
		{Name: "method", Type: "string", Default: "percent", Description: "Sizing method: fixed, percent, kelly"},
		{Name: "pct", Type: "number", Default: "2", Description: "Risk percent when method=percent (e.g. 2 means 2%)"},
		{Name: "stop_distance", Type: "number", Default: "0.05", Description: "Stop distance as decimal (e.g. 0.05 = 5%)"},
	}
}

func (n *PositionSizerNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any) (map[string]any, error) {
	pv := extractFloat(inputs["portfolio_value"])
	if pv == nil {
		return nil, fmt.Errorf("position_sizer: portfolio_value input is required")
	}

	method := getStringParam(params, "method", "percent")
	riskPct := getFloatParam(params, "pct", 2)
	stopDist := getFloatParam(params, "stop_distance", 0.05)

	portfolioValue := *pv
	riskPerTrade := 0.0
	if rpt := extractFloat(inputs["risk_per_trade"]); rpt != nil {
		riskPerTrade = *rpt
	}

	var positionSize, riskAmount float64

	switch method {
	case "fixed":
		positionSize = riskPerTrade
		riskAmount = riskPerTrade

	case "percent":
		riskAmount = portfolioValue * riskPct / 100.0
		if stopDist > 0 {
			positionSize = riskAmount / stopDist
		} else {
			positionSize = riskAmount
		}

	case "kelly":
		// Simplified Kelly: f* = win_prob - (1 - win_prob) / (win_loss_ratio)
		// Default conservative parameters: 55% win rate, 1:1 reward/risk
		winProb := getFloatParam(params, "kelly_win_prob", 0.55)
		winLossRatio := 1.0
		kellyFrac := winProb - (1-winProb)/winLossRatio
		// Cap at half-Kelly for safety
		if kellyFrac < 0 {
			kellyFrac = 0
		}
		kellyFrac = math.Min(kellyFrac, 0.25) // max 25% half-Kelly
		riskAmount = portfolioValue * kellyFrac
		if stopDist > 0 {
			positionSize = riskAmount / stopDist
		} else {
			positionSize = riskAmount
		}

	default:
		return nil, fmt.Errorf("position_sizer: unknown method %q, expected fixed/percent/kelly", method)
	}

	return map[string]any{
		"position_size": positionSize,
		"risk_amount":   riskAmount,
	}, nil
}

func (n *PositionSizerNode) Validate() error {
	method := getStringParam(n.params, "method", "percent")
	switch method {
	case "fixed", "percent", "kelly":
		return nil
	default:
		return fmt.Errorf("position_sizer: invalid method %q, expected fixed/percent/kelly", method)
	}
}
