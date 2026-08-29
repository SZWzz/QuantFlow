package capabilities

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"quantflow/internal/ai"
	"quantflow/internal/market"
	"strings"
)

// QuoteResult holds a stock quote returned by the quote_lookup capability.
type QuoteResult struct {
	Symbol string  `json:"symbol"`
	Price  float64 `json:"price"`
	Change float64 `json:"change"`
}

// marketReg is the package-level market adapter registry, set via SetMarketRegistry.
var marketReg *market.AdapterRegistry

// SetMarketRegistry sets the market adapter registry for quote capabilities.
func SetMarketRegistry(reg *market.AdapterRegistry) {
	marketReg = reg
}

// RegisterQuoteCapabilities registers quote_lookup and search_symbol capabilities.
func RegisterQuoteCapabilities(reg *ai.CapabilityRegistry) {
	if err := reg.Register(&ai.Capability{
		Name:        "quote_lookup",
		Description: "Get the current price and daily change for a stock symbol. Use this to check real-time market data.",
		Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {
					"symbol": {"type": "string", "description": "Stock ticker symbol, e.g. AAPL, 000001.SZ, 600519.SH"}
				},
				"required": ["symbol"]
			}`),
		Handler: func(ctx context.Context, args json.RawMessage) (string, error) {
			var params struct {
				Symbol string `json:"symbol"`
			}
			if err := json.Unmarshal(args, &params); err != nil {
				return "", fmt.Errorf("quote_lookup: %w", err)
			}
			if params.Symbol == "" {
				return "", fmt.Errorf("quote_lookup: symbol is required")
			}

			// Try real market data via AdapterRegistry; fall back to note.
			if marketReg != nil {
				marketName := market.MarketForSymbol(params.Symbol)
				quote, adapter, err := marketReg.FetchQuoteWithFallback(ctx, marketName, params.Symbol)
				if err == nil && quote != nil {
					return fmt.Sprintf("%s (via %s): price=%.2f, change=%.2f%%, open=%.2f, high=%.2f, low=%.2f, volume=%.0f",
						strings.ToUpper(params.Symbol), adapter,
						quote.Last, quote.ChangePct, quote.Open, quote.High, quote.Low, quote.Volume), nil
				}
			}

			return fmt.Sprintf("Quote for %s: real-time market data not available (market registry not wired).",
				strings.ToUpper(params.Symbol)), nil
		},
	}); err != nil {
		slog.Error("register capability failed", "name", "quote_lookup", "error", err)
	}

	if err := reg.Register(&ai.Capability{
		Name:        "search_symbol",
		Description: "Search for stock symbols by company name or ticker. Returns matching symbols.",
		Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {
					"query": {"type": "string", "description": "Company name or ticker to search for"}
				},
				"required": ["query"]
			}`),
		Handler: func(ctx context.Context, args json.RawMessage) (string, error) {
			var params struct {
				Query string `json:"query"`
			}
			if err := json.Unmarshal(args, &params); err != nil {
				return "", fmt.Errorf("search_symbol: %w", err)
			}
			query := strings.ToUpper(params.Query)
			// Common symbol mappings
			known := map[string]string{
				"APPLE":     "AAPL",
				"GOOGLE":    "GOOGL",
				"MICROSOFT": "MSFT",
				"TESLA":     "TSLA",
				"NVIDIA":    "NVDA",
				"阿里巴巴":      "BABA",
				"腾讯":        "0700.HK",
				"茅台":        "600519.SH",
				"平安":        "000001.SZ",
				"比亚迪":       "002594.SZ",
			}
			var matches []string
			for key, sym := range known {
				if strings.Contains(key, query) || strings.Contains(sym, query) {
					matches = append(matches, fmt.Sprintf("%s (%s)", sym, key))
				}
			}
			if len(matches) == 0 {
				return fmt.Sprintf("No symbols found for %q. Try company name or known ticker.", params.Query), nil
			}
			result, err := json.Marshal(matches)
			if err != nil {
				return "", fmt.Errorf("search_symbol: marshal: %w", err)
			}
			return string(result), nil
		},
	}); err != nil {
		slog.Error("register capability failed", "name", "search_symbol", "error", err)
	}
}
