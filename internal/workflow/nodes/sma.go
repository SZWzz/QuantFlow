// Package nodes provides built-in workflow node implementations.
package nodes

import (
	"context"
	"fmt"

	"quantflow/internal/workflow"
)

// SMANode computes a Simple Moving Average over its input series.
type SMANode struct {
	id     string
	params map[string]any
}

// NewSMANode creates a new SMANode.
func NewSMANode(id string, params map[string]any) (workflow.BaseNode, error) {
	return &SMANode{id: id, params: params}, nil
}

func (n *SMANode) ID() string       { return n.id }
func (n *SMANode) NodeType() string { return "sma" }
func (n *SMANode) Category() string { return "indicator" }

func (n *SMANode) InputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "input", Type: workflow.PortSeries, Required: true},
	}
}

func (n *SMANode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "output", Type: workflow.PortSeries, Required: false},
	}
}

func (n *SMANode) ParamSchema() []workflow.ParamDef {
	return []workflow.ParamDef{
		{Name: "period", Type: "int", Default: 20, Description: "SMA window size"},
	}
}

func (n *SMANode) Execute(ctx context.Context, inputs map[string]any, params map[string]any, nctx *workflow.NodeContext) (map[string]any, error) {
	raw, ok := inputs["input"]
	if !ok {
		return nil, fmt.Errorf("sma: missing required input 'input'")
	}
	series, ok := toFloat64Slice(raw)
	if !ok {
		return nil, fmt.Errorf("sma: input must be []float64, got %T", raw)
	}

	period := 20
	if p, ok := params["period"]; ok {
		switch v := p.(type) {
		case float64:
			period = int(v)
		case int:
			period = v
		}
	}

	result := sma(series, period)
	return map[string]any{"output": result}, nil
}

func (n *SMANode) Validate() error {
	period := 20
	if p, ok := n.params["period"]; ok {
		switch v := p.(type) {
		case float64:
			period = int(v)
		case int:
			period = v
		}
	}
	if period <= 0 {
		return fmt.Errorf("sma: period must be positive, got %d", period)
	}
	return nil
}

// sma computes the simple moving average.
// For the first (period-1) elements, returns the mean of available values.
func sma(data []float64, period int) []float64 {
	if len(data) == 0 || period <= 0 {
		return nil
	}
	result := make([]float64, len(data))
	var sum float64
	for i, v := range data {
		sum += v
		if i < period {
			result[i] = sum / float64(i+1)
		} else {
			sum -= data[i-period]
			result[i] = sum / float64(period)
		}
	}
	return result
}

// toFloat64Slice attempts to convert any to []float64.
func toFloat64Slice(raw any) ([]float64, bool) {
	result := extractFloat64Slice(raw)
	return result, result != nil
}


