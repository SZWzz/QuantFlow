package nodes

import (
	"context"
	"testing"
)

func TestArithmeticNode_Execute(t *testing.T) {
	node, err := NewArithmeticNode("arith1", nil)
	if err != nil {
		t.Fatalf("NewArithmeticNode() error = %v", err)
	}
	if node.NodeType() != "arithmetic" {
		t.Errorf("NodeType() = %q, want %q", node.NodeType(), "arithmetic")
	}
	tests := []struct {
		op     string
		a, b   []float64
		expect []float64
	}{
		{"add", []float64{1, 2, 3}, []float64{4, 5, 6}, []float64{5, 7, 9}},
		{"sub", []float64{5, 3, 1}, []float64{1, 2, 3}, []float64{4, 1, -2}},
		{"mul", []float64{2, 3, 4}, []float64{3, 4, 5}, []float64{6, 12, 20}},
		{"div", []float64{10, 20, 30}, []float64{2, 4, 5}, []float64{5, 5, 6}},
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

func TestArithmeticNode_DivByZero(t *testing.T) {
	node, err := NewArithmeticNode("arith1", nil)
	if err != nil {
		t.Fatalf("NewArithmeticNode() error = %v", err)
	}
	outputs, err := node.Execute(context.Background(), map[string]any{"a": []float64{5}, "b": []float64{0}}, map[string]any{"op": "div"}, nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	result := outputs["result"].([]float64)
	if result[0] != 0 {
		t.Errorf("result = %v, want 0 (div by zero)", result[0])
	}
}

func TestArithmeticNode_MissingInput(t *testing.T) {
	node, err := NewArithmeticNode("arith1", nil)
	if err != nil {
		t.Fatalf("NewArithmeticNode() error = %v", err)
	}
	_, err = node.Execute(context.Background(), map[string]any{"a": []float64{1}}, nil, nil)
	if err == nil {
		t.Error("expected error for missing b")
	}
}

func TestArithmeticNode_UnknownOp(t *testing.T) {
	node, err := NewArithmeticNode("arith1", nil)
	if err != nil {
		t.Fatalf("NewArithmeticNode() error = %v", err)
	}
	_, err = node.Execute(context.Background(), map[string]any{"a": []float64{1}, "b": []float64{2}}, map[string]any{"op": "mod"}, nil)
	if err == nil {
		t.Error("expected error for unknown op")
	}
}
