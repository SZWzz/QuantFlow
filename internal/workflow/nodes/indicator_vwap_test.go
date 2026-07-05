package nodes

import (
	"context"
	"testing"
)

func TestIndicatorVWAPNode_NoBridge(t *testing.T) {
	node, err := NewIndicatorVWAPNode("vwap1", nil)
	if err != nil {
		t.Fatalf("NewIndicatorVWAPNode() error = %v", err)
	}
	if node.NodeType() != "indicator_vwap" {
		t.Errorf("NodeType() = %q, want %q", node.NodeType(), "indicator_vwap")
	}
	outputs, err := node.Execute(context.Background(), map[string]any{"ohlcv": []float64{1, 2, 3}}, nil, nil)
	if err == nil {
		t.Fatal("expected error for missing python bridge")
	}
	val, ok := outputs["vwap"].([]float64)
	if !ok || len(val) != 0 {
		t.Errorf("expected empty []float64 for vwap")
	}
}
