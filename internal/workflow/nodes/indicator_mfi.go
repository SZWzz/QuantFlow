package nodes

import (
	"context"
	"fmt"
	"quantflow/internal/python"
	"quantflow/internal/workflow"
)

type IndicatorMFINode struct {
	id     string
	params map[string]any
}

func NewIndicatorMFINode(id string, params map[string]any) (workflow.BaseNode, error) {
	return &IndicatorMFINode{id: id, params: params}, nil
}

func (n *IndicatorMFINode) ID() string       { return n.id }
func (n *IndicatorMFINode) NodeType() string { return "indicator_mfi" }
func (n *IndicatorMFINode) Category() string { return "indicators" }

func (n *IndicatorMFINode) InputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{{Name: "ohlcv", Type: workflow.PortSeries, Required: true}}
}

func (n *IndicatorMFINode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{{Name: "mfi", Type: workflow.PortSeries}}
}

func (n *IndicatorMFINode) ParamSchema() []workflow.ParamDef {
	return []workflow.ParamDef{{Name: "n", Type: "int", Default: "14", Description: "MFI period"}}
}

func (n *IndicatorMFINode) Execute(ctx context.Context, inputs map[string]any, params map[string]any, nctx *workflow.NodeContext) (map[string]any, error) {
	if nctx == nil || nctx.Bridge == nil {
		return map[string]any{"mfi": []float64{}}, fmt.Errorf("python bridge required")
	}
	bridge := nctx.Bridge.(*python.PythonBridge)
	_ = bridge
	return map[string]any{"mfi": []float64{}}, nil
}

func (n *IndicatorMFINode) Validate() error { return nil }
