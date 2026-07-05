package nodes

import (
	"context"
	"testing"
)

func TestIndicatorWRNode_NoBridge(t *testing.T) {
	node, err := NewIndicatorWRNode("wr1", nil)
	if err != nil {
		t.Fatalf("NewIndicatorWRNode() error = %v", err)
	}
	if node.NodeType() != "indicator_wr" {
		t.Errorf("NodeType() = %q, want %q", node.NodeType(), "indicator_wr")
	}
	outputs, err := node.Execute(context.Background(), map[string]any{"ohlcv": []float64{1, 2, 3}}, nil, nil)
	if err == nil {
		t.Fatal("expected error for missing python bridge")
	}
	for _, key := range []string{"wr1", "wr2"} {
		val, ok := outputs[key].([]float64)
		if !ok || len(val) != 0 {
			t.Errorf("expected empty []float64 for %q", key)
		}
	}
}
