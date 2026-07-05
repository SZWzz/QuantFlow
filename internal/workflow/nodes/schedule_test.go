package nodes

import (
	"context"
	"testing"
)

func TestScheduleNode_Execute(t *testing.T) {
	node, err := NewScheduleNode("sched1", nil)
	if err != nil {
		t.Fatalf("NewScheduleNode() error = %v", err)
	}
	if node.NodeType() != "schedule" {
		t.Errorf("NodeType() = %q, want %q", node.NodeType(), "schedule")
	}
	outputs, err := node.Execute(context.Background(), map[string]any{}, map[string]any{"workflow_id": "wf1", "cron_expr": "0 9 * * 1-5"}, nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	taskID, ok := outputs["task_id"].(string)
	if !ok {
		t.Fatalf("expected string task_id, got %T", outputs["task_id"])
	}
	if taskID == "" {
		t.Error("task_id should not be empty")
	}
}

func TestScheduleNode_MissingWorkflowID(t *testing.T) {
	node, _err := NewScheduleNode("sched1", nil)
	if _err != nil {
		t.Fatalf("NewScheduleNode() error = %v", _err)
	}
	_, err := node.Execute(context.Background(), map[string]any{}, map[string]any{"cron_expr": "0 9 * * 1-5"}, nil)
	if err == nil {
		t.Error("expected error for missing workflow_id")
	}
}
