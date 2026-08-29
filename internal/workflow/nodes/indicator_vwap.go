package nodes

import (
	"context"
	"fmt"
	"quantflow/internal/python"
	"quantflow/internal/workflow"
)

type IndicatorVWAPNode struct {
	id     string
	params map[string]any
}

func NewIndicatorVWAPNode(id string, params map[string]any) (workflow.BaseNode, error) {
	return &IndicatorVWAPNode{id: id, params: params}, nil
}

func (n *IndicatorVWAPNode) ID() string       { return n.id }
func (n *IndicatorVWAPNode) NodeType() string { return "indicator_vwap" }
func (n *IndicatorVWAPNode) Category() string { return "indicators" }

func (n *IndicatorVWAPNode) InputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{{Name: "ohlcv", Type: workflow.PortSeries, Required: true}}
}

func (n *IndicatorVWAPNode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{{Name: "vwap", Type: workflow.PortSeries}}
}

func (n *IndicatorVWAPNode) ParamSchema() []workflow.ParamDef {
	return []workflow.ParamDef{}
}

func (n *IndicatorVWAPNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any, nctx *workflow.NodeContext) (map[string]any, error) {
	if nctx == nil || nctx.Bridge == nil {
		return map[string]any{"vwap": []float64{}}, fmt.Errorf("python bridge required")
	}
	bridge := nctx.Bridge.(*python.PythonBridge)
	_ = bridge
	return map[string]any{"vwap": []float64{}}, nil
}

func (n *IndicatorVWAPNode) Validate() error { return nil }
