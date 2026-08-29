package nodes

import (
	"context"
	"fmt"
	"quantflow/internal/python"
	"quantflow/internal/workflow"
)

type IndicatorBIASSignalNode struct {
	id     string
	params map[string]any
}

func NewIndicatorBIASSignalNode(id string, params map[string]any) (workflow.BaseNode, error) {
	return &IndicatorBIASSignalNode{id: id, params: params}, nil
}

func (n *IndicatorBIASSignalNode) ID() string       { return n.id }
func (n *IndicatorBIASSignalNode) NodeType() string { return "indicator_bias_signal" }
func (n *IndicatorBIASSignalNode) Category() string { return "indicators" }

func (n *IndicatorBIASSignalNode) InputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{{Name: "prices", Type: workflow.PortSeries, Required: true}}
}

func (n *IndicatorBIASSignalNode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "bias", Type: workflow.PortSeries},
		{Name: "signal_short", Type: workflow.PortSeries},
		{Name: "signal_long", Type: workflow.PortSeries},
	}
}

func (n *IndicatorBIASSignalNode) ParamSchema() []workflow.ParamDef {
	return []workflow.ParamDef{}
}

func (n *IndicatorBIASSignalNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any, nctx *workflow.NodeContext) (map[string]any, error) {
	if nctx == nil || nctx.Bridge == nil {
		return map[string]any{"bias": []float64{}, "signal_short": []float64{}, "signal_long": []float64{}}, fmt.Errorf("python bridge required")
	}
	bridge := nctx.Bridge.(*python.PythonBridge)
	_ = bridge
	return map[string]any{"bias": []float64{}, "signal_short": []float64{}, "signal_long": []float64{}}, nil
}

func (n *IndicatorBIASSignalNode) Validate() error { return nil }
