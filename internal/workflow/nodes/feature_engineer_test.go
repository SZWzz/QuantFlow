package nodes

import (
	"context"
	"testing"

	"quantflow/internal/workflow"
)

func TestFeatureEngineerNode_Registration(t *testing.T) {
	r := workflow.NewRegistry()
	r.RegisterWithCategory("feature_engineer", NewFeatureEngineerNode, "ml")

	node, err := r.Create("feature_engineer", "fe-1", map[string]any{
		"method":      "standardize",
		"fill_na":     "zero",
		"lag_periods": "1",
	})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if node.Category() != "ml" {
		t.Errorf("expected category 'ml', got '%s'", node.Category())
	}
}

func TestFeatureEngineerNode_Execute(t *testing.T) {
	node, _ := NewFeatureEngineerNode("fe-1", map[string]any{
		"method": "standardize",
	})
	inputs := map[string]any{
		"ohlcv_data": map[string][]float64{
			"close": {100, 101, 102, 103, 104},
		},
		"factors": map[string][]float64{
			"momentum_1m": {0.01, 0.02, 0.015, 0.03, 0.025},
		},
	}
	outputs, err := node.Execute(context.Background(), inputs, map[string]any{})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	matrix := outputs["feature_matrix"]
	if matrix == nil {
		t.Fatal("expected feature_matrix output")
	}
}

func TestFeatureEngineerNode_Validate(t *testing.T) {
	// Valid method
	_, err := NewFeatureEngineerNode("fe-1", map[string]any{"method": "minmax"})
	if err != nil {
		t.Errorf("unexpected error for valid method: %v", err)
	}

	// Invalid method
	_, err = NewFeatureEngineerNode("fe-2", map[string]any{"method": "invalid"})
	if err == nil {
		t.Error("expected validation error for invalid method")
	}
}
