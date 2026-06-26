package nodes

import (
	"context"
	"fmt"
	"sort"

	"quantflow/internal/workflow"
)

// RankNode computes cross-sectional rank of values within the entire series.
type RankNode struct {
	id     string
	params map[string]any
}

func NewRankNode(id string, params map[string]any) (workflow.BaseNode, error) {
	return &RankNode{id: id, params: params}, nil
}

func (n *RankNode) ID() string       { return n.id }
func (n *RankNode) NodeType() string  { return "rank" }
func (n *RankNode) Category() string  { return "alpha" }

func (n *RankNode) InputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{{Name: "values", Type: workflow.PortSeries, Required: true}}
}

func (n *RankNode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{{Name: "result", Type: workflow.PortSeries, Required: false}}
}

func (n *RankNode) ParamSchema() []workflow.ParamDef {
	return []workflow.ParamDef{
		{Name: "method", Type: "string", Default: "percentile",
			Description: "Rank method: percentile or minmax"},
	}
}

func (n *RankNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any, nctx *workflow.NodeContext) (map[string]any, error) {
	values := extractFloatSlice(inputs["values"])
	if values == nil {
		return nil, fmt.Errorf("rank: values input required")
	}
	if len(values) == 0 {
		return nil, fmt.Errorf("rank: values must not be empty")
	}

	method := getStringParam(params, "method", "percentile")
	nv := len(values)

	sorted := make([]float64, nv)
	copy(sorted, values)
	sort.Float64s(sorted)

	result := make([]float64, nv)
	switch method {
	case "percentile":
		for i, v := range values {
			// Count values strictly less than v for percentile rank
			count := 0
			for _, s := range sorted {
				if s < v {
					count++
				}
			}
			if nv > 1 {
				result[i] = float64(count) / float64(nv-1)
			}
		}
	case "minmax":
		min, max := sorted[0], sorted[nv-1]
		denom := max - min
		for i, v := range values {
			if denom == 0 {
				result[i] = 0.5
			} else {
				result[i] = (v - min) / denom
			}
		}
	default:
		return nil, fmt.Errorf("rank: unknown method %q, expected percentile or minmax", method)
	}

	return map[string]any{"result": result}, nil
}

func (n *RankNode) Validate() error {
	method := getStringParam(n.params, "method", "percentile")
	switch method {
	case "percentile", "minmax":
		return nil
	default:
		return fmt.Errorf("rank: invalid method %q, expected percentile or minmax", method)
	}
}
