package backtest

import (
	"math"
	"testing"
)

func TestPriceLimitFor_BoardRules(t *testing.T) {
	cases := []struct {
		symbol string
		ratio  float64
	}{
		{"600519", 0.10}, // 主板 贵州茅台
		{"000001", 0.10}, // 主板 平安银行
		{"300750", 0.20}, // 创业板 宁德时代
		{"301088", 0.20}, // 创业板
		{"688981", 0.20}, // 科创板 中芯国际
		{"830799", 0.30}, // 北交所
	}
	for _, c := range cases {
		got := PriceLimitFor(c.symbol)
		if math.Abs(got.Ratio-c.ratio) > 1e-9 {
			t.Errorf("PriceLimitFor(%s) = %v, want ratio %v", c.symbol, got, c.ratio)
		}
	}
}

func TestPriceLimit_LimitUpDown(t *testing.T) {
	r := PriceLimitRule{Ratio: 0.10}
	if up := r.LimitUp(10.0); math.Abs(up-11.0) > 1e-9 {
		t.Errorf("LimitUp(10) = %v, want 11.0", up)
	}
	if down := r.LimitDown(10.0); math.Abs(down-9.0) > 1e-9 {
		t.Errorf("LimitDown(10) = %v, want 9.0", down)
	}
	// No prevClose → no limit
	if up := r.LimitUp(0); up != 0 {
		t.Errorf("LimitUp(0) = %v, want 0", up)
	}
}

func TestPriceLimit_CanBuyCanSell(t *testing.T) {
	r := PriceLimitRule{Ratio: 0.10} // ±10%, prevClose=10 → [9, 11]

	// 涨停价 11.0 买入应被拒
	if r.CanBuy(11.0, 10.0) {
		t.Error("CanBuy at limit-up should be false")
	}
	// 10.5 买入允许
	if !r.CanBuy(10.5, 10.0) {
		t.Error("CanBuy at 10.5 should be true")
	}
	// 跌停价 9.0 卖出应被拒
	if r.CanSell(9.0, 10.0) {
		t.Error("CanSell at limit-down should be false")
	}
	// 9.5 卖出允许
	if !r.CanSell(9.5, 10.0) {
		t.Error("CanSell at 9.5 should be true")
	}
	// 首日无 prevClose → 不限制
	if !r.CanBuy(999, 0) || !r.CanSell(1, 0) {
		t.Error("no prevClose should allow any price")
	}
}
