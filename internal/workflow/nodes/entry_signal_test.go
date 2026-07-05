package nodes

import (
	"context"
	"testing"
)

func TestEntrySignalNode_Execute(t *testing.T) {
	node, err := NewEntrySignalNode("es1", nil)
	if err != nil {
		t.Fatalf("NewEntrySignalNode() error = %v", err)
	}
	if node.NodeType() != "entry_signal" {
		t.Errorf("NodeType() = %q, want %q", node.NodeType(), "entry_signal")
	}
	outputs, err := node.Execute(context.Background(), map[string]any{"condition": []float64{0, 1, 1, 0, 1}}, nil, nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	entry := outputs["entry"].([]float64)
	expected := []float64{0, 1, 0, 0, 1}
	for i, v := range expected {
		if entry[i] != v {
			t.Errorf("[%d] = %v, want %v", i, entry[i], v)
		}
	}
}

func TestEntrySignalNode_MissingInput(t *testing.T) {
	node, _err := NewEntrySignalNode("es1", nil)
	if _err != nil {
		t.Fatalf("NewEntrySignalNode() error = %v", _err)
	}
	_, err := node.Execute(context.Background(), map[string]any{}, nil, nil)
	if err == nil {
		t.Error("expected error for missing 'condition'")
	}
}
