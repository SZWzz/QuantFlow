package market

import "context"

// Adapter defines the interface for a market data source.
// Each adapter fetches data from a specific provider (Yahoo, EastMoney, etc.).
type Adapter interface {
	// Name returns the adapter's unique identifier (e.g., "yahoo", "eastmoney").
	Name() string

	// Markets returns the market types this adapter supports.
	// Values: "CN" (A-shares), "US" (US equities), "HK" (Hong Kong), "CRYPTO".
	Markets() []string

	// RequiresAuth returns true if this adapter needs an API token or credentials.
	RequiresAuth() bool

	// IsAvailable checks whether the adapter is currently usable
	// (network reachable, token valid, rate limit not exceeded).
	IsAvailable(ctx context.Context) bool

	// FetchQuote fetches a real-time quote snapshot for a symbol.
	FetchQuote(ctx context.Context, symbol string) (*QuoteSnapshot, error)

	// FetchOHLCV fetches OHLCV bars for a symbol within a date range.
	// start and end are Unix timestamps in seconds.
	// fqfactor controls price adjustment (复权): "" (不复权), "qfq" (前复权), "hfq" (后复权).
	// Only applicable to CN-market adapters (mootdx); other adapters should pass "".
	FetchOHLCV(ctx context.Context, symbol, interval, fqfactor string, start, end int64) ([]OHLCVBar, error)

	// HealthCheck checks if the adapter is operational.
	HealthCheck(ctx context.Context) error
}
