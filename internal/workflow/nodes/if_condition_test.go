package nodes

import (
	"context"
	"testing"
)

func TestIfConditionNode_Execute(t *testing.T) {
	node, err := NewIfConditionNode("if1", nil)
	if err != nil {
		t.Fatalf("NewIfConditionNode() error = %v", err)
	}
	if node.NodeType() != "if_condition" {
		t.Errorf("NodeType() = %q, want %q", node.NodeType(), "if_condition")
	}
	tests := []struct {
		op        string
		threshold float64
		value     float64
		expect    bool
	}{
		{"gt", 0, 5, true},
		{"gt", 0, -1, false},
		{"lt", 10, 5, true},
		{"gte", 5, 5, true},
		{"lte", 5, 5, true},
		{"eq", 5, 5, true},
		{"eq", 5, 6, false},
	}
	for _, tc := range tests {
		outputs, err := node.Execute(context.Background(), map[string]any{"condition_value": tc.value}, map[string]any{"op": tc.op, "threshold": tc.threshold}, nil)
		if err != nil {
			t.Fatalf("op=%q: %v", tc.op, err)
		}
		result := outputs["true_branch"].(bool)
		if result != tc.expect {
			t.Errorf("op=%q val=%v thresh=%v: got %v, want %v", tc.op, tc.value, tc.threshold, result, tc.expect)
		}
	}
}

func TestIfConditionNode_MissingInput(t *testing.T) {
	node, _err := NewIfConditionNode("if1", nil)
	if _err != nil {
		t.Fatalf("NewIfConditionNode() error = %v", _err)
	}
	_, err := node.Execute(context.Background(), map[string]any{}, nil, nil)
	if err == nil {
		t.Error("expected error for missing 'condition_value'")
	}
}

func TestIfConditionNode_InvalidOp(t *testing.T) {
	node, _err := NewIfConditionNode("if1", nil)
	if _err != nil {
		t.Fatalf("NewIfConditionNode() error = %v", _err)
	}
	_, err := node.Execute(context.Background(), map[string]any{"condition_value": float64(1)}, map[string]any{"op": "invalid"}, nil)
	if err == nil {
		t.Error("expected error for invalid op")
	}
}
