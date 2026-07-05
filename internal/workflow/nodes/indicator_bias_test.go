package nodes

import (
	"context"
	"testing"
)

func TestIndicatorBIASNode_NoBridge(t *testing.T) {
	node, err := NewIndicatorBIASNode("bias1", nil)
	if err != nil {
		t.Fatalf("NewIndicatorBIASNode() error = %v", err)
	}
	if node.NodeType() != "indicator_bias" {
		t.Errorf("NodeType() = %q, want %q", node.NodeType(), "indicator_bias")
	}
	outputs, err := node.Execute(context.Background(), map[string]any{"prices": []float64{1, 2, 3}}, nil, nil)
	if err == nil {
		t.Fatal("expected error for missing python bridge")
	}
	for _, key := range []string{"bias6", "bias12", "bias24"} {
		val, ok := outputs[key].([]float64)
		if !ok || len(val) != 0 {
			t.Errorf("expected empty []float64 for %q", key)
		}
	}
}
