package nodes

import (
	"context"
	"fmt"
	"log/slog"
	"quantflow/internal/research"
	"quantflow/internal/workflow"
)

// InsiderTradesNode analyzes insider trading activity for a symbol and produces a signal.
// Degrades to empty trades when the InsiderTradingService is not set.
type InsiderTradesNode struct {
	id     string
	params map[string]any
}

// NewInsiderTradesNode creates a new InsiderTradesNode.
func NewInsiderTradesNode(id string, params map[string]any) (workflow.BaseNode, error) {
	return &InsiderTradesNode{id: id, params: params}, nil
}

func (n *InsiderTradesNode) ID() string       { return n.id }
func (n *InsiderTradesNode) NodeType() string { return "insider_trades" }
func (n *InsiderTradesNode) Category() string { return "research" }

func (n *InsiderTradesNode) InputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "symbol", Type: workflow.PortString, Required: true},
	}
}

func (n *InsiderTradesNode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "transactions", Type: workflow.PortSeries, Required: false},
		{Name: "net_activity", Type: workflow.PortString, Required: false},
		{Name: "signal", Type: workflow.PortSignal, Required: false},
	}
}

func (n *InsiderTradesNode) ParamSchema() []workflow.ParamDef {
	return []workflow.ParamDef{
		{Name: "lookback_days", Type: "int", Default: 90, Description: "Number of days of insider trades to analyze"},
	}
}

func (n *InsiderTradesNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any, nctx *workflow.NodeContext) (map[string]any, error) {
	var insiderTradingService workflow.InsiderTradingServiceInterface
	if nctx != nil {
		insiderTradingService = nctx.InsiderTradingService
	}
	symbol, ok := inputs["symbol"].(string)
	if !ok || symbol == "" {
		return nil, fmt.Errorf("insider_trades: missing required input 'symbol'")
	}

	if insiderTradingService != nil {
		txns, err := insiderTradingService.GetInsiderTrades(ctx, symbol)
		if err != nil {
			slog.Warn("insider trading service returned error, using mock", "symbol", symbol, "error", err)
			return mockInsiderTradeOutput(symbol), nil
		}

		netActivity, signal := analyzeInsiderActivity(txns)

		return map[string]any{
			"transactions": txns,
			"net_activity": netActivity,
			"signal":       signal,
		}, nil
	}

	slog.Warn("insider trading service not set, using mock", "symbol", symbol)
	return mockInsiderTradeOutput(symbol), nil
}

func (n *InsiderTradesNode) Validate() error { return nil }

// analyzeInsiderActivity computes the net insider activity direction from a list of trades.
func analyzeInsiderActivity(txns []research.InsiderTransaction) (string, map[string]any) {
	if len(txns) == 0 {
		return "neutral", map[string]any{"action": "hold", "confidence": 0.0}
	}

	buyShares := int64(0)
	sellShares := int64(0)

	for _, t := range txns {
		switch t.Type {
		case "buy", "purchase":
			buyShares += t.Shares
		case "sell", "sale":
			sellShares += t.Shares
		}
	}

	totalShares := buyShares + sellShares
	if totalShares == 0 {
		return "neutral", map[string]any{"action": "hold", "confidence": 0.0}
	}

	buyRatio := float64(buyShares) / float64(totalShares)
	confidence := buyRatio
	if buyRatio < 0.5 {
		confidence = 1.0 - buyRatio
	}

	var activity string
	var action string

	switch {
	case buyRatio > 0.6:
		activity = "bullish"
		action = "buy"
	case buyRatio < 0.4:
		activity = "bearish"
		action = "sell"
	default:
		activity = "neutral"
		action = "hold"
	}

	return activity, map[string]any{
		"action":     action,
		"confidence": confidence,
	}
}

// mockInsiderTradeOutput returns empty insider trade data when no service is available.
func mockInsiderTradeOutput(symbol string) map[string]any {
	return map[string]any{
		"transactions": []research.InsiderTransaction{},
		"net_activity": "neutral",
		"signal": map[string]any{
			"action":     "hold",
			"confidence": 0.0,
		},
	}
}
