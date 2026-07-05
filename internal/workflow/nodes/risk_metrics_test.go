package nodes

import (
	"context"
	"testing"
)

func TestRiskMetricsNode_WithReturns(t *testing.T) {
	node, err := NewRiskMetricsNode("rm1", nil)
	if err != nil {
		t.Fatalf("NewRiskMetricsNode() error = %v", err)
	}
	if node.NodeType() != "risk_metrics" {
		t.Errorf("NodeType() = %q, want %q", node.NodeType(), "risk_metrics")
	}
	outputs, err := node.Execute(context.Background(), map[string]any{
		"portfolio": map[string]any{"AAPL": 50000.0, "GOOGL": 30000.0},
		"returns":   []float64{0.01, 0.02, -0.01, 0.03, 0.01, -0.02, 0.02, 0.01, 0.0, -0.01, 0.02, 0.03},
	}, nil, nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	metrics, ok := outputs["metrics"].(map[string]any)
	if !ok {
		t.Fatalf("expected map, got %T", outputs["metrics"])
	}
	if metrics["total_value"].(float64) != 80000 {
		t.Errorf("total_value = %v, want 80000", metrics["total_value"])
	}
	sharpe, ok := metrics["sharpe_ratio"].(float64)
	if !ok || sharpe == 0 {
		t.Errorf("expected non-zero sharpe_ratio, got %v", sharpe)
	}
	vol, ok := metrics["volatility"].(float64)
	if !ok || vol <= 0 {
		t.Errorf("expected positive volatility, got %v", vol)
	}
	dd, ok := metrics["max_drawdown"].(float64)
	if !ok || dd < 0 {
		t.Errorf("max_drawdown should be >= 0, got %v", dd)
	}
}

func TestRiskMetricsNode_MissingPortfolio(t *testing.T) {
	node, _err := NewRiskMetricsNode("rm1", nil)
	if _err != nil {
		t.Fatalf("NewRiskMetricsNode() error = %v", _err)
	}
	_, err := node.Execute(context.Background(), map[string]any{}, nil, nil)
	if err == nil {
		t.Error("expected error for missing portfolio")
	}
}

func TestRiskMetricsNode_NoReturns(t *testing.T) {
	node, err := NewRiskMetricsNode("rm1", nil)
	if err != nil {
		t.Fatalf("NewRiskMetricsNode() error = %v", err)
	}
	outputs, err := node.Execute(context.Background(), map[string]any{
		"portfolio": map[string]float64{"AAPL": 100000},
	}, nil, nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	metrics := outputs["metrics"].(map[string]any)
	if metrics["sharpe_ratio"].(float64) != 0 {
		t.Errorf("sharpe should be 0 when no returns, got %v", metrics["sharpe_ratio"])
	}
}

func TestRiskMetricsNode_UnsupportedPortfolio(t *testing.T) {
	node2, err := NewRiskMetricsNode("rm1", nil)
	if err != nil {
		t.Fatalf("NewRiskMetricsNode() error = %v", err)
	}
	_, err = node2.Execute(context.Background(), map[string]any{
		"portfolio": "invalid",
	}, nil, nil)
	if err == nil {
		t.Error("expected error for unsupported portfolio type")
	}
}
