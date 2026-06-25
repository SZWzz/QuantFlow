package market

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
)

// FallbackChains defines the priority-ordered list of adapter names for each market.
// The first available adapter in the chain is used; if it fails, the next is tried.
// Design aligned with astockpursue's FALLBACK_CHAINS.
var FallbackChains = map[string][]string{
	"CN":     {"tencent", "eastmoney", "mootdx", "sina", "tushare", "baidu", "akshare"},
	"US":     {"yahoo", "sina", "polygon", "finnhub"},
	"HK":     {"yahoo", "tencent", "sina", "akshare"},
	"CRYPTO": {"binance", "okx", "coingecko"},
}

// AdapterRegistry manages registered market data adapters and provides
// fallback-based fetching.
type AdapterRegistry struct {
	mu       sync.RWMutex
	adapters map[string]Adapter // name → adapter
}

// NewAdapterRegistry creates an empty AdapterRegistry.
func NewAdapterRegistry() *AdapterRegistry {
	return &AdapterRegistry{
		adapters: make(map[string]Adapter),
	}
}

// Register adds an adapter to the registry.
func (r *AdapterRegistry) Register(a Adapter) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.adapters[a.Name()] = a
}

// Get returns an adapter by name, or nil if not registered.
func (r *AdapterRegistry) Get(name string) Adapter {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.adapters[name]
}

// List returns all registered adapter names.
func (r *AdapterRegistry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.adapters))
	for name := range r.adapters {
		names = append(names, name)
	}
	return names
}

// Count returns the number of registered adapters.
func (r *AdapterRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.adapters)
}

// FetchQuoteWithFallback tries each adapter in the market's fallback chain
// until one succeeds. Returns the quote, the adapter name used, and any error.
func (r *AdapterRegistry) FetchQuoteWithFallback(ctx context.Context, market, symbol string) (*QuoteSnapshot, string, error) {
	chain, ok := FallbackChains[market]
	if !ok {
		return nil, "", fmt.Errorf("unknown market: %q", market)
	}

	var lastErr error
	for _, name := range chain {
		adapter := r.Get(name)
		if adapter == nil {
			slog.Debug("adapter not registered, skipping", "name", name, "market", market)
			continue
		}
		if !adapter.IsAvailable(ctx) {
			slog.Debug("adapter unavailable, skipping", "name", name, "market", market)
			continue
		}

		quote, err := RetryWithBudget(
			func() (*QuoteSnapshot, error) { return adapter.FetchQuote(ctx, symbol) },
			DefaultRetryConfig(name),
		)
		if err != nil {
			slog.Warn("adapter fetch quote failed, trying next", "name", name, "symbol", symbol, "error", err)
			lastErr = err
			continue
		}

		slog.Debug("quote fetched", "adapter", name, "symbol", symbol, "price", quote.Last)
		return quote, name, nil
	}

	if lastErr != nil {
		return nil, "", fmt.Errorf("all adapters failed for market %q symbol %q: %w", market, symbol, lastErr)
	}
	return nil, "", fmt.Errorf("no available adapter found for market %q symbol %q (chain: %v)", market, symbol, chain)
}

// FetchOHLCVWithFallback tries each adapter in the market's fallback chain
// until one succeeds. Returns OHLCV bars, the adapter name, and any error.
func (r *AdapterRegistry) FetchOHLCVWithFallback(ctx context.Context, market, symbol, interval string, start, end int64) ([]OHLCVBar, string, error) {
	interval = NormalizeInterval(interval)

	chain, ok := FallbackChains[market]
	if !ok {
		return nil, "", fmt.Errorf("unknown market: %q", market)
	}

	var lastErr error
	for _, name := range chain {
		adapter := r.Get(name)
		if adapter == nil {
			continue
		}
		if !adapter.IsAvailable(ctx) {
			continue
		}

		bars, err := RetryWithBudget(
			func() ([]OHLCVBar, error) { return adapter.FetchOHLCV(ctx, symbol, interval, start, end) },
			DefaultRetryConfig(name),
		)
		if err != nil {
			slog.Warn("adapter fetch ohlcv failed, trying next", "name", name, "symbol", symbol, "error", err)
			lastErr = err
			continue
		}

		return bars, name, nil
	}

	if lastErr != nil {
		return nil, "", fmt.Errorf("all adapters failed for market %q symbol %q: %w", market, symbol, lastErr)
	}
	return nil, "", fmt.Errorf("no available adapter found for market %q symbol %q", market, symbol)
}

// MarketForSymbol infers the market type from a symbol's suffix/prefix.
//
// Rules:
//   - .SZ / .SH / .BJ → CN (A-shares, including Beijing Stock Exchange)
//   - .HK → HK (Hong Kong)
//   - USDT / USDC / BTC / ETH / SOL / BNB suffix or bare base → CRYPTO
//   - Everything else → US (default)
func MarketForSymbol(symbol string) string {
	if len(symbol) >= 3 {
		suffix := symbol[len(symbol)-3:]
		if suffix == ".SZ" || suffix == ".SH" || suffix == ".BJ" {
			return "CN"
		}
		if suffix == ".HK" {
			return "HK"
		}
	}
	// Crypto: ends with a known stablecoin/base marker, or is a bare base.
	cryptoMarkers := []string{"USDT", "USDC", "BTC", "ETH", "SOL", "BNB"}
	for _, m := range cryptoMarkers {
		if len(symbol) >= len(m) && symbol[len(symbol)-len(m):] == m {
			return "CRYPTO"
		}
	}
	// Bare crypto base symbols (e.g., "BTC", "ETH")
	switch symbol {
	case "BTC", "ETH", "SOL", "BNB":
		return "CRYPTO"
	}
	return "US"
}
