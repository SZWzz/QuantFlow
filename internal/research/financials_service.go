package research

import (
	"context"
	"encoding/json"
	"log/slog"
	"strconv"
	"strings"

	"quantflow/internal/market/adapters"
)

// FinancialsService provides financial data and ratio computation.
// Priority: mootdx Finance (real-time quarterly snapshots) → Sina financials
// (balance sheet + income statement) → mock data.
type FinancialsService struct {
	sinaAdapter   *adapters.SinaFinancialsAdapter
	mootdxAdapter *adapters.MootdxAdapter
}

// NewFinancialsService creates a new FinancialsService.
// Both adapters may be nil for mock mode.
func NewFinancialsService(sinaAdapter *adapters.SinaFinancialsAdapter, mootdxAdapter *adapters.MootdxAdapter) *FinancialsService {
	return &FinancialsService{sinaAdapter: sinaAdapter, mootdxAdapter: mootdxAdapter}
}

// GetFinancials returns financial data for a symbol.
// Tries mootdx first (fast TCP, 37-field snapshot), falls back to Sina, then mock.
func (s *FinancialsService) GetFinancials(ctx context.Context, symbol string) (*FinancialData, error) {
	// 1. Try mootdx finance snapshot (fast, 37 fields including EPS/ROE/Profit)
	if s.mootdxAdapter != nil && s.mootdxAdapter.IsAvailable(ctx) {
		fd, err := s.fetchFromMootdx(ctx, symbol)
		if err == nil {
			slog.Debug("financials: fetched from mootdx", "symbol", symbol)
			return fd, nil
		}
		slog.Warn("financials: mootdx fetch failed, trying sina", "symbol", symbol, "error", err)
	}

	// 2. Fallback to Sina financials
	if s.sinaAdapter != nil {
		fd, err := s.fetchFromSina(ctx, symbol)
		if err == nil {
			slog.Debug("financials: fetched from sina", "symbol", symbol)
			return fd, nil
		}
		slog.Warn("financials: sina fetch failed, using mock", "symbol", symbol, "error", err)
		return s.mockFinancials(symbol), nil
	}

	slog.Debug("financials: no adapter, using mock", "symbol", symbol)
	return s.mockFinancials(symbol), nil
}

// fetchFromMootdx converts a mootdx finance snapshot into FinancialData.
func (s *FinancialsService) fetchFromMootdx(ctx context.Context, symbol string) (*FinancialData, error) {
	fin, err := s.mootdxAdapter.FetchFinance(ctx, symbol)
	if err != nil {
		return nil, err
	}

	fd := &FinancialData{
		Symbol:    symbol,
		Revenue:   fin.Income,
		NetIncome: fin.Profit,
		EPS:       fin.EPS,
	}

	// Fill additional fields from raw data
	if v, ok := fin.Raw["总资产"]; ok {
		fd.TotalAssets = parseFloatAny(v)
	}
	if v, ok := fin.Raw["净资产"]; ok {
		fd.TotalEquity = parseFloatAny(v)
	}

	return fd, nil
}

func parseFloatAny(v string) float64 {
	if f, err := strconv.ParseFloat(strings.TrimSpace(v), 64); err == nil {
		return f
	}
	return 0
}

// fetchFromSina fetches real financial data from Sina Finance and parses it.
func (s *FinancialsService) fetchFromSina(ctx context.Context, symbol string) (*FinancialData, error) {
	// Fetch latest income statement (利润表) — 1 period is enough for current data
	income, err := s.sinaAdapter.FetchIncomeStatement(ctx, symbol, 1)
	if err != nil {
		return nil, err
	}

	// Fetch latest balance sheet (资产负债表)
	balance, err := s.sinaAdapter.FetchBalanceSheet(ctx, symbol, 1)
	if err != nil {
		return nil, err
	}

	// Fetch latest cash flow statement (现金流量表)
	cashflow, err := s.sinaAdapter.FetchCashFlow(ctx, symbol, 1)
	if err != nil {
		return nil, err
	}

	fd := &FinancialData{Symbol: symbol}

	// Parse income statement items
	if len(income) > 0 {
		for _, item := range income[0].Items {
			v := parseSinaValue(item.Value)
			switch {
			case matchTitle(item.Title, "营业总收入", "营业总收入(元)", "营业收入"):
				fd.Revenue = v
			case matchTitle(item.Title, "净利润", "净利润(元)", "归属于母公司所有者的净利润"):
				fd.NetIncome = v
			case matchTitle(item.Title, "基本每股收益", "基本每股收益(元/股)"):
				fd.EPS = v
			}
		}
	}

	// Parse balance sheet items
	if len(balance) > 0 {
		for _, item := range balance[0].Items {
			v := parseSinaValue(item.Value)
			switch {
			case matchTitle(item.Title, "总资产", "资产总计"):
				fd.TotalAssets = v
			case matchTitle(item.Title, "所有者权益", "所有者权益(或股东权益)合计", "归属于母公司所有者权益合计"):
				fd.TotalEquity = v
			case matchTitle(item.Title, "负债合计", "负债和所有者权益") :
				// "负债合计" gives total liabilities; we need TotalDebt which is usually interest-bearing
				// Use "负债合计" as approximation for total debt
				if fd.TotalDebt == 0 {
					fd.TotalDebt = v
				}
			}
		}
	}

	// Parse cash flow items
	if len(cashflow) > 0 {
		for _, item := range cashflow[0].Items {
			v := parseSinaValue(item.Value)
			if matchTitle(item.Title, "经营活动产生的现金流量净额") {
				fd.FreeCashFlow = v
			}
		}
	}

	// Market cap is not in Sina financial statements — leave as 0 (can be filled by quote data)
	return fd, nil
}

// ComputeRatios calculates key financial ratios from financial data.
func (s *FinancialsService) ComputeRatios(data *FinancialData) *FinancialRatios {
	if data == nil {
		return &FinancialRatios{}
	}
	r := &FinancialRatios{}
	if data.EPS > 0 && data.MarketCap > 0 {
		// PE = MarketCap / NetIncome (more standard than EPS-based)
		if data.NetIncome > 0 {
			r.PE = data.MarketCap / data.NetIncome
		}
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

// mockFinancials returns mock financial data for development/testing.
func (s *FinancialsService) mockFinancials(symbol string) *FinancialData {
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
	}
}

// matchTitle checks if an item title contains any of the given keywords.
func matchTitle(title string, keywords ...string) bool {
	for _, kw := range keywords {
		if strings.Contains(title, kw) {
			return true
		}
	}
	return false
}

// parseSinaValue converts Sina's raw value (numeric string or float) to float64.
// Sina returns values as strings like "1.23e10" or numbers. Handles both.
func parseSinaValue(v interface{}) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case string:
		val = strings.TrimSpace(val)
		if val == "" || val == "--" || val == "null" {
			return 0
		}
		f, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return 0
		}
		return f
	case json.Number:
		f, _ := val.Float64()
		return f
	default:
		return 0
	}
}

