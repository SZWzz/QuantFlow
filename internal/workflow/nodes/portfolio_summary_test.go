package nodes

import (
	"context"
	"testing"
)

func TestPortfolioSummaryNode_NoOMS(t *testing.T) {
	node, err := NewPortfolioSummaryNode("ps1", nil)
	if err != nil {
		t.Fatalf("NewPortfolioSummaryNode() error = %v", err)
	}
	if node.NodeType() != "portfolio_summary" {
		t.Errorf("NodeType() = %q, want %q", node.NodeType(), "portfolio_summary")
	}
	outputs, err := node.Execute(context.Background(), map[string]any{}, nil, nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	summary, ok := outputs["summary"].(map[string]any)
	if !ok {
		t.Fatalf("expected map, got %T", outputs["summary"])
	}
	tv, ok := summary["total_value"].(int)
	if !ok {
		t.Errorf("total_value type = %T, want int", summary["total_value"])
	} else if tv != 0 {
		t.Errorf("total_value = %v, want 0", tv)
	}
}
