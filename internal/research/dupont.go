package research

import "math"

// DupontBreakdown decomposes ROE using the 3-factor Dupont formula.
type DupontBreakdown struct {
	Symbol           string  `json:"symbol"`
	ROE              float64 `json:"roe"`
	NetMargin        float64 `json:"net_margin"`
	AssetTurnover    float64 `json:"asset_turnover"`
	EquityMultiplier float64 `json:"equity_multiplier"`
	GrossMargin      float64 `json:"gross_margin"`
	EPS              float64 `json:"eps"`
}

// DupontTrend holds Dupont decomposition for a single period.
type DupontTrend struct {
	Period    string          `json:"period"`
	Breakdown DupontBreakdown `json:"breakdown"`
}

// PeerRadar contains radar chart data for peer comparison.
type PeerRadar struct {
	Symbol  string             `json:"symbol"`
	Name    string             `json:"name"`
	Metrics map[string]float64 `json:"metrics"`
}

// ComputeDupont calculates the 3-factor Dupont breakdown from financial data.
// ROE = (NetIncome/Revenue) × (Revenue/TotalAssets) × (TotalAssets/TotalEquity)
func ComputeDupont(fd *FinancialData) *DupontBreakdown {
	if fd == nil {
		return &DupontBreakdown{}
	}

	db := &DupontBreakdown{Symbol: fd.Symbol}

	// Net Margin = NetIncome / Revenue
	if fd.Revenue > 0 {
		db.NetMargin = math.Round(fd.NetIncome/fd.Revenue*10000) / 100
	}

	// Asset Turnover = Revenue / TotalAssets
	if fd.TotalAssets > 0 {
		db.AssetTurnover = math.Round(fd.Revenue/fd.TotalAssets*10000) / 100
	}

	// Equity Multiplier = TotalAssets / TotalEquity
	if fd.TotalEquity > 0 {
		db.EquityMultiplier = math.Round(fd.TotalAssets/fd.TotalEquity*100) / 100
	}

	// ROE = Net Income / Total Equity
	if fd.TotalEquity > 0 {
		db.ROE = math.Round(fd.NetIncome/fd.TotalEquity*10000) / 100
	}

	// Gross Margin = (Revenue - COGS) / Revenue (approximation: use net margin as proxy since COGS not available)
	db.GrossMargin = db.NetMargin

	db.EPS = fd.EPS

	return db
}

// ComputePeerRadar builds radar chart data for a symbol and its peers.
func ComputePeerRadar(symbol string, peers []string, getFD func(string) *FinancialData) []PeerRadar {
	var result []PeerRadar

	all := append([]string{symbol}, peers...)
	for _, sym := range all {
		fd := getFD(sym)
		if fd == nil {
			continue
		}
		db := ComputeDupont(fd)
		result = append(result, PeerRadar{
			Symbol: sym,
			Name:   sym,
			Metrics: map[string]float64{
				"ROE": db.ROE,
				"净利率": db.NetMargin,
				"周转率": db.AssetTurnover,
				"杠杆":  db.EquityMultiplier,
				"毛利率": db.GrossMargin,
				"EPS": db.EPS,
			},
		})
	}

	return result
}
