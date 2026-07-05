package nodes

import (
	"context"
	"testing"
)

func TestHoldSignalNode_Execute(t *testing.T) {
	node, err := NewHoldSignalNode("hs1", nil)
	if err != nil {
		t.Fatalf("NewHoldSignalNode() error = %v", err)
	}
	if node.NodeType() != "hold_signal" {
		t.Errorf("NodeType() = %q, want %q", node.NodeType(), "hold_signal")
	}
	outputs, err := node.Execute(context.Background(), map[string]any{"signal": []float64{1, 0, -1, 0, 1, 0, -1}}, nil, nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	position := outputs["position"].([]float64)
	expected := []float64{1, 1, 0, 0, 1, 1, 0}
	for i, v := range expected {
		if position[i] != v {
			t.Errorf("[%d] = %v, want %v", i, position[i], v)
		}
	}
}

func TestHoldSignalNode_MissingInput(t *testing.T) {
	node, _err := NewHoldSignalNode("hs1", nil)
	if _err != nil {
		t.Fatalf("NewHoldSignalNode() error = %v", _err)
	}
	_, err := node.Execute(context.Background(), map[string]any{}, nil, nil)
	if err == nil {
		t.Error("expected error for missing 'signal'")
	}
}
