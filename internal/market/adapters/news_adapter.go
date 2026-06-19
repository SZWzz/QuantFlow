// Package adapters provides market data, news, and research adapters.
package adapters

import "context"

// NewsArticle represents a single news article fetched from a data source.
type NewsArticle struct {
	Symbol  string `json:"symbol"`
	Title   string `json:"title"`
	Content string `json:"content"`
	Time    string `json:"time"`
	Source  string `json:"source"`
	URL     string `json:"url,omitempty"`
}

// NewsAdapter fetches financial news for sentiment analysis and NLP.
// Separate from market.Adapter (which is for quotes/OHLCV) because news
// has a different data shape and lifecycle.
type NewsAdapter interface {
	// Name returns the adapter's unique identifier (e.g. "eastmoney_news").
	Name() string

	// IsAvailable checks whether the upstream news API is reachable.
	IsAvailable(ctx context.Context) bool

	// FetchStockNews fetches recent news articles for a single stock.
	// limit caps the number of articles returned (max 50).
	FetchStockNews(ctx context.Context, symbol string, limit int) ([]NewsArticle, error)
}

// GlobalNewsAdapter extends NewsAdapter with market-wide news.
type GlobalNewsAdapter interface {
	NewsAdapter

	// FetchGlobalNews fetches general market news (7×24 rolling feed).
	FetchGlobalNews(ctx context.Context, limit int) ([]NewsArticle, error)
}
