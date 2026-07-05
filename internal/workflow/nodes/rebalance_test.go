package nodes

import (
	"context"
	"testing"
)

func TestRebalanceNode_Execute(t *testing.T) {
	node, err := NewRebalanceNode("rb1", nil)
	if err != nil {
		t.Fatalf("NewRebalanceNode() error = %v", err)
	}
	if node.NodeType() != "rebalance" {
		t.Errorf("NodeType() = %q, want %q", node.NodeType(), "rebalance")
	}
	outputs, err := node.Execute(context.Background(), map[string]any{"weights": []float64{0.5, 0.3, 0.4, 0.2, 0.6}}, map[string]any{"frequency": float64(2)}, nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	result := outputs["rebalanced"].([]float64)
	if len(result) != 5 {
		t.Fatalf("len = %d, want 5", len(result))
	}
	if result[0] != 0.5 || result[1] != 0.5 || result[2] != 0.4 {
		t.Errorf("rebalanced: got %v", result)
	}
}

func TestRebalanceNode_MissingInput(t *testing.T) {
	node, _err := NewRebalanceNode("rb1", nil)
	if _err != nil {
		t.Fatalf("NewRebalanceNode() error = %v", _err)
	}
	_, err := node.Execute(context.Background(), map[string]any{}, nil, nil)
	if err == nil {
		t.Error("expected error for missing 'weights' input")
	}
}
