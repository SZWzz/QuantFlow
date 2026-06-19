package portfolio

import (
	"testing"

	"quantflow/internal/trading"
)

func TestService_GetSummary(t *testing.T) {
	oms := trading.NewOMS()
	oms.PlaceOrder("AAPL", trading.SideBuy, trading.TypeLimit, 100, 150.0)

	var orderID string
	for _, o := range oms.GetOrders() {
		orderID = o.ID
	}

	oms.FillOrder(orderID, 100, 150.0)
	oms.UpdateMarketPrice("AAPL", 155.0)

	svc := NewService(oms)
	s := svc.GetSummary(50000.0)

	if s.TotalValue != 65500.0 {
		t.Errorf("total = %f, want 65500", s.TotalValue)
	}
	if s.TotalPnL != 500.0 {
		t.Errorf("pnl = %f, want 500", s.TotalPnL)
	}
}

func TestDetectMarket(t *testing.T) {
	tests := []struct {
		symbol   string
		expected string
	}{
		{"000001.SZ", "CN"},
		{"600519.SH", "CN"},
		{"00700.HK", "HK"},
		{"AAPL", "US"},
		{"BTCUSDT", "CRYPTO"},
	}
	for _, tt := range tests {
		if r := detectMarket(tt.symbol); r != tt.expected {
			t.Errorf("detectMarket(%q) = %q, want %q", tt.symbol, r, tt.expected)
		}
	}
}

func TestComputeMetrics(t *testing.T) {
	pnl := []*DailyPnL{
		{Date: "2024-01-01", TotalValue: 100000},
		{Date: "2024-01-02", TotalValue: 101000},
		{Date: "2024-01-03", TotalValue: 100500},
		{Date: "2024-01-04", TotalValue: 102000},
		{Date: "2024-01-05", TotalValue: 101800},
	}
	m := ComputeMetrics(pnl, 101800, 0.03)
	if m.TotalExposure != 101800 {
		t.Errorf("exposure = %f, want 101800", m.TotalExposure)
	}
	if m.DailyVol <= 0 {
		t.Error("daily vol should be > 0")
	}
}
