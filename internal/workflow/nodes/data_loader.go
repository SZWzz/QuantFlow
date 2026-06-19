package nodes

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"

	"quantflow/internal/market"
	"quantflow/internal/workflow"
)

// DataLoaderNode loads market data from external sources (CSV files for now).
type DataLoaderNode struct {
	id     string
	params map[string]any
}

// NewDataLoaderNode creates a new DataLoaderNode.
func NewDataLoaderNode(id string, params map[string]any) (workflow.BaseNode, error) {
	return &DataLoaderNode{id: id, params: params}, nil
}

func (n *DataLoaderNode) ID() string       { return n.id }
func (n *DataLoaderNode) NodeType() string { return "data_loader" }
func (n *DataLoaderNode) Category() string { return "data" }

func (n *DataLoaderNode) InputPorts() []workflow.PortDefinition { return nil }

func (n *DataLoaderNode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "ohlcv", Type: workflow.PortOHLCV, Required: false},
	}
}

func (n *DataLoaderNode) ParamSchema() []workflow.ParamDef {
	return []workflow.ParamDef{
		{Name: "source", Type: "string", Default: "csv", Description: "Data source type"},
		{Name: "path", Type: "string", Default: "", Description: "Path to CSV file"},
	}
}

func (n *DataLoaderNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any) (map[string]any, error) {
	source := "csv"
	path := ""
	if s, ok := n.params["source"]; ok {
		source = fmt.Sprint(s)
	}
	if p, ok := n.params["path"]; ok {
		path = fmt.Sprint(p)
	}
	if s, ok := params["source"]; ok {
		source = fmt.Sprint(s)
	}
	if p, ok := params["path"]; ok {
		path = fmt.Sprint(p)
	}

	switch source {
	case "csv":
		bars, err := loadCSV(path)
		if err != nil {
			return nil, fmt.Errorf("data_loader: %w", err)
		}
		return map[string]any{"ohlcv": bars}, nil
	default:
		return nil, fmt.Errorf("data_loader: unknown source %q", source)
	}
}

func (n *DataLoaderNode) Validate() error {
	path := ""
	if p, ok := n.params["path"]; ok {
		path = fmt.Sprint(p)
	}
	if path == "" {
		return fmt.Errorf("data_loader: path is required")
	}
	return nil
}

// loadCSV reads an OHLCV CSV file and returns the parsed bars.
func loadCSV(path string) ([]market.OHLCVBar, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open csv: %w", err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	header, err := r.Read()
	if err != nil {
		return nil, fmt.Errorf("read csv header: %w", err)
	}
	colIdx := make(map[string]int)
	for i, col := range header {
		colIdx[col] = i
	}

	var bars []market.OHLCVBar
	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read csv row: %w", err)
		}
		bars = append(bars, market.OHLCVBar{
			Symbol: "CSV",
			Date:   record[colIdx["date"]],
			Open:   parseFloat(record[colIdx["open"]]),
			High:   parseFloat(record[colIdx["high"]]),
			Low:    parseFloat(record[colIdx["low"]]),
			Close:  parseFloat(record[colIdx["close"]]),
			Volume: parseFloat(record[colIdx["volume"]]),
		})
	}
	return bars, nil
}

// parseFloat parses a string to float64, returning 0 on failure.
func parseFloat(s string) float64 {
	v, _ := strconv.ParseFloat(s, 64)
	return v
}
