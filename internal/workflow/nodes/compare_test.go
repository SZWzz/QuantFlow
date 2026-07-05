package nodes

import (
	"context"
	"testing"
)

func TestCompareNode_Execute(t *testing.T) {
	node, err := NewCompareNode("cmp1", nil)
	if err != nil {
		t.Fatalf("NewCompareNode() error = %v", err)
	}
	if node.NodeType() != "compare" {
		t.Errorf("NodeType() = %q, want %q", node.NodeType(), "compare")
	}
	tests := []struct {
		op     string
		a, b   []float64
		expect []float64
	}{
		{"gt", []float64{1, 2, 3}, []float64{0, 2, 4}, []float64{1, 0, 0}},
		{"lt", []float64{1, 2, 3}, []float64{0, 2, 4}, []float64{0, 0, 1}},
		{"eq", []float64{1, 2, 3}, []float64{1, 0, 3}, []float64{1, 0, 1}},
		{"neq", []float64{1, 2, 3}, []float64{1, 0, 3}, []float64{0, 1, 0}},
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

func TestCompareNode_MissingInput(t *testing.T) {
	node, _err := NewCompareNode("cmp1", nil)
	if _err != nil {
		t.Fatalf("NewCompareNode() error = %v", _err)
	}
	_, err := node.Execute(context.Background(), map[string]any{"a": []float64{1}}, nil, nil)
	if err == nil {
		t.Error("expected error for missing b")
	}
}
