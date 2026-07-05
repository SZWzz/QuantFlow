package nodes

import (
	"context"
	"fmt"
	"math"

	"quantflow/internal/workflow"
)

type RiskMetricsNode struct {
	id     string
	params map[string]any
}

func NewRiskMetricsNode(id string, params map[string]any) (workflow.BaseNode, error) {
	return &RiskMetricsNode{id: id, params: params}, nil
}

func (n *RiskMetricsNode) ID() string        { return n.id }
func (n *RiskMetricsNode) NodeType() string  { return "risk_metrics" }
func (n *RiskMetricsNode) Category() string  { return "risk" }

func (n *RiskMetricsNode) InputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "portfolio", Type: workflow.PortAny, Required: true},
		{Name: "returns", Type: workflow.PortSeries, Required: false},
	}
}

func (n *RiskMetricsNode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "metrics", Type: workflow.PortAny, Required: false},
	}
}

func (n *RiskMetricsNode) ParamSchema() []workflow.ParamDef {
	return []workflow.ParamDef{
		{Name: "risk_free_rate", Type: "number", Default: "0.02", Description: "Annual risk-free rate"},
	}
}

func (n *RiskMetricsNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any, nctx *workflow.NodeContext) (map[string]any, error) {
	portfolio := inputs["portfolio"]
	if portfolio == nil {
		return nil, fmt.Errorf("risk_metrics: portfolio input is required")
	}

	totalValue := 0.0
	switch p := portfolio.(type) {
	case []map[string]any:
		for _, pos := range p {
			shares, _ := toFloat64(pos["shares"])
			price, _ := toFloat64(pos["price"])
			totalValue += shares * price
		}
	case map[string]float64:
		for _, v := range p {
			totalValue += v
		}
	case map[string]any:
		for _, v := range p {
			val, _ := toFloat64(v)
			totalValue += val
		}
	default:
		return nil, fmt.Errorf("risk_metrics: unsupported portfolio type %T", portfolio)
	}

	metrics := map[string]any{"total_value": totalValue}

	returns := extractFloat64Slice(inputs["returns"])
	if len(returns) > 1 {
		n := float64(len(returns))
		var sum, sumSq float64
		for _, r := range returns {
			sum += r
			sumSq += r * r
		}
		mean := sum / n
		variance := (sumSq - sum*sum/n) / (n - 1)
		std := math.Sqrt(variance)

		rfr := getFloatParam(params, "risk_free_rate", 0.02)
		sharpe := (mean - rfr/252) / std

		var95 := -(mean - 1.645*std)

		var peak, maxDD float64
		cum := 1.0
		for _, r := range returns {
			cum *= 1 + r
			if cum > peak {
				peak = cum
			}
			dd := (peak - cum) / peak
			if dd > maxDD {
				maxDD = dd
			}
		}

		volatility := std * math.Sqrt(252)

		metrics["sharpe_ratio"] = sharpe
		metrics["max_drawdown"] = maxDD
		metrics["volatility"] = volatility
		metrics["var_95"] = var95
	} else {
		metrics["sharpe_ratio"] = 0.0
		metrics["max_drawdown"] = 0.0
		metrics["volatility"] = 0.0
		metrics["var_95"] = 0.0
	}

	return map[string]any{"metrics": metrics}, nil
}

func (n *RiskMetricsNode) Validate() error { return nil }

func toFloat64(v any) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case float32:
		return float64(val), true
	case int:
		return float64(val), true
	case int64:
		return float64(val), true
	default:
		return 0, false
	}
}
