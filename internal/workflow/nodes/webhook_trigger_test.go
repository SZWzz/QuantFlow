package nodes

import (
	"context"
	"testing"
)

func TestWebhookTriggerNode_Execute(t *testing.T) {
	node, err := NewWebhookTriggerNode("wh1", nil)
	if err != nil {
		t.Fatalf("NewWebhookTriggerNode() error = %v", err)
	}
	if node.NodeType() != "webhook_trigger" {
		t.Errorf("NodeType() = %q, want %q", node.NodeType(), "webhook_trigger")
	}
	outputs, err := node.Execute(context.Background(), map[string]any{}, map[string]any{"path": "/my-webhook"}, nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if outputs["payload"] != nil {
		t.Errorf("expected nil payload, got %v", outputs["payload"])
	}
	status, ok := outputs["status"].(string)
	if !ok || status == "" {
		t.Errorf("expected non-empty status, got %q", status)
	}
}
