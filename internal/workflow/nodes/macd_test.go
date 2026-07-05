package nodes

import (
	"context"
	"testing"
)

func TestMACDNode_Execute(t *testing.T) {
	node, err := NewMACDNode("macd1", nil)
	if err != nil {
		t.Fatalf("NewMACDNode() error = %v", err)
	}
	if node.NodeType() != "macd" {
		t.Errorf("NodeType() = %q, want %q", node.NodeType(), "macd")
	}
	inputs := map[string]any{"prices": []float64{10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27}}
	outputs, err := node.Execute(context.Background(), inputs, nil, nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	for _, key := range []string{"macd_line", "signal_line", "histogram"} {
		val, ok := outputs[key].([]float64)
		if !ok {
			t.Fatalf("expected []float64 for %q, got %T", key, outputs[key])
		}
		if len(val) == 0 {
			t.Errorf("%q is empty", key)
		}
	}
}

func TestMACDNode_MissingInput(t *testing.T) {
	node, _err := NewMACDNode("macd1", nil)
	if _err != nil {
		t.Fatalf("NewMACDNode() error = %v", _err)
	}
	_, err := node.Execute(context.Background(), map[string]any{}, nil, nil)
	if err == nil {
		t.Error("expected error for missing 'prices' input")
	}
}
