package nodes

import (
	"context"
	"testing"
)

func TestSatelliteNode_MockData(t *testing.T) {
	node, err := NewSatelliteNode("sat1", nil)
	if err != nil {
		t.Fatalf("NewSatelliteNode() error = %v", err)
	}
	if node.NodeType() != "satellite" {
		t.Errorf("NodeType() = %q, want %q", node.NodeType(), "satellite")
	}
	outputs, err := node.Execute(context.Background(), map[string]any{}, nil, nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	energySignal, ok := outputs["energy_signal"].(map[string]any)
	if !ok {
		t.Fatalf("expected map energy_signal, got %T", outputs["energy_signal"])
	}
	if _, ok := energySignal["action"]; !ok {
		t.Error("energy_signal missing 'action'")
	}
	solarGHI, ok := outputs["solar_ghi"].(float64)
	if !ok || solarGHI < 0 {
		t.Errorf("solar_ghi = %v, want >= 0", solarGHI)
	}
}

func TestSatelliteNode_WithRegion(t *testing.T) {
	node, _err := NewSatelliteNode("sat1", nil)
	if _err != nil {
		t.Fatalf("NewSatelliteNode() error = %v", _err)
	}
	outputs, err := node.Execute(context.Background(), map[string]any{"region": "texas"}, nil, nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if _, ok := outputs["energy_signal"]; !ok {
		t.Error("expected energy_signal in output")
	}
}
