package nodes

import (
	"testing"

	"quantflow/internal/workflow"
)

func TestTrainModelNode_Registration(t *testing.T) {
	r := workflow.NewRegistry()
	r.RegisterWithCategory("train_model", NewTrainModelNode, "ml")

	node, err := r.Create("train_model", "tm-1", map[string]any{
		"model_type":       "xgboost",
		"target_type":      "regression",
		"forecast_horizon": "5",
	})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if node.Category() != "ml" {
		t.Errorf("expected category 'ml', got '%s'", node.Category())
	}
}

func TestTrainModelNode_ParamValidation(t *testing.T) {
	_, err := NewTrainModelNode("tm-1", map[string]any{
		"model_type": "unsupported_model",
	})
	if err == nil {
		t.Error("expected validation error for unsupported model_type")
	}
}

func TestTrainModelNode_ValidModelTypes(t *testing.T) {
	validTypes := []string{"xgboost", "lightgbm", "lstm", "transformer", "ppo", "dqn", "sac", "garch"}
	for _, vt := range validTypes {
		_, err := NewTrainModelNode("tm-1", map[string]any{
			"model_type": vt,
		})
		if err != nil {
			t.Errorf("unexpected error for valid model_type '%s': %v", vt, err)
		}
	}
}
