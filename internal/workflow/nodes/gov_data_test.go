package nodes

import (
	"context"
	"testing"
)

func TestGovDataNode_MockData(t *testing.T) {
	node, err := NewGovDataNode("gd1", nil)
	if err != nil {
		t.Fatalf("NewGovDataNode() error = %v", err)
	}
	if node.NodeType() != "gov_data" {
		t.Errorf("NodeType() = %q, want %q", node.NodeType(), "gov_data")
	}
	outputs, err := node.Execute(context.Background(), map[string]any{}, nil, nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	macroSignal, ok := outputs["macro_signal"].(map[string]any)
	if !ok {
		t.Fatalf("expected map macro_signal, got %T", outputs["macro_signal"])
	}
	action, ok := macroSignal["action"].(string)
	if !ok || action == "" {
		t.Errorf("expected non-empty action, got %q", action)
	}
	if _, ok := outputs["latest_value"].(float64); !ok {
		t.Errorf("expected float64 latest_value")
	}
}

func TestGovDataNode_WithIndicator(t *testing.T) {
	node, _err := NewGovDataNode("gd1", nil)
	if _err != nil {
		t.Fatalf("NewGovDataNode() error = %v", _err)
	}
	outputs, err := node.Execute(context.Background(), map[string]any{"indicator": "GDP"}, nil, nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if _, ok := outputs["macro_signal"]; !ok {
		t.Error("expected macro_signal in output")
	}
}
