package nodes

import (
	"context"
	"testing"

	"quantflow/internal/workflow"
)

func TestSubWorkflowNode_MissingRunner(t *testing.T) {
	node, err := NewSubWorkflowNode("sw1", nil)
	if err != nil {
		t.Fatalf("NewSubWorkflowNode() error = %v", err)
	}
	if node.NodeType() != "sub_workflow" {
		t.Errorf("NodeType() = %q, want %q", node.NodeType(), "sub_workflow")
	}
	_, err = node.Execute(context.Background(), map[string]any{}, map[string]any{"workflow_id": "wf1"}, nil)
	if err == nil {
		t.Error("expected error when SubWorkflowRunner is nil")
	}
}

func TestSubWorkflowNode_MissingWorkflowID(t *testing.T) {
	node, _err := NewSubWorkflowNode("sw1", nil)
	if _err != nil {
		t.Fatalf("NewSubWorkflowNode() error = %v", _err)
	}
	_, err := node.Execute(context.Background(), map[string]any{}, map[string]any{}, nil)
	if err == nil {
		t.Error("expected error for missing workflow_id")
	}
}

func TestSubWorkflowNode_WithRunner(t *testing.T) {
	node, err := NewSubWorkflowNode("sw1", nil)
	if err != nil {
		t.Fatalf("NewSubWorkflowNode() error = %v", err)
	}
	nctx := &workflow.NodeContext{
		SubWorkflowRunner: func(ctx context.Context, workflowID string, inputs map[string]any) (map[string]any, error) {
			return map[string]any{"result": "ok"}, nil
		},
	}
	outputs, err := node.Execute(context.Background(), map[string]any{}, map[string]any{"workflow_id": "wf1"}, nctx)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if outputs["status"] != "completed" {
		t.Errorf("status = %v, want 'completed'", outputs["status"])
	}
	if outputs["result"] != "ok" {
		t.Errorf("result = %v, want 'ok'", outputs["result"])
	}
}
