package market

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// mockAdapter is a controllable adapter for testing the registry.
type mockAdapter struct {
	name         string
	markets      []string
	requiresAuth bool
	available    bool
	quoteResult  *QuoteSnapshot
	quoteErr     error
}

func (m *mockAdapter) Name() string                         { return m.name }
func (m *mockAdapter) Markets() []string                     { return m.markets }
func (m *mockAdapter) RequiresAuth() bool                    { return m.requiresAuth }
func (m *mockAdapter) IsAvailable(ctx context.Context) bool  { return m.available }
func (m *mockAdapter) HealthCheck(ctx context.Context) error { return nil }

func (m *mockAdapter) FetchQuote(ctx context.Context, symbol string) (*QuoteSnapshot, error) {
	return m.quoteResult, m.quoteErr
}

func (m *mockAdapter) FetchOHLCV(ctx context.Context, symbol, interval, _ string, start, end int64) ([]OHLCVBar, error) {
	return nil, nil
}

func TestRegistry_RegisterAndGet(t *testing.T) {
	r := NewAdapterRegistry()
	adapter := &mockAdapter{name: "test", markets: []string{"CN"}, available: true}

	r.Register(adapter)

	got := r.Get("test")
	if got == nil {
		t.Fatal("expected adapter to be registered")
	}
	if got.Name() != "test" {
		t.Errorf("Name = %q, want %q", got.Name(), "test")
	}
}

func TestRegistry_FallbackChainSelectsFirstAvailable(t *testing.T) {
	r := NewAdapterRegistry()

	// Register two adapters — use a 24/7 market to avoid trading-hours gate
	r.Register(&mockAdapter{
		name: "a", markets: []string{"CRYPTO"}, available: true,
		quoteResult: &QuoteSnapshot{Symbol: "TEST", Last: 100.0},
	})
	r.Register(&mockAdapter{
		name: "b", markets: []string{"CRYPTO"}, available: true,
		quoteResult: &QuoteSnapshot{Symbol: "TEST", Last: 200.0},
	})

	// Override fallback chain for testing
	origChain := FallbackChains["CRYPTO"]
	FallbackChains["CRYPTO"] = []string{"a", "b"}
	defer func() { FallbackChains["CRYPTO"] = origChain }()

	quote, source, err := r.FetchQuoteWithFallback(context.Background(), "CRYPTO", "TEST")
	if err != nil {
		t.Fatalf("FetchQuoteWithFallback error: %v", err)
	}
	if source != "a" {
		t.Errorf("source = %q, want %q (should use first available)", source, "a")
	}
	if quote.Last != 100.0 {
		t.Errorf("Last = %f, want 100.0", quote.Last)
	}
}

func TestRegistry_FallbackSkipsUnavailable(t *testing.T) {
	r := NewAdapterRegistry()

	r.Register(&mockAdapter{
		name: "a", markets: []string{"CRYPTO"}, available: false,
		quoteErr: errors.New("fetch failed"),
	})
	r.Register(&mockAdapter{
		name: "b", markets: []string{"CRYPTO"}, available: true,
		quoteResult: &QuoteSnapshot{Symbol: "TEST", Last: 200.0},
	})

	origChain := FallbackChains["CRYPTO"]
	FallbackChains["CRYPTO"] = []string{"a", "b"}
	defer func() { FallbackChains["CRYPTO"] = origChain }()

	quote, source, err := r.FetchQuoteWithFallback(context.Background(), "CRYPTO", "TEST")
	if err != nil {
		t.Fatalf("FetchQuoteWithFallback error: %v", err)
	}
	if source != "b" {
		t.Errorf("source = %q, want %q (should try unavailable a then fallback to b)", source, "b")
	}
	if quote.Last != 200.0 {
		t.Errorf("Last = %f", quote.Last)
	}
}

func TestRegistry_FallbackSkipsErrorAndTriesNext(t *testing.T) {
	r := NewAdapterRegistry()

	r.Register(&mockAdapter{
		name: "a", markets: []string{"CRYPTO"}, available: true,
		quoteErr: errors.New("network timeout"),
	})
	r.Register(&mockAdapter{
		name: "b", markets: []string{"CRYPTO"}, available: true,
		quoteResult: &QuoteSnapshot{Symbol: "TEST", Last: 300.0},
	})

	origChain := FallbackChains["CRYPTO"]
	FallbackChains["CRYPTO"] = []string{"a", "b"}
	defer func() { FallbackChains["CRYPTO"] = origChain }()

	quote, source, err := r.FetchQuoteWithFallback(context.Background(), "CRYPTO", "TEST")
	if err != nil {
		t.Fatalf("FetchQuoteWithFallback error: %v", err)
	}
	if source != "b" {
		t.Errorf("source = %q, want %q (should skip error from a)", source, "b")
	}
	if quote.Last != 300.0 {
		t.Errorf("Last = %f", quote.Last)
	}
}

func TestRegistry_AllFailed(t *testing.T) {
	r := NewAdapterRegistry()

	r.Register(&mockAdapter{
		name: "a", markets: []string{"CRYPTO"}, available: true,
		quoteErr: errors.New("fail a"),
	})
	r.Register(&mockAdapter{
		name: "b", markets: []string{"CRYPTO"}, available: true,
		quoteErr: errors.New("fail b"),
	})

	origChain := FallbackChains["CRYPTO"]
	FallbackChains["CRYPTO"] = []string{"a", "b"}
	defer func() { FallbackChains["CRYPTO"] = origChain }()

	_, _, err := r.FetchQuoteWithFallback(context.Background(), "CRYPTO", "TEST")
	if err == nil {
		t.Fatal("expected error when all adapters fail")
	}
}

// mockIndustryRankProvider implements Adapter + IndustryRankProvider for testing.
type mockIndustryRankProvider struct {
	name  string
	fail  bool
	ranks []IndustryRank
}

func (m *mockIndustryRankProvider) Name() string                         { return m.name }
func (m *mockIndustryRankProvider) Markets() []string                    { return []string{"ZZ"} }
func (m *mockIndustryRankProvider) RequiresAuth() bool                   { return false }
func (m *mockIndustryRankProvider) IsAvailable(ctx context.Context) bool { return true }
func (m *mockIndustryRankProvider) HealthCheck(ctx context.Context) error { return nil }
func (m *mockIndustryRankProvider) FetchQuote(ctx context.Context, symbol string) (*QuoteSnapshot, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockIndustryRankProvider) FetchOHLCV(ctx context.Context, symbol, interval, fqfactor string, start, end int64) ([]OHLCVBar, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockIndustryRankProvider) FetchIndustryRanks(ctx context.Context, market string, topN int) ([]IndustryRank, error) {
	if m.fail {
		return nil, fmt.Errorf("mock: %s failed", m.name)
	}
	if m.ranks != nil {
		return m.ranks, nil
	}
	return []IndustryRank{{Rank: 1, Name: "Mock", ChangePct: 1.0}}, nil
}

func TestFetchIndustryRanksWithFallback_UnknownMarket(t *testing.T) {
	reg := NewAdapterRegistry()
	reg.Register(&mockIndustryRankProvider{name: "test"})

	ranks, err := reg.FetchIndustryRanksWithFallback(context.Background(), "ZZ", 10)
	if err == nil {
		t.Fatal("expected error for unknown market")
	}
	if ranks != nil {
		t.Fatal("expected nil ranks for unknown market")
	}
}

func TestFetchIndustryRanksWithFallback_Success(t *testing.T) {
	reg := NewAdapterRegistry()
	reg.Register(&mockIndustryRankProvider{name: "finnhub"})

	origChain := IndustryRankChains["US"]
	IndustryRankChains["US"] = []string{"finnhub"}
	defer func() { IndustryRankChains["US"] = origChain }()

	ranks, err := reg.FetchIndustryRanksWithFallback(context.Background(), "US", 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ranks) == 0 {
		t.Fatal("expected non-empty ranks")
	}
	if ranks[0].Name != "Mock" {
		t.Errorf("ranks[0].Name = %q, want %q", ranks[0].Name, "Mock")
	}
}

func TestFetchIndustryRanksWithFallback_FallbackOnFailure(t *testing.T) {
	reg := NewAdapterRegistry()
	reg.Register(&mockIndustryRankProvider{name: "a", fail: true})
	reg.Register(&mockIndustryRankProvider{name: "b"})

	origChain := IndustryRankChains["US"]
	IndustryRankChains["US"] = []string{"a", "b"}
	defer func() { IndustryRankChains["US"] = origChain }()

	ranks, err := reg.FetchIndustryRanksWithFallback(context.Background(), "US", 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ranks) == 0 {
		t.Fatal("expected non-empty ranks from fallback")
	}
}

func TestFetchIndustryRanksWithFallback_AllFailed(t *testing.T) {
	reg := NewAdapterRegistry()
	reg.Register(&mockIndustryRankProvider{name: "a", fail: true})
	reg.Register(&mockIndustryRankProvider{name: "b", fail: true})

	origChain := IndustryRankChains["US"]
	IndustryRankChains["US"] = []string{"a", "b"}
	defer func() { IndustryRankChains["US"] = origChain }()

	_, err := reg.FetchIndustryRanksWithFallback(context.Background(), "US", 5)
	if err == nil {
		t.Fatal("expected error when all adapters fail")
	}
}

func TestRegistry_SaveLoadLastQuotes(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/last_quote.json"

	r := NewAdapterRegistry()
	r.SetLastQuotePath(path)

	// Initially empty
	if err := r.LoadLastQuotes(); err != nil {
		t.Fatal(err)
	}

	// Store a quote and save
	r.lastQuote.Store("CN:600519", &QuoteSnapshot{Symbol: "600519", Last: 1880.5, Volume: 50000})
	r.saveLastQuotes()

	// File should exist and be valid JSON
	r2 := NewAdapterRegistry()
	r2.SetLastQuotePath(path)
	if err := r2.LoadLastQuotes(); err != nil {
		t.Fatal(err)
	}
	val, ok := r2.lastQuote.Load("CN:600519")
	if !ok {
		t.Fatal("expected loaded quote")
	}
	q := val.(*QuoteSnapshot)
	if q.Last != 1880.5 {
		t.Fatalf("expected Last=1880.5, got %f", q.Last)
	}
	if q.Volume != 50000 {
		t.Fatalf("expected Volume=50000, got %f", q.Volume)
	}
}

func TestRegistry_LoadLastQuotes_MissingFile(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/nonexistent.json"

	r := NewAdapterRegistry()
	r.SetLastQuotePath(path)
	if err := r.LoadLastQuotes(); err != nil {
		t.Fatalf("expected no error for missing file, got %v", err)
	}
}

func TestMarketForSymbol(t *testing.T) {
	tests := []struct {
		symbol string
		market string
	}{
		{"000001.SZ", "CN"},
		{"600519.SH", "CN"},
		{"00700.HK", "HK"},
		{"BTCUSDT", "CRYPTO"},
		{"ETHUSDT", "CRYPTO"},
		{"AAPL", "US"},
		{"MSFT", "US"},
	}

	for _, tt := range tests {
		t.Run(tt.symbol, func(t *testing.T) {
			got := MarketForSymbol(tt.symbol)
			if got != tt.market {
				t.Errorf("MarketForSymbol(%q) = %q, want %q", tt.symbol, got, tt.market)
			}
		})
	}
}

// ── OHLCV cache freshness ───────────────────────────────────────

type ohlcvMockAdapter struct {
	mockAdapter
	bars  []OHLCVBar
	err   error
	calls int
}

func (m *ohlcvMockAdapter) FetchOHLCV(ctx context.Context, symbol, interval, _ string, start, end int64) ([]OHLCVBar, error) {
	m.calls++
	return m.bars, m.err
}

func newOHLCVTestRegistry(t *testing.T) (*AdapterRegistry, *OHLCVCache) {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	oc, err := NewOHLCVCache(db)
	if err != nil {
		t.Fatalf("NewOHLCVCache: %v", err)
	}
	r := NewAdapterRegistry()
	r.SetOHLCVCache(oc)
	t.Cleanup(func() { db.Close() })
	return r, oc
}

func withOHLCVChain(t *testing.T, names ...string) {
	t.Helper()
	orig := OHLCVChains["CN"]
	OHLCVChains["CN"] = map[string][]string{"stock": names, "index": names}
	t.Cleanup(func() { OHLCVChains["CN"] = orig })
}

func TestFetchOHLCV_StaleCacheTriggersRefetch(t *testing.T) {
	r, oc := newOHLCVTestRegistry(t)
	withOHLCVChain(t, "a")

	staleDate := time.Now().AddDate(0, 0, -10).Format("2006-01-02")
	if err := oc.Set("600519", "1D", []OHLCVBar{{Symbol: "600519", Date: staleDate, Open: 1, High: 1, Low: 1, Close: 1, Volume: 1}}); err != nil {
		t.Fatal(err)
	}

	freshDate := tsToDate(ohlcvExpectedLastBar("1D", time.Now()))
	fresh := []OHLCVBar{
		{Symbol: "600519", Date: staleDate, Open: 1, High: 1, Low: 1, Close: 1, Volume: 1},
		{Symbol: "600519", Date: time.Now().AddDate(0, 0, -5).Format("2006-01-02"), Open: 2, High: 2, Low: 2, Close: 2, Volume: 2},
		{Symbol: "600519", Date: freshDate, Open: 3, High: 3, Low: 3, Close: 3, Volume: 3},
	}
	a := &ohlcvMockAdapter{mockAdapter: mockAdapter{name: "a", markets: []string{"CN"}, available: true}, bars: fresh}
	r.Register(a)

	end := time.Now().Unix()
	start := end - 30*86400
	bars, source, err := r.FetchOHLCVWithFallback(context.Background(), "CN", "600519", "1d", "qfq", start, end)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.calls != 1 {
		t.Errorf("adapter calls = %d, want 1 (stale cache must refetch)", a.calls)
	}
	if source != "a" {
		t.Errorf("source = %q, want %q", source, "a")
	}
	if len(bars) != 3 || bars[len(bars)-1].Date != freshDate {
		t.Errorf("bars = %+v, want 3 bars ending at %s", bars, freshDate)
	}

	// Cache must now hold the fresh bars.
	cached, err := oc.Get("600519", "1D", start, end)
	if err != nil || len(cached) != 3 {
		t.Errorf("cached bars = %d (err %v), want 3", len(cached), err)
	}
}

func TestFetchOHLCV_FreshCacheServedWithoutAdapter(t *testing.T) {
	r, oc := newOHLCVTestRegistry(t)
	withOHLCVChain(t, "a")

	recent := OHLCVBar{Symbol: "600519", Date: tsToDate(ohlcvExpectedLastBar("1D", time.Now())), Open: 1, High: 1, Low: 1, Close: 1, Volume: 1}
	if err := oc.Set("600519", "1D", []OHLCVBar{recent}); err != nil {
		t.Fatal(err)
	}

	a := &ohlcvMockAdapter{mockAdapter: mockAdapter{name: "a", markets: []string{"CN"}, available: true}}
	r.Register(a)

	end := time.Now().Unix()
	bars, source, err := r.FetchOHLCVWithFallback(context.Background(), "CN", "600519", "1D", "", end-30*86400, end)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.calls != 0 {
		t.Errorf("adapter calls = %d, want 0 (fresh cache must be served)", a.calls)
	}
	if source != "cache" {
		t.Errorf("source = %q, want %q", source, "cache")
	}
	if len(bars) != 1 {
		t.Errorf("bars = %d, want 1", len(bars))
	}
}

func TestFetchOHLCV_StaleCacheAdapterFailureServesStale(t *testing.T) {
	r, oc := newOHLCVTestRegistry(t)
	withOHLCVChain(t, "a")

	staleDate := time.Now().AddDate(0, 0, -10).Format("2006-01-02")
	if err := oc.Set("600519", "1D", []OHLCVBar{{Symbol: "600519", Date: staleDate, Open: 1, High: 1, Low: 1, Close: 1, Volume: 1}}); err != nil {
		t.Fatal(err)
	}

	a := &ohlcvMockAdapter{mockAdapter: mockAdapter{name: "a", markets: []string{"CN"}, available: true}, err: errors.New("boom")}
	r.Register(a)

	end := time.Now().Unix()
	bars, source, err := r.FetchOHLCVWithFallback(context.Background(), "CN", "600519", "1D", "", end-30*86400, end)
	if err != nil {
		t.Fatalf("expected stale fallback instead of error, got: %v", err)
	}
	if source != "cache-stale" {
		t.Errorf("source = %q, want %q", source, "cache-stale")
	}
	if len(bars) != 1 || bars[0].Date != staleDate {
		t.Errorf("bars = %+v, want the stale %s bar", bars, staleDate)
	}
}

func TestFetchOHLCV_RefetchCooldown(t *testing.T) {
	r, oc := newOHLCVTestRegistry(t)
	withOHLCVChain(t, "a")

	staleDate := time.Now().AddDate(0, 0, -10).Format("2006-01-02")
	if err := oc.Set("600519", "1D", []OHLCVBar{{Symbol: "600519", Date: staleDate, Open: 1, High: 1, Low: 1, Close: 1, Volume: 1}}); err != nil {
		t.Fatal(err)
	}

	a := &ohlcvMockAdapter{mockAdapter: mockAdapter{name: "a", markets: []string{"CN"}, available: true}, err: errors.New("boom")}
	r.Register(a)

	end := time.Now().Unix()
	start := end - 30*86400

	// First call: stale cache → refetch attempted (and fails).
	_, _, _ = r.FetchOHLCVWithFallback(context.Background(), "CN", "600519", "1D", "", start, end)
	if a.calls != 1 {
		t.Fatalf("adapter calls = %d after first fetch, want 1", a.calls)
	}

	// Second call within cooldown: stale cache served without another refetch.
	bars, source, err := r.FetchOHLCVWithFallback(context.Background(), "CN", "600519", "1D", "", start, end)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.calls != 1 {
		t.Errorf("adapter calls = %d, want 1 (cooldown must suppress refetch)", a.calls)
	}
	if source != "cache" {
		t.Errorf("source = %q, want %q", source, "cache")
	}
	if len(bars) != 1 {
		t.Errorf("bars = %d, want 1", len(bars))
	}
}

func TestPrevTradingDay(t *testing.T) {
	loc := time.FixedZone("CST", 8*3600)
	cases := []struct {
		name string
		now  time.Time
		want time.Time
	}{
		{"weekday morning → previous weekday", time.Date(2026, 7, 15, 10, 0, 0, 0, loc), time.Date(2026, 7, 14, 0, 0, 0, 0, loc)},
		{"weekday after close → same day", time.Date(2026, 7, 15, 16, 0, 0, 0, loc), time.Date(2026, 7, 15, 0, 0, 0, 0, loc)},
		{"sunday → friday", time.Date(2026, 7, 19, 12, 0, 0, 0, loc), time.Date(2026, 7, 17, 0, 0, 0, 0, loc)},
		{"monday morning → friday", time.Date(2026, 7, 20, 9, 0, 0, 0, loc), time.Date(2026, 7, 17, 0, 0, 0, 0, loc)},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := prevTradingDay(c.now, loc)
			if got != c.want.Unix() {
				t.Errorf("prevTradingDay(%v) = %v, want %v", c.now, time.Unix(got, 0), c.want)
			}
		})
	}
}
