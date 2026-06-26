package nodes

import (
	"context"
	"fmt"
	"math"

	"quantflow/internal/workflow"
)

// StdDevNode computes rolling population standard deviation over N periods.
type StdDevNode struct {
	id     string
	params map[string]any
}

func NewStdDevNode(id string, params map[string]any) (workflow.BaseNode, error) {
	return &StdDevNode{id: id, params: params}, nil
}

func (n *StdDevNode) ID() string       { return n.id }
func (n *StdDevNode) NodeType() string  { return "std_dev" }
func (n *StdDevNode) Category() string  { return "alpha" }

func (n *StdDevNode) InputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{{Name: "values", Type: workflow.PortSeries, Required: true}}
}

func (n *StdDevNode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{{Name: "result", Type: workflow.PortSeries, Required: false}}
}

func (n *StdDevNode) ParamSchema() []workflow.ParamDef {
	return []workflow.ParamDef{
		{Name: "period", Type: "number", Default: "20", Description: "Rolling window size for std dev"},
	}
}

func (n *StdDevNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any, nctx *workflow.NodeContext) (map[string]any, error) {
	values := extractFloatSlice(inputs["values"])
	if values == nil {
		return nil, fmt.Errorf("std_dev: values input required")
	}

	period := int(getFloatParam(params, "period", 20))
	if period <= 0 || period > len(values) {
		return nil, fmt.Errorf("std_dev: period %d must be > 0 and <= len(values) %d", period, len(values))
	}

	result := make([]float64, len(values))
	for i := period - 1; i < len(values); i++ {
		sum, sumSq := 0.0, 0.0
		for j := i - period + 1; j <= i; j++ {
			sum += values[j]
		}
		mean := sum / float64(period)
		for j := i - period + 1; j <= i; j++ {
			diff := values[j] - mean
			sumSq += diff * diff
		}
		result[i] = math.Sqrt(sumSq / float64(period))
	}
	return map[string]any{"result": result}, nil
}

func (n *StdDevNode) Validate() error {
	period := int(getFloatParam(n.params, "period", 20))
	if period <= 0 {
		return fmt.Errorf("std_dev: period must be positive, got %d", period)
	}
	return nil
}
