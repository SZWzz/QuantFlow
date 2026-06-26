package nodes

import (
	"context"
	"fmt"

	"quantflow/internal/trading"
	"quantflow/internal/workflow"
)

// PlaceOrderNode submits an order to the trading OMS.
type PlaceOrderNode struct {
	id     string
	params map[string]any
}

// NewPlaceOrderNode creates a new PlaceOrderNode.
func NewPlaceOrderNode(id string, params map[string]any) (workflow.BaseNode, error) {
	return &PlaceOrderNode{id: id, params: params}, nil
}

func (n *PlaceOrderNode) ID() string        { return n.id }
func (n *PlaceOrderNode) NodeType() string  { return "place_order" }
func (n *PlaceOrderNode) Category() string  { return "trading" }

func (n *PlaceOrderNode) InputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "symbol", Type: workflow.PortString, Required: true},
		{Name: "quantity", Type: workflow.PortNumber, Required: true},
	}
}

func (n *PlaceOrderNode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "order_id", Type: workflow.PortString, Required: false},
		{Name: "status", Type: workflow.PortString, Required: false},
	}
}

func (n *PlaceOrderNode) ParamSchema() []workflow.ParamDef {
	return []workflow.ParamDef{
		{Name: "symbol", Type: "string", Default: "", Description: "Trading symbol"},
		{Name: "side", Type: "string", Default: "buy", Description: "buy or sell"},
		{Name: "order_type", Type: "string", Default: "market", Description: "market, limit, or stop"},
		{Name: "quantity", Type: "number", Default: "1", Description: "Order quantity"},
		{Name: "price", Type: "number", Default: "0", Description: "Limit price (0=market)"},
		{Name: "stop_price", Type: "number", Default: "0", Description: "Stop price"},
	}
}

func (n *PlaceOrderNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any, nctx *workflow.NodeContext) (map[string]any, error) {
	var oms *trading.OMS
	if nctx != nil {
		oms, _ = nctx.OMS.(*trading.OMS)
	}

	symbol := getStringParam(params, "symbol", "")
	if symbol == "" {
		if v, ok := inputs["symbol"]; ok {
			symbol = fmt.Sprintf("%v", v)
		}
	}
	if symbol == "" {
		return nil, fmt.Errorf("place_order: symbol is required")
	}

	side := trading.OrderSide(getStringParam(params, "side", "buy"))
	ot := trading.OrderType(getStringParam(params, "order_type", "market"))
	qty := getFloatParam(params, "quantity", 1)
	price := getFloatParam(params, "price", 0)
	stopPrice := getFloatParam(params, "stop_price", 0)

	if oms == nil {
		return map[string]any{"order_id": "sim-001", "status": "simulated"}, nil
	}

	var order *trading.Order
	var err error
	if oms.HasBroker() {
		order, err = oms.PlaceOrderLive(ctx, symbol, side, ot, qty, price, stopPrice)
	} else {
		order, err = oms.PlaceOrder(symbol, side, ot, qty, price)
	}
	if err != nil {
		return nil, fmt.Errorf("place_order: %w", err)
	}
	return map[string]any{"order_id": order.ID, "status": string(order.Status)}, nil
}

func (n *PlaceOrderNode) Validate() error {
	if getStringParam(n.params, "symbol", "") == "" {
		return fmt.Errorf("place_order: symbol required")
	}
	return nil
}
