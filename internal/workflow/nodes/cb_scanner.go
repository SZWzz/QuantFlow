package nodes

import (
	"context"
	"fmt"
	"quantflow/internal/market/adapters"
	"quantflow/internal/research"
	"quantflow/internal/workflow"
	"sort"
)

// CBScannerNode scans convertible bonds using dual-low strategy criteria.
type CBScannerNode struct {
	id      string
	adapter *adapters.EastMoneyCBAdapter
}

// NewCBScannerNode creates a convertible bond scanner node.
func NewCBScannerNode(id string, params map[string]any) (workflow.BaseNode, error) {
	return &CBScannerNode{
		id:      id,
		adapter: adapters.NewEastMoneyCBAdapter(),
	}, nil
}

func (n *CBScannerNode) ID() string                            { return n.id }
func (n *CBScannerNode) NodeType() string                      { return "cb_scanner" }
func (n *CBScannerNode) Category() string                      { return "research" }
func (n *CBScannerNode) InputPorts() []workflow.PortDefinition { return nil }
func (n *CBScannerNode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "results", Type: workflow.PortAny},
		{Name: "top_symbols", Type: workflow.PortString},
		{Name: "top_scores", Type: workflow.PortNumber},
	}
}

func (n *CBScannerNode) ParamSchema() []workflow.ParamDef {
	return []workflow.ParamDef{
		{Name: "max_price", Type: "float", Default: 150.0, Description: "最高转债价格"},
		{Name: "max_premium_rate", Type: "float", Default: 30.0, Description: "最高转股溢价率 (%)"},
		{Name: "min_years", Type: "float", Default: 0.5, Description: "最短剩余年限"},
		{Name: "top_n", Type: "int", Default: 10.0, Description: "返回前 N 只"},
	}
}

func (n *CBScannerNode) Validate() error { return nil }

func (n *CBScannerNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any, nctx *workflow.NodeContext) (map[string]any, error) {
	maxPrice := getFloatParam(params, "max_price", 150.0)
	maxPremium := getFloatParam(params, "max_premium_rate", 30.0)
	minYears := getFloatParam(params, "min_years", 0.5)
	topN := getIntParam(params, "top_n", 10)

	// Fetch CB data
	quotes, err := n.adapter.FetchCBList(ctx)
	if err != nil {
		return nil, fmt.Errorf("cb_scanner: fetch failed: %w", err)
	}

	// Analyze and rank
	analyzer := research.NewCBAnalyzer()
	results := analyzer.Screen(quotes, maxPrice, maxPremium, minYears)

	// Cap results
	if len(results) > topN {
		results = results[:topN]
	}

	// Extract top symbols and scores
	topSymbols := make([]string, 0, len(results))
	topScores := make([]float64, 0, len(results))
	for _, r := range results {
		topSymbols = append(topSymbols, r.Quote.Code)
		topScores = append(topScores, r.DualLowScore)
	}

	sort.Float64s(topScores)

	return map[string]any{
		"results":     results,
		"top_symbols": topSymbols,
		"top_scores":  topScores,
	}, nil
}
