package nodes

import (
	"context"
	"testing"
)

func TestRankSelectNode_TopN(t *testing.T) {
	node, err := NewRankSelectNode("rs1", nil)
	if err != nil {
		t.Fatalf("NewRankSelectNode() error = %v", err)
	}
	if node.NodeType() != "rank_select" {
		t.Errorf("NodeType() = %q, want %q", node.NodeType(), "rank_select")
	}
	outputs, err := node.Execute(context.Background(), map[string]any{"factor_values": []float64{10, 5, 20, 3, 15}}, map[string]any{"top_n": float64(2), "ascending": "false"}, nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	selected := outputs["selected"].([]float64)
	if len(selected) != 5 {
		t.Fatalf("len = %d, want 5", len(selected))
	}
	count := 0
	for _, v := range selected {
		if v == 1 {
			count++
		}
	}
	if count != 2 {
		t.Errorf("selected %d items, want 2", count)
	}
}

func TestRankSelectNode_BottomN(t *testing.T) {
	node, _err := NewRankSelectNode("rs1", nil)
	if _err != nil {
		t.Fatalf("NewRankSelectNode() error = %v", _err)
	}
	outputs, err := node.Execute(context.Background(), map[string]any{"factor_values": []float64{10, 5, 20, 3, 15}}, map[string]any{"top_n": float64(2), "ascending": "true"}, nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	selected := outputs["selected"].([]float64)
	count := 0
	for _, v := range selected {
		if v == 1 {
			count++
		}
	}
	if count != 2 {
		t.Errorf("selected %d items, want 2", count)
	}
}

func TestRankSelectNode_MissingInput(t *testing.T) {
	node, _err := NewRankSelectNode("rs1", nil)
	if _err != nil {
		t.Fatalf("NewRankSelectNode() error = %v", _err)
	}
	_, err := node.Execute(context.Background(), map[string]any{}, nil, nil)
	if err == nil {
		t.Error("expected error for missing 'factor_values' input")
	}
}
