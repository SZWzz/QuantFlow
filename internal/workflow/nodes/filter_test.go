package nodes

import (
	"context"
	"testing"
)

func TestFilterNode_Execute(t *testing.T) {
	node, err := NewFilterNode("f1", nil)
	if err != nil {
		t.Fatalf("NewFilterNode() error = %v", err)
	}
	if node.NodeType() != "filter" {
		t.Errorf("NodeType() = %q, want %q", node.NodeType(), "filter")
	}
	outputs, err := node.Execute(context.Background(), map[string]any{"series": []float64{1, 5, 3, 8, 2}}, map[string]any{"condition": "gt", "threshold": float64(3)}, nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	result := outputs["filtered"].([]float64)
	expected := []float64{5, 8}
	if len(result) != len(expected) {
		t.Fatalf("len = %d, want %d: %v", len(result), len(expected), result)
	}
	for i, v := range expected {
		if result[i] != v {
			t.Errorf("[%d] = %v, want %v", i, result[i], v)
		}
	}
}

func TestFilterNode_MissingInput(t *testing.T) {
	node, _err := NewFilterNode("f1", nil)
	if _err != nil {
		t.Fatalf("NewFilterNode() error = %v", _err)
	}
	_, err := node.Execute(context.Background(), map[string]any{}, nil, nil)
	if err == nil {
		t.Error("expected error for missing 'series' input")
	}
}
