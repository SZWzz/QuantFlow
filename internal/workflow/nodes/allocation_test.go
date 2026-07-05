package nodes

import (
	"context"
	"testing"
)

func TestAllocationNode_Equal(t *testing.T) {
	node, err := NewAllocationNode("al1", nil)
	if err != nil {
		t.Fatalf("NewAllocationNode() error = %v", err)
	}
	if node.NodeType() != "allocation" {
		t.Errorf("NodeType() = %q, want %q", node.NodeType(), "allocation")
	}
	outputs, err := node.Execute(context.Background(), map[string]any{
		"symbols":       []string{"AAPL", "GOOGL", "MSFT"},
		"total_capital": float64(100000),
	}, map[string]any{"method": "equal"}, nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	allocations, ok := outputs["allocations"].(map[string]float64)
	if !ok {
		t.Fatalf("expected map[string]float64, got %T", outputs["allocations"])
	}
	if len(allocations) != 3 {
		t.Fatalf("len = %d, want 3", len(allocations))
	}
	var total float64
	for _, v := range allocations {
		total += v
	}
	if total > 100000 || total <= 0 {
		t.Errorf("total allocation = %v, want ~100000", total)
	}
}

func TestAllocationNode_MissingSymbols(t *testing.T) {
	node, err := NewAllocationNode("al1", nil)
	if err != nil {
		t.Fatalf("NewAllocationNode() error = %v", err)
	}
	_, err = node.Execute(context.Background(), map[string]any{"total_capital": float64(100000)}, nil, nil)
	if err == nil {
		t.Error("expected error for missing symbols")
	}
}

func TestAllocationNode_MissingCapital(t *testing.T) {
	node, err := NewAllocationNode("al1", nil)
	if err != nil {
		t.Fatalf("NewAllocationNode() error = %v", err)
	}
	_, err = node.Execute(context.Background(), map[string]any{"symbols": []string{"AAPL"}}, nil, nil)
	if err == nil {
		t.Error("expected error for missing total_capital")
	}
}

func TestAllocationNode_EmptySymbols(t *testing.T) {
	node, err := NewAllocationNode("al1", nil)
	if err != nil {
		t.Fatalf("NewAllocationNode() error = %v", err)
	}
	_, err = node.Execute(context.Background(), map[string]any{"symbols": []string{}, "total_capital": float64(100000)}, nil, nil)
	if err == nil {
		t.Error("expected error for empty symbols")
	}
}
