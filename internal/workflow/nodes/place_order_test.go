package nodes

import (
	"context"
	"testing"
)

func TestPlaceOrderNode_Simulated(t *testing.T) {
	node, err := NewPlaceOrderNode("po1", nil)
	if err != nil {
		t.Fatalf("NewPlaceOrderNode() error = %v", err)
	}
	if node.NodeType() != "place_order" {
		t.Errorf("NodeType() = %q, want %q", node.NodeType(), "place_order")
	}
	outputs, err := node.Execute(context.Background(), map[string]any{}, map[string]any{"symbol": "AAPL", "side": "buy", "quantity": float64(100)}, nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	orderID, ok := outputs["order_id"].(string)
	if !ok || orderID == "" {
		t.Errorf("expected non-empty order_id, got %v", orderID)
	}
	status, ok := outputs["status"].(string)
	if !ok || status != "simulated" {
		t.Errorf("status = %v, want 'simulated'", status)
	}
}

func TestPlaceOrderNode_SymbolFromInput(t *testing.T) {
	node, _err := NewPlaceOrderNode("po1", nil)
	if _err != nil {
		t.Fatalf("NewPlaceOrderNode() error = %v", _err)
	}
	outputs, err := node.Execute(context.Background(), map[string]any{"symbol": "AAPL"}, map[string]any{"side": "buy", "quantity": float64(100)}, nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if outputs["order_id"].(string) == "" {
		t.Error("expected non-empty order_id")
	}
}

func TestPlaceOrderNode_MissingSymbol(t *testing.T) {
	node, err := NewPlaceOrderNode("po1", nil)
	if err != nil {
		t.Fatalf("NewPlaceOrderNode() error = %v", err)
	}
	_, err = node.Execute(context.Background(), map[string]any{}, map[string]any{}, nil)
	if err == nil {
		t.Error("expected error for missing symbol")
	}
}
