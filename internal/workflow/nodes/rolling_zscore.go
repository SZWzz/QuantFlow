package nodes

import (
	"context"
	"fmt"
	"math"

	"quantflow/internal/workflow"
)

// RollingZScoreNode computes rolling (value - mean) / stddev over N periods.
type RollingZScoreNode struct{ id string; params map[string]any }

func NewRollingZScoreNode(id string, params map[string]any) (workflow.BaseNode, error) {
	return &RollingZScoreNode{id: id, params: params}, nil
}

func (n *RollingZScoreNode) ID() string       { return n.id }
func (n *RollingZScoreNode) NodeType() string { return "rolling_zscore" }
func (n *RollingZScoreNode) Category() string { return "alpha" }

func (n *RollingZScoreNode) InputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "values", Type: workflow.PortSeries, Required: true},
	}
}

func (n *RollingZScoreNode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{{Name: "result", Type: workflow.PortSeries, Required: false}}
}

func (n *RollingZScoreNode) ParamSchema() []workflow.ParamDef {
	return []workflow.ParamDef{
		{Name: "period", Type: "number", Default: "20", Description: "Rolling window size"},
	}
}

func (n *RollingZScoreNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any) (map[string]any, error) {
	values := extractFloatSlice(inputs["values"])
	if values == nil {
		return nil, fmt.Errorf("rolling_zscore: values input is required")
	}

	period := int(getFloatParam(params, "period", 20))
	if period <= 1 {
		return nil, fmt.Errorf("rolling_zscore: period must be > 1, got %d", period)
	}
	if period > len(values) {
		return nil, fmt.Errorf("rolling_zscore: period %d > data length %d", period, len(values))
	}

	result := make([]float64, len(values))
	for i := period - 1; i < len(values); i++ {
		window := values[i-period+1 : i+1]
		// compute mean
		var sum float64
		for _, v := range window {
			sum += v
		}
		mean := sum / float64(period)
		// compute stddev
		var ssq float64
		for _, v := range window {
			d := v - mean
			ssq += d * d
		}
		std := math.Sqrt(ssq / float64(period))
		if std == 0 {
			result[i] = 0
		} else {
			result[i] = (values[i] - mean) / std
		}
	}

	return map[string]any{"result": result}, nil
}

func (n *RollingZScoreNode) Validate() error { return nil }
