package nodes

import (
	"context"
	"fmt"

	"quantflow/internal/python"
	"quantflow/internal/workflow"
)

type IndicatorOBVNode struct{ id string; params map[string]any }

func NewIndicatorOBVNode(id string, params map[string]any) (workflow.BaseNode, error) {
	return &IndicatorOBVNode{id: id, params: params}, nil
}

func (n *IndicatorOBVNode) ID() string       { return n.id }
func (n *IndicatorOBVNode) NodeType() string  { return "indicator_obv" }
func (n *IndicatorOBVNode) Category() string  { return "indicators" }

func (n *IndicatorOBVNode) InputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{{Name: "ohlcv", Type: workflow.PortSeries, Required: true}}
}

func (n *IndicatorOBVNode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{{Name: "obv", Type: workflow.PortSeries}}
}

func (n *IndicatorOBVNode) ParamSchema() []workflow.ParamDef {
	return []workflow.ParamDef{}
}

func (n *IndicatorOBVNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any, nctx *workflow.NodeContext) (map[string]any, error) {
	if nctx == nil || nctx.Bridge == nil {
		return map[string]any{"obv": []float64{}}, fmt.Errorf("python bridge required")
	}
	bridge := nctx.Bridge.(*python.PythonBridge)
	_ = bridge
	return map[string]any{"obv": []float64{}}, nil
}

func (n *IndicatorOBVNode) Validate() error { return nil }
