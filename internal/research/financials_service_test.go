package research

import (
	"context"
	"testing"
)

func TestFinancialsService_GetFinancials_MockFallback(t *testing.T) {
	svc := NewFinancialsService(nil, nil)
	data, err := svc.GetFinancials(context.Background(), "AAPL")
	if err != nil {
		t.Fatal(err)
	}
	if data == nil {
		t.Fatal("expected non-nil mock financial data")
	}
	if data.Symbol != "AAPL" {
		t.Errorf("expected symbol AAPL, got %s", data.Symbol)
	}
	if data.Revenue <= 0 {
		t.Error("expected positive revenue in mock data")
	}
	if data.EPS <= 0 {
		t.Error("expected positive EPS in mock data")
	}
}

func TestFinancialsService_ComputeRatios_NilData(t *testing.T) {
	svc := NewFinancialsService(nil, nil)
	ratios := svc.ComputeRatios(nil)
	if ratios == nil {
		t.Fatal("expected non-nil ratios even with nil input")
	}
}

func TestFinancialsService_ComputeRatios_ValidData(t *testing.T) {
	svc := NewFinancialsService(nil, nil)
	data := &FinancialData{
		Symbol:      "TEST",
		Revenue:     100_000_000_000,
		NetIncome:   25_000_000_000,
		EPS:         6.25,
		TotalAssets: 350_000_000_000,
		TotalEquity: 65_000_000_000,
		TotalDebt:   120_000_000_000,
		MarketCap:   2_500_000_000_000,
	}
	ratios := svc.ComputeRatios(data)
	if ratios.ROE <= 0 {
		t.Error("expected positive ROE")
	}
	if ratios.DebtToEquity <= 0 {
		t.Error("expected positive debt-to-equity")
	}
}
