package nodes

import (
	"context"
	"testing"
)

func TestRLTrainNode_Execute(t *testing.T) {
	node, err := NewRLTrainNode("rlt1", nil)
	if err != nil {
		t.Fatalf("NewRLTrainNode() error = %v", err)
	}
	if node.NodeType() != "rl_train" {
		t.Errorf("NodeType() = %q, want %q", node.NodeType(), "rl_train")
	}
	outputs, err := node.Execute(context.Background(), map[string]any{}, map[string]any{"algorithm": "ppo", "total_episodes": float64(100)}, nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	modelID, ok := outputs["model_id"].(string)
	if !ok || modelID == "" {
		t.Errorf("expected non-empty model_id, got %v", modelID)
	}
	reward, ok := outputs["reward_curve"].([]float64)
	if !ok || len(reward) != 0 {
		t.Errorf("expected empty reward_curve, got len=%d", len(reward))
	}
}

func TestRLTrainNode_InvalidAlgorithm(t *testing.T) {
	_, err := NewRLTrainNode("rlt1", map[string]any{"algorithm": "invalid"})
	if err == nil {
		t.Error("expected error for invalid algorithm")
	}
}
