// Package market provides real-time and historical market data access
// with automatic adapter selection and fallback. Key abstractions:
// Registry, MarketDataHub, and OffHoursCache.
package market

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"
)

// FallbackChains defines the priority-ordered list of adapter names for each market.
// Used by FetchQuoteWithFallback (quotes are symbol-type agnostic).
var FallbackChains = map[string][]string{
	"CN":     {"tencent", "mootdx", "eastmoney", "sina", "tushare", "baidu", "akshare"},
	"US":     {"yahoo", "sina", "finnhub"},
	"HK":     {"yahoo", "tencent", "sina", "akshare"},
	"CRYPTO": {"binance", "okx", "coingecko", "gateio"},
}

// OHLCVChains defines per-market, per-asset-type fallback chains for OHLCV data.
// CN stocks: tencent first (fast HTTP), mootdx as fallback.
// CN indices: tencent first, skip mootdx (TDX doesn't support index K-line).
// Other markets use FallbackChains (no stock/index split).
var OHLCVChains = map[string]map[string][]string{
	"CN": {
		// Mootdx excluded from stock OHLCV: TDX 7709 port's get_security_bars
		// returns None for historical K-lines (only live quote works).
		// Minute data uses Mootdx separately (MinuteChains) where it works.
		"stock": {"tencent", "eastmoney", "sina", "tushare", "baidu", "akshare"},
		"index": {"tencent", "eastmoney", "sina", "tushare", "baidu", "akshare"},
	},
}

// MinuteChains defines per-market, per-asset-type fallback chains for minute data.
// CN stocks: mootdx first (TDX near-real-time, no CDN cache), tencent as fallback.
// Tencent's HTTP API has CDN caching (~30-60s) that prevents 5s poller refresh.
// CN indices: tencent first (TDX doesn't support index minute data).
var MinuteChains = map[string]map[string][]string{
	"CN": {
		"stock": {"mootdx", "tencent"},
		"index": {"tencent", "mootdx"},
	},
}

// IsIndexSymbol detects CN index codes (e.g. 上证指数 000001.SH, 深证成指 399001.SZ).
// Returns false for non-CN symbols and individual stocks (e.g. 600519.SH, 000001.SZ).
func IsIndexSymbol(symbol string) bool {
	if MarketForSymbol(symbol) != "CN" {
		return false
	}
	code := symbol
	mkt := ""
	if len(symbol) >= 3 {
		switch suffix := symbol[len(symbol)-3:]; suffix {
		case ".SH", ".SS":
			code = symbol[:len(symbol)-3]
			mkt = "SH"
		case ".SZ":
			code = symbol[:len(symbol)-3]
			mkt = "SZ"
		case ".BJ":
			return false
		}
	}
	if len(code) != 6 {
		return false
	}
	for _, c := range code {
		if c < '0' || c > '9' {
			return false
		}
	}
	if mkt == "" {
		switch code[0] {
		case '5', '6', '9':
			mkt = "SH"
		case '0', '3':
			mkt = "SZ"
		default:
			return false
		}
	}
	switch mkt {
	case "SH":
		return code[:2] == "00" || code[:2] == "88"
	case "SZ":
		return code[:3] == "399"
	default:
		return false
	}
}

// chainForSymbol selects the asset-type-specific chain for the given symbol.
// If the market has a type-split chain map (e.g. OHLCVChains), it picks stock
// or index; otherwise it falls back to a default chain map (e.g. FallbackChains).
func chainForSymbol(symbol string, typeChains map[string]map[string][]string, defaultChains map[string][]string) []string {
	mkt := MarketForSymbol(symbol)
	if typeMap, ok := typeChains[mkt]; ok {
		if IsIndexSymbol(symbol) {
			if chain, ok := typeMap["index"]; ok {
				return chain
			}
		}
		if chain, ok := typeMap["stock"]; ok {
			return chain
		}
	}
	return defaultChains[mkt]
}

// quoteCacheTTL is the maximum age of a cached quote before it's considered stale.
const quoteCacheTTL = 5 * time.Second

type quoteCacheEntry struct {
	snapshot *QuoteSnapshot
	source   string
	expires  time.Time
}

// AdapterRegistry manages registered market data adapters and provides
// fallback-based fetching.
type AdapterRegistry struct {
	mu            sync.RWMutex
	adapters      map[string]Adapter // name → adapter
	quoteCache    map[string]*quoteCacheEntry
	lastQuote     sync.Map // market:symbol → *QuoteSnapshot (last known value, survives TTL)
	lastQuotePath string   // if set, last quotes are persisted to this JSON file
	saveMu        sync.Mutex
	saveTimer     *time.Timer
	ohlcvCache    *OHLCVCache // optional two-tier (LRU+SQLite) cache
}

// SetOHLCVCache attaches the OHLCV cache to the registry. Must be called
// before any FetchOHLCVWithFallback calls.
func (r *AdapterRegistry) SetOHLCVCache(c *OHLCVCache) {
	r.ohlcvCache = c
}

// SetLastQuotePath sets a file path for persisting last known quotes.
// Call before startup so that persisted quotes are loaded.
func (r *AdapterRegistry) SetLastQuotePath(path string) {
	r.lastQuotePath = path
}

// LoadLastQuotes reads persisted quotes from disk into the lastQuote map.
// No-op if the file does not exist.
func (r *AdapterRegistry) LoadLastQuotes() error {
	if r.lastQuotePath == "" {
		return nil
	}
	b, err := os.ReadFile(r.lastQuotePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("load last quotes: %w", err)
	}
	var data map[string]*QuoteSnapshot
	if err := json.Unmarshal(b, &data); err != nil {
		return fmt.Errorf("unmarshal last quotes: %w", err)
	}
	for k, v := range data {
		r.lastQuote.Store(k, v)
	}
	slog.Info("loaded persisted last quotes", "count", len(data), "path", r.lastQuotePath)
	return nil
}

// saveLastQuotes writes all lastQuote entries to disk atomically.
func (r *AdapterRegistry) saveLastQuotes() {
	if r.lastQuotePath == "" {
		return
	}
	r.saveMu.Lock()
	defer r.saveMu.Unlock()

	data := make(map[string]*QuoteSnapshot)
	r.lastQuote.Range(func(key, value any) bool {
		data[key.(string)] = value.(*QuoteSnapshot)
		return true
	})
	if len(data) == 0 {
		return
	}
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		slog.Warn("marshal last quotes", "error", err)
		return
	}
	tmpPath := r.lastQuotePath + ".tmp"
	if err := os.WriteFile(tmpPath, b, 0644); err != nil {
		slog.Warn("write last quotes tmp", "error", err)
		return
	}
	if err := os.Rename(tmpPath, r.lastQuotePath); err != nil {
		slog.Warn("rename last quotes", "error", err)
	}
	slog.Debug("saved last quotes", "count", len(data), "path", r.lastQuotePath)
}

func (r *AdapterRegistry) debouncedSaveQuotes() {
	r.saveMu.Lock()
	defer r.saveMu.Unlock()
	if r.saveTimer != nil {
		r.saveTimer.Stop()
	}
	r.saveTimer = time.AfterFunc(5*time.Second, func() {
		r.saveLastQuotes()
	})
}

// NewAdapterRegistry creates an empty AdapterRegistry.
func NewAdapterRegistry() *AdapterRegistry {
	return &AdapterRegistry{
		adapters:   make(map[string]Adapter),
		quoteCache: make(map[string]*quoteCacheEntry),
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
	// Check in-memory cache first (5s TTL)
	cacheKey := market + ":" + symbol
	r.mu.RLock()
	if entry, ok := r.quoteCache[cacheKey]; ok && time.Now().Before(entry.expires) {
		r.mu.RUnlock()
		return entry.snapshot, entry.source, nil
	}
	r.mu.RUnlock()

	// Outside trading hours: prefer cached data.  If no cache is available,
	// attempt a live fetch as last resort (adapters may still serve data).
	if !IsTradingHours(market) {
		if last, ok := r.lastQuote.Load(cacheKey); ok {
			return last.(*QuoteSnapshot), "cache", nil
		}
		// Continue to fallback chain — don't return error immediately
	}

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
			slog.Debug("adapter unavailable (probe failed), trying anyway", "name", name, "market", market)
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
		// Cache the result (short TTL) and persist last known value (no expiry)
		r.mu.Lock()
		r.quoteCache[cacheKey] = &quoteCacheEntry{snapshot: quote, source: name, expires: time.Now().Add(quoteCacheTTL)}
		r.mu.Unlock()
		r.lastQuote.Store(cacheKey, quote)
		r.debouncedSaveQuotes()
		return quote, name, nil
	}

	if lastErr != nil {
		return nil, "", fmt.Errorf("all adapters failed for market %q symbol %q: %w", market, symbol, lastErr)
	}
	return nil, "", fmt.Errorf("no available adapter found for market %q symbol %q (chain: %v)", market, symbol, chain)
}

// IndustryRankChains defines the priority-ordered list of adapter names for
// industry/sector ranking data for each market.
var IndustryRankChains = map[string][]string{
	"CN": {"eastmoney_signals"},
	"HK": {"tencent"},
	"US": {"finnhub"},
}

// FetchIndustryRanksWithFallback tries each adapter in the market's industry
// rank chain until one succeeds. Returns the ranked list and any error.
func (r *AdapterRegistry) FetchIndustryRanksWithFallback(ctx context.Context, market string, topN int) ([]IndustryRank, error) {
	chain, ok := IndustryRankChains[market]
	if !ok {
		return nil, fmt.Errorf("no industry rank chain for market %q", market)
	}

	var lastErr error
	for _, name := range chain {
		adapter := r.Get(name)
		if adapter == nil {
			slog.Debug("adapter not registered, skipping", "name", name, "market", market)
			continue
		}
		provider, ok := adapter.(IndustryRankProvider)
		if !ok {
			slog.Debug("adapter does not implement IndustryRankProvider", "name", name)
			continue
		}
		if !adapter.IsAvailable(ctx) {
			slog.Debug("adapter unavailable, skipping", "name", name, "market", market)
			continue
		}

		ranks, err := RetryWithBudget(
			func() ([]IndustryRank, error) { return provider.FetchIndustryRanks(ctx, market, topN) },
			DefaultRetryConfig(name),
		)
		if err != nil {
			slog.Warn("industry rank fetch failed, trying next", "name", name, "market", market, "error", err)
			lastErr = err
			continue
		}

		if len(ranks) > 0 {
			return ranks, nil
		}
	}

	if lastErr != nil {
		return nil, fmt.Errorf("all industry rank adapters failed for market %q: %w", market, lastErr)
	}
	return nil, fmt.Errorf("no available industry rank adapter for market %q (chain: %v)", market, chain)
}

// FetchOHLCVWithFallback tries each adapter in the market's fallback chain
// until one succeeds. Returns OHLCV bars, the adapter name, and any error.
// fqfactor controls price adjustment: "" (不复权), "qfq" (前复权), "hfq" (后复权).
//
// If the first successful adapter returns data that doesn't cover the requested
// start date (e.g. Tencent's 2000-bar cap), the function falls through to the
// next adapter automatically for a more complete dataset.
func (r *AdapterRegistry) FetchOHLCVWithFallback(ctx context.Context, market, symbol, interval, fqfactor string, start, end int64) ([]OHLCVBar, string, error) {
	interval = NormalizeInterval(interval)

	if r.ohlcvCache != nil {
		if cached, err := r.ohlcvCache.Get(symbol, interval, start, end); err == nil && len(cached) > 0 {
			slog.Debug("ohlcv cache hit", "symbol", symbol, "interval", interval, "bars", len(cached))
			return cached, "cache", nil
		}
	}

	chain := chainForSymbol(symbol, OHLCVChains, FallbackChains)
	if len(chain) == 0 {
		return nil, "", fmt.Errorf("unknown market: %q", market)
	}

	var lastErr error
	var bestBars []OHLCVBar
	var bestSource string
	for _, name := range chain {
		adapter := r.Get(name)
		if adapter == nil {
			slog.Debug("adapter not registered, skipping", "name", name, "market", market)
			continue
		}
		if !adapter.IsAvailable(ctx) {
			slog.Debug("adapter unavailable (probe failed), trying anyway", "name", name, "market", market)
		}

		bars, err := RetryWithBudget(
			func() ([]OHLCVBar, error) { return adapter.FetchOHLCV(ctx, symbol, interval, fqfactor, start, end) },
			DefaultRetryConfig(name),
		)
		if err != nil {
			slog.Warn("adapter fetch ohlcv failed, trying next", "name", name, "symbol", symbol, "error", err)
			lastErr = err
			continue
		}

		if len(bars) == 0 {
			slog.Info("ohlcv empty, trying next", "adapter", name, "symbol", symbol)
			continue
		}

		// Track the best (most complete) data seen so far.
		if len(bars) > len(bestBars) {
			bestBars = bars
			bestSource = name
		}

		// Check if this adapter's data covers most of the requested range.
		// A 90-day grace period accounts for stocks listed shortly after the
		// requested start date (e.g. 茅台 listed 2001-08-27 vs requested 2001-07-16).
		// If the data is clearly truncated (Tencent's 2000-bar cap), continue to
		// the next adapter. Only cache data after confirming adequate coverage.
		if start > 0 {
			firstTs := dateToTs(bars[0].Date)
			if firstTs == 0 || firstTs > start+90*86400 {
				slog.Info("ohlcv partial, trying next for fuller history",
					"adapter", name, "symbol", symbol, "bars", len(bars),
					"first", bars[0].Date, "need_start", time.Unix(start, 0).Format("2006-01-02"))
				continue
			}
		}

		if r.ohlcvCache != nil {
			if err := r.ohlcvCache.Set(symbol, interval, bars); err != nil {
				slog.Warn("ohlcv cache set failed", "symbol", symbol, "error", err)
			}
		}
		slog.Info("ohlcv fetched", "adapter", name, "symbol", symbol, "interval", interval, "bars", len(bars))
		return bars, name, nil
	}

	// All adapters exhausted — return the best (most bars) data we found.
	if len(bestBars) > 0 {
		slog.Info("ohlcv using best partial data", "adapter", bestSource, "symbol", symbol, "bars", len(bestBars),
			"first", bestBars[0].Date, "last", bestBars[len(bestBars)-1].Date)
		if r.ohlcvCache != nil {
			if err := r.ohlcvCache.Set(symbol, interval, bestBars); err != nil {
				slog.Warn("ohlcv cache set failed", "symbol", symbol, "error", err)
			}
		}
		return bestBars, bestSource, nil
	}

	if lastErr != nil {
		return nil, "", fmt.Errorf("all adapters failed for market %q symbol %q: %w", market, symbol, lastErr)
	}
	return nil, "", fmt.Errorf("no available adapter found for market %q symbol %q", market, symbol)
}

// FetchMinuteWithFallback tries each adapter in the market's minute data fallback
// chain until one succeeds. Uses MinuteChains which split by stock/index.
func (r *AdapterRegistry) FetchMinuteWithFallback(ctx context.Context, market, symbol string) ([]MinuteTick, string, error) {
	chain := chainForSymbol(symbol, MinuteChains, nil)
	if len(chain) == 0 {
		return nil, "", fmt.Errorf("no minute chain for market %q symbol %q", market, symbol)
	}

	var lastErr error
	for _, name := range chain {
		adapter := r.Get(name)
		if adapter == nil {
			slog.Debug("adapter not registered, skipping", "name", name, "market", market)
			continue
		}
		mp, ok := adapter.(MinuteLineProvider)
		if !ok {
			slog.Debug("adapter does not implement MinuteLineProvider", "name", name)
			continue
		}

		ticks, err := mp.FetchMinuteLine(symbol)
		if err != nil {
			slog.Warn("adapter fetch minute failed, trying next", "name", name, "symbol", symbol, "error", err)
			lastErr = err
			continue
		}
		if len(ticks) > 0 {
			return ticks, name, nil
		}
	}

	if lastErr != nil {
		return nil, "", fmt.Errorf("all minute adapters failed for %s: %w", symbol, lastErr)
	}
	return nil, "", fmt.Errorf("no minute data available for %s (chain: %v)", symbol, chain)
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
	// 6-digit numeric → CN A-share (bare code, e.g. "600519")
	if len(symbol) == 6 {
		isNum := true
		for _, c := range symbol {
			if c < '0' || c > '9' {
				isNum = false
				break
			}
		}
		if isNum {
			return "CN"
		}
	}
	return "US"
}
