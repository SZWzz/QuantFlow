package nodes

import (
	"context"
	"fmt"

	"quantflow/internal/python"
	"quantflow/internal/workflow"
)

type IndicatorBBINode struct{ id string; params map[string]any }

func NewIndicatorBBINode(id string, params map[string]any) (workflow.BaseNode, error) {
	return &IndicatorBBINode{id: id, params: params}, nil
}

func (n *IndicatorBBINode) ID() string       { return n.id }
func (n *IndicatorBBINode) NodeType() string  { return "indicator_bbi" }
func (n *IndicatorBBINode) Category() string  { return "indicators" }

func (n *IndicatorBBINode) InputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{{Name: "prices", Type: workflow.PortSeries, Required: true}}
}

func (n *IndicatorBBINode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{{Name: "bbi", Type: workflow.PortSeries}}
}

func (n *IndicatorBBINode) ParamSchema() []workflow.ParamDef {
	return []workflow.ParamDef{}
}

func (n *IndicatorBBINode) Execute(ctx context.Context, inputs map[string]any, params map[string]any, nctx *workflow.NodeContext) (map[string]any, error) {
	if nctx == nil || nctx.Bridge == nil {
		return map[string]any{"bbi": []float64{}}, fmt.Errorf("python bridge required")
	}
	bridge := nctx.Bridge.(*python.PythonBridge)
	_ = bridge
	return map[string]any{"bbi": []float64{}}, nil
}

func (n *IndicatorBBINode) Validate() error { return nil }
