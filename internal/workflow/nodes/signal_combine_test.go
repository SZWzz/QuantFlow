package nodes

import (
	"context"
	"testing"
)

func TestSignalCombineNode_Majority(t *testing.T) {
	node, err := NewSignalCombineNode("sc1", nil)
	if err != nil {
		t.Fatalf("NewSignalCombineNode() error = %v", err)
	}
	if node.NodeType() != "signal_combine" {
		t.Errorf("NodeType() = %q, want %q", node.NodeType(), "signal_combine")
	}
	outputs, err := node.Execute(context.Background(), map[string]any{"signals": [][]float64{{1, 0, -1}, {1, 1, 0}, {0, 0, -1}}}, map[string]any{"method": "majority"}, nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	combined := outputs["combined"].([]float64)
	if len(combined) != 3 {
		t.Fatalf("len = %d, want 3", len(combined))
	}
	if combined[0] != 1 || combined[2] != -1 {
		t.Errorf("majority: got %v, expected [1 0 -1]", combined)
	}
}

func TestSignalCombineNode_And(t *testing.T) {
	node, _err := NewSignalCombineNode("sc1", nil)
	if _err != nil {
		t.Fatalf("NewSignalCombineNode() error = %v", _err)
	}
	outputs, err := node.Execute(context.Background(), map[string]any{"signals": [][]float64{{1, 1, -1}, {1, 0, -1}}}, map[string]any{"method": "and"}, nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	combined := outputs["combined"].([]float64)
	if combined[0] != 1 || combined[2] != -1 {
		t.Errorf("and: got %v, expected [1 0 -1]", combined)
	}
}

func TestSignalCombineNode_MissingInput(t *testing.T) {
	node, _err := NewSignalCombineNode("sc1", nil)
	if _err != nil {
		t.Fatalf("NewSignalCombineNode() error = %v", _err)
	}
	_, err := node.Execute(context.Background(), map[string]any{}, nil, nil)
	if err == nil {
		t.Error("expected error for missing 'signals'")
	}
}
