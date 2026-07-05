package nodes

import (
	"context"
	"testing"
)

func TestAgentNode_MissingProfileManager(t *testing.T) {
	node, err := NewAgentNode("ag1", nil)
	if err != nil {
		t.Fatalf("NewAgentNode() error = %v", err)
	}
	if node.NodeType() != "agent" {
		t.Errorf("NodeType() = %q, want %q", node.NodeType(), "agent")
	}
	_, err = node.Execute(context.Background(), map[string]any{"prompt": "analyze"}, nil, nil)
	if err == nil {
		t.Error("expected error for missing profile manager")
	}
}

func TestAgentNode_InvalidProfile(t *testing.T) {
	node, err := NewAgentNode("ag1", map[string]any{"profile": "nonexistent"})
	if err != nil {
		t.Fatalf("NewAgentNode() error = %v", err)
	}
	err = node.Validate()
	if err == nil {
		t.Error("expected error for invalid profile")
	}
}
