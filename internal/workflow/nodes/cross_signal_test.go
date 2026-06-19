package nodes

import (
	"context"
	"testing"
)

func TestCrossSignalNode_GoldenCross(t *testing.T) {
	node, _ := NewCrossSignalNode("cs", nil)
	fast := []float64{1, 2, 3, 5, 7}
	slow := []float64{2, 2, 2, 3, 4}
	outputs, err := node.Execute(context.Background(), map[string]any{"fast": fast, "slow": slow}, nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	signals, ok := outputs["signal"].([]SignalEvent)
	if !ok {
		t.Fatalf("signal is %T", outputs["signal"])
	}
	if len(signals) != 1 {
		t.Fatalf("len = %d, want 1", len(signals))
	}
	if signals[0].Direction != "buy" {
		t.Errorf("direction = %q, want buy", signals[0].Direction)
	}
}

func TestCrossSignalNode_DeathCross(t *testing.T) {
	node, _ := NewCrossSignalNode("cs", nil)
	fast := []float64{7, 5, 3, 2, 1}
	slow := []float64{4, 4, 4, 3, 2}
	outputs, err := node.Execute(context.Background(), map[string]any{"fast": fast, "slow": slow}, nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	signals, ok := outputs["signal"].([]SignalEvent)
	if !ok {
		t.Fatalf("signal is %T", outputs["signal"])
	}
	if len(signals) == 0 {
		t.Fatal("expected at least one sell signal")
	}
	if signals[0].Direction != "sell" {
		t.Errorf("direction = %q, want sell", signals[0].Direction)
	}
}
