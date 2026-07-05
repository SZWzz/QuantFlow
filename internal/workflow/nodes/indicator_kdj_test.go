package nodes

import (
	"context"
	"testing"
)

func TestIndicatorKDJNode_NoBridge(t *testing.T) {
	node, err := NewIndicatorKDJNode("kdj1", nil)
	if err != nil {
		t.Fatalf("NewIndicatorKDJNode() error = %v", err)
	}
	if node.NodeType() != "indicator_kdj" {
		t.Errorf("NodeType() = %q, want %q", node.NodeType(), "indicator_kdj")
	}
	outputs, err := node.Execute(context.Background(), map[string]any{"ohlcv": []float64{1, 2, 3}}, nil, nil)
	if err == nil {
		t.Fatal("expected error for missing python bridge")
	}
	for _, key := range []string{"k", "d", "j"} {
		val, ok := outputs[key].([]float64)
		if !ok || len(val) != 0 {
			t.Errorf("expected empty []float64 for %q", key)
		}
	}
}
