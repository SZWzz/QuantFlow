package nodes

import (
	"context"
	"testing"
)

func TestStrategyNode_Execute(t *testing.T) {
	node, err := NewStrategyNode("strat1", nil)
	if err != nil {
		t.Fatalf("NewStrategyNode() error = %v", err)
	}
	if node.NodeType() != "strategy" {
		t.Errorf("NodeType() = %q, want %q", node.NodeType(), "strategy")
	}
	outputs, err := node.Execute(context.Background(), map[string]any{}, map[string]any{"signal_type": "sma_cross", "fast_period": float64(5), "slow_period": float64(20)}, nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	config, ok := outputs["strategy_config"].(map[string]any)
	if !ok {
		t.Fatalf("expected map, got %T", outputs["strategy_config"])
	}
	if config["signal_type"] != "sma_cross" {
		t.Errorf("signal_type = %v, want 'sma_cross'", config["signal_type"])
	}
	if config["fast_period"] != 5 {
		t.Errorf("fast_period = %v, want 5", config["fast_period"])
	}
}

func TestStrategyNode_WithSignals(t *testing.T) {
	node, err := NewStrategyNode("strat1", nil)
	if err != nil {
		t.Fatalf("NewStrategyNode() error = %v", err)
	}
	outputs, err := node.Execute(context.Background(), map[string]any{
		"factor_signals": []float64{1, 0, 1},
		"exit_signals":   []float64{0, -1, 0},
	}, map[string]any{"signal_type": "sma_cross"}, nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	signals, ok := outputs["signals"].([]float64)
	if !ok {
		t.Fatalf("expected []float64, got %T", outputs["signals"])
	}
	if len(signals) != 3 {
		t.Fatalf("len = %d, want 3", len(signals))
	}
	if signals[0] != 1 || signals[1] != -1 {
		t.Errorf("signals = %v", signals)
	}
}
