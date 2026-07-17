package main

import (
	"context"
	"fmt"

	"quantflow/internal/market"
	"quantflow/internal/trading"
)

// GetPriceBand computes price band (valuation channel) for a symbol.
func (a *App) GetPriceBand(ctx context.Context, symbol, market, interval string, lookbackDays int) (*trading.BandResult, error) {
	if a.marketReg == nil {
		return nil, fmt.Errorf("market registry not initialized")
	}

	end := int64(0) // now
	start := int64(0)
	if lookbackDays > 0 {
		// approximate: lookback in trading days
		start = -int64(lookbackDays * 24 * 3600)
	}

	bars, _, err := a.marketReg.FetchOHLCVWithFallback(ctx, market, symbol, interval, "", start, end)
	if err != nil {
		return nil, fmt.Errorf("fetch ohlcv: %w", err)
	}

	// Convert market.OHLCVBar to trading.OHLCVBar
	tradingBars := make([]trading.OHLCVBar, len(bars))
	for i, b := range bars {
		tradingBars[i] = trading.OHLCVBar{
			Symbol: b.Symbol,
			Date:   b.Date,
			Open:   b.Open,
			High:   b.High,
			Low:    b.Low,
			Close:  b.Close,
			Volume: b.Volume,
		}
	}

	return trading.ComputePriceBand(symbol, tradingBars), nil
}

// Ensure imports are used
var _ = market.OHLCVBar{}
