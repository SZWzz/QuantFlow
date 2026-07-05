package nodes

import (
	"context"
	"testing"
)

func TestRLEnvNode_Execute(t *testing.T) {
	node, err := NewRLEnvNode("rle1", nil)
	if err != nil {
		t.Fatalf("NewRLEnvNode() error = %v", err)
	}
	if node.NodeType() != "rl_env" {
		t.Errorf("NodeType() = %q, want %q", node.NodeType(), "rl_env")
	}
	outputs, err := node.Execute(context.Background(), map[string]any{}, map[string]any{"window_size": float64(20), "action_type": "discrete", "initial_cash": float64(10000)}, nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	config, ok := outputs["env_config"].(map[string]any)
	if !ok {
		t.Fatalf("expected map, got %T", outputs["env_config"])
	}
	if config["window_size"] != 20 {
		t.Errorf("window_size = %v, want 20", config["window_size"])
	}
	if config["action_type"] != "discrete" {
		t.Errorf("action_type = %v, want 'discrete'", config["action_type"])
	}
	if config["initial_cash"] != 10000 {
		t.Errorf("initial_cash = %v, want 10000", config["initial_cash"])
	}
}

func TestRLEnvNode_InvalidActionType(t *testing.T) {
	_, err := NewRLEnvNode("rle1", map[string]any{"action_type": "invalid"})
	if err == nil {
		t.Error("expected error for invalid action_type")
	}
}
