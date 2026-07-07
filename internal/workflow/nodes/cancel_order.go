package nodes

import (
	"context"
	"fmt"

	"quantflow/internal/workflow"
)

// CancelOrderNode cancels an existing order by ID.
type CancelOrderNode struct {
	id     string
	params map[string]any
}

// NewCancelOrderNode creates a new CancelOrderNode.
func NewCancelOrderNode(id string, params map[string]any) (workflow.BaseNode, error) {
	return &CancelOrderNode{id: id, params: params}, nil
}

func (n *CancelOrderNode) ID() string        { return n.id }
func (n *CancelOrderNode) NodeType() string  { return "cancel_order" }
func (n *CancelOrderNode) Category() string  { return "trading" }

func (n *CancelOrderNode) InputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "order_id", Type: workflow.PortString, Required: true},
	}
}

func (n *CancelOrderNode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "success", Type: workflow.PortBoolean, Required: false},
	}
}

func (n *CancelOrderNode) ParamSchema() []workflow.ParamDef {
	return []workflow.ParamDef{
		{Name: "order_id", Type: "string", Default: "", Description: "Order ID to cancel"},
	}
}

func (n *CancelOrderNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any, nctx *workflow.NodeContext) (map[string]any, error) {
	var oms workflow.OMSService
	if nctx != nil {
		oms = nctx.OMS
	}

	orderID := getStringParam(params, "order_id", "")
	if orderID == "" {
		if v, ok := inputs["order_id"]; ok {
			orderID = fmt.Sprintf("%v", v)
		}
	}
	if orderID == "" {
		return nil, fmt.Errorf("cancel_order: order_id is required")
	}
	if oms != nil {
		if err := oms.CancelOrder(orderID); err != nil {
			return map[string]any{"success": false, "error": err.Error()}, nil
		}
	}
	return map[string]any{"success": true}, nil
}

func (n *CancelOrderNode) Validate() error { return nil }
