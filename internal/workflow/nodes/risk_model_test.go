package nodes

import (
	"context"
	"math"
	"testing"

	"quantflow/internal/normalize"
	"quantflow/internal/workflow"
)

func TestRiskModelNode_Validate(t *testing.T) {
	tests := []struct {
		modelType string
		valid     bool
	}{
		{"garch", true},
		{"gjr_garch", true},
		{"egarch", true},
		{"covariance", true},
		{"invalid", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.modelType, func(t *testing.T) {
			params := map[string]any{"model_type": tt.modelType}
			node, err := NewRiskModelNode("test-1", params)
			if tt.valid {
				if err != nil {
					t.Errorf("expected valid, got error: %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("expected error for invalid type %q", tt.modelType)
				}
			}
			if node != nil && node.ID() != "test-1" {
				t.Errorf("expected id 'test-1', got %q", node.ID())
			}
		})
	}
}

func TestRiskModelNode_PortDefinitions(t *testing.T) {
	node, err := NewRiskModelNode("test-1", map[string]any{"model_type": "garch"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if node.NodeType() != "risk_model" {
		t.Errorf("NodeType() = %q, want 'risk_model'", node.NodeType())
	}
	if node.Category() != "risk" {
		t.Errorf("Category() = %q, want 'risk'", node.Category())
	}

	inputs := node.InputPorts()
	if len(inputs) != 1 || inputs[0].Name != "returns_data" {
		t.Errorf("InputPorts() missing 'returns_data' port")
	}

	outputs := node.OutputPorts()
	outNames := make(map[string]bool)
	for _, o := range outputs {
		outNames[o.Name] = true
	}
	for _, name := range []string{"volatility", "covariance_matrix", "model_metrics"} {
		if !outNames[name] {
			t.Errorf("OutputPorts() missing %q port", name)
		}
	}

	params := node.ParamSchema()
	paramNames := make(map[string]bool)
	for _, p := range params {
		paramNames[p.Name] = true
	}
	for _, name := range []string{"model_type", "method", "p", "q"} {
		if !paramNames[name] {
			t.Errorf("ParamSchema() missing %q param", name)
		}
	}
}

func TestRiskModelNode_Execute_FallbackVolatility(t *testing.T) {
	bars := []normalize.OHLCVBar{
		{Symbol: "TEST", Date: "2024-01-01", Open: 100, High: 101, Low: 99, Close: 100, Volume: 1000},
		{Symbol: "TEST", Date: "2024-01-02", Open: 101, High: 102, Low: 100, Close: 101, Volume: 1000},
		{Symbol: "TEST", Date: "2024-01-03", Open: 102, High: 103, Low: 101, Close: 102, Volume: 1000},
	}

	node, err := NewRiskModelNode("test-1", map[string]any{"model_type": "garch"})
	if err != nil {
		t.Fatalf("NewRiskModelNode: %v", err)
	}

	outputs, err := node.Execute(context.Background(), map[string]any{
		"returns_data": bars,
	}, map[string]any{"model_type": "garch"}, &workflow.NodeContext{})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	vol, ok := outputs["volatility"].([]float64)
	if !ok || len(vol) == 0 {
		t.Fatal("expected volatility []float64 output")
	}

	metrics, ok := outputs["model_metrics"].(map[string]float64)
	if !ok {
		t.Fatal("expected model_metrics map output")
	}

	if metrics["data_points"] != 2 {
		t.Errorf("expected 2 data_points (returns from 3 bars), got %v", metrics["data_points"])
	}
}

func TestRiskModelNode_Execute_PreComputedReturns(t *testing.T) {
	returns := []float64{0.01, -0.005, 0.02, -0.01}

	node, _ := NewRiskModelNode("test-1", map[string]any{"model_type": "covariance"})
	outputs, err := node.Execute(context.Background(), map[string]any{
		"returns_data": returns,
	}, map[string]any{"model_type": "covariance", "method": "ledoit_wolf"}, &workflow.NodeContext{})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	vol := outputs["volatility"].([]float64)
	if len(vol) == 0 {
		t.Fatal("expected volatility output")
	}
}

func TestRiskModelNode_Execute_InsufficientData(t *testing.T) {
	node, _ := NewRiskModelNode("test-1", map[string]any{"model_type": "garch"})

	bars := []normalize.OHLCVBar{
		{Symbol: "TEST", Date: "2024-01-01", Open: 100, High: 101, Low: 99, Close: 100, Volume: 1000},
	}

	_, err := node.Execute(context.Background(), map[string]any{
		"returns_data": bars,
	}, map[string]any{"model_type": "garch"}, &workflow.NodeContext{})

	if err == nil {
		t.Fatal("expected error for insufficient data")
	}
}

func TestRiskModelNode_ParamDefaults(t *testing.T) {
	node, err := NewRiskModelNode("test-1", map[string]any{})
	if err != nil {
		t.Fatalf("NewRiskModelNode with empty params: %v", err)
	}
	if node.NodeType() != "risk_model" {
		t.Errorf("NodeType() = %q", node.NodeType())
	}
}

func TestStdDev(t *testing.T) {
	tests := []struct {
		name     string
		values   []float64
		expected float64
	}{
		{"empty", []float64{}, 0},
		{"single", []float64{5.0}, 0},
		{"two", []float64{1.0, 3.0}, 1.0},
		{"uniform", []float64{2.0, 2.0, 2.0}, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stdDev(tt.values)
			if math.Abs(got-tt.expected) > 1e-9 {
				t.Errorf("stdDev(%v) = %v, want %v", tt.values, got, tt.expected)
			}
		})
	}
}

func TestReturnsFromBars(t *testing.T) {
	bars := []normalize.OHLCVBar{
		{Symbol: "T", Date: "2024-01-01", Close: 100},
		{Symbol: "T", Date: "2024-01-02", Close: 101},
		{Symbol: "T", Date: "2024-01-03", Close: 99},
	}

	returns := returnsFromBars(bars)
	if len(returns) != 2 {
		t.Fatalf("expected 2 returns, got %d", len(returns))
	}
	if math.Abs(returns[0]-0.01) > 1e-9 {
		t.Errorf("returns[0] = %v, want 0.01", returns[0])
	}
	if math.Abs(returns[1]+0.01980198) > 1e-6 {
		t.Errorf("returns[1] = %v, want ~-0.0198", returns[1])
	}
}
