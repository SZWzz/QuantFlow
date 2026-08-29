package nodes

import (
	"context"
	"fmt"
	"quantflow/internal/python"
	"quantflow/internal/workflow"
)

type IndicatorBRARNode struct {
	id     string
	params map[string]any
}

func NewIndicatorBRARNode(id string, params map[string]any) (workflow.BaseNode, error) {
	return &IndicatorBRARNode{id: id, params: params}, nil
}

func (n *IndicatorBRARNode) ID() string       { return n.id }
func (n *IndicatorBRARNode) NodeType() string { return "indicator_brar" }
func (n *IndicatorBRARNode) Category() string { return "indicators" }

func (n *IndicatorBRARNode) InputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{{Name: "ohlcv", Type: workflow.PortSeries, Required: true}}
}

func (n *IndicatorBRARNode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "br", Type: workflow.PortSeries},
		{Name: "ar", Type: workflow.PortSeries},
	}
}

func (n *IndicatorBRARNode) ParamSchema() []workflow.ParamDef {
	return []workflow.ParamDef{{Name: "n", Type: "int", Default: "26", Description: "BRAR period"}}
}

func (n *IndicatorBRARNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any, nctx *workflow.NodeContext) (map[string]any, error) {
	if nctx == nil || nctx.Bridge == nil {
		return map[string]any{"br": []float64{}, "ar": []float64{}}, fmt.Errorf("python bridge required")
	}
	bridge := nctx.Bridge.(*python.PythonBridge)
	_ = bridge
	return map[string]any{"br": []float64{}, "ar": []float64{}}, nil
}

func (n *IndicatorBRARNode) Validate() error { return nil }
