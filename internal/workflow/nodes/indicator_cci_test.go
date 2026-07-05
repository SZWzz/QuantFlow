package nodes

import (
	"context"
	"testing"
)

func TestIndicatorCCINode_NoBridge(t *testing.T) {
	node, err := NewIndicatorCCINode("cci1", nil)
	if err != nil {
		t.Fatalf("NewIndicatorCCINode() error = %v", err)
	}
	if node.NodeType() != "indicator_cci" {
		t.Errorf("NodeType() = %q, want %q", node.NodeType(), "indicator_cci")
	}
	outputs, err := node.Execute(context.Background(), map[string]any{"ohlcv": []float64{1, 2, 3}}, nil, nil)
	if err == nil {
		t.Fatal("expected error for missing python bridge")
	}
	val, ok := outputs["cci"].([]float64)
	if !ok || len(val) != 0 {
		t.Errorf("expected empty []float64 for cci")
	}
}
