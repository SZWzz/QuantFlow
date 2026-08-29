package nodes

import (
	"context"
	"fmt"
	"quantflow/internal/python"
	"quantflow/internal/workflow"
)

type IndicatorBIASNode struct {
	id     string
	params map[string]any
}

func NewIndicatorBIASNode(id string, params map[string]any) (workflow.BaseNode, error) {
	return &IndicatorBIASNode{id: id, params: params}, nil
}

func (n *IndicatorBIASNode) ID() string       { return n.id }
func (n *IndicatorBIASNode) NodeType() string { return "indicator_bias" }
func (n *IndicatorBIASNode) Category() string { return "indicators" }

func (n *IndicatorBIASNode) InputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{{Name: "prices", Type: workflow.PortSeries, Required: true}}
}

func (n *IndicatorBIASNode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "bias6", Type: workflow.PortSeries},
		{Name: "bias12", Type: workflow.PortSeries},
		{Name: "bias24", Type: workflow.PortSeries},
	}
}

func (n *IndicatorBIASNode) ParamSchema() []workflow.ParamDef {
	return []workflow.ParamDef{}
}

func (n *IndicatorBIASNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any, nctx *workflow.NodeContext) (map[string]any, error) {
	if nctx == nil || nctx.Bridge == nil {
		return map[string]any{"bias6": []float64{}, "bias12": []float64{}, "bias24": []float64{}}, fmt.Errorf("python bridge required")
	}
	bridge := nctx.Bridge.(*python.PythonBridge)
	_ = bridge
	return map[string]any{"bias6": []float64{}, "bias12": []float64{}, "bias24": []float64{}}, nil
}

func (n *IndicatorBIASNode) Validate() error { return nil }
