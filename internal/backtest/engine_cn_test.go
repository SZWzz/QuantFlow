package backtest

import (
	"context"
	"quantflow/internal/trading"
	"testing"
)

// TestCNEngine_RejectsBuyAtLimitUp 验证涨停价买入被拒。
// 构造两日行情：day0 close=10，day1 close=11（涨停）。
// day1 strategy 发 buy 信号，应被涨跌停校验拒绝，无成交记录。
func TestCNEngine_RejectsBuyAtLimitUp(t *testing.T) {
	bars := []trading.OHLCVBar{
		{Symbol: "600519", Date: "2026-06-01", Open: 10, High: 10, Low: 10, Close: 10, Volume: 1000},
		{Symbol: "600519", Date: "2026-06-02", Open: 11, High: 11, Low: 11, Close: 11, Volume: 1000},
	}

	// day1 (涨停) 发出买入信号
	var barIdx int
	strategy := Strategy{
		ID:   "test-limit-up",
		Name: "limit-up test",
		SignalFunc: func(openPrice float64, prevBar *trading.OHLCVBar, portfolio *Portfolio) *trading.Signal {
			barIdx++
			if barIdx == 2 {
				return &trading.Signal{Direction: "buy", Quantity: 100}
			}
			return nil
		},
	}

	engine := NewCNEngine(Config{InitialCash: 100000})
	result, err := engine.Run(context.Background(), strategy, bars)
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}

	// 应无任何买入成交（day1 涨停封板买不进）
	buys := 0
	for _, tr := range result.Trades {
		if tr.Side == "buy" {
			buys++
		}
	}
	if buys != 0 {
		t.Errorf("expected 0 buys at limit-up, got %d", buys)
	}
}

// TestCNEngine_AllowsNormalBuy 验证非涨停价正常买入。
func TestCNEngine_AllowsNormalBuy(t *testing.T) {
	bars := []trading.OHLCVBar{
		{Symbol: "600519", Date: "2026-06-01", Open: 10, High: 10, Low: 10, Close: 10, Volume: 1000},
		{Symbol: "600519", Date: "2026-06-02", Open: 10.5, High: 10.5, Low: 10.5, Close: 10.5, Volume: 1000},
	}

	var barIdx int
	strategy := Strategy{
		ID:   "test-normal-buy",
		Name: "normal buy test",
		SignalFunc: func(openPrice float64, prevBar *trading.OHLCVBar, portfolio *Portfolio) *trading.Signal {
			barIdx++
			if barIdx == 2 {
				return &trading.Signal{Direction: "buy", Quantity: 100}
			}
			return nil
		},
	}

	engine := NewCNEngine(Config{InitialCash: 100000})
	result, err := engine.Run(context.Background(), strategy, bars)
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}

	buys := 0
	for _, tr := range result.Trades {
		if tr.Side == "buy" {
			buys++
		}
	}
	if buys != 1 {
		t.Errorf("expected 1 normal buy, got %d", buys)
	}
}

// TestCNEngine_RejectsSellAtLimitDown 验证跌停价卖出被拒（信号路径）。
// day0 买入建仓 close=10，day1 close=9（跌停 ±10%）。
// day1 strategy 发 sell 信号，应被跌停校验拒绝，无卖出成交。
func TestCNEngine_RejectsSellAtLimitDown(t *testing.T) {
	bars := []trading.OHLCVBar{
		{Symbol: "600519", Date: "2026-06-01", Open: 10, High: 10, Low: 10, Close: 10, Volume: 1000},
		{Symbol: "600519", Date: "2026-06-02", Open: 9, High: 9, Low: 9, Close: 9, Volume: 1000},
	}

	var barIdx int
	strategy := Strategy{
		ID:   "test-limit-down-sell",
		Name: "limit-down sell test",
		SignalFunc: func(openPrice float64, prevBar *trading.OHLCVBar, portfolio *Portfolio) *trading.Signal {
			barIdx++
			if barIdx == 1 {
				return &trading.Signal{Direction: "buy", Quantity: 100}
			}
			if barIdx == 2 && portfolio.Positions["600519"] > 0 {
				return &trading.Signal{Direction: "sell", Quantity: 100}
			}
			return nil
		},
	}

	engine := NewCNEngine(Config{InitialCash: 100000})
	result, err := engine.Run(context.Background(), strategy, bars)
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}

	sells := 0
	for _, tr := range result.Trades {
		if tr.Side == "sell" {
			sells++
		}
	}
	if sells != 0 {
		t.Errorf("expected 0 sells at limit-down, got %d", sells)
	}
}

// TestCNEngine_ChiNextLimitUp 验证创业板 ±20% 在引擎层生效。
// 300750 day0 close=10，day1 close=12（涨停 ±20%，10*1.2=12）。
// day1 buy 信号应被拒。
func TestCNEngine_ChiNextLimitUp(t *testing.T) {
	bars := []trading.OHLCVBar{
		{Symbol: "300750", Date: "2026-06-01", Open: 10, High: 10, Low: 10, Close: 10, Volume: 1000},
		{Symbol: "300750", Date: "2026-06-02", Open: 12, High: 12, Low: 12, Close: 12, Volume: 1000},
	}

	var barIdx int
	strategy := Strategy{
		ID:   "test-chinext-limit",
		Name: "chinext limit test",
		SignalFunc: func(openPrice float64, prevBar *trading.OHLCVBar, portfolio *Portfolio) *trading.Signal {
			barIdx++
			if barIdx == 2 {
				return &trading.Signal{Direction: "buy", Quantity: 100}
			}
			return nil
		},
	}

	engine := NewCNEngine(Config{InitialCash: 100000})
	result, err := engine.Run(context.Background(), strategy, bars)
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}

	buys := 0
	for _, tr := range result.Trades {
		if tr.Side == "buy" {
			buys++
		}
	}
	if buys != 0 {
		t.Errorf("expected 0 buys at ChiNext limit-up (±20%%), got %d", buys)
	}
}

// TestCNEngine_RejectsStopLossAtLimitDown 验证跌停日止损单无法成交。
// day0 买入建仓 close=10，day1 close=9（跌停）。
// 风控配置 stop-loss 触发价 9.5，day1 应触发止损但被跌停校验拒绝。
func TestCNEngine_RejectsStopLossAtLimitDown(t *testing.T) {
	bars := []trading.OHLCVBar{
		{Symbol: "600519", Date: "2026-06-01", Open: 10, High: 10, Low: 10, Close: 10, Volume: 1000},
		{Symbol: "600519", Date: "2026-06-02", Open: 9, High: 9, Low: 9, Close: 9, Volume: 1000},
	}

	// day1 买入建仓，day2 跌停触发止损
	strategy := Strategy{
		ID:   "test-stoploss-limit-down",
		Name: "stoploss at limit-down test",
		SignalFunc: func(openPrice float64, prevBar *trading.OHLCVBar, portfolio *Portfolio) *trading.Signal {
			return &trading.Signal{Direction: "buy", Quantity: 100}
		},
		RiskConfig: trading.RiskConfig{
			StopLossPct: 0.05, // 5% stop-loss → triggers at 9.5, day2 close=9 触发
		},
	}

	engine := NewCNEngine(Config{InitialCash: 100000})
	result, err := engine.Run(context.Background(), strategy, bars)
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}

	sells := 0
	for _, tr := range result.Trades {
		if tr.Side == "sell" {
			sells++
		}
	}
	if sells != 0 {
		t.Errorf("expected 0 sells (stop-loss rejected at limit-down), got %d", sells)
	}
}
