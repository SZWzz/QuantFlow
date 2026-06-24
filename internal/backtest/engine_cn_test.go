package backtest

import (
	"context"
	"testing"

	"quantflow/internal/trading"
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
	strategy := Strategy{
		ID:   "test-limit-up",
		Name: "limit-up test",
		SignalFunc: func(bar trading.OHLCVBar, portfolio *Portfolio) *trading.Signal {
			if bar.Date == "2026-06-02" {
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

	strategy := Strategy{
		ID:   "test-normal-buy",
		Name: "normal buy test",
		SignalFunc: func(bar trading.OHLCVBar, portfolio *Portfolio) *trading.Signal {
			if bar.Date == "2026-06-02" {
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
