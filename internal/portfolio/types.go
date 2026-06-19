package portfolio

// Summary is a snapshot of the portfolio's total value, cash, market value, and P&L.
type Summary struct {
	TotalValue  float64 `json:"total_value"`
	CashBalance float64 `json:"cash_balance"`
	MarketValue float64 `json:"market_value"`
	TotalPnL    float64 `json:"total_pnl"`
	TotalPnLPct float64 `json:"total_pnl_pct"`
	DailyPnL    float64 `json:"daily_pnl"`
	DailyPnLPct float64 `json:"daily_pnl_pct"`
}

// PositionDetail represents a single position with computed analytics.
type PositionDetail struct {
	Symbol      string  `json:"symbol"`
	Quantity    float64 `json:"quantity"`
	AvgPrice    float64 `json:"avg_price"`
	MarketPrice float64 `json:"market_price"`
	PnL         float64 `json:"pnl"`
	PnLPct      float64 `json:"pnl_pct"`
	Market      string  `json:"market"`
	Currency    string  `json:"currency"`
	CostBasis   float64 `json:"cost_basis"`
	AllocPct    float64 `json:"alloc_pct"`
}

// Allocation breaks down the portfolio by market, sector, and currency.
type Allocation struct {
	ByMarket   map[string]float64 `json:"by_market"`
	BySector   map[string]float64 `json:"by_sector"`
	ByCurrency map[string]float64 `json:"by_currency"`
}

// DailyPnL is a single day's portfolio snapshot persisted in SQLite.
type DailyPnL struct {
	Date        string  `json:"date"`
	TotalValue  float64 `json:"total_value"`
	Cash        float64 `json:"cash"`
	MarketValue float64 `json:"market_value"`
	PnL         float64 `json:"pnl"`
	PnLPct      float64 `json:"pnl_pct"`
}
