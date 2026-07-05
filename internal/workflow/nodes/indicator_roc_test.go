package nodes

import (
	"context"
	"testing"
)

func TestIndicatorROCNode_NoBridge(t *testing.T) {
	node, err := NewIndicatorROCNode("roc1", nil)
	if err != nil {
		t.Fatalf("NewIndicatorROCNode() error = %v", err)
	}
	if node.NodeType() != "indicator_roc" {
		t.Errorf("NodeType() = %q, want %q", node.NodeType(), "indicator_roc")
	}
	outputs, err := node.Execute(context.Background(), map[string]any{"prices": []float64{1, 2, 3}}, nil, nil)
	if err == nil {
		t.Fatal("expected error for missing python bridge")
	}
	val, ok := outputs["roc"].([]float64)
	if !ok || len(val) != 0 {
		t.Errorf("expected empty []float64 for roc")
	}
}
