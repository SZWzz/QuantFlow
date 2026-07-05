package nodes

import (
	"context"
	"testing"
)

func TestMergeNode_Outer(t *testing.T) {
	node, err := NewMergeNode("m1", nil)
	if err != nil {
		t.Fatalf("NewMergeNode() error = %v", err)
	}
	if node.NodeType() != "merge" {
		t.Errorf("NodeType() = %q, want %q", node.NodeType(), "merge")
	}
	outputs, err := node.Execute(context.Background(), map[string]any{"series_a": []float64{1, 2, 3}, "series_b": []float64{4, 5}}, map[string]any{"how": "outer"}, nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	result := outputs["merged"].([]float64)
	if len(result) != 3 {
		t.Fatalf("len = %d, want 3", len(result))
	}
	if result[0] != 2.5 || result[1] != 3.5 {
		t.Errorf("outer merge: got %v", result)
	}
}

func TestMergeNode_Inner(t *testing.T) {
	node, _err := NewMergeNode("m1", nil)
	if _err != nil {
		t.Fatalf("NewMergeNode() error = %v", _err)
	}
	outputs, err := node.Execute(context.Background(), map[string]any{"series_a": []float64{1, 2, 3}, "series_b": []float64{4, 5}}, map[string]any{"how": "inner"}, nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	result := outputs["merged"].([]float64)
	if len(result) != 3 {
		t.Fatalf("len = %d, want 3", len(result))
	}
}

func TestMergeNode_MissingInput(t *testing.T) {
	node, _err := NewMergeNode("m1", nil)
	if _err != nil {
		t.Fatalf("NewMergeNode() error = %v", _err)
	}
	_, err := node.Execute(context.Background(), map[string]any{"series_a": []float64{1}}, nil, nil)
	if err == nil {
		t.Error("expected error for missing series_b")
	}
}
