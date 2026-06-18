package research

import "context"

// FinancialsService provides financial data and ratio computation.
type FinancialsService struct{}

// NewFinancialsService creates a new FinancialsService.
func NewFinancialsService() *FinancialsService {
	return &FinancialsService{}
}

// GetFinancials returns mock financial data for a symbol.
func (s *FinancialsService) GetFinancials(ctx context.Context, symbol string) (*FinancialData, error) {
	return &FinancialData{
		Symbol:       symbol,
		Revenue:      100_000_000_000,
		NetIncome:    25_000_000_000,
		EPS:          6.25,
		TotalAssets:  350_000_000_000,
		TotalEquity:  65_000_000_000,
		TotalDebt:    120_000_000_000,
		FreeCashFlow: 20_000_000_000,
		MarketCap:    2_500_000_000_000,
	}, nil
}

// ComputeRatios calculates key financial ratios from financial data.
func (s *FinancialsService) ComputeRatios(data *FinancialData) *FinancialRatios {
	if data == nil {
		return &FinancialRatios{}
	}
	r := &FinancialRatios{}
	if data.EPS > 0 {
		r.PE = data.MarketCap / (data.EPS * 4_000_000_000)
	}
	if data.TotalEquity > 0 {
		r.PB = data.MarketCap / data.TotalEquity
		r.ROE = data.NetIncome / data.TotalEquity
	}
	if data.TotalAssets > 0 {
		r.ROA = data.NetIncome / data.TotalAssets
	}
	if data.TotalEquity > 0 && data.TotalDebt > 0 {
		r.DebtToEquity = data.TotalDebt / data.TotalEquity
	}
	if data.Revenue > 0 {
		r.NetMargin = data.NetIncome / data.Revenue
	}
	return r
}
