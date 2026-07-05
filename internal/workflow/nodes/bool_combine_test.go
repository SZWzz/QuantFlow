package nodes

import (
	"context"
	"testing"
)

func TestBoolCombineNode_Execute(t *testing.T) {
	node, err := NewBoolCombineNode("bc1", nil)
	if err != nil {
		t.Fatalf("NewBoolCombineNode() error = %v", err)
	}
	if node.NodeType() != "bool_combine" {
		t.Errorf("NodeType() = %q, want %q", node.NodeType(), "bool_combine")
	}
	tests := []struct {
		op     string
		a, b   []float64
		expect []float64
	}{
		{"and", []float64{1, 1, 0}, []float64{1, 0, 1}, []float64{1, 0, 0}},
		{"or", []float64{1, 0, 0}, []float64{0, 1, 0}, []float64{1, 1, 0}},
		{"xor", []float64{1, 1, 0}, []float64{0, 1, 0}, []float64{1, 0, 0}},
	}
	for _, tc := range tests {
		outputs, err := node.Execute(context.Background(), map[string]any{"a": tc.a, "b": tc.b}, map[string]any{"op": tc.op}, nil)
		if err != nil {
			t.Fatalf("op=%q: %v", tc.op, err)
		}
		result := outputs["result"].([]float64)
		for i, v := range tc.expect {
			if result[i] != v {
				t.Errorf("op=%q [%d] = %v, want %v", tc.op, i, result[i], v)
			}
		}
	}
}

func TestBoolCombineNode_MissingInput(t *testing.T) {
	node, err := NewBoolCombineNode("bc1", nil)
	if err != nil {
		t.Fatalf("NewBoolCombineNode() error = %v", err)
	}
	_, err = node.Execute(context.Background(), map[string]any{"a": []float64{1}}, nil, nil)
	if err == nil {
		t.Error("expected error for missing b")
	}
}
