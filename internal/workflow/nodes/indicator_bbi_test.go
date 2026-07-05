package nodes

import (
	"context"
	"testing"
)

func TestIndicatorBBINode_NoBridge(t *testing.T) {
	node, err := NewIndicatorBBINode("bbi1", nil)
	if err != nil {
		t.Fatalf("NewIndicatorBBINode() error = %v", err)
	}
	if node.NodeType() != "indicator_bbi" {
		t.Errorf("NodeType() = %q, want %q", node.NodeType(), "indicator_bbi")
	}
	outputs, err := node.Execute(context.Background(), map[string]any{"prices": []float64{1, 2, 3}}, nil, nil)
	if err == nil {
		t.Fatal("expected error for missing python bridge")
	}
	val, ok := outputs["bbi"].([]float64)
	if !ok || len(val) != 0 {
		t.Errorf("expected empty []float64 for bbi")
	}
}
