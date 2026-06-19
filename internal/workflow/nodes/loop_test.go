package nodes

import (
	"context"
	"testing"
)

func TestLoopNode_StringArray(t *testing.T) {
	node, _ := NewLoopNode("loop1", nil)
	outputs, err := node.Execute(context.Background(), map[string]any{"items": []string{"a", "b", "c"}}, nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	batched, ok := outputs["batched"].([]any)
	if !ok {
		t.Fatalf("batched is %T", outputs["batched"])
	}
	if len(batched) != 3 {
		t.Fatalf("len = %d, want 3", len(batched))
	}
}

func TestLoopNode_AnyArray(t *testing.T) {
	node, _ := NewLoopNode("loop2", nil)
	outputs, err := node.Execute(context.Background(), map[string]any{"items": []any{1, "two", 3.0}}, nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	batched, ok := outputs["batched"].([]any)
	if !ok {
		t.Fatalf("batched is %T", outputs["batched"])
	}
	if len(batched) != 3 {
		t.Fatalf("len = %d, want 3", len(batched))
	}
}
