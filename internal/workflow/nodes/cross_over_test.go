package nodes

import (
	"context"
	"testing"
)

func TestCrossOverNode_GoldenCross(t *testing.T) {
	node, err := NewCrossOverNode("co1", nil)
	if err != nil {
		t.Fatalf("NewCrossOverNode() error = %v", err)
	}
	if node.NodeType() != "cross_over" {
		t.Errorf("NodeType() = %q, want %q", node.NodeType(), "cross_over")
	}
	fast := []float64{1, 2, 3, 5, 7}
	slow := []float64{2, 2, 2, 3, 4}
	outputs, err := node.Execute(context.Background(), map[string]any{"fast": fast, "slow": slow}, nil, nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	cross := outputs["cross"].([]float64)
	if len(cross) != 5 {
		t.Fatalf("len = %d, want 5", len(cross))
	}
	if cross[2] != 1 {
		t.Errorf("cross[2] = %v, want 1 (golden cross)", cross[2])
	}
}

func TestCrossOverNode_DeathCross(t *testing.T) {
	fast := []float64{7, 5, 3, 2, 1}
	slow := []float64{4, 4, 4, 3, 2}
	node, _err := NewCrossOverNode("co1", nil)
	if _err != nil {
		t.Fatalf("NewCrossOverNode() error = %v", _err)
	}
	outputs, err := node.Execute(context.Background(), map[string]any{"fast": fast, "slow": slow}, nil, nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	cross := outputs["cross"].([]float64)
	if cross[2] != -1 {
		t.Errorf("cross[2] = %v, want -1 (death cross)", cross[2])
	}
}

func TestCrossOverNode_MissingInput(t *testing.T) {
	node, _err := NewCrossOverNode("co1", nil)
	if _err != nil {
		t.Fatalf("NewCrossOverNode() error = %v", _err)
	}
	_, err := node.Execute(context.Background(), map[string]any{"fast": []float64{1}}, nil, nil)
	if err == nil {
		t.Error("expected error for missing slow")
	}
}

func TestCrossOverNode_EmptyInput(t *testing.T) {
	node, _err := NewCrossOverNode("co1", nil)
	if _err != nil {
		t.Fatalf("NewCrossOverNode() error = %v", _err)
	}
	_, err := node.Execute(context.Background(), map[string]any{"fast": []float64{}, "slow": []float64{}}, nil, nil)
	if err == nil {
		t.Error("expected error for empty inputs")
	}
}
