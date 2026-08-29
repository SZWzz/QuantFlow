package nodes

import (
	"context"
	"fmt"
	"quantflow/internal/workflow"
)

// ResampleNode resamples a time series to a different frequency using last-price aggregation.
type ResampleNode struct {
	id     string
	params map[string]any
}

// NewResampleNode creates a new ResampleNode.
func NewResampleNode(id string, params map[string]any) (workflow.BaseNode, error) {
	return &ResampleNode{id: id, params: params}, nil
}

func (n *ResampleNode) ID() string       { return n.id }
func (n *ResampleNode) NodeType() string { return "resample" }
func (n *ResampleNode) Category() string { return "data" }

func (n *ResampleNode) InputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{{Name: "series", Type: workflow.PortSeries, Required: true}}
}

func (n *ResampleNode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{{Name: "resampled", Type: workflow.PortSeries, Required: false}}
}

func (n *ResampleNode) ParamSchema() []workflow.ParamDef {
	return []workflow.ParamDef{
		{Name: "rule", Type: "string", Default: "1d", Description: "Resample rule: 1d, 1h, 1w, 1M"},
	}
}

func (n *ResampleNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any, nctx *workflow.NodeContext) (map[string]any, error) {
	rule := getStringParam(params, "rule", "1d")
	series := extractFloatSlice(inputs["series"])
	if series == nil {
		return nil, fmt.Errorf("resample: series input is required")
	}

	// Determine bucket size from rule (in number of data points)
	bucketSize := len(series)
	switch rule {
	case "1h":
		bucketSize = max(1, len(series)/24)
	case "1d":
		bucketSize = max(1, len(series)/7)
	case "1w":
		bucketSize = max(1, len(series)/4)
	case "1M":
		bucketSize = max(1, len(series)/2)
	}

	numBuckets := (len(series) + bucketSize - 1) / bucketSize
	result := make([]float64, 0, numBuckets)
	for i := 0; i < len(series); i += bucketSize {
		end := i + bucketSize
		if end > len(series) {
			end = len(series)
		}
		// Last price in bucket
		result = append(result, series[end-1])
	}
	return map[string]any{"resampled": result}, nil
}

func (n *ResampleNode) Validate() error {
	rule := getStringParam(n.params, "rule", "1d")
	switch rule {
	case "1h", "1d", "1w", "1M":
		return nil
	default:
		return fmt.Errorf("resample: invalid rule %q, expected 1h/1d/1w/1M", rule)
	}
}
