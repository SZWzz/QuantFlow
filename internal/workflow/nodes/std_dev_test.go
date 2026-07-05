package nodes

import (
	"context"
	"testing"
)

func TestStdDevNode_Execute(t *testing.T) {
	node, err := NewStdDevNode("sd1", nil)
	if err != nil {
		t.Fatalf("NewStdDevNode() error = %v", err)
	}
	if node.NodeType() != "std_dev" {
		t.Errorf("NodeType() = %q, want %q", node.NodeType(), "std_dev")
	}
	outputs, err := node.Execute(context.Background(), map[string]any{"values": []float64{10, 12, 14, 16, 18, 20}}, map[string]any{"period": float64(3)}, nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	result := outputs["result"].([]float64)
	if len(result) != 6 {
		t.Fatalf("len = %d, want 6", len(result))
	}
	for i := 0; i < 2; i++ {
		if result[i] != 0 {
			t.Errorf("result[%d] = %v, want 0 (insufficient data)", i, result[i])
		}
	}
	if result[5] <= 0 {
		t.Errorf("result[5] = %v, want positive std dev", result[5])
	}
}

func TestStdDevNode_MissingInput(t *testing.T) {
	node, _err := NewStdDevNode("sd1", nil)
	if _err != nil {
		t.Fatalf("NewStdDevNode() error = %v", _err)
	}
	_, err := node.Execute(context.Background(), map[string]any{}, nil, nil)
	if err == nil {
		t.Error("expected error for missing 'values' input")
	}
}
