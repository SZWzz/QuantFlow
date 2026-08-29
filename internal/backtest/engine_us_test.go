package backtest

import (
	"context"
	"math"
	"testing"
	"time"

	"quantflow/internal/trading"
)

// TestUSEngine_RoundTrip 验证美股基本买卖闭环与单边佣金模型。
// day1 open=10 买入 100 股（佣金 0.1% → 成本 1001），
// day2 open=11 卖出（收入 1100*0.999=1098.9），PnL ≈ 97.9。
func TestUSEngine_RoundTrip(t *testing.T) {
	bars := []trading.OHLCVBar{
		{Symbol: "AAPL", Date: "2026-06-01", Open: 10, High: 10, Low: 10, Close: 10, Volume: 10000},
		{Symbol: "AAPL", Date: "2026-06-02", Open: 11, High: 11, Low: 11, Close: 11, Volume: 10000},
	}

	var barIdx int
	strategy := Strategy{
		ID:   "test-us-roundtrip",
		Name: "us roundtrip",
		SignalFunc: func(openPrice float64, prevBar *trading.OHLCVBar, portfolio *Portfolio) *trading.Signal {
			barIdx++
			switch barIdx {
			case 1:
				return &trading.Signal{Direction: "buy", Quantity: 100}
			case 2:
				return &trading.Signal{Direction: "sell", Quantity: 100}
			}
			return nil
		},
	}

	engine := NewUSEngine(Config{InitialCash: 100000})
	result, err := engine.Run(context.Background(), strategy, bars)
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}

	if len(result.Trades) != 2 {
		t.Fatalf("expected 2 trades, got %d", len(result.Trades))
	}
	sell := result.Trades[1]
	// 毛盈亏 +100，扣除双边佣金后 ≈ 97.9
	if math.Abs(sell.PnL-97.9) > 0.5 {
		t.Errorf("expected PnL ≈ 97.9, got %.4f", sell.PnL)
	}
}

// TestUSEngine_FractionalDefaultQuantity 验证美股碎股：信号数量为 0 时默认 1 股。
func TestUSEngine_FractionalDefaultQuantity(t *testing.T) {
	bars := []trading.OHLCVBar{
		{Symbol: "AAPL", Date: "2026-06-01", Open: 10, High: 10, Low: 10, Close: 10, Volume: 10000},
	}

	strategy := Strategy{
		ID:   "test-us-fractional",
		Name: "us fractional",
		SignalFunc: func(openPrice float64, prevBar *trading.OHLCVBar, portfolio *Portfolio) *trading.Signal {
			return &trading.Signal{Direction: "buy", Quantity: 0}
		},
	}

	engine := NewUSEngine(Config{InitialCash: 100000})
	result, err := engine.Run(context.Background(), strategy, bars)
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if len(result.Trades) != 1 || result.Trades[0].Quantity != 1 {
		t.Errorf("expected 1 trade of 1 share, got %+v", result.Trades)
	}
}

// TestUSEngine_NoData 验证空数据返回错误。
func TestUSEngine_NoData(t *testing.T) {
	engine := NewUSEngine(Config{InitialCash: 100000})
	if _, err := engine.Run(context.Background(), Strategy{ID: "x"}, nil); err == nil {
		t.Error("expected error for empty bars, got nil")
	}
}

// TestPDTTracker_DayTradesIn5Days 验证 PDT 5 个交易日滚动窗口计数。
func TestPDTTracker_DayTradesIn5Days(t *testing.T) {
	day := func(d string) time.Time {
		dt, _ := time.Parse("2006-01-02", d)
		return dt
	}
	// 6 个连续交易日
	tradingDates := []time.Time{day("2026-06-01"), day("2026-06-02"), day("2026-06-03"), day("2026-06-04"), day("2026-06-05"), day("2026-06-08")}

	tracker := newPDTTracker()
	for _, d := range tradingDates[:4] {
		tracker.recordDayTrade(d)
	}

	if got := tracker.dayTradesIn5Days(day("2026-06-05"), tradingDates); got != 4 {
		t.Errorf("expected 4 day trades in 5-day window, got %d", got)
	}
	// 在 6/8 回看 5 个交易日：窗口边界为 6/1（含），4 笔仍全部在窗口内
	if got := tracker.dayTradesIn5Days(day("2026-06-08"), tradingDates); got != 4 {
		t.Errorf("expected 4 day trades on 6/8 (boundary inclusive), got %d", got)
	}
	// 加入更早的 5/29 后窗口滑动：5/29 的那笔被排除
	tradingDates = append([]time.Time{day("2026-05-29")}, tradingDates...)
	tracker.recordDayTrade(day("2026-05-29"))
	if got := tracker.dayTradesIn5Days(day("2026-06-08"), tradingDates); got != 4 {
		t.Errorf("expected 4 day trades after window slides past 5/29, got %d", got)
	}
}

// TestPDTTracker_IsPDT 验证 PDT 触发条件：≥4 次日内交易 且 净值 < $25,000。
func TestPDTTracker_IsPDT(t *testing.T) {
	day := func(d string) time.Time {
		dt, _ := time.Parse("2006-01-02", d)
		return dt
	}
	tradingDates := []time.Time{day("2026-06-01"), day("2026-06-02"), day("2026-06-03"), day("2026-06-04"), day("2026-06-05")}

	tracker := newPDTTracker()
	for _, d := range tradingDates[:4] {
		tracker.recordDayTrade(d)
	}

	if !tracker.isPDT(day("2026-06-05"), 20000, tradingDates) {
		t.Error("expected PDT triggered: 4 day trades + equity < $25k")
	}
	if tracker.isPDT(day("2026-06-05"), 30000, tradingDates) {
		t.Error("expected PDT NOT triggered: equity >= $25k exempt")
	}

	light := newPDTTracker()
	for _, d := range tradingDates[:3] {
		light.recordDayTrade(d)
	}
	if light.isPDT(day("2026-06-05"), 20000, tradingDates) {
		t.Error("expected PDT NOT triggered: only 3 day trades")
	}
}

// TestExtractTradingDates 验证交易日提取：去重 + 排序 + 忽略非法日期。
func TestExtractTradingDates(t *testing.T) {
	bars := []trading.OHLCVBar{
		{Date: "2026-06-03"}, {Date: "2026-06-01"}, {Date: "2026-06-03"},
		{Date: "not-a-date"}, {Date: "2026-06-02"},
	}
	dates := extractTradingDates(bars)
	if len(dates) != 3 {
		t.Fatalf("expected 3 unique valid dates, got %d", len(dates))
	}
	for i := 1; i < len(dates); i++ {
		if !dates[i].After(dates[i-1]) {
			t.Errorf("dates not sorted: %v", dates)
		}
	}
	if dates[0].Format("2006-01-02") != "2026-06-01" {
		t.Errorf("expected first date 2026-06-01, got %s", dates[0].Format("2006-01-02"))
	}
}

// TestUSEngine_StopLossSellsNextDay 验证止损在次日真实成交。
// 日线回测中买入与止损卖出不可能落在同一根 bar（止损检查先于信号），
// 因此次日止损卖出不构成日内交易，PDT 计数为 0。
func TestUSEngine_StopLossSellsNextDay(t *testing.T) {
	bars := []trading.OHLCVBar{
		{Symbol: "AAPL", Date: "2026-06-01", Open: 10, High: 10, Low: 10, Close: 10, Volume: 10000},
		{Symbol: "AAPL", Date: "2026-06-02", Open: 9, High: 9, Low: 9, Close: 9, Volume: 10000},
	}

	var barIdx int
	strategy := Strategy{
		ID:   "test-us-stoploss",
		Name: "us stop-loss next day",
		SignalFunc: func(openPrice float64, prevBar *trading.OHLCVBar, portfolio *Portfolio) *trading.Signal {
			barIdx++
			if barIdx == 1 {
				return &trading.Signal{Direction: "buy", Quantity: 100}
			}
			return nil
		},
		RiskConfig: trading.RiskConfig{StopLossPct: 0.05}, // 5% 止损 → 9.5，day2 close=9 触发
	}

	engine := NewUSEngine(Config{InitialCash: 100000})
	result, err := engine.Run(context.Background(), strategy, bars)
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}

	sells := 0
	for _, tr := range result.Trades {
		if tr.Side == "sell" {
			sells++
			if tr.Price != 9 {
				t.Errorf("stop-loss should fill at close=9, got %.2f", tr.Price)
			}
		}
	}
	if sells != 1 {
		t.Errorf("expected 1 stop-loss sell, got %d", sells)
	}
	// 次日卖出 → 不是日内交易
	if got := len(engine.pdt.trades); got != 0 {
		t.Errorf("expected 0 recorded day trades (next-day sell), got %d", got)
	}
}
