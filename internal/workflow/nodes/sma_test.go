package nodes

import (
	"context"
	"quantflow/internal/workflow"
	"testing"
)

func TestSMANode_Execute(t *testing.T) {
	node, err := NewSMANode("sma1", map[string]any{"period": float64(3)})
	if err != nil {
		t.Fatalf("NewSMANode() error = %v", err)
	}
	if node.ID() != "sma1" {
		t.Errorf("ID() = %q, want %q", node.ID(), "sma1")
	}
	if node.NodeType() != "sma" {
		t.Errorf("NodeType() = %q, want %q", node.NodeType(), "sma")
	}
	if node.Category() != "indicator" {
		t.Errorf("Category() = %q, want %q", node.Category(), "indicator")
	}

	inputs := map[string]any{"input": []float64{1.0, 2.0, 3.0, 4.0, 5.0}}
	params := map[string]any{"period": float64(3)}

	outputs, err := node.Execute(context.Background(), inputs, params, nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	result, ok := outputs["output"].([]float64)
	if !ok {
		t.Fatalf("output is %T, want []float64", outputs["output"])
	}
	// SMA(3) of [1,2,3,4,5] -> [1, 1.5, 2, 3, 4]
	expected := []float64{1, 1.5, 2, 3, 4}
	if len(result) != len(expected) {
		t.Fatalf("len = %d, want %d", len(result), len(expected))
	}
	for i, v := range expected {
		if result[i] != v {
			t.Errorf("[%d] = %v, want %v", i, result[i], v)
		}
	}
}

func TestSMANode_PortDefinitions(t *testing.T) {
	node, err := NewSMANode("sma1", nil)
	if err != nil {
		t.Fatalf("NewSMANode() error = %v", err)
	}
	inputs := node.InputPorts()
	if len(inputs) != 1 || inputs[0].Name != "input" {
		t.Errorf("InputPorts: %+v, want 1 port named 'input'", inputs)
	}
	if inputs[0].Type != workflow.PortSeries {
		t.Errorf("input port type = %q, want %q", inputs[0].Type, workflow.PortSeries)
	}
}
