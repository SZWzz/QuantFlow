package nodes

import (
	"context"
	"testing"
)

func TestIndicatorDMINode_NoBridge(t *testing.T) {
	node, err := NewIndicatorDMINode("dmi1", nil)
	if err != nil {
		t.Fatalf("NewIndicatorDMINode() error = %v", err)
	}
	if node.NodeType() != "indicator_dmi" {
		t.Errorf("NodeType() = %q, want %q", node.NodeType(), "indicator_dmi")
	}
	outputs, err := node.Execute(context.Background(), map[string]any{"ohlcv": []float64{1, 2, 3}}, nil, nil)
	if err == nil {
		t.Fatal("expected error for missing python bridge")
	}
	for _, key := range []string{"pdi", "mdi", "adx", "adxr"} {
		val, ok := outputs[key].([]float64)
		if !ok || len(val) != 0 {
			t.Errorf("expected empty []float64 for %q", key)
		}
	}
}
