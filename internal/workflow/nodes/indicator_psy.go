package nodes

import (
	"context"
	"fmt"
	"quantflow/internal/python"
	"quantflow/internal/workflow"
)

type IndicatorPSYNode struct {
	id     string
	params map[string]any
}

func NewIndicatorPSYNode(id string, params map[string]any) (workflow.BaseNode, error) {
	return &IndicatorPSYNode{id: id, params: params}, nil
}

func (n *IndicatorPSYNode) ID() string       { return n.id }
func (n *IndicatorPSYNode) NodeType() string { return "indicator_psy" }
func (n *IndicatorPSYNode) Category() string { return "indicators" }

func (n *IndicatorPSYNode) InputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{{Name: "prices", Type: workflow.PortSeries, Required: true}}
}

func (n *IndicatorPSYNode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{{Name: "psy", Type: workflow.PortSeries}}
}

func (n *IndicatorPSYNode) ParamSchema() []workflow.ParamDef {
	return []workflow.ParamDef{
		{Name: "n", Type: "int", Default: "12", Description: "PSY period"},
	}
}

func (n *IndicatorPSYNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any, nctx *workflow.NodeContext) (map[string]any, error) {
	if nctx == nil || nctx.Bridge == nil {
		return map[string]any{"psy": []float64{}}, fmt.Errorf("python bridge required")
	}
	bridge := nctx.Bridge.(*python.PythonBridge)
	_ = bridge
	return map[string]any{"psy": []float64{}}, nil
}

func (n *IndicatorPSYNode) Validate() error { return nil }
