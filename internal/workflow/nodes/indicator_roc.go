package nodes

import (
	"context"
	"fmt"
	"quantflow/internal/python"
	"quantflow/internal/workflow"
)

type IndicatorROCNode struct {
	id     string
	params map[string]any
}

func NewIndicatorROCNode(id string, params map[string]any) (workflow.BaseNode, error) {
	return &IndicatorROCNode{id: id, params: params}, nil
}

func (n *IndicatorROCNode) ID() string       { return n.id }
func (n *IndicatorROCNode) NodeType() string { return "indicator_roc" }
func (n *IndicatorROCNode) Category() string { return "indicators" }

func (n *IndicatorROCNode) InputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{{Name: "prices", Type: workflow.PortSeries, Required: true}}
}

func (n *IndicatorROCNode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{{Name: "roc", Type: workflow.PortSeries}}
}

func (n *IndicatorROCNode) ParamSchema() []workflow.ParamDef {
	return []workflow.ParamDef{
		{Name: "n", Type: "int", Default: "12", Description: "ROC period"},
	}
}

func (n *IndicatorROCNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any, nctx *workflow.NodeContext) (map[string]any, error) {
	if nctx == nil || nctx.Bridge == nil {
		return map[string]any{"roc": []float64{}}, fmt.Errorf("python bridge required")
	}
	bridge := nctx.Bridge.(*python.PythonBridge)
	_ = bridge
	return map[string]any{"roc": []float64{}}, nil
}

func (n *IndicatorROCNode) Validate() error { return nil }
