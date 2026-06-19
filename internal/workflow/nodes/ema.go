package nodes

import (
	"context"
	"fmt"
	"quantflow/internal/workflow"
)

type EMANode struct{ id string; params map[string]any }

func NewEMANode(id string, params map[string]any) (workflow.BaseNode, error) {
	return &EMANode{id: id, params: params}, nil
}

func (n *EMANode) ID() string       { return n.id }
func (n *EMANode) NodeType() string  { return "ema" }
func (n *EMANode) Category() string  { return "indicator" }

func (n *EMANode) InputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{{Name: "prices", Type: workflow.PortSeries, Required: true}}
}

func (n *EMANode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{{Name: "ema", Type: workflow.PortSeries, Required: false}}
}

func (n *EMANode) ParamSchema() []workflow.ParamDef {
	return []workflow.ParamDef{{Name: "period", Type: "number", Default: "20", Description: "EMA period"}}
}

func (n *EMANode) Execute(ctx context.Context, inputs map[string]any, params map[string]any) (map[string]any, error) {
	prices := extractFloatSlice(inputs["prices"])
	if prices == nil { return nil, fmt.Errorf("ema: prices input required") }
	period := int(getFloatParam(params, "period", 20))
	return map[string]any{"ema": ema(prices, period)}, nil
}

func (n *EMANode) Validate() error { return nil }
