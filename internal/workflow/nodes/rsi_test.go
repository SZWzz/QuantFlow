package nodes

import (
	"context"
	"math"
	"testing"
)

func TestRSINode_Execute(t *testing.T) {
	node, err := NewRSINode("rsi1", nil)
	if err != nil {
		t.Fatalf("NewRSINode() error = %v", err)
	}
	if node.NodeType() != "rsi" {
		t.Errorf("NodeType() = %q, want %q", node.NodeType(), "rsi")
	}
	inputs := map[string]any{"prices": []float64{44, 44.34, 44.09, 43.61, 44.33, 44.83, 45.10, 45.42, 45.84, 46.08, 45.89, 46.03, 45.61, 46.28, 46.28, 46.00, 46.03, 46.41, 46.22, 46.21}}
	outputs, err := node.Execute(context.Background(), inputs, nil, nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	rsi, ok := outputs["rsi"].([]float64)
	if !ok {
		t.Fatalf("expected []float64, got %T", outputs["rsi"])
	}
	if len(rsi) != 20 {
		t.Errorf("len = %d, want 20", len(rsi))
	}
	for i := 0; i < 14; i++ {
		if !math.IsNaN(rsi[i]) {
			t.Errorf("rsi[%d] = %v, want NaN (warmup)", i, rsi[i])
		}
	}
}

func TestRSINode_MissingInput(t *testing.T) {
	node, _err := NewRSINode("rsi1", nil)
	if _err != nil {
		t.Fatalf("NewRSINode() error = %v", _err)
	}
	_, err := node.Execute(context.Background(), map[string]any{}, nil, nil)
	if err == nil {
		t.Error("expected error for missing 'prices' input")
	}
}

func TestRSINode_PeriodTooLarge(t *testing.T) {
	node, _err := NewRSINode("rsi1", nil)
	if _err != nil {
		t.Fatalf("NewRSINode() error = %v", _err)
	}
	_, err := node.Execute(context.Background(), map[string]any{"prices": []float64{1, 2}}, nil, nil)
	if err == nil {
		t.Error("expected error for period >= data length")
	}
}
