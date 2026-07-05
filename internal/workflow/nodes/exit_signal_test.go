package nodes

import (
	"context"
	"testing"
)

func TestExitSignalNode_Execute(t *testing.T) {
	node, err := NewExitSignalNode("xs1", nil)
	if err != nil {
		t.Fatalf("NewExitSignalNode() error = %v", err)
	}
	if node.NodeType() != "exit_signal" {
		t.Errorf("NodeType() = %q, want %q", node.NodeType(), "exit_signal")
	}
	outputs, err := node.Execute(context.Background(), map[string]any{"condition": []float64{1, 1, 0, 0, 1, 0}}, nil, nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	exit := outputs["exit"].([]float64)
	expected := []float64{0, 0, -1, 0, 0, -1}
	for i, v := range expected {
		if exit[i] != v {
			t.Errorf("[%d] = %v, want %v", i, exit[i], v)
		}
	}
}

func TestExitSignalNode_MissingInput(t *testing.T) {
	node, _err := NewExitSignalNode("xs1", nil)
	if _err != nil {
		t.Fatalf("NewExitSignalNode() error = %v", _err)
	}
	_, err := node.Execute(context.Background(), map[string]any{}, nil, nil)
	if err == nil {
		t.Error("expected error for missing 'condition'")
	}
}
