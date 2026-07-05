package nodes

import (
	"context"
	"testing"
)

func TestEMANode_Execute(t *testing.T) {
	node, err := NewEMANode("ema1", nil)
	if err != nil {
		t.Fatalf("NewEMANode() error = %v", err)
	}
	if node.ID() != "ema1" {
		t.Errorf("ID() = %q, want %q", node.ID(), "ema1")
	}
	if node.NodeType() != "ema" {
		t.Errorf("NodeType() = %q, want %q", node.NodeType(), "ema")
	}
	inputs := map[string]any{"prices": []float64{1, 2, 3, 4, 5}}
	params := map[string]any{"period": float64(3)}
	outputs, err := node.Execute(context.Background(), inputs, params, nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	ema, ok := outputs["ema"].([]float64)
	if !ok {
		t.Fatalf("expected []float64, got %T", outputs["ema"])
	}
	if len(ema) != 5 {
		t.Errorf("len = %d, want 5", len(ema))
	}
}

func TestEMANode_MissingInput(t *testing.T) {
	node, _err := NewEMANode("ema1", nil)
	if _err != nil {
		t.Fatalf("NewEMANode() error = %v", _err)
	}
	_, err := node.Execute(context.Background(), map[string]any{}, nil, nil)
	if err == nil {
		t.Error("expected error for missing 'prices' input")
	}
}
