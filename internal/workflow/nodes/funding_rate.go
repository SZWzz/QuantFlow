package nodes

import (
	"context"
	"quantflow/internal/workflow"
)

type FundingRateNode struct {
	id     string
	params map[string]any
}

func NewFundingRateNode(id string, params map[string]any) (workflow.BaseNode, error) {
	return &FundingRateNode{id: id, params: params}, nil
}

func (n *FundingRateNode) ID() string       { return n.id }
func (n *FundingRateNode) NodeType() string { return "funding_rate" }
func (n *FundingRateNode) Category() string { return "data" }

func (n *FundingRateNode) InputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{{Name: "symbol", Type: workflow.PortString, Required: true}}
}

func (n *FundingRateNode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "rate", Type: workflow.PortNumber},
		{Name: "next_funding_time", Type: workflow.PortNumber},
	}
}
func (n *FundingRateNode) ParamSchema() []workflow.ParamDef { return nil }
func (n *FundingRateNode) Validate() error                  { return nil }

func (n *FundingRateNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any, nctx *workflow.NodeContext) (map[string]any, error) {
	return map[string]any{"rate": 0.0, "next_funding_time": 0}, nil
}
