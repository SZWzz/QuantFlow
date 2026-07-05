package nodes

import (
	"context"
	"testing"
)

func TestNotifyNode_Execute(t *testing.T) {
	node, err := NewNotifyNode("n1", nil)
	if err != nil {
		t.Fatalf("NewNotifyNode() error = %v", err)
	}
	if node.NodeType() != "notify" {
		t.Errorf("NodeType() = %q, want %q", node.NodeType(), "notify")
	}
	outputs, err := node.Execute(context.Background(), map[string]any{}, map[string]any{"title": "Test", "body": "Hello"}, nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	success, ok := outputs["success"].(bool)
	if !ok || !success {
		t.Errorf("success = %v, want true", success)
	}
}

func TestNotifyNode_MissingTitle(t *testing.T) {
	node, _err := NewNotifyNode("n1", nil)
	if _err != nil {
		t.Fatalf("NewNotifyNode() error = %v", _err)
	}
	_, err := node.Execute(context.Background(), map[string]any{}, map[string]any{}, nil)
	if err == nil {
		t.Error("expected error for missing title")
	}
}

func TestNotifyNode_TitleFromInput(t *testing.T) {
	node, _err := NewNotifyNode("n1", nil)
	if _err != nil {
		t.Fatalf("NewNotifyNode() error = %v", _err)
	}
	outputs, err := node.Execute(context.Background(), map[string]any{"message": "input msg"}, map[string]any{}, nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if outputs["success"] != true {
		t.Error("expected success")
	}
}
