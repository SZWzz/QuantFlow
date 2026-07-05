package nodes

import (
	"context"
	"testing"
)

func TestDeltaNode_Execute(t *testing.T) {
	node, err := NewDeltaNode("d1", nil)
	if err != nil {
		t.Fatalf("NewDeltaNode() error = %v", err)
	}
	if node.NodeType() != "delta" {
		t.Errorf("NodeType() = %q, want %q", node.NodeType(), "delta")
	}
	outputs, err := node.Execute(context.Background(), map[string]any{"values": []float64{10, 12, 15, 20}}, map[string]any{"period": float64(1)}, nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	result := outputs["result"].([]float64)
	if len(result) != 4 {
		t.Fatalf("len = %d, want 4", len(result))
	}
	if result[0] != 0 || result[1] != 2 || result[2] != 3 || result[3] != 5 {
		t.Errorf("unexpected result: %v", result)
	}
}

func TestDeltaNode_MissingInput(t *testing.T) {
	node, _err := NewDeltaNode("d1", nil)
	if _err != nil {
		t.Fatalf("NewDeltaNode() error = %v", _err)
	}
	_, err := node.Execute(context.Background(), map[string]any{}, nil, nil)
	if err == nil {
		t.Error("expected error for missing 'values' input")
	}
}

func TestDeltaNode_InvalidPeriod(t *testing.T) {
	node, _err := NewDeltaNode("d1", nil)
	if _err != nil {
		t.Fatalf("NewDeltaNode() error = %v", _err)
	}
	_, err := node.Execute(context.Background(), map[string]any{"values": []float64{1, 2, 3}}, map[string]any{"period": float64(5)}, nil)
	if err == nil {
		t.Error("expected error for period >= len(values)")
	}
}
