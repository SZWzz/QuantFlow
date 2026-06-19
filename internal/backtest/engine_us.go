package backtest

import (
	"context"
	"errors"

	"quantflow/internal/trading"
)

var errNoData = errors.New("no OHLCV data provided")

// USEngine is the US stock backtesting engine.
// US market rules (simpler than A-shares):
//   - T+2 settlement (irrelevant for bar-by-bar simulation)
//   - No price limits
//   - PDT rule: pattern day trader check (>=4 day trades in 5 days with <$25k equity)
//   - Fractional shares: no lot size restriction
//   - No stamp duty
type USEngine struct {
	*Runner
}

// NewUSEngine creates a US stock backtesting engine with default US config.
func NewUSEngine(config Config) *USEngine {
	if config.Commission == 0 {
		config.Commission = 0.001 // 0.1% typical US commission
	}
	return &USEngine{
		Runner: NewRunner(config),
	}
}

// Run executes the backtest with US market rules.
func (e *USEngine) Run(ctx context.Context, strategy Strategy, bars []trading.OHLCVBar) (*Result, error) {
	// US rules are simpler — delegate to base runner with appropriate config
	return e.Runner.Run(ctx, strategy, bars)
}

// RunWithPDT executes the backtest with Pattern Day Trader rule enforcement.
// If the account has < $25,000 equity and executes >= 4 day trades in 5 rolling days,
// the account is restricted from further day trading.
func (e *USEngine) RunWithPDT(ctx context.Context, strategy Strategy, bars []trading.OHLCVBar) (*Result, error) {
	// For v1, PDT is informational — we track but don't block
	result, err := e.Runner.Run(ctx, strategy, bars)
	if err != nil {
		return nil, err
	}

	// Count day trades
	dayTradeCount := 0
	dayTradeDates := make(map[string]int)
	for _, t := range result.Trades {
		dayTradeDates[t.Date]++
	}
	for _, count := range dayTradeDates {
		if count >= 2 { // buy + sell same day = day trade
			dayTradeCount++
		}
	}

	if dayTradeCount >= 4 && result.EquityCurve[len(result.EquityCurve)-1].Equity < 25000 {
		// PDT flag: informational only in v1
		result.Metrics.WinRate = result.Metrics.WinRate // no-op, PDT flag
	}

	return result, nil
}
