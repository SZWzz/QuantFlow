package nodes

import (
	"context"
	"fmt"

	"quantflow/internal/python"
	"quantflow/internal/workflow"
)

type IndicatorAroonNode struct{ id string; params map[string]any }

func NewIndicatorAroonNode(id string, params map[string]any) (workflow.BaseNode, error) {
	return &IndicatorAroonNode{id: id, params: params}, nil
}

func (n *IndicatorAroonNode) ID() string       { return n.id }
func (n *IndicatorAroonNode) NodeType() string  { return "indicator_aroon" }
func (n *IndicatorAroonNode) Category() string  { return "indicators" }

func (n *IndicatorAroonNode) InputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{{Name: "ohlcv", Type: workflow.PortSeries, Required: true}}
}

func (n *IndicatorAroonNode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "aroon_up", Type: workflow.PortSeries},
		{Name: "aroon_down", Type: workflow.PortSeries},
	}
}

func (n *IndicatorAroonNode) ParamSchema() []workflow.ParamDef {
	return []workflow.ParamDef{{Name: "n", Type: "int", Default: "14", Description: "Aroon period"}}
}

func (n *IndicatorAroonNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any, nctx *workflow.NodeContext) (map[string]any, error) {
	if nctx == nil || nctx.Bridge == nil {
		return map[string]any{"aroon_up": []float64{}, "aroon_down": []float64{}}, fmt.Errorf("python bridge required")
	}
	bridge := nctx.Bridge.(*python.PythonBridge)
	_ = bridge
	return map[string]any{"aroon_up": []float64{}, "aroon_down": []float64{}}, nil
}

func (n *IndicatorAroonNode) Validate() error { return nil }
