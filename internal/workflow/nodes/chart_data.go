package nodes

import (
	"context"
	"encoding/json"
	"fmt"

	"quantflow/internal/workflow"
)

// ChartDataNode converts input series data into an ECharts-ready JSON string.
// Supports line, bar, pie, and scatter chart types.
type ChartDataNode struct {
	id     string
	params map[string]any
}

// NewChartDataNode creates a new ChartDataNode.
func NewChartDataNode(id string, params map[string]any) (workflow.BaseNode, error) {
	return &ChartDataNode{id: id, params: params}, nil
}

func (n *ChartDataNode) ID() string       { return n.id }
func (n *ChartDataNode) NodeType() string { return "chart_data" }
func (n *ChartDataNode) Category() string { return "output" }

func (n *ChartDataNode) InputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "data", Type: workflow.PortSeries, Required: true},
	}
}

func (n *ChartDataNode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "chart_json", Type: workflow.PortString, Required: false},
	}
}

func (n *ChartDataNode) ParamSchema() []workflow.ParamDef {
	return []workflow.ParamDef{
		{Name: "chart_type", Type: "string", Default: "line",
			Description: "Chart type: line, bar, pie, scatter"},
		{Name: "title", Type: "string", Default: "",
			Description: "Chart title text"},
	}
}

// echartsOption is the minimal ECharts option structure we emit.
type echartsOption struct {
	Title  echartsTitle    `json:"title"`
	XAxis  echartsXAxis    `json:"xAxis"`
	Series []echartsSeries `json:"series"`
}

type echartsTitle struct {
	Text string `json:"text,omitempty"`
}

type echartsXAxis struct {
	Type string   `json:"type"`
	Data []string `json:"data,omitempty"`
}

type echartsSeries struct {
	Type string    `json:"type"`
	Data []float64 `json:"data"`
	Name string    `json:"name,omitempty"`
}

func (n *ChartDataNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any) (map[string]any, error) {
	chartType := getStringParam(params, "chart_type", "line")
	title := getStringParam(params, "title", "")

	validTypes := map[string]bool{"line": true, "bar": true, "pie": true, "scatter": true}
	if !validTypes[chartType] {
		return nil, fmt.Errorf("chart_data: unknown chart_type %q, expected line/bar/pie/scatter", chartType)
	}

	data := extractFloatSlice(inputs["data"])
	if data == nil {
		return nil, fmt.Errorf("chart_data: data input is required")
	}

	// Build x-axis labels (index-based)
	xLabels := make([]string, len(data))
	for i := range data {
		xLabels[i] = fmt.Sprintf("%d", i)
	}

	option := echartsOption{
		Title: echartsTitle{Text: title},
		XAxis: echartsXAxis{Type: "category", Data: xLabels},
		Series: []echartsSeries{
			{Type: chartType, Data: data},
		},
	}

	jsonBytes, err := json.Marshal(option)
	if err != nil {
		return nil, fmt.Errorf("chart_data: failed to marshal JSON: %w", err)
	}

	return map[string]any{"chart_json": string(jsonBytes)}, nil
}

func (n *ChartDataNode) Validate() error {
	chartType := getStringParam(n.params, "chart_type", "line")
	validTypes := map[string]bool{"line": true, "bar": true, "pie": true, "scatter": true}
	if !validTypes[chartType] {
		return fmt.Errorf("chart_data: invalid chart_type %q, expected line/bar/pie/scatter", chartType)
	}
	return nil
}
