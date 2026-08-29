package nodes

import (
	"context"
	"quantflow/internal/trading"
	"quantflow/internal/workflow"
)

// OrderQueryNode queries orders and trades from the trading OMS.
type OrderQueryNode struct {
	id     string
	params map[string]any
}

// NewOrderQueryNode creates a new OrderQueryNode.
func NewOrderQueryNode(id string, params map[string]any) (workflow.BaseNode, error) {
	return &OrderQueryNode{id: id, params: params}, nil
}

func (n *OrderQueryNode) ID() string       { return n.id }
func (n *OrderQueryNode) NodeType() string { return "order_query" }
func (n *OrderQueryNode) Category() string { return "trading" }

func (n *OrderQueryNode) InputPorts() []workflow.PortDefinition { return nil }

func (n *OrderQueryNode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "orders", Type: workflow.PortSeries, Required: false},
		{Name: "trades", Type: workflow.PortSeries, Required: false},
	}
}

func (n *OrderQueryNode) ParamSchema() []workflow.ParamDef {
	return []workflow.ParamDef{
		{Name: "status", Type: "string", Default: "", Description: "Filter by status"},
	}
}

func (n *OrderQueryNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any, nctx *workflow.NodeContext) (map[string]any, error) {
	var oms workflow.OMSService
	if nctx != nil {
		oms = nctx.OMS
	}

	if oms == nil {
		return map[string]any{"orders": []*trading.Order{}, "trades": []*trading.Trade{}}, nil
	}
	status := getStringParam(params, "status", "")
	orders := oms.GetOrders()
	var filtered []*trading.Order
	for _, o := range orders {
		if status == "" || string(o.Status) == status {
			filtered = append(filtered, o)
		}
	}
	return map[string]any{"orders": filtered, "trades": oms.GetTrades()}, nil
}

func (n *OrderQueryNode) Validate() error { return nil }
