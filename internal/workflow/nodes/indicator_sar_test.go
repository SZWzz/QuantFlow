package nodes

import (
	"context"
	"testing"
)

func TestIndicatorSARNode_NoBridge(t *testing.T) {
	node, err := NewIndicatorSARNode("sar1", nil)
	if err != nil {
		t.Fatalf("NewIndicatorSARNode() error = %v", err)
	}
	if node.NodeType() != "indicator_sar" {
		t.Errorf("NodeType() = %q, want %q", node.NodeType(), "indicator_sar")
	}
	outputs, err := node.Execute(context.Background(), map[string]any{"ohlcv": []float64{1, 2, 3}}, nil, nil)
	if err == nil {
		t.Fatal("expected error for missing python bridge")
	}
	val, ok := outputs["sar"].([]float64)
	if !ok || len(val) != 0 {
		t.Errorf("expected empty []float64 for sar")
	}
}
