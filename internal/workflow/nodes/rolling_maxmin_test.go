package nodes

import (
	"context"
	"testing"
)

func TestRollingMaxMinNode_Max(t *testing.T) {
	node, err := NewRollingMaxMinNode("rm1", nil)
	if err != nil {
		t.Fatalf("NewRollingMaxMinNode() error = %v", err)
	}
	if node.NodeType() != "rolling_maxmin" {
		t.Errorf("NodeType() = %q, want %q", node.NodeType(), "rolling_maxmin")
	}
	outputs, err := node.Execute(context.Background(), map[string]any{"values": []float64{1, 3, 2, 5, 4}}, map[string]any{"period": float64(3), "mode": "max"}, nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	result := outputs["result"].([]float64)
	if len(result) != 5 {
		t.Fatalf("len = %d, want 5", len(result))
	}
	if result[4] != 5 {
		t.Errorf("result[4] = %v, want 5 (rolling max)", result[4])
	}
}

func TestRollingMaxMinNode_Min(t *testing.T) {
	node, _err := NewRollingMaxMinNode("rm1", nil)
	if _err != nil {
		t.Fatalf("NewRollingMaxMinNode() error = %v", _err)
	}
	outputs, err := node.Execute(context.Background(), map[string]any{"values": []float64{3, 1, 4, 2, 5}}, map[string]any{"period": float64(3), "mode": "min"}, nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	result := outputs["result"].([]float64)
	if result[4] != 2 {
		t.Errorf("result[4] = %v, want 2 (rolling min)", result[4])
	}
}

func TestRollingMaxMinNode_MissingInput(t *testing.T) {
	node, _err := NewRollingMaxMinNode("rm1", nil)
	if _err != nil {
		t.Fatalf("NewRollingMaxMinNode() error = %v", _err)
	}
	_, err := node.Execute(context.Background(), map[string]any{}, nil, nil)
	if err == nil {
		t.Error("expected error for missing 'values' input")
	}
}

func TestRollingMaxMinNode_PeriodTooLarge(t *testing.T) {
	node, _err := NewRollingMaxMinNode("rm1", nil)
	if _err != nil {
		t.Fatalf("NewRollingMaxMinNode() error = %v", _err)
	}
	_, err := node.Execute(context.Background(), map[string]any{"values": []float64{1, 2}}, map[string]any{"period": float64(5)}, nil)
	if err == nil {
		t.Error("expected error for period > data length")
	}
}
