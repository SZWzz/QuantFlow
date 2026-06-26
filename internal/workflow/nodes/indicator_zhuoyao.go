package nodes

import (
	"context"
	"fmt"

	"quantflow/internal/python"
	"quantflow/internal/workflow"
)

type IndicatorZhuoyaoNode struct{ id string; params map[string]any }

func NewIndicatorZhuoyaoNode(id string, params map[string]any) (workflow.BaseNode, error) {
	return &IndicatorZhuoyaoNode{id: id, params: params}, nil
}

func (n *IndicatorZhuoyaoNode) ID() string       { return n.id }
func (n *IndicatorZhuoyaoNode) NodeType() string  { return "indicator_zhuoyao" }
func (n *IndicatorZhuoyaoNode) Category() string  { return "indicators" }

func (n *IndicatorZhuoyaoNode) InputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{{Name: "ohlcv", Type: workflow.PortSeries, Required: true}}
}

func (n *IndicatorZhuoyaoNode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "zhuoyao_20", Type: workflow.PortSeries},
		{Name: "zhuoyao_60", Type: workflow.PortSeries},
		{Name: "zhuoyao_120", Type: workflow.PortSeries},
	}
}

func (n *IndicatorZhuoyaoNode) ParamSchema() []workflow.ParamDef {
	return []workflow.ParamDef{}
}

func (n *IndicatorZhuoyaoNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any, nctx *workflow.NodeContext) (map[string]any, error) {
	if nctx == nil || nctx.Bridge == nil {
		return map[string]any{"zhuoyao_20": []float64{}, "zhuoyao_60": []float64{}, "zhuoyao_120": []float64{}}, fmt.Errorf("python bridge required")
	}
	bridge := nctx.Bridge.(*python.PythonBridge)
	_ = bridge
	return map[string]any{"zhuoyao_20": []float64{}, "zhuoyao_60": []float64{}, "zhuoyao_120": []float64{}}, nil
}

func (n *IndicatorZhuoyaoNode) Validate() error { return nil }
