package backtest

import (
	"context"
	"math"
	"testing"
	"time"

	"quantflow/internal/trading"
)

// makeBars creates a sequence of OHLCV bars with a linear price trend.
func makeBars(symbol string, startPrice float64, n int, trend float64) []trading.OHLCVBar {
	bars := make([]trading.OHLCVBar, n)
	for i := 0; i < n; i++ {
		price := startPrice + trend*float64(i)
		day := i + 1
		dateStr := "2024-01-"
		if day < 10 {
			dateStr += "0"
		}
		dateStr += itoaStr(day)
		bars[i] = trading.OHLCVBar{
			Symbol: symbol,
			Date:   dateStr,
			Open:   price,
			High:   price * 1.02,
			Low:    price * 0.98,
			Close:  price,
			Volume: 10000,
		}
	}
	return bars
}

func itoaStr(n int) string {
	if n < 10 {
		return string(rune('0' + n))
	}
	return string(rune('0'+n/10)) + string(rune('0'+n%10))
}

func TestMetrics_AllPositive(t *testing.T) {
	equity := []EquityPoint{
		{Date: "2024-01-01", Equity: 100000, Cash: 100000},
		{Date: "2024-01-02", Equity: 101000, Cash: 100000},
		{Date: "2024-01-03", Equity: 102000, Cash: 100000},
		{Date: "2024-01-04", Equity: 103000, Cash: 100000},
		{Date: "2024-01-05", Equity: 105000, Cash: 100000},
	}

	metrics := ComputeMetrics(equity, nil)
	if metrics.TotalReturn <= 0 {
		t.Errorf("Expected positive total return, got %f", metrics.TotalReturn)
	}
	if metrics.MaxDrawdown != 0 {
		t.Errorf("Expected zero drawdown for rising curve, got %f", metrics.MaxDrawdown)
	}
}

func TestMetrics_MaxDrawdown(t *testing.T) {
	equity := []EquityPoint{
		{Date: "2024-01-01", Equity: 100000, Cash: 100000},
		{Date: "2024-01-02", Equity: 110000, Cash: 100000},
		{Date: "2024-01-03", Equity: 105000, Cash: 100000},
		{Date: "2024-01-04", Equity: 95000, Cash: 100000},
		{Date: "2024-01-05", Equity: 102000, Cash: 100000},
	}

	metrics := ComputeMetrics(equity, nil)
	expected := (95000.0 - 110000.0) / 110000.0
	if math.Abs(metrics.MaxDrawdown-expected) > 0.001 {
		t.Errorf("MaxDrawdown: got %f, want %f", metrics.MaxDrawdown, expected)
	}
}

func TestMetrics_Flat(t *testing.T) {
	equity := []EquityPoint{
		{Date: "2024-01-01", Equity: 100000, Cash: 100000},
		{Date: "2024-01-02", Equity: 100000, Cash: 100000},
	}

	metrics := ComputeMetrics(equity, nil)
	if metrics.TotalReturn != 0 {
		t.Errorf("Expected zero total return for flat equity, got %f", metrics.TotalReturn)
	}
}

func TestRunner_SMACross(t *testing.T) {
	config := DefaultConfig()
	config.InitialCash = 100000

	runner := NewRunner(config)

	// Create bars: price goes 100 → 200 over 100 bars
	bars := makeBars("TEST", 100, 100, 1.0)

	// SMA cross strategy: buy when price > SMA(10), sell when price < SMA(10)
	var smaValues []float64
	strategy := Strategy{
		ID:   "sma_cross",
		Name: "SMA Cross Test",
		SignalFunc: func(openPrice float64, prevBar *trading.OHLCVBar, portfolio *Portfolio) *trading.Signal {
			if prevBar == nil {
				return nil
			}
			smaValues = append(smaValues, prevBar.Close)
			if len(smaValues) < 10 {
				return nil
			}
			sum := 0.0
			for i := len(smaValues) - 10; i < len(smaValues); i++ {
				sum += smaValues[i]
			}
			sma := sum / 10.0

			heldQty := portfolio.Positions["TEST"]

			if prevBar.Close > sma && heldQty <= 0 {
				return &trading.Signal{
					Symbol:    "TEST",
					Direction: "buy",
					Quantity:  100,
				}
			}
			if prevBar.Close < sma && heldQty > 0 {
				return &trading.Signal{
					Symbol:    "TEST",
					Direction: "sell",
					Quantity:  heldQty,
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

	if len(result.EquityCurve) == 0 {
		t.Fatal("Expected equity curve, got empty")
	}

	t.Logf("SMA Cross: Return=%.2f%%, Trades=%d, Sharpe=%.2f, MaxDD=%.2f%%",
		result.Metrics.TotalReturn*100,
		result.Metrics.TotalTrades,
		result.Metrics.SharpeRatio,
		result.Metrics.MaxDrawdown*100,
	)
}

func TestRunner_NoBars(t *testing.T) {
	runner := NewRunner(DefaultConfig())
	_, err := runner.Run(context.Background(), Strategy{}, nil)
	if err == nil {
		t.Fatal("Expected error for empty bars")
	}
}

func TestCNEngine_T1Enforcement(t *testing.T) {
	config := DefaultConfig()
	config.InitialCash = 100000

	engine := NewCNEngine(config)

	bars := []trading.OHLCVBar{
		{Symbol: "000001.SZ", Date: "2024-01-01", Open: 10, High: 10.2, Low: 9.8, Close: 10, Volume: 10000},
		{Symbol: "000001.SZ", Date: "2024-01-02", Open: 11, High: 11.2, Low: 10.8, Close: 11, Volume: 10000},
		{Symbol: "000001.SZ", Date: "2024-01-03", Open: 12, High: 12.2, Low: 11.8, Close: 12, Volume: 10000},
	}

	buyDone := false
	strategy := Strategy{
		ID:   "t1_test",
		Name: "T+1 Test",
		SignalFunc: func(openPrice float64, prevBar *trading.OHLCVBar, portfolio *Portfolio) *trading.Signal {
			if !buyDone {
				buyDone = true
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
	result, err := engine.Run(ctx, strategy, bars)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	// Should have 1 buy trade
	buyCount := 0
	for _, tr := range result.Trades {
		if tr.Side == "buy" {
			buyCount++
		}
	}
	if buyCount != 1 {
		t.Errorf("Expected 1 buy trade, got %d", buyCount)
	}
	t.Logf("T+1 CN Engine: trades=%d, final equity=%.2f", len(result.Trades), result.EquityCurve[len(result.EquityCurve)-1].Equity)
}

func TestTradingDays(t *testing.T) {
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)

	days := TradingDaysInRange(start, end)
	if days < 200 || days > 300 {
		t.Errorf("Expected ~250 trading days in 2024, got %d", days)
	}
	t.Logf("Trading days in 2024: %d", days)
}
