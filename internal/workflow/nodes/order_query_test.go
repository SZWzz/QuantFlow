package nodes

import (
	"context"
	"quantflow/internal/trading"
	"testing"
)

func TestOrderQueryNode_NoOMS(t *testing.T) {
	node, err := NewOrderQueryNode("oq1", nil)
	if err != nil {
		t.Fatalf("NewOrderQueryNode() error = %v", err)
	}
	if node.NodeType() != "order_query" {
		t.Errorf("NodeType() = %q, want %q", node.NodeType(), "order_query")
	}
	outputs, err := node.Execute(context.Background(), map[string]any{}, nil, nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	orders, ok := outputs["orders"].([]*trading.Order)
	if !ok || len(orders) != 0 {
		t.Errorf("expected empty orders slice")
	}
	trades, ok := outputs["trades"].([]*trading.Trade)
	if !ok || len(trades) != 0 {
		t.Errorf("expected empty trades slice")
	}
}
