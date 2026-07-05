package nodes

import (
	"context"
	"testing"
)

func TestRollingZScoreNode_Execute(t *testing.T) {
	node, err := NewRollingZScoreNode("rz1", nil)
	if err != nil {
		t.Fatalf("NewRollingZScoreNode() error = %v", err)
	}
	if node.NodeType() != "rolling_zscore" {
		t.Errorf("NodeType() = %q, want %q", node.NodeType(), "rolling_zscore")
	}
	outputs, err := node.Execute(context.Background(), map[string]any{"values": []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}}, map[string]any{"period": float64(5)}, nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	result := outputs["result"].([]float64)
	if len(result) != 10 {
		t.Fatalf("len = %d, want 10", len(result))
	}
	for i := 0; i < 4; i++ {
		if result[i] != 0 {
			t.Errorf("result[%d] = %v, want 0 (insufficient data)", i, result[i])
		}
	}
}

func TestRollingZScoreNode_MissingInput(t *testing.T) {
	node, _err := NewRollingZScoreNode("rz1", nil)
	if _err != nil {
		t.Fatalf("NewRollingZScoreNode() error = %v", _err)
	}
	_, err := node.Execute(context.Background(), map[string]any{}, nil, nil)
	if err == nil {
		t.Error("expected error for missing 'values' input")
	}
}

func TestRollingZScoreNode_PeriodTooSmall(t *testing.T) {
	node, _err := NewRollingZScoreNode("rz1", nil)
	if _err != nil {
		t.Fatalf("NewRollingZScoreNode() error = %v", _err)
	}
	_, err := node.Execute(context.Background(), map[string]any{"values": []float64{1, 2, 3}}, map[string]any{"period": float64(1)}, nil)
	if err == nil {
		t.Error("expected error for period <= 1")
	}
}
