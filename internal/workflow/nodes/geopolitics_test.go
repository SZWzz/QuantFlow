package nodes

import (
	"context"
	"testing"
)

func TestGeopoliticsNode_MockData(t *testing.T) {
	node, err := NewGeopoliticsNode("geo1", nil)
	if err != nil {
		t.Fatalf("NewGeopoliticsNode() error = %v", err)
	}
	if node.NodeType() != "geopolitics" {
		t.Errorf("NodeType() = %q, want %q", node.NodeType(), "geopolitics")
	}
	outputs, err := node.Execute(context.Background(), map[string]any{}, nil, nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	riskSignal, ok := outputs["risk_signal"].(map[string]any)
	if !ok {
		t.Fatalf("expected map risk_signal, got %T", outputs["risk_signal"])
	}
	if _, ok := riskSignal["action"]; !ok {
		t.Error("risk_signal missing 'action'")
	}
	riskScore, ok := outputs["risk_score"].(float64)
	if !ok || riskScore < 0 {
		t.Errorf("risk_score = %v, want >= 0", riskScore)
	}
}

func TestGeopoliticsNode_WithTopic(t *testing.T) {
	node, _err := NewGeopoliticsNode("geo1", nil)
	if _err != nil {
		t.Fatalf("NewGeopoliticsNode() error = %v", _err)
	}
	outputs, err := node.Execute(context.Background(), map[string]any{"topic": "taiwan-strait"}, nil, nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if _, ok := outputs["risk_signal"]; !ok {
		t.Error("expected risk_signal in output")
	}
}
