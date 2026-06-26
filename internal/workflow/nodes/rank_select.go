package nodes

import (
	"context"
	"fmt"
	"sort"

	"quantflow/internal/workflow"
)

// RankSelectNode selects the top N (or bottom N) stocks based on factor values,
// producing a boolean 1/0 mask array.
type RankSelectNode struct {
	id     string
	params map[string]any
}

// NewRankSelectNode creates a new RankSelectNode.
func NewRankSelectNode(id string, params map[string]any) (workflow.BaseNode, error) {
	return &RankSelectNode{id: id, params: params}, nil
}

func (n *RankSelectNode) ID() string       { return n.id }
func (n *RankSelectNode) NodeType() string { return "rank_select" }
func (n *RankSelectNode) Category() string { return "signal" }

func (n *RankSelectNode) InputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "factor_values", Type: workflow.PortSeries, Required: true},
	}
}

func (n *RankSelectNode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "selected", Type: workflow.PortSeries, Required: false},
	}
}

func (n *RankSelectNode) ParamSchema() []workflow.ParamDef {
	return []workflow.ParamDef{
		{Name: "top_n", Type: "number", Default: 10, Description: "Number of top stocks to select"},
		{Name: "ascending", Type: "bool", Default: "false", Description: "If true, select bottom N (lowest values)"},
	}
}

func (n *RankSelectNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any, nctx *workflow.NodeContext) (map[string]any, error) {
	values := extractFloatSlice(inputs["factor_values"])
	if values == nil {
		return nil, fmt.Errorf("rank_select: factor_values input is required")
	}
	if len(values) == 0 {
		return nil, fmt.Errorf("rank_select: factor_values must not be empty")
	}

	topN := int(getFloatParam(params, "top_n", 10))
	ascending := getStringParam(params, "ascending", "false") == "true"
	if topN <= 0 {
		topN = 1
	}
	if topN > len(values) {
		topN = len(values)
	}

	// Sort a copy to find the threshold value for top N.
	sorted := make([]float64, len(values))
	copy(sorted, values)
	sort.Float64s(sorted)

	// Determine the cutoff value: the Nth from the appropriate end.
	var cutoff float64
	if ascending {
		cutoff = sorted[topN-1] // top N lowest: the largest selected value
	} else {
		cutoff = sorted[len(sorted)-topN] // top N highest: the smallest selected value
	}

	// Build boolean mask: 1 if selected, 0 otherwise.
	selected := make([]float64, len(values))
	for i, v := range values {
		if ascending {
			if v <= cutoff {
				selected[i] = 1
			}
		} else {
			if v >= cutoff {
				selected[i] = 1
			}
		}
	}

	return map[string]any{"selected": selected}, nil
}

func (n *RankSelectNode) Validate() error {
	topN := int(getFloatParam(n.params, "top_n", 10))
	if topN <= 0 {
		return fmt.Errorf("rank_select: top_n must be > 0, got %d", topN)
	}
	return nil
}
