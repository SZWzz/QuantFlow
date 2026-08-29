package nodes

import (
	"quantflow/internal/workflow"
	"testing"
)

func TestPredictNode_Registration(t *testing.T) {
	r := workflow.NewRegistry()
	r.RegisterWithCategory("predict", NewPredictNode, "ml")

	node, err := r.Create("predict", "pred-1", nil)
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if node.Category() != "ml" {
		t.Errorf("expected category 'ml', got '%s'", node.Category())
	}
}

func TestPredictNode_Ports(t *testing.T) {
	node, _ := NewPredictNode("pred-1", nil)

	inputs := node.InputPorts()
	if len(inputs) != 2 {
		t.Errorf("expected 2 input ports, got %d", len(inputs))
	}

	outputs := node.OutputPorts()
	if len(outputs) != 1 {
		t.Errorf("expected 1 output port, got %d", len(outputs))
	}
	if outputs[0].Name != "predictions" {
		t.Errorf("expected output port 'predictions', got '%s'", outputs[0].Name)
	}
}

func TestPredictNode_Validate(t *testing.T) {
	// PredictNode has no params, so Validate always passes
	node, err := NewPredictNode("pred-1", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := node.Validate(); err != nil {
		t.Errorf("unexpected validation error: %v", err)
	}
}
