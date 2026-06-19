package nodes

import (
	"context"
	"testing"

	"quantflow/internal/workflow"
)

func TestEvaluateModelNode_Registration(t *testing.T) {
	r := workflow.NewRegistry()
	r.RegisterWithCategory("evaluate_model", NewEvaluateModelNode, "ml")

	node, err := r.Create("evaluate_model", "eval-1", nil)
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if node.Category() != "ml" {
		t.Errorf("expected category 'ml', got '%s'", node.Category())
	}
}

func TestEvaluateModelNode_Execute(t *testing.T) {
	node, _ := NewEvaluateModelNode("eval-1", nil)

	inputs := map[string]any{
		"model_id": "model-123",
		"predictions": map[string][]float64{
			"value": {1.0, 2.0, 3.0, 4.0, 5.0},
		},
		"actual": map[string][]float64{
			"value": {1.1, 1.9, 3.2, 3.8, 5.1},
		},
	}
	outputs, err := node.Execute(context.Background(), inputs, map[string]any{})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	report := outputs["evaluation_report"]
	if report == nil {
		t.Fatal("expected evaluation_report output")
	}

	metrics, ok := report.(map[string][]float64)
	if !ok {
		t.Fatalf("expected map[string][]float64, got %T", report)
	}

	// Verify all expected metrics are present
	for _, key := range []string{"mse", "mae", "rmse", "ic"} {
		if vals, exists := metrics[key]; !exists || len(vals) != 1 {
			t.Errorf("expected metric '%s' with 1 value, got %v", key, vals)
		}
	}

	// Verify MSE is approximately correct: ((0.1)^2+(0.1)^2+(-0.2)^2+(0.2)^2+(-0.1)^2)/5 = 0.022
	if metrics["mse"][0] < 0.01 || metrics["mse"][0] > 0.04 {
		t.Errorf("unexpected MSE: %f", metrics["mse"][0])
	}
}

func TestEvaluateModelNode_LengthMismatch(t *testing.T) {
	node, _ := NewEvaluateModelNode("eval-1", nil)

	inputs := map[string]any{
		"model_id": "model-123",
		"predictions": map[string][]float64{
			"value": {1.0, 2.0, 3.0},
		},
		"actual": map[string][]float64{
			"value": {1.0, 2.0},
		},
	}
	_, err := node.Execute(context.Background(), inputs, map[string]any{})
	if err == nil {
		t.Error("expected error for length mismatch")
	}
}

func TestEvaluateModelNode_Validate(t *testing.T) {
	node, err := NewEvaluateModelNode("eval-1", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := node.Validate(); err != nil {
		t.Errorf("unexpected validation error: %v", err)
	}
}
