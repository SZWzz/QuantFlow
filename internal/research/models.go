// Package research provides sentiment analysis, financial research, and stock analysis services.
// All services degrade gracefully when Python sidecar is unavailable, returning mock data.
package research

import "time"

// SentimentOutput is the Go-domain representation of a sentiment analysis result.
type SentimentOutput struct {
	Symbol      string   `json:"symbol"`
	Score       float64  `json:"score"`
	Label       string   `json:"label"`
	Confidence  float64  `json:"confidence"`
	Keywords    []string `json:"keywords"`
	Entities    []string `json:"entities"`
	Source      string   `json:"source"`
	ComputeTime float64  `json:"compute_time_ms"`
}

// FinancialData holds key financial metrics for a stock.
type FinancialData struct {
	Symbol       string  `json:"symbol"`
	Revenue      float64 `json:"revenue"`
	NetIncome    float64 `json:"net_income"`
	EPS          float64 `json:"eps"`
	TotalAssets  float64 `json:"total_assets"`
	TotalEquity  float64 `json:"total_equity"`
	TotalDebt    float64 `json:"total_debt"`
	FreeCashFlow float64 `json:"free_cash_flow"`
	MarketCap    float64 `json:"market_cap"`
}

// FinancialRatios holds computed financial ratios.
type FinancialRatios struct {
	PE           float64 `json:"pe_ratio"`
	PB           float64 `json:"pb_ratio"`
	ROE          float64 `json:"roe"`
	ROA          float64 `json:"roa"`
	DebtToEquity float64 `json:"debt_to_equity"`
	NetMargin    float64 `json:"net_margin"`
}

// PeerComparisonData holds peer comparison metrics for one peer company.
type PeerComparisonData struct {
	Symbol        string  `json:"symbol"`
	Name          string  `json:"name"`
	MarketCap     float64 `json:"market_cap"`
	PE            float64 `json:"pe_ratio"`
	RevenueGrowth float64 `json:"revenue_growth"`
	NetMargin     float64 `json:"margin"`
	ROE           float64 `json:"roe"`
}

// AnalystEstimate holds a single analyst rating for a stock.
type AnalystEstimate struct {
	Analyst    string  `json:"analyst"`
	Firm       string  `json:"firm"`
	Rating     string  `json:"rating"`
	TargetLow  float64 `json:"target_low"`
	TargetHigh float64 `json:"target_high"`
	Date       string  `json:"date"`
}

// InsiderTransaction represents a single insider trade.
type InsiderTransaction struct {
	Name   string  `json:"name"`
	Role   string  `json:"role"`
	Type   string  `json:"type"`
	Shares int64   `json:"shares"`
	Price  float64 `json:"price"`
	Value  float64 `json:"value"`
	Date   string  `json:"date"`
}

// FinancialsBundle groups financial data and computed ratios under a single JSON key.
// The frontend expects { data: FinancialData, ratios: FinancialRatios } nested under "financials".
type FinancialsBundle struct {
	Data   *FinancialData   `json:"data,omitempty"`
	Ratios *FinancialRatios `json:"ratios,omitempty"`
}

// StockResearchResult aggregates all research dimensions for a symbol.
type StockResearchResult struct {
	Symbol      string                 `json:"symbol"`
	UpdatedAt   time.Time              `json:"updated_at"`
	Overview    map[string]interface{} `json:"overview,omitempty"`
	Financials  *FinancialsBundle      `json:"financials,omitempty"`
	Sentiment   *SentimentOutput       `json:"sentiment,omitempty"`
	Peers       []PeerComparisonData   `json:"peers,omitempty"`
	Estimates   []AnalystEstimate      `json:"estimates,omitempty"`
	InsiderTxns []InsiderTransaction   `json:"insider,omitempty"`
}

// CongressTrade represents a congress member's stock trade.
type CongressTrade struct {
	Name    string `json:"name"`
	Chamber string `json:"chamber"`
	Party   string `json:"party"`
	Symbol  string `json:"symbol"`
	Type    string `json:"type"`
	Amount  string `json:"amount"`
	Date    string `json:"date"`
}
