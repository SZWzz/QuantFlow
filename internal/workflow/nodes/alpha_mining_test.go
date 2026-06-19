package nodes

import (
	"testing"

	"quantflow/internal/workflow"
)

func TestAlphaMiningNode_Registration(t *testing.T) {
	r := workflow.NewRegistry()
	r.RegisterWithCategory("alpha_mining", NewAlphaMiningNode, "ml")

	node, err := r.Create("alpha_mining", "am-1", map[string]any{
		"population_size": "50",
		"generations":     "10",
		"top_k":           "5",
	})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if node.Category() != "ml" {
		t.Errorf("expected category 'ml', got '%s'", node.Category())
	}
}

func TestAlphaMiningNode_Ports(t *testing.T) {
	node, err := NewAlphaMiningNode("am-1", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	inputs := node.InputPorts()
	if len(inputs) != 2 {
		t.Errorf("expected 2 input ports, got %d", len(inputs))
	}
	if inputs[0].Name != "factor_pool" {
		t.Errorf("expected first input port 'factor_pool', got '%s'", inputs[0].Name)
	}
	if inputs[1].Name != "ohlcv_data" {
		t.Errorf("expected second input port 'ohlcv_data', got '%s'", inputs[1].Name)
	}

	outputs := node.OutputPorts()
	if len(outputs) != 2 {
		t.Errorf("expected 2 output ports, got %d", len(outputs))
	}
	if outputs[0].Name != "new_factors" {
		t.Errorf("expected first output port 'new_factors', got '%s'", outputs[0].Name)
	}
	if outputs[1].Name != "factor_scores" {
		t.Errorf("expected second output port 'factor_scores', got '%s'", outputs[1].Name)
	}
}

func TestAlphaMiningNode_ParamSchema(t *testing.T) {
	node, err := NewAlphaMiningNode("am-1", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	schema := node.ParamSchema()
	if len(schema) != 6 {
		t.Errorf("expected 6 params, got %d", len(schema))
	}

	paramNames := make(map[string]bool)
	for _, p := range schema {
		paramNames[p.Name] = true
	}
	expectedParams := []string{"population_size", "generations", "top_k", "crossover_rate", "mutation_rate", "fitness_metric"}
	for _, name := range expectedParams {
		if !paramNames[name] {
			t.Errorf("missing param: %s", name)
		}
	}
}

func TestAlphaMiningNode_Validate(t *testing.T) {
	// AlphaMiningNode's Validate always passes (no constrained params)
	node, err := NewAlphaMiningNode("am-1", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := node.Validate(); err != nil {
		t.Errorf("unexpected validation error: %v", err)
	}
}
