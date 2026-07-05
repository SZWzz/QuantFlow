package nodes

import (
	"context"
	"testing"
)

func TestIndicatorBRARNode_NoBridge(t *testing.T) {
	node, err := NewIndicatorBRARNode("brar1", nil)
	if err != nil {
		t.Fatalf("NewIndicatorBRARNode() error = %v", err)
	}
	if node.NodeType() != "indicator_brar" {
		t.Errorf("NodeType() = %q, want %q", node.NodeType(), "indicator_brar")
	}
	outputs, err := node.Execute(context.Background(), map[string]any{"ohlcv": []float64{1, 2, 3}}, nil, nil)
	if err == nil {
		t.Fatal("expected error for missing python bridge")
	}
	for _, key := range []string{"br", "ar"} {
		val, ok := outputs[key].([]float64)
		if !ok || len(val) != 0 {
			t.Errorf("expected empty []float64 for %q", key)
		}
	}
}
