package nodes

import (
	"context"
	"fmt"

	"quantflow/internal/python"
	"quantflow/internal/workflow"
)

type IndicatorCCINode struct{ id string; params map[string]any }

func NewIndicatorCCINode(id string, params map[string]any) (workflow.BaseNode, error) {
	return &IndicatorCCINode{id: id, params: params}, nil
}

func (n *IndicatorCCINode) ID() string       { return n.id }
func (n *IndicatorCCINode) NodeType() string  { return "indicator_cci" }
func (n *IndicatorCCINode) Category() string  { return "indicators" }

func (n *IndicatorCCINode) InputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{{Name: "ohlcv", Type: workflow.PortSeries, Required: true}}
}

func (n *IndicatorCCINode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{{Name: "cci", Type: workflow.PortSeries}}
}

func (n *IndicatorCCINode) ParamSchema() []workflow.ParamDef {
	return []workflow.ParamDef{{Name: "n", Type: "int", Default: "14", Description: "CCI period"}}
}

func (n *IndicatorCCINode) Execute(ctx context.Context, inputs map[string]any, params map[string]any, nctx *workflow.NodeContext) (map[string]any, error) {
	if nctx == nil || nctx.Bridge == nil {
		return map[string]any{"cci": []float64{}}, fmt.Errorf("python bridge required")
	}
	bridge := nctx.Bridge.(*python.PythonBridge)
	_ = bridge
	return map[string]any{"cci": []float64{}}, nil
}

func (n *IndicatorCCINode) Validate() error { return nil }
