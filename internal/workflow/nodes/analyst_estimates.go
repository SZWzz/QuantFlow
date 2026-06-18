package nodes

import (
	"context"
	"fmt"
	"log/slog"

	"quantflow/internal/research"
	"quantflow/internal/workflow"
)

// AnalystEstimatesNode fetches analyst ratings and target prices for a symbol.
// Degrades to empty estimates when the AnalystEstimatesService is not set.
type AnalystEstimatesNode struct {
	id     string
	params map[string]any
}

// NewAnalystEstimatesNode creates a new AnalystEstimatesNode.
func NewAnalystEstimatesNode(id string, params map[string]any) (workflow.BaseNode, error) {
	return &AnalystEstimatesNode{id: id, params: params}, nil
}

func (n *AnalystEstimatesNode) ID() string       { return n.id }
func (n *AnalystEstimatesNode) NodeType() string { return "analyst_estimates" }
func (n *AnalystEstimatesNode) Category() string { return "research" }

func (n *AnalystEstimatesNode) InputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "symbol", Type: workflow.PortString, Required: true},
	}
}

func (n *AnalystEstimatesNode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "ratings", Type: workflow.PortSeries, Required: false},
		{Name: "target_price", Type: workflow.PortNumber, Required: false},
		{Name: "consensus", Type: workflow.PortString, Required: false},
	}
}

func (n *AnalystEstimatesNode) ParamSchema() []workflow.ParamDef {
	return []workflow.ParamDef{
		{Name: "min_rating", Type: "string", Default: "", Description: "Minimum rating filter (e.g., Buy, Hold)"},
	}
}

func (n *AnalystEstimatesNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any) (map[string]any, error) {
	symbol, ok := inputs["symbol"].(string)
	if !ok || symbol == "" {
		return nil, fmt.Errorf("analyst_estimates: missing required input 'symbol'")
	}

	if analystEstimatesService != nil {
		estimates, err := analystEstimatesService.GetEstimates(ctx, symbol)
		if err != nil {
			slog.Warn("analyst estimates service returned error, using mock", "symbol", symbol, "error", err)
			return mockAnalystEstimateOutput(symbol), nil
		}

		avgTargetPrice := computeAverageTargetPrice(estimates)
		consensus := computeConsensusRating(estimates)

		return map[string]any{
			"ratings":      estimates,
			"target_price": avgTargetPrice,
			"consensus":    consensus,
		}, nil
	}

	slog.Warn("analyst estimates service not set, using mock", "symbol", symbol)
	return mockAnalystEstimateOutput(symbol), nil
}

func (n *AnalystEstimatesNode) Validate() error { return nil }

// computeAverageTargetPrice returns the mean of target prices across the high-low range.
func computeAverageTargetPrice(estimates []research.AnalystEstimate) float64 {
	if len(estimates) == 0 {
		return 0.0
	}
	var sum float64
	for _, e := range estimates {
		avg := (e.TargetLow + e.TargetHigh) / 2.0
		sum += avg
	}
	return sum / float64(len(estimates))
}

// computeConsensusRating determines the overall consensus from individual analyst ratings.
func computeConsensusRating(estimates []research.AnalystEstimate) string {
	if len(estimates) == 0 {
		return "neutral"
	}

	buyCount := 0
	sellCount := 0
	holdCount := 0

	for _, e := range estimates {
		switch e.Rating {
		case "buy", "strong_buy", "outperform":
			buyCount++
		case "sell", "strong_sell", "underperform":
			sellCount++
		default:
			holdCount++
		}
	}

	total := buyCount + sellCount + holdCount
	if total == 0 {
		return "neutral"
	}

	buyRatio := float64(buyCount) / float64(total)
	sellRatio := float64(sellCount) / float64(total)

	if buyRatio > 0.5 {
		return "buy"
	}
	if sellRatio > 0.5 {
		return "sell"
	}
	if buyRatio > sellRatio && buyRatio > 0.3 {
		return "overweight"
	}
	if sellRatio > buyRatio && sellRatio > 0.3 {
		return "underweight"
	}

	return "hold"
}

// mockAnalystEstimateOutput returns empty analyst data when no service is available.
func mockAnalystEstimateOutput(symbol string) map[string]any {
	return map[string]any{
		"ratings":      []research.AnalystEstimate{},
		"target_price": 0.0,
		"consensus":    "neutral",
	}
}
