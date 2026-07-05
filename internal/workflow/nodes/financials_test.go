package nodes

import (
	"context"
	"testing"
)

func TestFinancialsNode_MockData(t *testing.T) {
	node, err := NewFinancialsNode("f1", nil)
	if err != nil {
		t.Fatalf("NewFinancialsNode() error = %v", err)
	}
	if node.NodeType() != "financials" {
		t.Errorf("NodeType() = %q, want %q", node.NodeType(), "financials")
	}
	outputs, err := node.Execute(context.Background(), map[string]any{"symbol": "AAPL"}, nil, nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	fd, ok := outputs["financial_data"].(map[string]any)
	if !ok {
		t.Fatalf("expected map, got %T", outputs["financial_data"])
	}
	if fd["symbol"] != "AAPL" {
		t.Errorf("symbol = %v, want 'AAPL'", fd["symbol"])
	}
	ratios, ok := outputs["ratios"].(map[string]any)
	if !ok {
		t.Fatalf("expected map ratios, got %T", outputs["ratios"])
	}
	if ratios["pe_ratio"] != float64(0) {
		t.Errorf("pe_ratio = %v, want 0", ratios["pe_ratio"])
	}
}

func TestFinancialsNode_MissingSymbol(t *testing.T) {
	node, _err := NewFinancialsNode("f1", nil)
	if _err != nil {
		t.Fatalf("NewFinancialsNode() error = %v", _err)
	}
	_, err := node.Execute(context.Background(), map[string]any{}, nil, nil)
	if err == nil {
		t.Error("expected error for missing symbol")
	}
}
