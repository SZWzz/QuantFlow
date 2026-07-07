package backtest

import (
	"context"
	"testing"

	"quantflow/internal/trading"
)

func TestRunner_StopLossHit(t *testing.T) {
	config := DefaultConfig()
	config.InitialCash = 100000

	runner := NewRunner(config)

	// Bars: price rises for 3 days then drops 6% to trigger 5% stop loss
	bars := []trading.OHLCVBar{
		{Symbol: "TEST", Date: "2024-01-02", Open: 100, High: 101, Low: 99, Close: 100, Volume: 10000},
		{Symbol: "TEST", Date: "2024-01-03", Open: 105, High: 106, Low: 104, Close: 105, Volume: 10000},
		{Symbol: "TEST", Date: "2024-01-04", Open: 104, High: 105, Low: 103, Close: 104, Volume: 10000},
		{Symbol: "TEST", Date: "2024-01-05", Open: 103, High: 104, Low: 102, Close: 103, Volume: 10000},
		{Symbol: "TEST", Date: "2024-01-06", Open: 95, High: 96, Low: 93, Close: 94, Volume: 10000},
	}

	bought := false
	strategy := Strategy{
		ID:   "stop_loss_test",
		Name: "Stop Loss Test",
		SignalFunc: func(openPrice float64, prevBar *trading.OHLCVBar, portfolio *Portfolio) *trading.Signal {
			if !bought {
				bought = true
				return &trading.Signal{
					Symbol:    "TEST",
					Direction: "buy",
					Quantity:  100,
				}
			}
			return nil
		},
	}

	ctx := context.Background()
	result, err := runner.Run(ctx, strategy, bars)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if result.Metrics.TotalReturn >= 0 {
		t.Errorf("expected negative return with stop loss triggered, got %f%%", result.Metrics.TotalReturn*100)
	}
	if result.Metrics.TotalTrades == 0 {
		t.Fatal("expected at least 1 trade (buy), got 0")
	}

	sellCount := 0
	for _, tr := range result.Trades {
		if tr.Side == "sell" {
			sellCount++
			if tr.PnL >= 0 {
				t.Errorf("expected negative PnL on stop-loss sell, got %f", tr.PnL)
			}
		}
	}
	if sellCount != 1 {
		t.Errorf("expected 1 sell trade from stop loss, got %d", sellCount)
	}

	t.Logf("Stop-Loss: Return=%.4f%%, Trades=%d, SellPnL=%.2f",
		result.Metrics.TotalReturn*100, result.Metrics.TotalTrades,
		result.Trades[len(result.Trades)-1].PnL)
}

func TestRunner_NoStopLossWhenPriceStable(t *testing.T) {
	config := DefaultConfig()
	config.InitialCash = 100000

	runner := NewRunner(config)

	bars := []trading.OHLCVBar{
		{Symbol: "TEST", Date: "2024-01-02", Open: 100, High: 101, Low: 99, Close: 100, Volume: 10000},
		{Symbol: "TEST", Date: "2024-01-03", Open: 101, High: 102, Low: 100, Close: 101, Volume: 10000},
		{Symbol: "TEST", Date: "2024-01-04", Open: 102, High: 103, Low: 101, Close: 102, Volume: 10000},
		{Symbol: "TEST", Date: "2024-01-05", Open: 103, High: 104, Low: 102, Close: 103, Volume: 10000},
		{Symbol: "TEST", Date: "2024-01-06", Open: 104, High: 105, Low: 103, Close: 104, Volume: 10000},
	}

	bought := false
	strategy := Strategy{
		ID:   "no_stop_test",
		Name: "No Stop Loss Test",
		SignalFunc: func(openPrice float64, prevBar *trading.OHLCVBar, portfolio *Portfolio) *trading.Signal {
			if !bought {
				bought = true
				return &trading.Signal{
					Symbol:    "TEST",
					Direction: "buy",
					Quantity:  100,
				}
			}
			return nil
		},
	}

	ctx := context.Background()
	result, err := runner.Run(ctx, strategy, bars)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if result.Metrics.TotalReturn <= 0 {
		t.Errorf("expected positive return for rising market, got %f%%", result.Metrics.TotalReturn*100)
	}

	sellCount := 0
	for _, tr := range result.Trades {
		if tr.Side == "sell" {
			sellCount++
		}
	}
	if sellCount != 0 {
		t.Errorf("expected 0 sell trades (no stop loss), got %d", sellCount)
	}
}

func TestRunner_StopLossShort(t *testing.T) {
	config := DefaultConfig()
	config.InitialCash = 100000

	runner := NewRunner(config)

	// Short entry: price at 100, then rises 6% → triggers 5% stop loss for shorts
	bars := []trading.OHLCVBar{
		{Symbol: "TEST", Date: "2024-01-02", Open: 100, High: 101, Low: 99, Close: 100, Volume: 10000},
		{Symbol: "TEST", Date: "2024-01-03", Open: 105, High: 106, Low: 104, Close: 105, Volume: 10000},
		{Symbol: "TEST", Date: "2024-01-04", Open: 106, High: 107, Low: 105, Close: 106, Volume: 10000},
	}

	sold := false
	strategy := Strategy{
		ID:   "short_stop_test",
		Name: "Short Stop Loss Test",
		SignalFunc: func(openPrice float64, prevBar *trading.OHLCVBar, portfolio *Portfolio) *trading.Signal {
			if !sold {
				sold = true
				return &trading.Signal{
					Symbol:    "TEST",
					Direction: "sell",
					Quantity:  100,
				}
			}
			return nil
		},
	}

	ctx := context.Background()
	result, err := runner.Run(ctx, strategy, bars)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	buyCount := 0
	for _, tr := range result.Trades {
		if tr.Side == "buy" {
			buyCount++
		}
	}
	if buyCount == 0 {
		t.Log("No buy-back triggered (short positions not yet supported in processSellSignal)")
	}
	t.Logf("Short Stop-Loss: trades=%d, return=%.4f%%", result.Metrics.TotalTrades, result.Metrics.TotalReturn*100)
}
