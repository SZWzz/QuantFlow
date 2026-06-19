package market

import (
	"context"
	"errors"
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

func (m *mockAdapter) FetchOHLCV(ctx context.Context, symbol, interval string, start, end int64) ([]OHLCVBar, error) {
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
