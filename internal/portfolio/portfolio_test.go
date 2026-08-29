package portfolio

import (
	"math"
	"quantflow/internal/trading"
	"testing"
)

func TestService_EmptyPortfolio(t *testing.T) {
	oms := trading.NewOMS()
	svc := NewService(oms)
	s := svc.GetSummary()
	if s.TotalValue != 0 {
		t.Errorf("TotalValue = %f, want 0", s.TotalValue)
	}
	if s.CashBalance != 0 {
		t.Errorf("CashBalance = %f, want 0", s.CashBalance)
	}
	if s.MarketValue != 0 {
		t.Errorf("MarketValue = %f, want 0", s.MarketValue)
	}
	if s.TotalPnL != 0 {
		t.Errorf("TotalPnL = %f, want 0", s.TotalPnL)
	}
}

func TestService_MultiplePositions_SortedByAllocation(t *testing.T) {
	oms := trading.NewOMS()
	oms.GetCashLedger().Deposit(200000)

	for _, order := range []struct {
		sym   string
		qty   float64
		price float64
	}{
		{"AAPL", 100, 150.0},
		{"GOOGL", 50, 200.0},
		{"MSFT", 200, 50.0},
	} {
		o, err := oms.PlaceOrder(order.sym, trading.SideBuy, trading.TypeMarket, "", order.qty, 0)
		if err != nil {
			t.Fatal(err)
		}
		oms.FillOrder(o.ID, order.qty, order.price)
	}

	oms.UpdateMarketPrice("AAPL", 180.0)
	oms.UpdateMarketPrice("GOOGL", 220.0)
	oms.UpdateMarketPrice("MSFT", 60.0)

	svc := NewService(oms)
	details := svc.GetPositions()

	if len(details) != 3 {
		t.Fatalf("expected 3 positions, got %d", len(details))
	}

	// Should be sorted by allocation descending
	if details[0].AllocPct < details[1].AllocPct || details[1].AllocPct < details[2].AllocPct {
		t.Error("positions not sorted by allocation descending")
		for i, d := range details {
			t.Logf("  [%d] %s: qty=%.0f alloc=%.2f%%", i, d.Symbol, d.Quantity, d.AllocPct)
		}
	}

	totalAlloc := details[0].AllocPct + details[1].AllocPct + details[2].AllocPct
	if math.Abs(totalAlloc-100.0) > 0.01 {
		t.Errorf("total allocation = %.2f%%, want 100%%", totalAlloc)
	}
}

func TestService_GetSummaryWithPositions(t *testing.T) {
	oms := trading.NewOMS()
	oms.GetCashLedger().Deposit(100000)

	o, _ := oms.PlaceOrder("AAPL", trading.SideBuy, trading.TypeMarket, "", 100, 0)
	oms.FillOrder(o.ID, 100, 150.0)
	oms.UpdateMarketPrice("AAPL", 160.0)

	svc := NewService(oms)
	s := svc.GetSummary()

	if s.MarketValue != 16000 {
		t.Errorf("MarketValue = %f, want 16000", s.MarketValue)
	}
	if s.TotalPnL != 1000 {
		t.Errorf("TotalPnL = %f, want 1000", s.TotalPnL)
	}
}

func TestService_GetAllocation(t *testing.T) {
	oms := trading.NewOMS()
	oms.GetCashLedger().Deposit(100000)

	for _, order := range []struct {
		sym   string
		qty   float64
		price float64
	}{
		{"000001.SZ", 1000, 10.0},
		{"00700.HK", 500, 50.0},
		{"AAPL", 100, 150.0},
	} {
		o, _ := oms.PlaceOrder(order.sym, trading.SideBuy, trading.TypeMarket, "", order.qty, 0)
		oms.FillOrder(o.ID, order.qty, order.price)
	}

	svc := NewService(oms)
	alloc := svc.GetAllocation()

	if len(alloc.ByMarket) == 0 {
		t.Fatal("expected non-empty ByMarket allocation")
	}
	if _, ok := alloc.ByMarket["CN"]; !ok {
		t.Error("expected CN market in allocation")
	}
	if _, ok := alloc.ByMarket["HK"]; !ok {
		t.Error("expected HK market in allocation")
	}
	if _, ok := alloc.ByMarket["US"]; !ok {
		t.Error("expected US market in allocation")
	}
	if _, ok := alloc.ByCurrency["CNY"]; !ok {
		t.Error("expected CNY currency in allocation")
	}
	if _, ok := alloc.ByCurrency["HKD"]; !ok {
		t.Error("expected HKD currency in allocation")
	}
	if _, ok := alloc.ByCurrency["USD"]; !ok {
		t.Error("expected USD currency in allocation")
	}
}

func TestService_PersistWithNilRepo(t *testing.T) {
	oms := trading.NewOMS()
	svc := NewService(oms)
	if err := svc.RecordDailySnapshot(); err != nil {
		t.Error("expected no error when repo is nil:", err)
	}
	history, err := svc.GetPnLHistory(30)
	if err != nil {
		t.Error("expected no error when repo is nil:", err)
	}
	if history != nil {
		t.Error("expected nil history when repo is nil")
	}
}

func TestComputeMetrics_EmptyData(t *testing.T) {
	m := ComputeMetrics(nil, 100000, 0.03)
	if m == nil {
		t.Fatal("expected non-nil metrics for nil input")
	}
	if m.TotalExposure != 100000 {
		t.Errorf("TotalExposure = %f, want 100000", m.TotalExposure)
	}

	m = ComputeMetrics([]*DailyPnL{}, 100000, 0.03)
	if m.TotalExposure != 100000 {
		t.Errorf("TotalExposure = %f, want 100000", m.TotalExposure)
	}
}

func TestComputeMetrics_SingleDay(t *testing.T) {
	pnl := []*DailyPnL{
		{Date: "2024-01-01", TotalValue: 100000},
	}
	m := ComputeMetrics(pnl, 100000, 0.03)
	if m.TotalExposure != 100000 {
		t.Errorf("TotalExposure = %f, want 100000", m.TotalExposure)
	}
}

func TestComputeMetrics_RisingMarket(t *testing.T) {
	pnl := []*DailyPnL{
		{Date: "2024-01-05", TotalValue: 105000},
		{Date: "2024-01-04", TotalValue: 104000},
		{Date: "2024-01-03", TotalValue: 103000},
		{Date: "2024-01-02", TotalValue: 102000},
		{Date: "2024-01-01", TotalValue: 100000},
	}
	m := ComputeMetrics(pnl, 105000, 0.02)

	if m.TotalExposure != 105000 {
		t.Errorf("TotalExposure = %f, want 105000", m.TotalExposure)
	}
	if m.SharpeRatio <= 0 {
		t.Errorf("expected positive Sharpe for rising market, got %f", m.SharpeRatio)
	}
	if m.MaxDrawdown != 0 {
		t.Errorf("expected 0 drawdown for strictly rising market, got %f", m.MaxDrawdown)
	}
	if m.DailyVol <= 0 {
		t.Error("expected positive daily volatility")
	}
}

func TestComputeMetrics_FallingMarket(t *testing.T) {
	pnl := []*DailyPnL{
		{Date: "2024-01-05", TotalValue: 90000},
		{Date: "2024-01-04", TotalValue: 93000},
		{Date: "2024-01-03", TotalValue: 96000},
		{Date: "2024-01-02", TotalValue: 98000},
		{Date: "2024-01-01", TotalValue: 100000},
	}
	m := ComputeMetrics(pnl, 90000, 0.02)

	if m.TotalExposure != 90000 {
		t.Errorf("TotalExposure = %f, want 90000", m.TotalExposure)
	}
	if m.SharpeRatio >= 0 {
		t.Errorf("expected negative Sharpe for falling market, got %f", m.SharpeRatio)
	}
	if m.MaxDrawdown <= 0 {
		t.Errorf("expected positive drawdown for falling market, got %f", m.MaxDrawdown)
	}
}

func TestComputeMetrics_MaxDrawdownDates(t *testing.T) {
	pnl := []*DailyPnL{
		{Date: "2024-01-06", TotalValue: 102000},
		{Date: "2024-01-05", TotalValue: 98000},
		{Date: "2024-01-04", TotalValue: 95000},
		{Date: "2024-01-03", TotalValue: 110000},
		{Date: "2024-01-02", TotalValue: 105000},
		{Date: "2024-01-01", TotalValue: 100000},
	}
	m := ComputeMetrics(pnl, 102000, 0.02)

	if m.MaxDrawdown <= 0 {
		t.Error("expected positive max drawdown")
	}
	if m.MaxDDStart == "" {
		t.Error("expected non-empty max drawdown start date")
	}
	if m.MaxDDEnd == "" {
		t.Error("expected non-empty max drawdown end date")
	}
	t.Logf("MaxDD: %.4f, Start: %s, End: %s", m.MaxDrawdown, m.MaxDDStart, m.MaxDDEnd)
}

func TestComputeMetrics_VolatilityValues(t *testing.T) {
	pnl := []*DailyPnL{
		{Date: "2024-01-10", TotalValue: 105000},
		{Date: "2024-01-09", TotalValue: 102000},
		{Date: "2024-01-08", TotalValue: 108000},
		{Date: "2024-01-07", TotalValue: 101000},
		{Date: "2024-01-06", TotalValue: 107000},
		{Date: "2024-01-05", TotalValue: 100000},
		{Date: "2024-01-04", TotalValue: 106000},
		{Date: "2024-01-03", TotalValue: 99000},
		{Date: "2024-01-02", TotalValue: 104000},
		{Date: "2024-01-01", TotalValue: 100000},
	}
	m := ComputeMetrics(pnl, 105000, 0.02)

	if m.DailyVol <= 0 {
		t.Error("expected positive daily volatility")
	}
	if m.AnnualVol <= m.DailyVol {
		t.Error("expected annual vol > daily vol (annualized by sqrt(252))")
	}
	if m.SortinoRatio == 0 && m.SharpeRatio != 0 {
		t.Error("Sortino should be non-zero when Sharpe is non-zero")
	}
}
