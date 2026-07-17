package trading

import (
	"testing"
)

func TestGenerateDailyReport_Basic(t *testing.T) {
	oms := NewOMS()

	// Place and fill some trades
	order1, _ := oms.PlaceOrder("AAPL", SideBuy, TypeMarket, "", 100, 195.0)
	oms.FillOrder(order1.ID, 100, 195.5)
	trades := oms.GetTrades()
	trades[0].Commission = 5.0

	order2, _ := oms.PlaceOrder("TSLA", SideBuy, TypeMarket, "", 50, 245.0)
	oms.FillOrder(order2.ID, 50, 246.0)
	trades = oms.GetTrades()
	trades[1].Commission = 3.0

	// Update market prices for P&L
	oms.UpdateMarketPrice("AAPL", 198.0)
	oms.UpdateMarketPrice("TSLA", 250.0)

	report := GenerateDailyReport(oms, "2026-07-16")
	if report.Date != "2026-07-16" {
		t.Errorf("expected date 2026-07-16, got %s", report.Date)
	}
	if report.Trades != 2 {
		t.Errorf("expected 2 trades, got %d", report.Trades)
	}
	if report.Commission != 8.0 {
		t.Errorf("expected commission 8.0, got %f", report.Commission)
	}
	if len(report.Positions) != 2 {
		t.Errorf("expected 2 positions, got %d", len(report.Positions))
	}
}

func TestGenerateDailyReport_Empty(t *testing.T) {
	oms := NewOMS()
	report := GenerateDailyReport(oms, "2026-07-16")
	if report.Date != "2026-07-16" {
		t.Errorf("expected date 2026-07-16, got %s", report.Date)
	}
	if report.Trades != 0 {
		t.Errorf("expected 0 trades, got %d", report.Trades)
	}
	if report.DayPNL != 0 {
		t.Errorf("expected 0 P&L, got %f", report.DayPNL)
	}
}

func TestEncodeDecodeDailyReport(t *testing.T) {
	oms := NewOMS()
	order, _ := oms.PlaceOrder("AAPL", SideBuy, TypeMarket, "", 100, 195.0)
	oms.FillOrder(order.ID, 100, 195.5)
	oms.UpdateMarketPrice("AAPL", 198.0)

	report := GenerateDailyReport(oms, "2026-07-16")

	data, err := EncodeDailyReport(report)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	decoded, err := DecodeDailyReport(data)
	if err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if decoded.Date != report.Date {
		t.Errorf("date mismatch: %s vs %s", decoded.Date, report.Date)
	}
	if decoded.Trades != report.Trades {
		t.Errorf("trades mismatch: %d vs %d", decoded.Trades, report.Trades)
	}
}
