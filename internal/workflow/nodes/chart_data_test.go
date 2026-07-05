package nodes

import (
	"context"
	"encoding/json"
	"testing"
)

func TestChartDataNode_Line(t *testing.T) {
	node, err := NewChartDataNode("cd1", nil)
	if err != nil {
		t.Fatalf("NewChartDataNode() error = %v", err)
	}
	if node.NodeType() != "chart_data" {
		t.Errorf("NodeType() = %q, want %q", node.NodeType(), "chart_data")
	}
	outputs, err := node.Execute(context.Background(), map[string]any{"data": []float64{1, 2, 3}}, map[string]any{"chart_type": "line", "title": "Test"}, nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	jsonStr, ok := outputs["chart_json"].(string)
	if !ok {
		t.Fatalf("expected string, got %T", outputs["chart_json"])
	}
	var parsed map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		t.Fatalf("invalid JSON output: %v", err)
	}
	series, ok := parsed["series"].([]any)
	if !ok || len(series) == 0 {
		t.Fatal("expected non-empty series array")
	}
	first := series[0].(map[string]any)
	if first["type"] != "line" {
		t.Errorf("series type = %v, want 'line'", first["type"])
	}
}

func TestChartDataNode_MissingInput(t *testing.T) {
	node, _err := NewChartDataNode("cd1", nil)
	if _err != nil {
		t.Fatalf("NewChartDataNode() error = %v", _err)
	}
	_, err := node.Execute(context.Background(), map[string]any{}, nil, nil)
	if err == nil {
		t.Error("expected error for missing 'data' input")
	}
}

func TestChartDataNode_InvalidType(t *testing.T) {
	node, _err := NewChartDataNode("cd1", nil)
	if _err != nil {
		t.Fatalf("NewChartDataNode() error = %v", _err)
	}
	_, err := node.Execute(context.Background(), map[string]any{"data": []float64{1}}, map[string]any{"chart_type": "invalid"}, nil)
	if err == nil {
		t.Error("expected error for invalid chart_type")
	}
}
