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
		// mootdx added: TDX TCP port's get_security_bars
		// returns None for historical K-lines (only live quote works).
		// Minute data uses Mootdx separately (MinuteChains) where it works.
		"stock": {"tencent", "sina", "tushare", "baidu", "akshare", "eastmoney"},
		"index": {"tencent", "sina", "tushare", "baidu", "akshare", "eastmoney"},
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
	"HK": {
		"stock": {"akshare_hk_minute", "qos", "yahoo"},
	},
	"US": {
		"stock": {"yahoo"},
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

// ohlcvRefetchCooldown bounds how often a stale OHLCV cache entry triggers a
// full adapter refetch (e.g. during holidays when no newer bars exist yet).
const ohlcvRefetchCooldown = 10 * time.Minute

type quoteCacheEntry struct {
	snapshot *QuoteSnapshot
	source   string
	expires  time.Time
}

// AdapterRegistry manages registered market data adapters and provides
// fallback-based fetching.
type AdapterRegistry struct {
	mu             sync.RWMutex
	adapters       map[string]Adapter // name → adapter
	quoteCache     map[string]*quoteCacheEntry
	lastQuote      sync.Map // market:symbol → *QuoteSnapshot (last known value, survives TTL)
	lastQuotePath  string   // if set, last quotes are persisted to this JSON file
	saveMu         sync.Mutex
	saveTimer      *time.Timer
	ohlcvCache     *OHLCVCache // optional two-tier (LRU+SQLite) cache
	ohlcvFetchMu   sync.Mutex
	ohlcvLastFetch map[string]time.Time // "symbol:interval" → last adapter refetch attempt
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
	if err := os.WriteFile(tmpPath, b, 0o600); err != nil {
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
		adapters:       make(map[string]Adapter),
		quoteCache:     make(map[string]*quoteCacheEntry),
		ohlcvLastFetch: make(map[string]time.Time),
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

	var stale []OHLCVBar
	if r.ohlcvCache != nil {
		if cached, err := r.ohlcvCache.Get(symbol, interval, start, end); err == nil && len(cached) > 0 {
			// Serve cache only when it reaches the latest expected bar.
			// A stale cache (e.g. daily bars lagging behind) triggers a refetch,
			// bounded by a cooldown so holidays/off-hours don't hammer adapters.
			if ohlcvCacheFresh(interval, cached, end, time.Now()) || !r.markOhlcvRefetch(symbol, interval) {
				slog.Debug("ohlcv cache hit", "symbol", symbol, "interval", interval, "bars", len(cached))
				return cached, "cache", nil
			}
			stale = cached
			slog.Debug("ohlcv cache stale, refetching", "symbol", symbol, "interval", interval,
				"last", cached[len(cached)-1].Date)
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

	// Refetch failed — fall back to the stale cache so the user keeps
	// last-known data instead of an error.
	if len(stale) > 0 {
		slog.Info("ohlcv refetch failed, serving stale cache", "symbol", symbol, "interval", interval)
		return stale, "cache-stale", nil
	}

	if lastErr != nil {
		return nil, "", fmt.Errorf("all adapters failed for market %q symbol %q: %w", market, symbol, lastErr)
	}
	return nil, "", fmt.Errorf("no available adapter found for market %q symbol %q", market, symbol)
}

// markOhlcvRefetch reports whether a stale-cache refetch should be attempted
// now for the symbol/interval, stamping the attempt time. Returns false while
// within ohlcvRefetchCooldown of the last attempt.
func (r *AdapterRegistry) markOhlcvRefetch(symbol, interval string) bool {
	key := symbol + ":" + interval
	r.ohlcvFetchMu.Lock()
	defer r.ohlcvFetchMu.Unlock()
	if t, ok := r.ohlcvLastFetch[key]; ok && time.Since(t) < ohlcvRefetchCooldown {
		return false
	}
	r.ohlcvLastFetch[key] = time.Now()
	return true
}

// ohlcvCacheFresh reports whether cached bars reach the latest bar adapters
// could plausibly have for the interval. end is the query's range end; when
// it's in the past, freshness is anchored there instead of now.
func ohlcvCacheFresh(interval string, bars []OHLCVBar, end int64, now time.Time) bool {
	last := dateToTs(bars[len(bars)-1].Date)
	if last == 0 {
		return false
	}
	anchor := now
	if end > 0 && end < now.Unix() {
		anchor = time.Unix(end, 0)
	}
	return last >= ohlcvExpectedLastBar(interval, anchor)
}

// ohlcvExpectedLastBar returns the unix timestamp of the latest bar adapters
// could plausibly have for interval at time n (Asia/Shanghai). Weekends are
// skipped; holidays are not modeled — they cause bounded extra refetches via
// ohlcvRefetchCooldown.
func ohlcvExpectedLastBar(interval string, n time.Time) int64 {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		loc = time.FixedZone("CST", 8*3600)
	}
	n = n.In(loc)
	switch interval {
	case "1m", "5m", "15m", "30m", "1H", "1h":
		step := map[string]int64{"1m": 60, "5m": 300, "15m": 900, "30m": 1800, "1H": 3600, "1h": 3600}[interval]
		mins := n.Hour()*60 + n.Minute()
		switch {
		case mins < 9*60: // before open → previous trading day 15:00
			d := prevWeekday(n.AddDate(0, 0, -1))
			return time.Date(d.Year(), d.Month(), d.Day(), 15, 0, 0, 0, loc).Unix()
		case mins > 15*60+30: // after close → today (or last weekday) 15:00
			d := prevWeekday(n)
			return time.Date(d.Year(), d.Month(), d.Day(), 15, 0, 0, 0, loc).Unix()
		default: // trading hours → allow the still-forming bar to be absent
			return n.Unix() - 2*step
		}
	case "1W":
		return prevTradingDay(n, loc) - 6*86400
	case "1M":
		return prevTradingDay(n, loc) - 30*86400
	default: // 1D
		return prevTradingDay(n, loc)
	}
}

// prevTradingDay returns the CST-midnight timestamp of the most recent trading
// day at time n: today if it is a weekday past the 15:30 close, otherwise the
// previous weekday.
func prevTradingDay(n time.Time, loc *time.Location) int64 {
	d := n
	if n.Hour() < 15 || (n.Hour() == 15 && n.Minute() < 30) {
		d = n.AddDate(0, 0, -1)
	}
	for d.Weekday() == time.Saturday || d.Weekday() == time.Sunday {
		d = d.AddDate(0, 0, -1)
	}
	return time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, loc).Unix()
}

// prevWeekday rolls n back to the nearest weekday (n itself if already one).
func prevWeekday(n time.Time) time.Time {
	d := n
	for d.Weekday() == time.Saturday || d.Weekday() == time.Sunday {
		d = d.AddDate(0, 0, -1)
	}
	return d
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
	// 1-5 digit numeric → HK (bare code, e.g. "00700", "5", "9988")
	if len(symbol) >= 1 && len(symbol) <= 5 {
		isNum := true
		for _, c := range symbol {
			if c < '0' || c > '9' {
				isNum = false
				break
			}
		}
		if isNum {
			return "HK"
		}
	}
	return "US"
}
