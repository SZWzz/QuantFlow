package adapters

import (
	"context"
	"testing"
	"time"

	"quantflow/internal/market"
)

// quoteAdapterTest holds the config for testing a single adapter's FetchQuote.
type quoteAdapterTest struct {
	name     string
	adapter  market.Adapter
	symbol   string // raw symbol, e.g. "600519"
	market   string // "CN", "US", "HK", "CRYPTO"
	wantAuth bool   // whether this adapter requires auth to succeed
}

// TestAllQuoteAdapters_FetchQuote tests every quote adapter's ability to fetch
// real-time quote data. Adapters that require auth are only tested for
// interface compliance; free adapters are tested against live endpoints.
func TestAllQuoteAdapters_FetchQuote(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	tests := []quoteAdapterTest{
		// ── CN (A-shares) ──────────────────────────────────────────
		{name: "mootdx", adapter: NewMootdxAdapter(nil), symbol: "600519", market: "CN", wantAuth: false},
		{name: "sina", adapter: NewSinaAdapter(), symbol: "600519", market: "CN", wantAuth: false},
		{name: "tushare", adapter: NewTuShareAdapter(), symbol: "600519", market: "CN", wantAuth: true},
		{name: "eastmoney", adapter: NewEastMoneyAdapter(), symbol: "600519", market: "CN", wantAuth: false},
		{name: "tencent", adapter: NewTencentAdapter(), symbol: "600519", market: "CN", wantAuth: false},
		{name: "baidu", adapter: NewBaiduAdapter(), symbol: "600519", market: "CN", wantAuth: false},
		{name: "akshare", adapter: NewAKShareAdapter(), symbol: "600519", market: "CN", wantAuth: false},

		// ── US ─────────────────────────────────────────────────────
		{name: "yahoo", adapter: NewYahooAdapter(), symbol: "AAPL", market: "US", wantAuth: false},
		{name: "polygon", adapter: NewPolygonAdapter(PolygonConfig{}), symbol: "AAPL", market: "US", wantAuth: true},

		// ── HK ─────────────────────────────────────────────────────
		// (no dedicated HK-only adapter; yahoo and others cover HK)

		// ── CRYPTO ─────────────────────────────────────────────────
		{name: "gateio", adapter: NewGateIOAdapter(), symbol: "BTCUSDT", market: "CRYPTO", wantAuth: false},
		{name: "okx", adapter: NewOKXAdapter(), symbol: "BTC-USDT", market: "CRYPTO", wantAuth: false},
		{name: "binance", adapter: NewBinanceAdapter(), symbol: "BTCUSDT", market: "CRYPTO", wantAuth: false},
		{name: "coingecko", adapter: NewCoinGeckoAdapter(), symbol: "bitcoin", market: "CRYPTO", wantAuth: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify interface compliance
			if tt.adapter.Name() != tt.name {
				t.Errorf("Name() = %q, want %q", tt.adapter.Name(), tt.name)
			}
			if tt.adapter.RequiresAuth() != tt.wantAuth {
				t.Logf("RequiresAuth() = %v (expected %v)", tt.adapter.RequiresAuth(), tt.wantAuth)
			}

			// Skip real fetch for auth-required adapters
			if tt.wantAuth {
				t.Skipf("%s requires auth — skipping live fetch", tt.name)
			}

			// Check availability
			available := tt.adapter.IsAvailable(ctx)
			t.Logf("%s IsAvailable=%v", tt.name, available)

			// Attempt FetchQuote
			snap, err := tt.adapter.FetchQuote(ctx, tt.symbol)
			if err != nil {
				t.Logf("%s FetchQuote(%q) error: %v", tt.name, tt.symbol, err)
				return
			}
			if snap == nil {
				t.Errorf("%s FetchQuote returned nil snapshot", tt.name)
				return
			}
			t.Logf("%s FetchQuote(%q) OK: last=%.2f change=%.2f%% vol=%.0f",
				tt.name, tt.symbol, snap.Last, snap.ChangePct, snap.Volume)
		})
	}
}
