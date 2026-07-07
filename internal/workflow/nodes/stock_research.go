package nodes

import (
	"context"
	"fmt"
	"log/slog"

	"quantflow/internal/research"
	"quantflow/internal/workflow"
)

// StockResearchNode performs comprehensive stock research across multiple dimensions
// (overview, financials, sentiment, peers, estimates, insider trading).
// Degrades gracefully to mock data when services are not set.
type StockResearchNode struct {
	id     string
	params map[string]any
}

// NewStockResearchNode creates a new StockResearchNode.
func NewStockResearchNode(id string, params map[string]any) (workflow.BaseNode, error) {
	return &StockResearchNode{id: id, params: params}, nil
}

func (n *StockResearchNode) ID() string       { return n.id }
func (n *StockResearchNode) NodeType() string { return "stock_research" }
func (n *StockResearchNode) Category() string { return "research" }

func (n *StockResearchNode) InputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "symbol", Type: workflow.PortString, Required: true},
	}
}

func (n *StockResearchNode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "overview", Type: workflow.PortSeries, Required: false},
		{Name: "financials", Type: workflow.PortSeries, Required: false},
		{Name: "sentiment", Type: workflow.PortSeries, Required: false},
		{Name: "peers", Type: workflow.PortSeries, Required: false},
		{Name: "estimates", Type: workflow.PortSeries, Required: false},
		{Name: "insider", Type: workflow.PortSeries, Required: false},
	}
}

func (n *StockResearchNode) ParamSchema() []workflow.ParamDef {
	return []workflow.ParamDef{
		{Name: "tabs", Type: "string_array", Default: []string{"overview", "financials", "sentiment", "peers", "estimates", "insider"}, Description: "Research tabs to compute"},
	}
}

func (n *StockResearchNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any, nctx *workflow.NodeContext) (map[string]any, error) {
	var financialsService workflow.FinancialsServiceInterface
	var sentimentEngine workflow.SentimentEngineService
	var peerComparisonService workflow.PeerComparisonServiceInterface
	var analystEstimatesService workflow.AnalystEstimatesServiceInterface
	var insiderTradingService workflow.InsiderTradingServiceInterface
	if nctx != nil {
		financialsService = nctx.FinancialsService
		sentimentEngine = nctx.SentimentEngine
		peerComparisonService = nctx.PeerComparisonService
		analystEstimatesService = nctx.AnalystEstimatesService
		insiderTradingService = nctx.InsiderTradingService
	}

	symbol, ok := inputs["symbol"].(string)
	if !ok || symbol == "" {
		return nil, fmt.Errorf("stock_research: missing required input 'symbol'")
	}

	output := map[string]any{}

	output["overview"] = map[string]any{
		"symbol": symbol, "name": symbol, "sector": "Technology",
		"industry": "Software", "market_cap": 2.5e12, "source": "mock",
	}

	if financialsService != nil {
		fd, _ := financialsService.GetFinancials(ctx, symbol)
		ratios := financialsService.ComputeRatios(fd)
		output["financials"] = map[string]any{"data": fd, "ratios": ratios, "source": "mock"}
	} else {
		output["financials"] = map[string]any{"source": "mock", "data": nil}
	}

	if sentimentEngine != nil {
		s, err := sentimentEngine.AnalyzeSentiment(ctx, symbol, "", "news", "en")
		if err == nil {
			output["sentiment"] = s
		} else {
			output["sentiment"] = map[string]any{"label": "neutral", "source": "mock"}
		}
	} else {
		output["sentiment"] = map[string]any{"label": "neutral", "source": "mock"}
	}

	if peerComparisonService != nil {
		peers, _ := peerComparisonService.GetPeers(ctx, symbol)
		output["peers"] = peers
	} else {
		output["peers"] = []research.PeerComparisonData{}
	}

	if analystEstimatesService != nil {
		est, _ := analystEstimatesService.GetEstimates(ctx, symbol)
		output["estimates"] = est
	} else {
		output["estimates"] = []research.AnalystEstimate{}
	}

	if insiderTradingService != nil {
		txns, _ := insiderTradingService.GetInsiderTrades(ctx, symbol)
		output["insider"] = txns
	} else {
		output["insider"] = []research.InsiderTransaction{}
	}

	slog.Debug("stock_research completed", "symbol", symbol)
	return output, nil
}

func (n *StockResearchNode) Validate() error { return nil }
