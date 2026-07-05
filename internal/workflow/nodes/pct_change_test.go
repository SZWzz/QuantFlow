package nodes

import (
	"context"
	"testing"
)

func TestPctChangeNode_Execute(t *testing.T) {
	node, err := NewPctChangeNode("pc1", nil)
	if err != nil {
		t.Fatalf("NewPctChangeNode() error = %v", err)
	}
	if node.NodeType() != "pct_change" {
		t.Errorf("NodeType() = %q, want %q", node.NodeType(), "pct_change")
	}
	outputs, err := node.Execute(context.Background(), map[string]any{"values": []float64{100, 110, 121}}, map[string]any{"period": float64(1)}, nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	result := outputs["result"].([]float64)
	if len(result) != 3 {
		t.Fatalf("len = %d, want 3", len(result))
	}
	if result[0] != 0 {
		t.Errorf("result[0] = %v, want 0", result[0])
	}
	if result[1] != 0.1 {
		t.Errorf("result[1] = %v, want 0.1", result[1])
	}
}

func TestPctChangeNode_MissingInput(t *testing.T) {
	node, _err := NewPctChangeNode("pc1", nil)
	if _err != nil {
		t.Fatalf("NewPctChangeNode() error = %v", _err)
	}
	_, err := node.Execute(context.Background(), map[string]any{}, nil, nil)
	if err == nil {
		t.Error("expected error for missing 'values' input")
	}
}
