package nodes

import (
	"context"
	"testing"
)

func TestCancelOrderNode_Simulated(t *testing.T) {
	node, err := NewCancelOrderNode("co1", nil)
	if err != nil {
		t.Fatalf("NewCancelOrderNode() error = %v", err)
	}
	if node.NodeType() != "cancel_order" {
		t.Errorf("NodeType() = %q, want %q", node.NodeType(), "cancel_order")
	}
	outputs, err := node.Execute(context.Background(), map[string]any{}, map[string]any{"order_id": "ord-001"}, nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	success, ok := outputs["success"].(bool)
	if !ok || !success {
		t.Errorf("success = %v, want true", success)
	}
}

func TestCancelOrderNode_OrderIDFromInput(t *testing.T) {
	node, _err := NewCancelOrderNode("co1", nil)
	if _err != nil {
		t.Fatalf("NewCancelOrderNode() error = %v", _err)
	}
	outputs, err := node.Execute(context.Background(), map[string]any{"order_id": "ord-001"}, nil, nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if outputs["success"] != true {
		t.Error("expected success")
	}
}

func TestCancelOrderNode_MissingOrderID(t *testing.T) {
	node, _err := NewCancelOrderNode("co1", nil)
	if _err != nil {
		t.Fatalf("NewCancelOrderNode() error = %v", _err)
	}
	_, err := node.Execute(context.Background(), map[string]any{}, map[string]any{}, nil)
	if err == nil {
		t.Error("expected error for missing order_id")
	}
}
