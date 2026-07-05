package nodes

import (
	"context"
	"testing"
)

func TestIndicatorBIASSignalNode_NoBridge(t *testing.T) {
	node, err := NewIndicatorBIASSignalNode("bs1", nil)
	if err != nil {
		t.Fatalf("NewIndicatorBIASSignalNode() error = %v", err)
	}
	if node.NodeType() != "indicator_bias_signal" {
		t.Errorf("NodeType() = %q, want %q", node.NodeType(), "indicator_bias_signal")
	}
	outputs, err := node.Execute(context.Background(), map[string]any{"prices": []float64{1, 2, 3}}, nil, nil)
	if err == nil {
		t.Fatal("expected error for missing python bridge")
	}
	for _, key := range []string{"bias", "signal_short", "signal_long"} {
		val, ok := outputs[key].([]float64)
		if !ok || len(val) != 0 {
			t.Errorf("expected empty []float64 for %q", key)
		}
	}
}
