package nodes

import (
	"context"
	"testing"
	"time"
)

func TestWaitNode_Execute(t *testing.T) {
	node, err := NewWaitNode("w1", nil)
	if err != nil {
		t.Fatalf("NewWaitNode() error = %v", err)
	}
	if node.NodeType() != "wait" {
		t.Errorf("NodeType() = %q, want %q", node.NodeType(), "wait")
	}
	start := time.Now()
	outputs, err := node.Execute(context.Background(), map[string]any{}, map[string]any{"duration_sec": float64(0.01)}, nil)
	duration := time.Since(start)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if duration < 5*time.Millisecond {
		t.Errorf("wait was too short: %v", duration)
	}
	if outputs != nil {
		t.Errorf("expected nil output, got %v", outputs)
	}
}

func TestWaitNode_Zero(t *testing.T) {
	node, _err := NewWaitNode("w1", nil)
	if _err != nil {
		t.Fatalf("NewWaitNode() error = %v", _err)
	}
	outputs, err := node.Execute(context.Background(), map[string]any{}, map[string]any{"duration_sec": float64(-1)}, nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if outputs != nil {
		t.Errorf("expected nil output, got %v", outputs)
	}
}

func TestWaitNode_CancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	node, _err := NewWaitNode("w1", nil)
	if _err != nil {
		t.Fatalf("NewWaitNode() error = %v", _err)
	}
	_, err := node.Execute(ctx, map[string]any{}, map[string]any{"duration_sec": float64(10)}, nil)
	if err == nil {
		t.Error("expected error from cancelled context")
	}
}
