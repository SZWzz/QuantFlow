package nodes

import (
	"context"
	"fmt"
	"quantflow/internal/python"
	"quantflow/internal/workflow"
)

type IndicatorWRNode struct {
	id     string
	params map[string]any
}

func NewIndicatorWRNode(id string, params map[string]any) (workflow.BaseNode, error) {
	return &IndicatorWRNode{id: id, params: params}, nil
}

func (n *IndicatorWRNode) ID() string       { return n.id }
func (n *IndicatorWRNode) NodeType() string { return "indicator_wr" }
func (n *IndicatorWRNode) Category() string { return "indicators" }

func (n *IndicatorWRNode) InputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{{Name: "ohlcv", Type: workflow.PortSeries, Required: true}}
}

func (n *IndicatorWRNode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "wr1", Type: workflow.PortSeries},
		{Name: "wr2", Type: workflow.PortSeries},
	}
}

func (n *IndicatorWRNode) ParamSchema() []workflow.ParamDef {
	return []workflow.ParamDef{
		{Name: "n1", Type: "int", Default: "10", Description: "WR period 1"},
		{Name: "n2", Type: "int", Default: "6", Description: "WR period 2"},
	}
}

func (n *IndicatorWRNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any, nctx *workflow.NodeContext) (map[string]any, error) {
	if nctx == nil || nctx.Bridge == nil {
		return map[string]any{"wr1": []float64{}, "wr2": []float64{}}, fmt.Errorf("python bridge required")
	}
	bridge := nctx.Bridge.(*python.PythonBridge)
	_ = bridge
	return map[string]any{"wr1": []float64{}, "wr2": []float64{}}, nil
}

func (n *IndicatorWRNode) Validate() error { return nil }
