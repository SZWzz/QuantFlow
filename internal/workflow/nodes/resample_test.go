package nodes

import (
	"context"
	"testing"
)

func TestResampleNode_Execute(t *testing.T) {
	node, err := NewResampleNode("r1", nil)
	if err != nil {
		t.Fatalf("NewResampleNode() error = %v", err)
	}
	if node.NodeType() != "resample" {
		t.Errorf("NodeType() = %q, want %q", node.NodeType(), "resample")
	}
	outputs, err := node.Execute(context.Background(), map[string]any{"series": []float64{1, 2, 3, 4, 5, 6, 7}}, map[string]any{"rule": "1d"}, nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	result := outputs["resampled"].([]float64)
	if len(result) == 0 {
		t.Fatal("expected non-empty resampled output")
	}
}

func TestResampleNode_MissingInput(t *testing.T) {
	node, _err := NewResampleNode("r1", nil)
	if _err != nil {
		t.Fatalf("NewResampleNode() error = %v", _err)
	}
	_, err := node.Execute(context.Background(), map[string]any{}, nil, nil)
	if err == nil {
		t.Error("expected error for missing 'series' input")
	}
}
