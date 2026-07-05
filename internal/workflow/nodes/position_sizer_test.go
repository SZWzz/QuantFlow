package nodes

import (
	"context"
	"testing"
)

func TestPositionSizerNode_Percent(t *testing.T) {
	node, err := NewPositionSizerNode("ps1", nil)
	if err != nil {
		t.Fatalf("NewPositionSizerNode() error = %v", err)
	}
	if node.NodeType() != "position_sizer" {
		t.Errorf("NodeType() = %q, want %q", node.NodeType(), "position_sizer")
	}
	outputs, err := node.Execute(context.Background(), map[string]any{"portfolio_value": float64(100000)}, map[string]any{"method": "percent", "pct": float64(2), "stop_distance": float64(0.05)}, nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	posSize := outputs["position_size"].(float64)
	riskAmt := outputs["risk_amount"].(float64)
	if riskAmt != 2000 {
		t.Errorf("risk_amount = %v, want 2000", riskAmt)
	}
	if posSize != 40000 {
		t.Errorf("position_size = %v, want 40000 (2000/0.05)", posSize)
	}
}

func TestPositionSizerNode_Fixed(t *testing.T) {
	node, _err := NewPositionSizerNode("ps1", nil)
	if _err != nil {
		t.Fatalf("NewPositionSizerNode() error = %v", _err)
	}
	outputs, err := node.Execute(context.Background(), map[string]any{"portfolio_value": float64(100000), "risk_per_trade": float64(500)}, map[string]any{"method": "fixed"}, nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if outputs["position_size"].(float64) != 500 {
		t.Errorf("position_size = %v, want 500", outputs["position_size"])
	}
}

func TestPositionSizerNode_MissingPortfolioValue(t *testing.T) {
	node, _err := NewPositionSizerNode("ps1", nil)
	if _err != nil {
		t.Fatalf("NewPositionSizerNode() error = %v", _err)
	}
	_, err := node.Execute(context.Background(), map[string]any{}, nil, nil)
	if err == nil {
		t.Error("expected error for missing portfolio_value")
	}
}
