package nodes

import (
	"context"
	"fmt"

	"quantflow/internal/workflow"
)

// MergeNode merges two data series by index alignment (inner/outer join).
type MergeNode struct{ id string; params map[string]any }

// NewMergeNode creates a new MergeNode.
func NewMergeNode(id string, params map[string]any) (workflow.BaseNode, error) {
	return &MergeNode{id: id, params: params}, nil
}

func (n *MergeNode) ID() string       { return n.id }
func (n *MergeNode) NodeType() string { return "merge" }
func (n *MergeNode) Category() string { return "data" }

func (n *MergeNode) InputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "series_a", Type: workflow.PortSeries, Required: true},
		{Name: "series_b", Type: workflow.PortSeries, Required: true},
	}
}

func (n *MergeNode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{{Name: "merged", Type: workflow.PortSeries, Required: false}}
}

func (n *MergeNode) ParamSchema() []workflow.ParamDef {
	return []workflow.ParamDef{
		{Name: "how", Type: "string", Default: "outer", Description: "Merge method: inner or outer"},
	}
}

func (n *MergeNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any, nctx *workflow.NodeContext) (map[string]any, error) {
	how := getStringParam(params, "how", "outer")
	a := extractFloatSlice(inputs["series_a"])
	b := extractFloatSlice(inputs["series_b"])
	if a == nil || b == nil {
		return nil, fmt.Errorf("merge: series_a and series_b are required")
	}
	m := len(a)
	if how == "outer" && len(b) > m {
		m = len(b)
	}
	result := make([]float64, m)
	for i := 0; i < m; i++ {
		var va, vb float64
		if i < len(a) { va = a[i] }
		if i < len(b) { vb = b[i] }
		if how == "inner" && (i >= len(a) || i >= len(b)) { continue }
		result[i] = (va + vb) / 2
	}
	return map[string]any{"merged": result}, nil
}

func (n *MergeNode) Validate() error {
	how := getStringParam(n.params, "how", "outer")
	if how != "inner" && how != "outer" {
		return fmt.Errorf("merge: how must be 'inner' or 'outer', got %q", how)
	}
	return nil
}
