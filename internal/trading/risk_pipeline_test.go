package trading

import (
	"sync"
	"testing"
)

func TestCheckDrawdown_Normal(t *testing.T) {
	r := NewRiskPipeline(RiskConfig{PeakEquity: 100000, MaxDrawdownPct: 0.20})
	err := r.CheckDrawdown(90000)
	if err != nil {
		t.Error("expected no error for 10% drawdown:", err)
	}
}

func TestCheckDrawdown_Exceeds(t *testing.T) {
	r := NewRiskPipeline(RiskConfig{PeakEquity: 100000, MaxDrawdownPct: 0.20})
	err := r.CheckDrawdown(75000)
	if err == nil {
		t.Error("expected error for 25% drawdown")
	}
}

func TestCheckDrawdown_NewPeak(t *testing.T) {
	r := NewRiskPipeline(RiskConfig{PeakEquity: 100000, MaxDrawdownPct: 0.20})
	err := r.CheckDrawdown(110000)
	if err != nil {
		t.Fatal("unexpected error on new peak:", err)
	}
	err = r.CheckDrawdown(90000)
	if err != nil {
		t.Error("expected no error after peak update, got:", err)
	}
}

func TestCheckDrawdown_ZeroPeak(t *testing.T) {
	r := NewRiskPipeline(RiskConfig{PeakEquity: 0, MaxDrawdownPct: 0.20})
	err := r.CheckDrawdown(50000)
	if err != nil {
		t.Error("expected no error when peak is 0:", err)
	}
}

func TestCheckDrawdown_ExactBoundary(t *testing.T) {
	r := NewRiskPipeline(RiskConfig{PeakEquity: 100000, MaxDrawdownPct: 0.20})
	err := r.CheckDrawdown(80000)
	if err != nil {
		t.Error("exact 20% drawdown boundary should pass:", err)
	}
	err = r.CheckDrawdown(79999)
	if err == nil {
		t.Error("expected error for 20.001% drawdown")
	}
}

func TestCheckDrawdown_ConcurrentSafe(t *testing.T) {
	r := NewRiskPipeline(RiskConfig{PeakEquity: 100000, MaxDrawdownPct: 0.20})
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r.CheckDrawdown(95000)
		}()
	}
	wg.Wait()
}

func TestCheckOrder_MaxPosition(t *testing.T) {
	r := NewRiskPipeline(RiskConfig{MaxPositionPct: 0.25})
	// 100 * 500 = 50000 = 50% of 100000 > 25% → should fail
	err := r.CheckOrder(&Order{Quantity: 100, Price: 500.0}, nil, 100000)
	if err == nil {
		t.Error("expected error for exceeding max position")
	}
}

func TestCheckOrder_MaxPositionOK(t *testing.T) {
	r := NewRiskPipeline(RiskConfig{MaxPositionPct: 0.25})
	// 50 * 400 = 20000 = 20% of 100000 < 25% → should pass
	err := r.CheckOrder(&Order{Quantity: 50, Price: 400.0}, nil, 100000)
	if err != nil {
		t.Error("expected no error for position within limit:", err)
	}
}

func TestCheckOrder_ZeroPortfolio(t *testing.T) {
	r := NewRiskPipeline(RiskConfig{MaxPositionPct: 0.25})
	err := r.CheckOrder(&Order{Quantity: 10, Price: 100.0}, nil, 0)
	if err == nil {
		t.Error("expected error for zero portfolio value")
	}
}

func TestCheckOrder_MarketZeroPriceWithPosition(t *testing.T) {
	r := NewRiskPipeline(RiskConfig{MaxPositionPct: 0.25})
	pos := &Position{Symbol: "AAPL", Quantity: 100, AvgPrice: 200.0, MarketPrice: 205.0}
	// Market order with zero price, position has MarketPrice → uses position.MarketPrice
	// 100 * 205 = 20500 = 20.5% of 100000 < 25% → should pass
	err := r.CheckOrder(&Order{Quantity: 100, OrderType: TypeMarket, Price: 0}, pos, 100000)
	if err != nil {
		t.Error("expected no error for market order with position reference:", err)
	}
}

func TestCheckOrder_MarketZeroPriceExceedsWithPosition(t *testing.T) {
	r := NewRiskPipeline(RiskConfig{MaxPositionPct: 0.25})
	pos := &Position{Symbol: "AAPL", Quantity: 100, AvgPrice: 200.0, MarketPrice: 205.0}
	// 200 * 500 = 100000 = 100% > 25% → should fail
	err := r.CheckOrder(&Order{Quantity: 500, OrderType: TypeMarket, Price: 0}, pos, 100000)
	if err == nil {
		t.Error("expected error for market order exceeding max position")
	}
}

func TestCheckOrder_Disabled(t *testing.T) {
	r := NewRiskPipeline(RiskConfig{})
	// MaxPositionPct = 0 means disabled
	err := r.CheckOrder(&Order{Quantity: 1000, Price: 1000}, nil, 1000)
	if err != nil {
		t.Error("expected no error when max position check is disabled:", err)
	}
}

func TestCheckStopLoss_Long(t *testing.T) {
	r := NewRiskPipeline(RiskConfig{StopLossPct: 0.05})
	pos := &Position{Symbol: "AAPL", Quantity: 100, AvgPrice: 200.0}

	if !r.CheckStopLoss(pos, 189.0) {
		t.Error("stop loss should trigger at 5.5% drop")
	}
	if r.CheckStopLoss(pos, 191.0) {
		t.Error("stop loss should not trigger at 4.5% drop")
	}
}

func TestCheckStopLoss_Short(t *testing.T) {
	r := NewRiskPipeline(RiskConfig{StopLossPct: 0.05})
	pos := &Position{Symbol: "AAPL", Quantity: -100, AvgPrice: 200.0}

	if !r.CheckStopLoss(pos, 211.0) {
		t.Error("stop loss should trigger for short at 5.5% rise")
	}
	if r.CheckStopLoss(pos, 209.0) {
		t.Error("stop loss should not trigger for short at 4.5% rise")
	}
}

func TestCheckStopLoss_Disabled(t *testing.T) {
	r := NewRiskPipeline(RiskConfig{})
	pos := &Position{Symbol: "AAPL", Quantity: 100, AvgPrice: 200.0}
	if r.CheckStopLoss(pos, 1.0) {
		t.Error("stop loss should be disabled when StopLossPct == 0")
	}
}

func TestCheckStopLoss_NilPosition(t *testing.T) {
	r := NewRiskPipeline(RiskConfig{StopLossPct: 0.05})
	if r.CheckStopLoss(nil, 100.0) {
		t.Error("stop loss should not trigger on nil position")
	}
}

func TestCheckStopLoss_ZeroQty(t *testing.T) {
	r := NewRiskPipeline(RiskConfig{StopLossPct: 0.05})
	pos := &Position{Symbol: "AAPL", Quantity: 0, AvgPrice: 200.0}
	if r.CheckStopLoss(pos, 100.0) {
		t.Error("stop loss should not trigger on zero quantity position")
	}
}

func TestCheckTakeProfit_Long(t *testing.T) {
	r := NewRiskPipeline(RiskConfig{TakeProfitPct: 0.15})
	pos := &Position{Symbol: "AAPL", Quantity: 100, AvgPrice: 200.0}

	if !r.CheckTakeProfit(pos, 231.0) {
		t.Error("take profit should trigger at 15.5% gain")
	}
	if r.CheckTakeProfit(pos, 229.0) {
		t.Error("take profit should not trigger at 14.5% gain")
	}
}

func TestCheckTakeProfit_Short(t *testing.T) {
	r := NewRiskPipeline(RiskConfig{TakeProfitPct: 0.15})
	pos := &Position{Symbol: "AAPL", Quantity: -100, AvgPrice: 200.0}

	if !r.CheckTakeProfit(pos, 169.0) {
		t.Error("take profit should trigger for short at 15.5% drop")
	}
	if r.CheckTakeProfit(pos, 171.0) {
		t.Error("take profit should not trigger for short at 14.5% drop")
	}
}

func TestCheckTakeProfit_Disabled(t *testing.T) {
	r := NewRiskPipeline(RiskConfig{})
	pos := &Position{Symbol: "AAPL", Quantity: 100, AvgPrice: 200.0}
	if r.CheckTakeProfit(pos, 1000.0) {
		t.Error("take profit should be disabled when TakeProfitPct == 0")
	}
}

func TestDefaultRiskConfig(t *testing.T) {
	cfg := DefaultRiskConfig()
	if cfg.MaxPositionPct != 0.25 {
		t.Errorf("MaxPositionPct = %f, want 0.25", cfg.MaxPositionPct)
	}
	if cfg.StopLossPct != 0.05 {
		t.Errorf("StopLossPct = %f, want 0.05", cfg.StopLossPct)
	}
	if cfg.TakeProfitPct != 0.15 {
		t.Errorf("TakeProfitPct = %f, want 0.15", cfg.TakeProfitPct)
	}
	if cfg.MaxDrawdownPct != 0.20 {
		t.Errorf("MaxDrawdownPct = %f, want 0.20", cfg.MaxDrawdownPct)
	}
}
