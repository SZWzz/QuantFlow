package nodes

import (
	"context"
	"fmt"

	"quantflow/internal/python"
	"quantflow/internal/workflow"
)

type IndicatorKDJNode struct{ id string; params map[string]any }

func NewIndicatorKDJNode(id string, params map[string]any) (workflow.BaseNode, error) {
	return &IndicatorKDJNode{id: id, params: params}, nil
}

func (n *IndicatorKDJNode) ID() string       { return n.id }
func (n *IndicatorKDJNode) NodeType() string  { return "indicator_kdj" }
func (n *IndicatorKDJNode) Category() string  { return "indicators" }

func (n *IndicatorKDJNode) InputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{{Name: "ohlcv", Type: workflow.PortSeries, Required: true}}
}

func (n *IndicatorKDJNode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "k", Type: workflow.PortSeries},
		{Name: "d", Type: workflow.PortSeries},
		{Name: "j", Type: workflow.PortSeries},
	}
}

func (n *IndicatorKDJNode) ParamSchema() []workflow.ParamDef {
	return []workflow.ParamDef{
		{Name: "n", Type: "int", Default: "9", Description: "KDJ period"},
		{Name: "m1", Type: "int", Default: "3", Description: "K smoothing"},
		{Name: "m2", Type: "int", Default: "3", Description: "D smoothing"},
	}
}

func (n *IndicatorKDJNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any, nctx *workflow.NodeContext) (map[string]any, error) {
	if nctx == nil || nctx.Bridge == nil {
		return map[string]any{"k": []float64{}, "d": []float64{}, "j": []float64{}}, fmt.Errorf("python bridge required")
	}
	bridge := nctx.Bridge.(*python.PythonBridge)
	_ = bridge
	return map[string]any{"k": []float64{}, "d": []float64{}, "j": []float64{}}, nil
}

func (n *IndicatorKDJNode) Validate() error { return nil }
