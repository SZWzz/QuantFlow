package nodes

import (
	"context"
	"fmt"
	"math"

	"quantflow/internal/workflow"
)

type RSINode struct{ id string; params map[string]any }

func NewRSINode(id string, params map[string]any) (workflow.BaseNode, error) {
	return &RSINode{id: id, params: params}, nil
}

func (n *RSINode) ID() string       { return n.id }
func (n *RSINode) NodeType() string  { return "rsi" }
func (n *RSINode) Category() string  { return "indicator" }

func (n *RSINode) InputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{{Name: "prices", Type: workflow.PortSeries, Required: true}}
}

func (n *RSINode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{{Name: "rsi", Type: workflow.PortSeries, Required: false}}
}

func (n *RSINode) ParamSchema() []workflow.ParamDef {
	return []workflow.ParamDef{{Name: "period", Type: "number", Default: "14", Description: "RSI period"}}
}

func (n *RSINode) Execute(ctx context.Context, inputs map[string]any, params map[string]any, nctx *workflow.NodeContext) (map[string]any, error) {
	prices := extractFloatSlice(inputs["prices"])
	if prices == nil { return nil, fmt.Errorf("rsi: prices input required") }

	period := int(getFloatParam(params, "period", 14))
	if period >= len(prices) { return nil, fmt.Errorf("rsi: period %d >= data length %d", period, len(prices)) }

	rsi := make([]float64, len(prices))
	// Fill warmup period with NaN so downstream can distinguish "no data"
	// from "value is zero" (critical for z-score normalization and ML features).
	for i := 0; i < period && i < len(rsi); i++ {
		rsi[i] = math.NaN()
	}
	var gains, losses float64
	for i := 1; i <= period; i++ {
		diff := prices[i] - prices[i-1]
		if diff > 0 { gains += diff } else { losses += -diff }
	}
	avgGain := gains / float64(period)
	avgLoss := losses / float64(period)

	for i := period; i < len(prices); i++ {
		if avgLoss == 0 { rsi[i] = 100 } else { rsi[i] = 100 - (100 / (1 + avgGain/avgLoss)) }
		diff := prices[i] - prices[i-1]
		var gain, loss float64
		if diff > 0 { gain = diff } else { loss = -diff }
		avgGain = (avgGain*float64(period-1) + gain) / float64(period)
		avgLoss = (avgLoss*float64(period-1) + loss) / float64(period)
	}

	return map[string]any{"rsi": rsi}, nil
}

func (n *RSINode) Validate() error { return nil }
