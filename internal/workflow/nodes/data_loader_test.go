package nodes

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"quantflow/internal/market"
)

func TestDataLoaderNode_CSV(t *testing.T) {
	tmp := t.TempDir()
	csvPath := filepath.Join(tmp, "test.csv")
	os.WriteFile(csvPath, []byte("date,open,high,low,close,volume\n2024-01-01,100,110,95,105,1000\n2024-01-02,105,115,100,110,1200\n"), 0644)

	node, err := NewDataLoaderNode("loader1", map[string]any{"source": "csv", "path": csvPath})
	if err != nil {
		t.Fatalf("NewDataLoaderNode() error = %v", err)
	}

	outputs, err := node.Execute(context.Background(), nil, nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	ohlcv, ok := outputs["ohlcv"]
	if !ok {
		t.Fatal("missing 'ohlcv' output")
	}
	data, ok := ohlcv.([]market.OHLCVBar)
	if !ok {
		t.Fatalf("ohlcv is %T, want []market.OHLCVBar", ohlcv)
	}
	if len(data) != 2 {
		t.Fatalf("len = %d, want 2", len(data))
	}
	if data[0].Close != 105 {
		t.Errorf("data[0].Close = %v, want 105", data[0].Close)
	}
	if data[1].Volume != 1200 {
		t.Errorf("data[1].Volume = %v, want 1200", data[1].Volume)
	}
}

func TestDataLoaderNode_PortDefinitions(t *testing.T) {
	node, _ := NewDataLoaderNode("dl", map[string]any{"source": "csv", "path": "dummy.csv"})
	inputs := node.InputPorts()
	if len(inputs) != 0 {
		t.Errorf("InputPorts should be empty, got %d", len(inputs))
	}
	outputs := node.OutputPorts()
	if len(outputs) != 1 || outputs[0].Name != "ohlcv" {
		t.Errorf("OutputPorts: %+v, want 1 port named 'ohlcv'", outputs)
	}
}
