package nodes

import (
	"context"
	"testing"
)

func TestIndicatorOBVNode_NoBridge(t *testing.T) {
	node, err := NewIndicatorOBVNode("obv1", nil)
	if err != nil {
		t.Fatalf("NewIndicatorOBVNode() error = %v", err)
	}
	if node.NodeType() != "indicator_obv" {
		t.Errorf("NodeType() = %q, want %q", node.NodeType(), "indicator_obv")
	}
	outputs, err := node.Execute(context.Background(), map[string]any{"ohlcv": []float64{1, 2, 3}}, nil, nil)
	if err == nil {
		t.Fatal("expected error for missing python bridge")
	}
	val, ok := outputs["obv"].([]float64)
	if !ok || len(val) != 0 {
		t.Errorf("expected empty []float64 for obv")
	}
}
