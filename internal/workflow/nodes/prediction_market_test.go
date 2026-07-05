package nodes

import (
	"context"
	"testing"
)

func TestPredictionMarketNode_MockData(t *testing.T) {
	node, err := NewPredictionMarketNode("pm1", nil)
	if err != nil {
		t.Fatalf("NewPredictionMarketNode() error = %v", err)
	}
	if node.NodeType() != "prediction_market" {
		t.Errorf("NodeType() = %q, want %q", node.NodeType(), "prediction_market")
	}
	outputs, err := node.Execute(context.Background(), map[string]any{}, nil, nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	signal, ok := outputs["signal"].(map[string]any)
	if !ok {
		t.Fatalf("expected map signal, got %T", outputs["signal"])
	}
	if _, ok := signal["action"]; !ok {
		t.Error("signal missing 'action'")
	}
	if _, ok := outputs["signal_summary"]; !ok {
		t.Error("expected signal_summary")
	}
}

func TestPredictionMarketNode_WithCategory(t *testing.T) {
	node, _err := NewPredictionMarketNode("pm1", nil)
	if _err != nil {
		t.Fatalf("NewPredictionMarketNode() error = %v", _err)
	}
	outputs, err := node.Execute(context.Background(), map[string]any{"category": "elections"}, nil, nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if _, ok := outputs["signal"]; !ok {
		t.Error("expected signal in output")
	}
}
