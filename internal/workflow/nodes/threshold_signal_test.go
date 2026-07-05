package nodes

import (
	"context"
	"testing"
)

func TestThresholdSignalNode_Execute(t *testing.T) {
	node, err := NewThresholdSignalNode("ts1", nil)
	if err != nil {
		t.Fatalf("NewThresholdSignalNode() error = %v", err)
	}
	if node.NodeType() != "threshold_signal" {
		t.Errorf("NodeType() = %q, want %q", node.NodeType(), "threshold_signal")
	}
	outputs, err := node.Execute(context.Background(), map[string]any{"values": []float64{10, 50, 90}}, map[string]any{"upper": float64(80), "lower": float64(20)}, nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	signals := outputs["signal"].([]float64)
	expected := []float64{1, 0, -1}
	for i, v := range expected {
		if signals[i] != v {
			t.Errorf("[%d] = %v, want %v", i, signals[i], v)
		}
	}
}

func TestThresholdSignalNode_MissingInput(t *testing.T) {
	node, _err := NewThresholdSignalNode("ts1", nil)
	if _err != nil {
		t.Fatalf("NewThresholdSignalNode() error = %v", _err)
	}
	_, err := node.Execute(context.Background(), map[string]any{}, nil, nil)
	if err == nil {
		t.Error("expected error for missing 'values' input")
	}
}
