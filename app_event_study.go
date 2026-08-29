package main

import (
	"context"
	"fmt"
	"quantflow/internal/research"
)

// ComputeEventStudy calculates CAR around an event date.
func (a *App) ComputeEventStudy(ctx context.Context, symbol, market, interval string, eventDate string, window int) (*research.EventStudyResult, error) {
	if a.eventStudySvc == nil {
		return nil, fmt.Errorf("event study service not initialized")
	}
	if a.marketReg == nil {
		return nil, fmt.Errorf("market registry not initialized")
	}

	// Fetch stock OHLCV
	start := int64(0)
	end := int64(0)

	stockBars, _, err := a.marketReg.FetchOHLCVWithFallback(ctx, market, symbol, interval, "", start, end)
	if err != nil {
		return nil, fmt.Errorf("fetch stock OHLCV: %w", err)
	}

	// Determine benchmark symbol
	benchSymbol := "000001.SH"
	switch market {
	case "HK":
		benchSymbol = "HSI"
	case "US":
		benchSymbol = "SPY"
	case "CRYPTO":
		benchSymbol = "BTCUSDT"
	}

	benchBars, _, err := a.marketReg.FetchOHLCVWithFallback(ctx, market, benchSymbol, interval, "", start, end)
	if err != nil {
		// Use stock bars as fallback benchmark
		benchBars = stockBars
	}

	return a.eventStudySvc.ComputeCAR(ctx, stockBars, benchBars, eventDate, window)
}
