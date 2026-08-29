//go:build integration
// +build integration

package workflow_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"quantflow/internal/workflow"
	"quantflow/internal/workflow/nodes"
)

func TestIntegration_SMACrossFromFile(t *testing.T) {
	tmp := t.TempDir()
	csvPath := filepath.Join(tmp, "data.csv")
	os.WriteFile(csvPath, []byte(`date,open,high,low,close,volume
2024-01-01,100,110,95,105,1000
2024-01-02,105,115,100,110,1200
2024-01-03,110,120,105,115,1300
2024-01-04,115,125,110,120,1400
2024-01-05,120,130,115,125,1500`), 0o644)

	reg := workflow.NewRegistry()
	reg.RegisterWithCategory("data_loader", nodes.NewDataLoaderNode, "data")
	reg.RegisterWithCategory("sma", nodes.NewSMANode, "indicator")
	reg.RegisterWithCategory("cross_signal", nodes.NewCrossSignalNode, "signal")
	reg.RegisterWithCategory("log_output", nodes.NewLogOutputNode, "output")

	engine, err := workflow.NewEngine(reg, 64)
	if err != nil {
		t.Fatalf("NewEngine() error = %v", err)
	}

	wf := &workflow.Workflow{
		ID:   "integration-test",
		Name: "SMA Cross Integration",
		Nodes: []workflow.NodeInstance{
			{ID: "loader", NodeType: "data_loader", Params: map[string]any{"source": "csv", "path": csvPath}},
			{ID: "fast", NodeType: "sma", Params: map[string]any{"period": 2}},
			{ID: "slow", NodeType: "sma", Params: map[string]any{"period": 3}},
			{ID: "signal", NodeType: "cross_signal"},
			{ID: "log", NodeType: "log_output"},
		},
		Edges: []workflow.Edge{
			{FromNode: "loader", FromPort: "ohlcv", ToNode: "fast", ToPort: "input"},
			{FromNode: "loader", FromPort: "ohlcv", ToNode: "slow", ToPort: "input"},
			{FromNode: "fast", FromPort: "output", ToNode: "signal", ToPort: "fast"},
			{FromNode: "slow", FromPort: "output", ToNode: "signal", ToPort: "slow"},
			{FromNode: "signal", FromPort: "signal", ToNode: "log", ToPort: "input"},
		},
	}

	result, err := engine.Execute(context.Background(), wf)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.Status != "completed" {
		t.Errorf("Status = %q, want completed", result.Status)
	}

	var signalResult *workflow.NodeResult
	for i := range result.NodeResults {
		if result.NodeResults[i].NodeID == "signal" {
			signalResult = &result.NodeResults[i]
			break
		}
	}
	if signalResult == nil {
		t.Fatal("missing signal node result")
	}
	if signalResult.Status != "completed" {
		t.Errorf("signal status = %q, want completed", signalResult.Status)
	}
}
