package nodes

import (
	"context"
	"testing"
)

func TestScaleNode_ZScore(t *testing.T) {
	node, err := NewScaleNode("s1", nil)
	if err != nil {
		t.Fatalf("NewScaleNode() error = %v", err)
	}
	if node.NodeType() != "scale" {
		t.Errorf("NodeType() = %q, want %q", node.NodeType(), "scale")
	}
	outputs, err := node.Execute(context.Background(), map[string]any{"values": []float64{1, 2, 3, 4, 5}}, map[string]any{"method": "zscore"}, nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	result := outputs["result"].([]float64)
	if len(result) != 5 {
		t.Fatalf("len = %d, want 5", len(result))
	}
	var sum float64
	for _, v := range result {
		sum += v
	}
	if sum > 1e-10 {
		t.Errorf("z-scores should sum to ~0, got %v", sum)
	}
}

func TestScaleNode_MinMax(t *testing.T) {
	node, _err := NewScaleNode("s1", nil)
	if _err != nil {
		t.Fatalf("NewScaleNode() error = %v", _err)
	}
	outputs, err := node.Execute(context.Background(), map[string]any{"values": []float64{1, 3, 5}}, map[string]any{"method": "minmax"}, nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	result := outputs["result"].([]float64)
	if result[0] != 0 || result[1] != 0.5 || result[2] != 1 {
		t.Errorf("minmax: got %v, want [0 0.5 1]", result)
	}
}

func TestScaleNode_Identical(t *testing.T) {
	node, _err := NewScaleNode("s1", nil)
	if _err != nil {
		t.Fatalf("NewScaleNode() error = %v", _err)
	}
	outputs, err := node.Execute(context.Background(), map[string]any{"values": []float64{5, 5, 5}}, map[string]any{"method": "zscore"}, nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	result := outputs["result"].([]float64)
	for _, v := range result {
		if v != 0 {
			t.Errorf("zscore of identical values should be 0, got %v", v)
		}
	}
}

func TestScaleNode_MissingInput(t *testing.T) {
	node, _err := NewScaleNode("s1", nil)
	if _err != nil {
		t.Fatalf("NewScaleNode() error = %v", _err)
	}
	_, err := node.Execute(context.Background(), map[string]any{}, nil, nil)
	if err == nil {
		t.Error("expected error for missing 'values' input")
	}
}

func TestScaleNode_Empty(t *testing.T) {
	node, _err := NewScaleNode("s1", nil)
	if _err != nil {
		t.Fatalf("NewScaleNode() error = %v", _err)
	}
	_, err := node.Execute(context.Background(), map[string]any{"values": []float64{}}, nil, nil)
	if err == nil {
		t.Error("expected error for empty values")
	}
}
