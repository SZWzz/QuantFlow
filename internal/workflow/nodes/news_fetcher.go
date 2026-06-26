package nodes

import (
	"context"
	"fmt"
	"strings"

	"quantflow/internal/market/adapters"
	"quantflow/internal/workflow"
)

// NewsFetcherNode fetches financial news for a stock symbol and outputs
// the concatenated text for downstream NLP sentiment analysis.
//
// Inputs:  symbol (string, required) — 6-digit CN stock code
// Outputs: news_text (string) — concatenated article titles + content for NLP
//          articles  ([]adapters.NewsArticle) — structured article list
//
// Typical chain: NewsFetcherNode → SentimentNode (connect news_text port)
type NewsFetcherNode struct {
	id     string
	params map[string]any
}

// Compile-time check.
var _ workflow.BaseNode = (*NewsFetcherNode)(nil)

// NewNewsFetcherNode creates a new NewsFetcherNode.
func NewNewsFetcherNode(id string, params map[string]any) (workflow.BaseNode, error) {
	return &NewsFetcherNode{id: id, params: params}, nil
}

func (n *NewsFetcherNode) ID() string       { return n.id }
func (n *NewsFetcherNode) NodeType() string { return "news_fetcher" }
func (n *NewsFetcherNode) Category() string { return "research" }

func (n *NewsFetcherNode) InputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "symbol", Type: workflow.PortString, Required: true},
		{Name: "limit", Type: workflow.PortNumber, Required: false},
	}
}

func (n *NewsFetcherNode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "news_text", Type: workflow.PortString, Required: false},
		{Name: "articles", Type: workflow.PortSeries, Required: false},
	}
}

func (n *NewsFetcherNode) ParamSchema() []workflow.ParamDef {
	return []workflow.ParamDef{
		{Name: "source", Type: "string", Default: "eastmoney", Description: "News source: eastmoney"},
		{Name: "language", Type: "string", Default: "zh", Description: "News language: zh, en"},
	}
}

// Execute fetches news and concatenates the text for NLP consumption.
func (n *NewsFetcherNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any, nctx *workflow.NodeContext) (map[string]any, error) {
	symbol, ok := inputs["symbol"].(string)
	if !ok || symbol == "" {
		return nil, fmt.Errorf("news_fetcher: missing required input 'symbol'")
	}

	limit := 5
	if l, ok := inputs["limit"].(float64); ok && l > 0 {
		limit = int(l)
	}

	var articles []adapters.NewsArticle
	var err error

	if newsAdapter != nil {
		articles, err = newsAdapter.FetchStockNews(ctx, symbol, limit)
	} else {
		articles = mockNewsArticles(symbol, limit)
	}
	if err != nil || len(articles) == 0 {
		articles = mockNewsArticles(symbol, limit)
	}

	// Concatenate titles and content into a single text blob for NLP
	parts := make([]string, 0, len(articles)*2)
	for _, a := range articles {
		if a.Title != "" {
			parts = append(parts, a.Title)
		}
		if a.Content != "" {
			parts = append(parts, a.Content)
		}
	}
	newsText := strings.Join(parts, ". ")

	return map[string]any{
		"news_text": newsText,
		"articles":  articles,
	}, nil
}

func (n *NewsFetcherNode) Validate() error { return nil }

// mockNewsArticles provides fallback mock news when no adapter is available.
func mockNewsArticles(symbol string, limit int) []adapters.NewsArticle {
	return []adapters.NewsArticle{
		{
			Symbol:  symbol,
			Title:   fmt.Sprintf("Mock news for %s", symbol),
			Content: "No live news adapter configured. This is mock data for development.",
			Time:    "2026-01-01",
			Source:  "mock",
		},
	}
}
