package nodes

import (
	"context"
	"fmt"

	"quantflow/internal/python"
	"quantflow/internal/workflow"
)

type IndicatorATRNode struct{ id string; params map[string]any }

func NewIndicatorATRNode(id string, params map[string]any) (workflow.BaseNode, error) {
	return &IndicatorATRNode{id: id, params: params}, nil
}

func (n *IndicatorATRNode) ID() string       { return n.id }
func (n *IndicatorATRNode) NodeType() string  { return "indicator_atr" }
func (n *IndicatorATRNode) Category() string  { return "indicators" }

func (n *IndicatorATRNode) InputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{{Name: "ohlcv", Type: workflow.PortSeries, Required: true}}
}

func (n *IndicatorATRNode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{{Name: "atr", Type: workflow.PortSeries}}
}

func (n *IndicatorATRNode) ParamSchema() []workflow.ParamDef {
	return []workflow.ParamDef{{Name: "n", Type: "int", Default: "14", Description: "ATR period"}}
}

func (n *IndicatorATRNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any, nctx *workflow.NodeContext) (map[string]any, error) {
	if nctx == nil || nctx.Bridge == nil {
		return map[string]any{"atr": []float64{}}, fmt.Errorf("python bridge required")
	}
	bridge := nctx.Bridge.(*python.PythonBridge)
	_ = bridge
	return map[string]any{"atr": []float64{}}, nil
}

func (n *IndicatorATRNode) Validate() error { return nil }
