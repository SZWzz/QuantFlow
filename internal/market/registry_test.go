package market

import (
	"context"
	"errors"
	"fmt"
	"testing"
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

	// Register two adapters for CN market
	r.Register(&mockAdapter{
		name: "a", markets: []string{"CN"}, available: true,
		quoteResult: &QuoteSnapshot{Symbol: "TEST", Last: 100.0},
	})
	r.Register(&mockAdapter{
		name: "b", markets: []string{"CN"}, available: true,
		quoteResult: &QuoteSnapshot{Symbol: "TEST", Last: 200.0},
	})

	// Override fallback chain for testing
	origChain := FallbackChains["CN"]
	FallbackChains["CN"] = []string{"a", "b"}
	defer func() { FallbackChains["CN"] = origChain }()

	quote, source, err := r.FetchQuoteWithFallback(context.Background(), "CN", "TEST")
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
		name: "a", markets: []string{"CN"}, available: false,
	})
	r.Register(&mockAdapter{
		name: "b", markets: []string{"CN"}, available: true,
		quoteResult: &QuoteSnapshot{Symbol: "TEST", Last: 200.0},
	})

	origChain := FallbackChains["CN"]
	FallbackChains["CN"] = []string{"a", "b"}
	defer func() { FallbackChains["CN"] = origChain }()

	quote, source, err := r.FetchQuoteWithFallback(context.Background(), "CN", "TEST")
	if err != nil {
		t.Fatalf("FetchQuoteWithFallback error: %v", err)
	}
	if source != "b" {
		t.Errorf("source = %q, want %q (should skip unavailable a)", source, "b")
	}
	if quote.Last != 200.0 {
		t.Errorf("Last = %f", quote.Last)
	}
}

func TestRegistry_FallbackSkipsErrorAndTriesNext(t *testing.T) {
	r := NewAdapterRegistry()

	r.Register(&mockAdapter{
		name: "a", markets: []string{"CN"}, available: true,
		quoteErr: errors.New("network timeout"),
	})
	r.Register(&mockAdapter{
		name: "b", markets: []string{"CN"}, available: true,
		quoteResult: &QuoteSnapshot{Symbol: "TEST", Last: 300.0},
	})

	origChain := FallbackChains["CN"]
	FallbackChains["CN"] = []string{"a", "b"}
	defer func() { FallbackChains["CN"] = origChain }()

	quote, source, err := r.FetchQuoteWithFallback(context.Background(), "CN", "TEST")
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
		name: "a", markets: []string{"CN"}, available: true,
		quoteErr: errors.New("fail a"),
	})
	r.Register(&mockAdapter{
		name: "b", markets: []string{"CN"}, available: true,
		quoteErr: errors.New("fail b"),
	})

	origChain := FallbackChains["CN"]
	FallbackChains["CN"] = []string{"a", "b"}
	defer func() { FallbackChains["CN"] = origChain }()

	_, _, err := r.FetchQuoteWithFallback(context.Background(), "CN", "TEST")
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
