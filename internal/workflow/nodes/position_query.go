package nodes

import (
	"context"

	"quantflow/internal/trading"
	"quantflow/internal/workflow"
)

// PositionQueryNode queries current positions from the trading OMS.
type PositionQueryNode struct {
	id     string
	params map[string]any
}

// NewPositionQueryNode creates a new PositionQueryNode.
func NewPositionQueryNode(id string, params map[string]any) (workflow.BaseNode, error) {
	return &PositionQueryNode{id: id, params: params}, nil
}

func (n *PositionQueryNode) ID() string        { return n.id }
func (n *PositionQueryNode) NodeType() string  { return "position_query" }
func (n *PositionQueryNode) Category() string  { return "trading" }

func (n *PositionQueryNode) InputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "symbol", Type: workflow.PortString, Required: false},
	}
}

func (n *PositionQueryNode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "positions", Type: workflow.PortSeries, Required: false},
		{Name: "count", Type: workflow.PortNumber, Required: false},
	}
}

func (n *PositionQueryNode) ParamSchema() []workflow.ParamDef {
	return []workflow.ParamDef{
		{Name: "symbol", Type: "string", Default: "", Description: "Optional symbol filter"},
	}
}

func (n *PositionQueryNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any, nctx *workflow.NodeContext) (map[string]any, error) {
	var oms workflow.OMSService
	if nctx != nil {
		oms = nctx.OMS
	}

	if oms == nil {
		return map[string]any{"positions": []*trading.Position{}, "count": 0}, nil
	}
	symbol := getStringParam(params, "symbol", "")
	if symbol != "" {
		pos := oms.GetPosition(symbol)
		if pos != nil {
			return map[string]any{"positions": []*trading.Position{pos}, "count": 1}, nil
		}
		return map[string]any{"positions": []*trading.Position{}, "count": 0}, nil
	}
	positions := oms.GetAllPositions()
	return map[string]any{"positions": positions, "count": len(positions)}, nil
}

func (n *PositionQueryNode) Validate() error { return nil }
