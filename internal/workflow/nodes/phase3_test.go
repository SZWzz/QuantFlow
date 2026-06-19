package nodes

import (
	"context"
	"testing"

	"quantflow/internal/backtest"
	"quantflow/internal/market"
	"quantflow/internal/workflow"
)

func TestFactorNode(t *testing.T) {
	node, err := NewFactorNode("factor1", map[string]any{
		"factor_name": "momentum_20d",
		"symbols":     "000001.SZ",
	})
	if err != nil {
		t.Fatalf("NewFactorNode failed: %v", err)
	}

	if node.NodeType() != "factor" {
		t.Errorf("NodeType = %q, want %q", node.NodeType(), "factor")
	}
	if node.Category() != "alpha" {
		t.Errorf("Category = %q, want %q", node.Category(), "alpha")
	}

	// Test execution
	inputs := map[string]any{"ohlcv": []float64{10, 11, 12, 13, 14}}
	outputs, err := node.Execute(context.Background(), inputs, map[string]any{
		"factor_name": "momentum_20d",
	})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	metadata, ok := outputs["metadata"].(map[string]any)
	if !ok {
		t.Fatalf("metadata is not map[string]any: %T", outputs["metadata"])
	}
	if metadata["factor_name"] != "momentum_20d" {
		t.Errorf("factor_name = %v", metadata["factor_name"])
	}
}

func TestFactorNode_Validate(t *testing.T) {
	// Valid
	n, _ := NewFactorNode("f1", map[string]any{"factor_name": "rsi_14"})
	if err := n.Validate(); err != nil {
		t.Errorf("expected valid, got: %v", err)
	}

	// Missing factor_name
	n2, _ := NewFactorNode("f2", map[string]any{})
	if err := n2.Validate(); err == nil {
		t.Error("expected error for missing factor_name")
	}
}

func TestStrategyNode(t *testing.T) {
	node, err := NewStrategyNode("strat1", map[string]any{
		"signal_type": "sma_cross",
		"fast_period": 5,
		"slow_period": 20,
	})
	if err != nil {
		t.Fatalf("NewStrategyNode failed: %v", err)
	}

	if node.NodeType() != "strategy" {
		t.Errorf("NodeType = %q", node.NodeType())
	}
	if node.Category() != "strategy" {
		t.Errorf("Category = %q", node.Category())
	}

	outputs, err := node.Execute(context.Background(), nil, map[string]any{
		"signal_type": "sma_cross",
	})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	config, ok := outputs["strategy_config"].(map[string]any)
	if !ok {
		t.Fatalf("strategy_config not found in outputs")
	}
	if config["signal_type"] != "sma_cross" {
		t.Errorf("signal_type = %v", config["signal_type"])
	}
}

func TestStrategyNode_Validate(t *testing.T) {
	// Valid types
	for _, st := range []string{"sma_cross", "rsi_threshold", "momentum", "custom"} {
		n, _ := NewStrategyNode("s", map[string]any{"signal_type": st})
		if err := n.Validate(); err != nil {
			t.Errorf("expected valid for %q, got: %v", st, err)
		}
	}

	// Invalid type
	n, _ := NewStrategyNode("s", map[string]any{"signal_type": "invalid_type"})
	if err := n.Validate(); err == nil {
		t.Error("expected error for invalid signal_type")
	}
}

func TestBacktestNode(t *testing.T) {
	node, err := NewBacktestNode("bt1", map[string]any{
		"market":       "CN",
		"initial_cash": 100000,
	})
	if err != nil {
		t.Fatalf("NewBacktestNode failed: %v", err)
	}

	if node.NodeType() != "backtest" {
		t.Errorf("NodeType = %q", node.NodeType())
	}
	if node.Category() != "backtest" {
		t.Errorf("Category = %q", node.Category())
	}
}

func TestBacktestNode_Execute(t *testing.T) {
	// Create some test OHLCV bars (market.OHLCVBar is the canonical type).
	bars := []market.OHLCVBar{
		{Symbol: "TEST", Date: "2024-01-01", Open: 10, High: 10.2, Low: 9.8, Close: 10, Volume: 1000},
		{Symbol: "TEST", Date: "2024-01-02", Open: 11, High: 11.2, Low: 10.8, Close: 11, Volume: 1000},
		{Symbol: "TEST", Date: "2024-01-03", Open: 12, High: 12.2, Low: 11.8, Close: 12, Volume: 1000},
		{Symbol: "TEST", Date: "2024-01-04", Open: 13, High: 13.2, Low: 12.8, Close: 13, Volume: 1000},
		{Symbol: "TEST", Date: "2024-01-05", Open: 14, High: 14.2, Low: 13.8, Close: 14, Volume: 1000},
	}

	node, _ := NewBacktestNode("bt2", map[string]any{"market": "CN"})
	outputs, err := node.Execute(context.Background(), map[string]any{
		"ohlcv_data": bars,
	}, map[string]any{
		"market":       "CN",
		"initial_cash": 100000,
	})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Verify outputs
	if outputs["equity_curve"] == nil {
		t.Error("equity_curve is nil")
	}

	result, ok := outputs["result"].(*backtest.Result)
	if !ok {
		t.Fatalf("result is not *backtest.Result: %T", outputs["result"])
	}
	if len(result.EquityCurve) != 5 {
		t.Errorf("expected 5 equity points, got %d", len(result.EquityCurve))
	}
	if len(result.Trades) == 0 {
		t.Error("expected at least 1 trade (buy and hold)")
	}

	metrics, ok := outputs["metrics"].(backtest.Metrics)
	if !ok {
		t.Fatalf("metrics is not backtest.Metrics: %T", outputs["metrics"])
	}
	t.Logf("Backtest: Return=%.2f%%, Trades=%d, Sharpe=%.2f",
		metrics.TotalReturn*100, metrics.TotalTrades, metrics.SharpeRatio)
}

func TestBacktestNode_Validate(t *testing.T) {
	for _, m := range []string{"CN", "US", "HK", "CRYPTO"} {
		n, _ := NewBacktestNode("b", map[string]any{"market": m})
		if err := n.Validate(); err != nil {
			t.Errorf("expected valid for market %q, got: %v", m, err)
		}
	}

	n, _ := NewBacktestNode("b", map[string]any{"market": "INVALID"})
	if err := n.Validate(); err == nil {
		t.Error("expected error for invalid market")
	}
}

func TestAgentNode_Creation(t *testing.T) {
	node, err := NewAgentNode("agent1", map[string]any{
		"profile": "general",
	})
	if err != nil {
		t.Fatalf("NewAgentNode failed: %v", err)
	}

	if node.NodeType() != "agent" {
		t.Errorf("NodeType = %q, want %q", node.NodeType(), "agent")
	}
	if node.Category() != "ai" {
		t.Errorf("Category = %q, want %q", node.Category(), "ai")
	}
}

func TestAgentNode_Ports(t *testing.T) {
	node, _ := NewAgentNode("a1", nil)

	hasPrompt := false
	for _, p := range node.InputPorts() {
		if p.Name == "prompt" && p.Required {
			hasPrompt = true
		}
	}
	if !hasPrompt {
		t.Error("prompt input port should be required")
	}

	hasResult := false
	for _, p := range node.OutputPorts() {
		if p.Name == "result" {
			hasResult = true
		}
	}
	if !hasResult {
		t.Error("result output port missing")
	}
}

func TestAgentNode_Validate(t *testing.T) {
	for _, p := range []string{"general", "quant_analyst", "trader", "research_assistant"} {
		n, _ := NewAgentNode("a", map[string]any{"profile": p})
		if err := n.Validate(); err != nil {
			t.Errorf("expected valid for profile %q, got: %v", p, err)
		}
	}

	n, _ := NewAgentNode("a", map[string]any{"profile": "invalid_profile"})
	if err := n.Validate(); err == nil {
		t.Error("expected error for invalid profile")
	}
}

func TestAgentNode_Registration(t *testing.T) {
	registry := workflow.NewRegistry()
	RegisterAll(registry)

	if !registry.Has("agent") {
		t.Error("agent node not registered")
	}
}

func TestPhase3NodeRegistration(t *testing.T) {
	registry := workflow.NewRegistry()
	RegisterAll(registry)

	// Verify new nodes are registered
	for _, nodeType := range []string{"factor", "strategy", "backtest"} {
		if !registry.Has(nodeType) {
			t.Errorf("node type %q not registered", nodeType)
		} else {
			t.Logf("Node %q: registered", nodeType)
		}
	}
}
