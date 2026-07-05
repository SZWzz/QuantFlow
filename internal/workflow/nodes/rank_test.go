package nodes

import (
	"context"
	"testing"
)

func TestRankNode_Percentile(t *testing.T) {
	node, err := NewRankNode("rk1", nil)
	if err != nil {
		t.Fatalf("NewRankNode() error = %v", err)
	}
	if node.NodeType() != "rank" {
		t.Errorf("NodeType() = %q, want %q", node.NodeType(), "rank")
	}
	outputs, err := node.Execute(context.Background(), map[string]any{"values": []float64{1, 2, 3, 4, 5}}, map[string]any{"method": "percentile"}, nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	result := outputs["result"].([]float64)
	if len(result) != 5 {
		t.Fatalf("len = %d, want 5", len(result))
	}
	if result[0] != 0 || result[4] != 1 {
		t.Errorf("percentile rank: got %v, expected first=0 last=1", result)
	}
}

func TestRankNode_MinMax(t *testing.T) {
	node, _err := NewRankNode("rk1", nil)
	if _err != nil {
		t.Fatalf("NewRankNode() error = %v", _err)
	}
	outputs, err := node.Execute(context.Background(), map[string]any{"values": []float64{10, 20, 30}}, map[string]any{"method": "minmax"}, nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	result := outputs["result"].([]float64)
	if result[0] != 0 || result[1] != 0.5 || result[2] != 1 {
		t.Errorf("minmax: got %v, want [0 0.5 1]", result)
	}
}

func TestRankNode_MissingInput(t *testing.T) {
	node, _err := NewRankNode("rk1", nil)
	if _err != nil {
		t.Fatalf("NewRankNode() error = %v", _err)
	}
	_, err := node.Execute(context.Background(), map[string]any{}, nil, nil)
	if err == nil {
		t.Error("expected error for missing 'values' input")
	}
}

func TestRankNode_Empty(t *testing.T) {
	node, _err := NewRankNode("rk1", nil)
	if _err != nil {
		t.Fatalf("NewRankNode() error = %v", _err)
	}
	_, err := node.Execute(context.Background(), map[string]any{"values": []float64{}}, nil, nil)
	if err == nil {
		t.Error("expected error for empty values")
	}
}
