package nodes

import (
	"context"
	"testing"
)

func TestStopLossNode_Fixed(t *testing.T) {
	node, err := NewStopLossNode("sl1", nil)
	if err != nil {
		t.Fatalf("NewStopLossNode() error = %v", err)
	}
	if node.NodeType() != "stop_loss" {
		t.Errorf("NodeType() = %q, want %q", node.NodeType(), "stop_loss")
	}
	outputs, err := node.Execute(context.Background(), map[string]any{"price": float64(95), "entry_price": float64(100)}, map[string]any{"stop_pct": float64(5), "trailing": "false"}, nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if outputs["triggered"].(float64) != 1 {
		t.Errorf("triggered = %v, want 1", outputs["triggered"])
	}
	if outputs["stop_price"].(float64) != 95 {
		t.Errorf("stop_price = %v, want 95", outputs["stop_price"])
	}
}

func TestStopLossNode_NotTriggered(t *testing.T) {
	node, _err := NewStopLossNode("sl1", nil)
	if _err != nil {
		t.Fatalf("NewStopLossNode() error = %v", _err)
	}
	outputs, err := node.Execute(context.Background(), map[string]any{"price": float64(98), "entry_price": float64(100)}, map[string]any{"stop_pct": float64(5), "trailing": "false"}, nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if outputs["triggered"].(float64) != 0 {
		t.Errorf("triggered = %v, want 0", outputs["triggered"])
	}
}

func TestStopLossNode_Trailing(t *testing.T) {
	node, _err := NewStopLossNode("sl1", nil)
	if _err != nil {
		t.Fatalf("NewStopLossNode() error = %v", _err)
	}
	outputs, err := node.Execute(context.Background(), map[string]any{"price": float64(110), "entry_price": float64(100)}, map[string]any{"stop_pct": float64(5), "trailing": "true"}, nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if outputs["triggered"].(float64) != 0 {
		t.Errorf("triggered = %v, want 0 (price is higher than entry)", outputs["triggered"])
	}
}

func TestStopLossNode_MissingInput(t *testing.T) {
	node, _err := NewStopLossNode("sl1", nil)
	if _err != nil {
		t.Fatalf("NewStopLossNode() error = %v", _err)
	}
	_, err := node.Execute(context.Background(), map[string]any{"price": float64(100)}, nil, nil)
	if err == nil {
		t.Error("expected error for missing entry_price")
	}
}
