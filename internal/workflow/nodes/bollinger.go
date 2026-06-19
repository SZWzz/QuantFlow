package nodes

import (
	"context"
	"fmt"
	"math"
	"quantflow/internal/workflow"
)

type BollingerNode struct{ id string; params map[string]any }

func NewBollingerNode(id string, params map[string]any) (workflow.BaseNode, error) {
	return &BollingerNode{id: id, params: params}, nil
}

func (n *BollingerNode) ID() string       { return n.id }
func (n *BollingerNode) NodeType() string  { return "bollinger" }
func (n *BollingerNode) Category() string  { return "indicator" }

func (n *BollingerNode) InputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{{Name: "prices", Type: workflow.PortSeries, Required: true}}
}

func (n *BollingerNode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "upper", Type: workflow.PortSeries, Required: false},
		{Name: "middle", Type: workflow.PortSeries, Required: false},
		{Name: "lower", Type: workflow.PortSeries, Required: false},
	}
}

func (n *BollingerNode) ParamSchema() []workflow.ParamDef {
	return []workflow.ParamDef{
		{Name: "period", Type: "number", Default: "20", Description: "SMA period"},
		{Name: "multiplier", Type: "number", Default: "2", Description: "Standard deviation multiplier"},
	}
}

func (n *BollingerNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any) (map[string]any, error) {
	prices := extractFloatSlice(inputs["prices"])
	if prices == nil { return nil, fmt.Errorf("bollinger: prices input required") }

	period := int(getFloatParam(params, "period", 20))
	k := getFloatParam(params, "multiplier", 2)

	nP := len(prices)
	upper, middle, lower := make([]float64, nP), make([]float64, nP), make([]float64, nP)
	// Fill warmup period with NaN — same rationale as RSI (distinguish "no data" from zero).
	for i := 0; i < period-1 && i < nP; i++ {
		upper[i] = math.NaN()
		middle[i] = math.NaN()
		lower[i] = math.NaN()
	}

	for i := period - 1; i < nP; i++ {
		sum, sumSq := 0.0, 0.0
		for j := i - period + 1; j <= i; j++ {
			sum += prices[j]
		}
		mean := sum / float64(period)
		for j := i - period + 1; j <= i; j++ {
			diff := prices[j] - mean
			sumSq += diff * diff
		}
		stddev := math.Sqrt(sumSq / float64(period))
		middle[i] = mean
		upper[i] = mean + k*stddev
		lower[i] = mean - k*stddev
	}

	return map[string]any{"upper": upper, "middle": middle, "lower": lower}, nil
}

func (n *BollingerNode) Validate() error { return nil }
