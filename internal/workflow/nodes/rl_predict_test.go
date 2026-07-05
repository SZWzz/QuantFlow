package nodes

import (
	"context"
	"testing"
)

func TestRLPredictNode_Execute(t *testing.T) {
	node, err := NewRLPredictNode("rlp1", nil)
	if err != nil {
		t.Fatalf("NewRLPredictNode() error = %v", err)
	}
	if node.NodeType() != "rl_predict" {
		t.Errorf("NodeType() = %q, want %q", node.NodeType(), "rl_predict")
	}
	outputs, err := node.Execute(context.Background(), map[string]any{
		"model_id":    "rl_ppo_100",
		"observation": []float64{1, 2, 3, 4, 5},
	}, nil, nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	action, ok := outputs["action"].(int)
	if !ok || action != 1 {
		t.Errorf("action = %v, want 1 (hold)", action)
	}
	if outputs["action_value"].(float64) != 0 {
		t.Errorf("action_value = %v, want 0", outputs["action_value"])
	}
}
