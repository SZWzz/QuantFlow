package nodes

import (
	"context"
	"testing"
)

func TestIfElseNode_Execute(t *testing.T) {
	node, err := NewIfElseNode("ie1", nil)
	if err != nil {
		t.Fatalf("NewIfElseNode() error = %v", err)
	}
	if node.NodeType() != "if_else" {
		t.Errorf("NodeType() = %q, want %q", node.NodeType(), "if_else")
	}
	outputs, err := node.Execute(context.Background(), map[string]any{
		"condition": []float64{1, 0, -1, 2},
		"a":         []float64{10, 20, 30, 40},
		"b":         []float64{100, 200, 300, 400},
	}, nil, nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	result := outputs["result"].([]float64)
	expected := []float64{10, 200, 300, 40}
	for i, v := range expected {
		if result[i] != v {
			t.Errorf("[%d] = %v, want %v", i, result[i], v)
		}
	}
}

func TestIfElseNode_MissingInput(t *testing.T) {
	node, _err := NewIfElseNode("ie1", nil)
	if _err != nil {
		t.Fatalf("NewIfElseNode() error = %v", _err)
	}
	_, err := node.Execute(context.Background(), map[string]any{"condition": []float64{1}}, nil, nil)
	if err == nil {
		t.Error("expected error for missing a/b")
	}
}
