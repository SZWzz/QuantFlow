package nodes

import (
	"context"
	"testing"
)

func TestAnalystEstimatesNode_MockData(t *testing.T) {
	node, err := NewAnalystEstimatesNode("ae1", nil)
	if err != nil {
		t.Fatalf("NewAnalystEstimatesNode() error = %v", err)
	}
	if node.NodeType() != "analyst_estimates" {
		t.Errorf("NodeType() = %q, want %q", node.NodeType(), "analyst_estimates")
	}
	outputs, err := node.Execute(context.Background(), map[string]any{"symbol": "AAPL"}, nil, nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	targetPrice, ok := outputs["target_price"].(float64)
	if !ok || targetPrice != 0 {
		t.Errorf("target_price = %v, want 0", targetPrice)
	}
	consensus, ok := outputs["consensus"].(string)
	if !ok || consensus != "neutral" {
		t.Errorf("consensus = %q, want 'neutral'", consensus)
	}
}

func TestAnalystEstimatesNode_MissingSymbol(t *testing.T) {
	node, err := NewAnalystEstimatesNode("ae1", nil)
	if err != nil {
		t.Fatalf("NewAnalystEstimatesNode() error = %v", err)
	}
	_, err = node.Execute(context.Background(), map[string]any{}, nil, nil)
	if err == nil {
		t.Error("expected error for missing symbol")
	}
}
