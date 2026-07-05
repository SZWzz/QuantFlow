package nodes

import (
	"context"
	"testing"
)

func TestStockResearchNode_MockData(t *testing.T) {
	node, err := NewStockResearchNode("sr1", nil)
	if err != nil {
		t.Fatalf("NewStockResearchNode() error = %v", err)
	}
	if node.NodeType() != "stock_research" {
		t.Errorf("NodeType() = %q, want %q", node.NodeType(), "stock_research")
	}
	outputs, err := node.Execute(context.Background(), map[string]any{"symbol": "AAPL"}, nil, nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	overview, ok := outputs["overview"].(map[string]any)
	if !ok {
		t.Fatalf("expected map overview, got %T", outputs["overview"])
	}
	if overview["symbol"] != "AAPL" {
		t.Errorf("symbol = %v, want 'AAPL'", overview["symbol"])
	}
	financials, ok := outputs["financials"].(map[string]any)
	if !ok || financials["source"] != "mock" {
		t.Errorf("expected mock financials, got %v", financials)
	}
	sentiment, ok := outputs["sentiment"].(map[string]any)
	if !ok || sentiment["source"] != "mock" {
		t.Errorf("expected mock sentiment, got %v", sentiment)
	}
}

func TestStockResearchNode_MissingSymbol(t *testing.T) {
	node, _err := NewStockResearchNode("sr1", nil)
	if _err != nil {
		t.Fatalf("NewStockResearchNode() error = %v", _err)
	}
	_, err := node.Execute(context.Background(), map[string]any{}, nil, nil)
	if err == nil {
		t.Error("expected error for missing symbol")
	}
}
