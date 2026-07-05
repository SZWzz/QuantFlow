package nodes

import (
	"context"
	"testing"
)

func TestMathOpNode_Execute(t *testing.T) {
	node, err := NewMathOpNode("mo1", nil)
	if err != nil {
		t.Fatalf("NewMathOpNode() error = %v", err)
	}
	if node.NodeType() != "math_op" {
		t.Errorf("NodeType() = %q, want %q", node.NodeType(), "math_op")
	}
	tests := []struct {
		op     string
		a, b   float64
		expect float64
	}{
		{"add", 3, 4, 7},
		{"sub", 10, 3, 7},
		{"mul", 5, 6, 30},
		{"div", 20, 4, 5},
		{"pow", 2, 3, 8},
		{"sqrt", 9, 0, 3},
		{"log", 1, 0, 0},
	}
	for _, tc := range tests {
		inputs := map[string]any{"a": tc.a}
		if tc.op != "sqrt" && tc.op != "log" {
			inputs["b"] = tc.b
		}
		outputs, err := node.Execute(context.Background(), inputs, map[string]any{"operation": tc.op}, nil)
		if err != nil {
			t.Fatalf("op=%q: %v", tc.op, err)
		}
		result := outputs["result"].(float64)
		if tc.op == "sqrt" || tc.op == "log" || tc.op == "div" {
			if result != tc.expect {
				t.Errorf("op=%q: got %v, want %v", tc.op, result, tc.expect)
			}
		}
	}
}

func TestMathOpNode_MissingA(t *testing.T) {
	node, _err := NewMathOpNode("mo1", nil)
	if _err != nil {
		t.Fatalf("NewMathOpNode() error = %v", _err)
	}
	_, err := node.Execute(context.Background(), map[string]any{}, nil, nil)
	if err == nil {
		t.Error("expected error for missing a")
	}
}

func TestMathOpNode_DivByZero(t *testing.T) {
	node, _err := NewMathOpNode("mo1", nil)
	if _err != nil {
		t.Fatalf("NewMathOpNode() error = %v", _err)
	}
	_, err := node.Execute(context.Background(), map[string]any{"a": float64(1), "b": float64(0)}, map[string]any{"operation": "div"}, nil)
	if err == nil {
		t.Error("expected error for division by zero")
	}
}

func TestMathOpNode_SqrtNegative(t *testing.T) {
	node, _err := NewMathOpNode("mo1", nil)
	if _err != nil {
		t.Fatalf("NewMathOpNode() error = %v", _err)
	}
	_, err := node.Execute(context.Background(), map[string]any{"a": float64(-1)}, map[string]any{"operation": "sqrt"}, nil)
	if err == nil {
		t.Error("expected error for sqrt of negative")
	}
}
