package nodes

import (
	"context"
	"fmt"

	"quantflow/internal/python"
	"quantflow/internal/workflow"
)

type IndicatorDMINode struct{ id string; params map[string]any }

func NewIndicatorDMINode(id string, params map[string]any) (workflow.BaseNode, error) {
	return &IndicatorDMINode{id: id, params: params}, nil
}

func (n *IndicatorDMINode) ID() string       { return n.id }
func (n *IndicatorDMINode) NodeType() string  { return "indicator_dmi" }
func (n *IndicatorDMINode) Category() string  { return "indicators" }

func (n *IndicatorDMINode) InputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{{Name: "ohlcv", Type: workflow.PortSeries, Required: true}}
}

func (n *IndicatorDMINode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "pdi", Type: workflow.PortSeries},
		{Name: "mdi", Type: workflow.PortSeries},
		{Name: "adx", Type: workflow.PortSeries},
		{Name: "adxr", Type: workflow.PortSeries},
	}
}

func (n *IndicatorDMINode) ParamSchema() []workflow.ParamDef {
	return []workflow.ParamDef{
		{Name: "n", Type: "int", Default: "14", Description: "DMI period"},
		{Name: "m", Type: "int", Default: "6", Description: "ADXR smoothing"},
	}
}

func (n *IndicatorDMINode) Execute(ctx context.Context, inputs map[string]any, params map[string]any, nctx *workflow.NodeContext) (map[string]any, error) {
	if nctx == nil || nctx.Bridge == nil {
		return map[string]any{"pdi": []float64{}, "mdi": []float64{}, "adx": []float64{}, "adxr": []float64{}}, fmt.Errorf("python bridge required")
	}
	bridge := nctx.Bridge.(*python.PythonBridge)
	_ = bridge
	return map[string]any{"pdi": []float64{}, "mdi": []float64{}, "adx": []float64{}, "adxr": []float64{}}, nil
}

func (n *IndicatorDMINode) Validate() error { return nil }
