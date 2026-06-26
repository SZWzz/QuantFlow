package nodes

import (
	"context"
	"fmt"

	"quantflow/internal/python"
	"quantflow/internal/workflow"
)

type IndicatorSARNode struct{ id string; params map[string]any }

func NewIndicatorSARNode(id string, params map[string]any) (workflow.BaseNode, error) {
	return &IndicatorSARNode{id: id, params: params}, nil
}

func (n *IndicatorSARNode) ID() string       { return n.id }
func (n *IndicatorSARNode) NodeType() string  { return "indicator_sar" }
func (n *IndicatorSARNode) Category() string  { return "indicators" }

func (n *IndicatorSARNode) InputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{{Name: "ohlcv", Type: workflow.PortSeries, Required: true}}
}

func (n *IndicatorSARNode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{{Name: "sar", Type: workflow.PortSeries}}
}

func (n *IndicatorSARNode) ParamSchema() []workflow.ParamDef {
	return []workflow.ParamDef{}
}

func (n *IndicatorSARNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any, nctx *workflow.NodeContext) (map[string]any, error) {
	if nctx == nil || nctx.Bridge == nil {
		return map[string]any{"sar": []float64{}}, fmt.Errorf("python bridge required")
	}
	bridge := nctx.Bridge.(*python.PythonBridge)
	_ = bridge
	return map[string]any{"sar": []float64{}}, nil
}

func (n *IndicatorSARNode) Validate() error { return nil }
