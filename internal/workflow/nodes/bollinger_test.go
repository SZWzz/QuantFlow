package nodes

import (
	"context"
	"math"
	"testing"
)

func TestBollingerNode_Execute(t *testing.T) {
	node, err := NewBollingerNode("boll1", nil)
	if err != nil {
		t.Fatalf("NewBollingerNode() error = %v", err)
	}
	if node.NodeType() != "bollinger" {
		t.Errorf("NodeType() = %q, want %q", node.NodeType(), "bollinger")
	}
	prices := []float64{10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30}
	inputs := map[string]any{"prices": prices}
	params := map[string]any{"period": float64(5), "multiplier": float64(2)}
	outputs, err := node.Execute(context.Background(), inputs, params, nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	for _, key := range []string{"upper", "middle", "lower"} {
		val, ok := outputs[key].([]float64)
		if !ok {
			t.Fatalf("expected []float64 for %q, got %T", key, outputs[key])
		}
		if len(val) != len(prices) {
			t.Errorf("%q len = %d, want %d", key, len(val), len(prices))
		}
	}
	upper := outputs["upper"].([]float64)
	middle := outputs["middle"].([]float64)
	lower := outputs["lower"].([]float64)
	for i := 0; i < 4; i++ {
		if !math.IsNaN(upper[i]) {
			t.Errorf("upper[%d] = %v, want NaN (warmup)", i, upper[i])
		}
	}
	if middle[len(middle)-1] <= 0 {
		t.Errorf("expected positive middle value, got %v", middle[len(middle)-1])
	}
	if upper[len(upper)-1] <= middle[len(middle)-1] {
		t.Errorf("upper %v should be > middle %v", upper[len(upper)-1], middle[len(middle)-1])
	}
	if lower[len(lower)-1] >= middle[len(middle)-1] {
		t.Errorf("lower %v should be < middle %v", lower[len(lower)-1], middle[len(middle)-1])
	}
}

func TestBollingerNode_MissingInput(t *testing.T) {
	node, err := NewBollingerNode("boll1", nil)
	if err != nil {
		t.Fatalf("NewBollingerNode() error = %v", err)
	}
	_, err = node.Execute(context.Background(), map[string]any{}, nil, nil)
	if err == nil {
		t.Error("expected error for missing 'prices' input")
	}
}
