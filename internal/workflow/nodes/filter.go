package nodes

import (
	"context"
	"fmt"
	"quantflow/internal/workflow"
)

// FilterNode filters a series by a condition on a named column.
type FilterNode struct {
	id     string
	params map[string]any
}

// NewFilterNode creates a new FilterNode.
func NewFilterNode(id string, params map[string]any) (workflow.BaseNode, error) {
	return &FilterNode{id: id, params: params}, nil
}

func (n *FilterNode) ID() string       { return n.id }
func (n *FilterNode) NodeType() string { return "filter" }
func (n *FilterNode) Category() string { return "data" }

func (n *FilterNode) InputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{{Name: "series", Type: workflow.PortSeries, Required: true}}
}

func (n *FilterNode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{{Name: "filtered", Type: workflow.PortSeries, Required: false}}
}

func (n *FilterNode) ParamSchema() []workflow.ParamDef {
	return []workflow.ParamDef{
		{Name: "column", Type: "string", Default: "", Description: "Field name to filter on"},
		{Name: "condition", Type: "string", Default: "gt", Description: "Condition: gt, lt, gte, lte, eq"},
		{Name: "threshold", Type: "float", Default: 0, Description: "Threshold value to compare against"},
	}
}

func (n *FilterNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any, nctx *workflow.NodeContext) (map[string]any, error) {
	series := extractFloatSlice(inputs["series"])
	if series == nil {
		return nil, fmt.Errorf("filter: series input is required")
	}
	cond := getStringParam(params, "condition", "gt")
	threshold := getFloatParam(params, "threshold", 0)

	var result []float64
	for _, v := range series {
		keep := false
		switch cond {
		case "gt":
			keep = v > threshold
		case "lt":
			keep = v < threshold
		case "gte":
			keep = v >= threshold
		case "lte":
			keep = v <= threshold
		case "eq":
			keep = v == threshold
		default:
			keep = v > threshold
		}
		if keep {
			result = append(result, v)
		}
	}
	return map[string]any{"filtered": result}, nil
}

func (n *FilterNode) Validate() error {
	cond := getStringParam(n.params, "condition", "gt")
	switch cond {
	case "gt", "lt", "gte", "lte", "eq":
		return nil
	default:
		return fmt.Errorf("filter: invalid condition %q", cond)
	}
}
