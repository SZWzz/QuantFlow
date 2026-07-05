package nodes

import (
	"context"
	"testing"
)

func TestInsiderTradesNode_MockData(t *testing.T) {
	node, err := NewInsiderTradesNode("it1", nil)
	if err != nil {
		t.Fatalf("NewInsiderTradesNode() error = %v", err)
	}
	if node.NodeType() != "insider_trades" {
		t.Errorf("NodeType() = %q, want %q", node.NodeType(), "insider_trades")
	}
	outputs, err := node.Execute(context.Background(), map[string]any{"symbol": "AAPL"}, nil, nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	netActivity, ok := outputs["net_activity"].(string)
	if !ok || netActivity != "neutral" {
		t.Errorf("net_activity = %q, want 'neutral'", netActivity)
	}
	signal, ok := outputs["signal"].(map[string]any)
	if !ok || signal["action"] != "hold" {
		t.Errorf("signal action = %v, want 'hold'", signal["action"])
	}
}

func TestInsiderTradesNode_MissingSymbol(t *testing.T) {
	node, _err := NewInsiderTradesNode("it1", nil)
	if _err != nil {
		t.Fatalf("NewInsiderTradesNode() error = %v", _err)
	}
	_, err := node.Execute(context.Background(), map[string]any{}, nil, nil)
	if err == nil {
		t.Error("expected error for missing symbol")
	}
}
