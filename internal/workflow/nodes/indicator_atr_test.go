package nodes

import (
	"context"
	"testing"
)

func TestIndicatorATRNode_NoBridge(t *testing.T) {
	node, err := NewIndicatorATRNode("atr1", nil)
	if err != nil {
		t.Fatalf("NewIndicatorATRNode() error = %v", err)
	}
	if node.NodeType() != "indicator_atr" {
		t.Errorf("NodeType() = %q, want %q", node.NodeType(), "indicator_atr")
	}
	outputs, err := node.Execute(context.Background(), map[string]any{"ohlcv": []float64{1, 2, 3}}, nil, nil)
	if err == nil {
		t.Fatal("expected error for missing python bridge")
	}
	val, ok := outputs["atr"].([]float64)
	if !ok || len(val) != 0 {
		t.Errorf("expected empty []float64 for atr")
	}
}
