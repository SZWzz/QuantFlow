package nodes

import (
	"context"
	"fmt"
	"math"

	"quantflow/internal/workflow"
)

// ScaleNode normalizes a series via z-score or min-max scaling.
type ScaleNode struct {
	id     string
	params map[string]any
}

func NewScaleNode(id string, params map[string]any) (workflow.BaseNode, error) {
	return &ScaleNode{id: id, params: params}, nil
}

func (n *ScaleNode) ID() string       { return n.id }
func (n *ScaleNode) NodeType() string  { return "scale" }
func (n *ScaleNode) Category() string  { return "alpha" }

func (n *ScaleNode) InputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{{Name: "values", Type: workflow.PortSeries, Required: true}}
}

func (n *ScaleNode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{{Name: "result", Type: workflow.PortSeries, Required: false}}
}

func (n *ScaleNode) ParamSchema() []workflow.ParamDef {
	return []workflow.ParamDef{
		{Name: "method", Type: "string", Default: "zscore",
			Description: "Scaling method: zscore or minmax"},
	}
}

func (n *ScaleNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any, nctx *workflow.NodeContext) (map[string]any, error) {
	values := extractFloatSlice(inputs["values"])
	if values == nil {
		return nil, fmt.Errorf("scale: values input required")
	}
	if len(values) == 0 {
		return nil, fmt.Errorf("scale: values must not be empty")
	}

	method := getStringParam(params, "method", "zscore")
	nv := len(values)

	// Compute mean
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	mean := sum / float64(nv)

	result := make([]float64, nv)
	switch method {
	case "zscore":
		// Compute population std deviation
		sumSq := 0.0
		for _, v := range values {
			diff := v - mean
			sumSq += diff * diff
		}
		std := math.Sqrt(sumSq / float64(nv))
		if std == 0 {
			// All values identical; z-scores are all 0
			return map[string]any{"result": result}, nil
		}
		for i, v := range values {
			result[i] = (v - mean) / std
		}
	case "minmax":
		min := values[0]
		max := values[0]
		for _, v := range values {
			if v < min {
				min = v
			}
			if v > max {
				max = v
			}
		}
		denom := max - min
		if denom == 0 {
			for i := range result {
				result[i] = 0.5
			}
		} else {
			for i, v := range values {
				result[i] = (v - min) / denom
			}
		}
	default:
		return nil, fmt.Errorf("scale: unknown method %q, expected zscore or minmax", method)
	}

	return map[string]any{"result": result}, nil
}

func (n *ScaleNode) Validate() error {
	method := getStringParam(n.params, "method", "zscore")
	switch method {
	case "zscore", "minmax":
		return nil
	default:
		return fmt.Errorf("scale: invalid method %q, expected zscore or minmax", method)
	}
}
