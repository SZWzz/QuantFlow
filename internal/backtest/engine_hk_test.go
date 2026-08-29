package backtest

import (
	"context"
	"math"
	"quantflow/internal/trading"
	"testing"
)

// TestHKEngine_RoundTripChargesFeesBothSides 验证 HK 双边收费模型。
// day1 open=10 买入 100 股，day2 open=11 卖出，毛盈亏 +100。
// 双边总费率 = commission 0.03% + stamp duty 0.13% + exchange/SFC fee 0.00843%。
// 买入成本 = 1000 * 1.0016843 = 1001.68（费用计入持仓成本）
// 卖出收入 = 1100 * (1 - 0.0016843) = 1098.15
// PnL = 1098.15 - 1001.68 ≈ +96.46 — 双边费用合计约 3.54 被正确扣除。
func TestHKEngine_RoundTripChargesFeesBothSides(t *testing.T) {
	bars := []trading.OHLCVBar{
		{Symbol: "00700", Date: "2026-06-01", Open: 10, High: 10, Low: 10, Close: 10, Volume: 10000},
		{Symbol: "00700", Date: "2026-06-02", Open: 11, High: 11, Low: 11, Close: 11, Volume: 10000},
	}

	var barIdx int
	strategy := Strategy{
		ID:   "test-hk-roundtrip",
		Name: "hk roundtrip",
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

	engine := NewHKEngine(Config{InitialCash: 100000})
	result, err := engine.Run(context.Background(), strategy, bars)
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}

	if len(result.Trades) != 2 {
		t.Fatalf("expected 2 trades, got %d", len(result.Trades))
	}
	sell := result.Trades[1]
	if sell.Side != "sell" {
		t.Fatalf("expected second trade to be sell, got %s", sell.Side)
	}
	// 毛盈亏 +100，双边费用 ≈ 3.54 → 净盈亏 ≈ +96.46
	if sell.PnL >= 100 {
		t.Errorf("expected fees to reduce gross profit below 100, got PnL %.4f", sell.PnL)
	}
	if math.Abs(sell.PnL-96.46) > 0.1 {
		t.Errorf("expected PnL ≈ 96.46 (fee model), got %.4f", sell.PnL)
	}
}

// TestHKEngine_LotSizeRounding 验证整手取整：信号 250 股按每手 100 取整为 200。
func TestHKEngine_LotSizeRounding(t *testing.T) {
	bars := []trading.OHLCVBar{
		{Symbol: "00700", Date: "2026-06-01", Open: 10, High: 10, Low: 10, Close: 10, Volume: 10000},
	}

	strategy := Strategy{
		ID:   "test-hk-lot",
		Name: "hk lot rounding",
		SignalFunc: func(openPrice float64, prevBar *trading.OHLCVBar, portfolio *Portfolio) *trading.Signal {
			return &trading.Signal{Direction: "buy", Quantity: 250}
		},
	}

	engine := NewHKEngine(Config{InitialCash: 100000})
	result, err := engine.Run(context.Background(), strategy, bars)
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}

	if len(result.Trades) != 1 {
		t.Fatalf("expected 1 trade, got %d", len(result.Trades))
	}
	if result.Trades[0].Quantity != 200 {
		t.Errorf("expected lot-rounded quantity 200, got %v", result.Trades[0].Quantity)
	}
}

// TestHKEngine_SubLotQuantityRejected 验证不足一手的信号不成交。
func TestHKEngine_SubLotQuantityRejected(t *testing.T) {
	bars := []trading.OHLCVBar{
		{Symbol: "00700", Date: "2026-06-01", Open: 10, High: 10, Low: 10, Close: 10, Volume: 10000},
	}

	strategy := Strategy{
		ID:   "test-hk-sublot",
		Name: "hk sub-lot rejected",
		SignalFunc: func(openPrice float64, prevBar *trading.OHLCVBar, portfolio *Portfolio) *trading.Signal {
			return &trading.Signal{Direction: "buy", Quantity: 50}
		},
	}

	engine := NewHKEngine(Config{InitialCash: 100000})
	result, err := engine.Run(context.Background(), strategy, bars)
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if len(result.Trades) != 0 {
		t.Errorf("expected 0 trades for sub-lot quantity, got %d", len(result.Trades))
	}
}

// TestHKEngine_InsufficientCash 验证现金不足时买入被拒。
func TestHKEngine_InsufficientCash(t *testing.T) {
	bars := []trading.OHLCVBar{
		{Symbol: "00700", Date: "2026-06-01", Open: 10, High: 10, Low: 10, Close: 10, Volume: 10000},
	}

	strategy := Strategy{
		ID:   "test-hk-nocash",
		Name: "hk insufficient cash",
		SignalFunc: func(openPrice float64, prevBar *trading.OHLCVBar, portfolio *Portfolio) *trading.Signal {
			return &trading.Signal{Direction: "buy", Quantity: 100}
		},
	}

	engine := NewHKEngine(Config{InitialCash: 500}) // 100 股 @10 需 ~1001.68
	result, err := engine.Run(context.Background(), strategy, bars)
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if len(result.Trades) != 0 {
		t.Errorf("expected 0 trades with insufficient cash, got %d", len(result.Trades))
	}
}

// TestHKEngine_NoData 验证空数据返回错误。
func TestHKEngine_NoData(t *testing.T) {
	engine := NewHKEngine(Config{InitialCash: 100000})
	_, err := engine.Run(context.Background(), Strategy{ID: "x"}, nil)
	if err == nil {
		t.Error("expected error for empty bars, got nil")
	}
}

// TestHKEngine_StopLossSellsAtClose 验证 HK 止损路径：触发后按收盘价卖出并扣除费用。
func TestHKEngine_StopLossSellsAtClose(t *testing.T) {
	bars := []trading.OHLCVBar{
		{Symbol: "00700", Date: "2026-06-01", Open: 10, High: 10, Low: 10, Close: 10, Volume: 10000},
		{Symbol: "00700", Date: "2026-06-02", Open: 9, High: 9, Low: 9, Close: 9, Volume: 10000},
	}

	var barIdx int
	strategy := Strategy{
		ID:   "test-hk-stoploss",
		Name: "hk stop loss",
		SignalFunc: func(openPrice float64, prevBar *trading.OHLCVBar, portfolio *Portfolio) *trading.Signal {
			barIdx++
			if barIdx == 1 {
				return &trading.Signal{Direction: "buy", Quantity: 100}
			}
			return nil
		},
		RiskConfig: trading.RiskConfig{StopLossPct: 0.05}, // 5% 止损 → 9.5，day2 close=9 触发
	}

	engine := NewHKEngine(Config{InitialCash: 100000})
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
}
