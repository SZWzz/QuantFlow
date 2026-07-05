package nodes

import (
	"context"
	"testing"
)

func TestAlertNode_Triggered(t *testing.T) {
	node, err := NewAlertNode("a1", nil)
	if err != nil {
		t.Fatalf("NewAlertNode() error = %v", err)
	}
	if node.NodeType() != "alert" {
		t.Errorf("NodeType() = %q, want %q", node.NodeType(), "alert")
	}
	outputs, err := node.Execute(context.Background(), map[string]any{"value": float64(100)}, map[string]any{"condition": "gt", "threshold": float64(50)}, nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if outputs["triggered"] != true {
		t.Error("expected triggered=true")
	}
	if outputs["value"].(float64) != 100 {
		t.Errorf("value = %v, want 100", outputs["value"])
	}
}

func TestAlertNode_NotTriggered(t *testing.T) {
	node, err := NewAlertNode("a1", nil)
	if err != nil {
		t.Fatalf("NewAlertNode() error = %v", err)
	}
	outputs, err := node.Execute(context.Background(), map[string]any{"value": float64(10)}, map[string]any{"condition": "gt", "threshold": float64(50)}, nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if outputs["triggered"] != false {
		t.Error("expected triggered=false")
	}
}

func TestAlertNode_AllOps(t *testing.T) {
	node, err := NewAlertNode("a1", nil)
	if err != nil {
		t.Fatalf("NewAlertNode() error = %v", err)
	}
	tests := []struct {
		cond      string
		value     float64
		threshold float64
		expect    bool
	}{
		{"gt", 10, 5, true},
		{"lt", 3, 5, true},
		{"gte", 5, 5, true},
		{"lte", 5, 5, true},
		{"eq", 5, 5, true},
	}
	for _, tc := range tests {
		outputs, err := node.Execute(context.Background(), map[string]any{"value": tc.value}, map[string]any{"condition": tc.cond, "threshold": tc.threshold}, nil)
		if err != nil {
			t.Fatalf("cond=%q: %v", tc.cond, err)
		}
		if outputs["triggered"] != tc.expect {
			t.Errorf("cond=%q val=%v thresh=%v: got %v, want %v", tc.cond, tc.value, tc.threshold, outputs["triggered"], tc.expect)
		}
	}
}
