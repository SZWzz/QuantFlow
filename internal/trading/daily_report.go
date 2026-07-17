package trading

import (
	"encoding/json"
	"math"
	"sort"
)

// DailyReport summarizes a day's trading activity.
type DailyReport struct {
	Date            string            `json:"date"`
	MarketValue     float64           `json:"market_value"`
	DayPNL          float64           `json:"day_pnl"`
	DayPNLPercent   float64           `json:"day_pnl_percent"`
	TotalPNL        float64           `json:"total_pnl"`
	TotalPNLPercent float64           `json:"total_pnl_percent"`
	Trades          int               `json:"trades"`
	Commission      float64           `json:"commission"`
	Tax             float64           `json:"tax"`
	MaxDrawdown     float64           `json:"max_drawdown"`
	BestTrade       *TradeSummary     `json:"best_trade"`
	WorstTrade      *TradeSummary     `json:"worst_trade"`
	Positions       []PositionSummary `json:"positions"`
	Notes           string            `json:"notes"`
}

// TradeSummary summarizes a single trade for report display.
type TradeSummary struct {
	Symbol   string  `json:"symbol"`
	Side     string  `json:"side"`
	Quantity float64 `json:"quantity"`
	Price    float64 `json:"price"`
	PnL      float64 `json:"pnl"`
}

// PositionSummary is a condensed position for report display.
type PositionSummary struct {
	Symbol    string  `json:"symbol"`
	Quantity  float64 `json:"quantity"`
	MarketVal float64 `json:"market_val"`
	PnL       float64 `json:"pnl"`
	PnLPct    float64 `json:"pnl_pct"`
}

// GenerateDailyReport creates a DailyReport from OMS data for a given date.
func GenerateDailyReport(oms *OMS, date string) *DailyReport {
	trades := oms.GetTrades()
	positions := oms.GetAllPositions()

	report := &DailyReport{
		Date:      date,
		Trades:    len(trades),
		Positions: make([]PositionSummary, 0, len(positions)),
	}

	var totalCommission, totalTax float64
	var bestPnL, worstPnL float64 = math.Inf(-1), math.Inf(1)
	var bestTrade, worstTrade *TradeSummary

	for _, t := range trades {
		totalCommission += t.Commission
		totalTax += t.StampTax
		// PnL for each trade (simplified: position-level P&L is captured via positions)
		pnl := 0.0
		if t.Side == SideSell {
			pnl = (t.Price - 0) * t.Quantity // approximate for report
		}
		ts := &TradeSummary{
			Symbol:   t.Symbol,
			Side:     string(t.Side),
			Quantity: t.Quantity,
			Price:    t.Price,
			PnL:      pnl,
		}
		if pnl > bestPnL {
			bestPnL = pnl
			bestTrade = ts
		}
		if pnl < worstPnL {
			worstPnL = pnl
			worstTrade = ts
		}
	}

	report.Commission = totalCommission
	report.Tax = totalTax
	if bestTrade != nil {
		report.BestTrade = bestTrade
	}
	if worstTrade != nil {
		report.WorstTrade = worstTrade
	}

	// Position summaries and day P&L
	var totalMarketVal, totalDayPnL, totalRealizedPnL float64
	for _, p := range positions {
		mktVal := p.Quantity * p.MarketPrice
		posPnLPct := 0.0
		if p.AvgPrice > 0 {
			posPnLPct = (p.MarketPrice - p.AvgPrice) / p.AvgPrice * 100
		}
		report.Positions = append(report.Positions, PositionSummary{
			Symbol:    p.Symbol,
			Quantity:  p.Quantity,
			MarketVal: mktVal,
			PnL:       p.PnL,
			PnLPct:    posPnLPct,
		})
		totalMarketVal += mktVal
		totalDayPnL += p.PnL
		totalRealizedPnL += p.RealizedPnl
	}

	// Sort positions by market value descending
	sort.Slice(report.Positions, func(i, j int) bool {
		return report.Positions[i].MarketVal > report.Positions[j].MarketVal
	})

	report.MarketValue = totalMarketVal
	report.DayPNL = totalDayPnL
	report.TotalPNL = totalRealizedPnL

	if totalMarketVal > 0 {
		report.DayPNLPercent = totalDayPnL / totalMarketVal * 100
	}
	if totalMarketVal > 0 && totalRealizedPnL != 0 {
		report.TotalPNLPercent = totalRealizedPnL / totalMarketVal * 100
	}

	return report
}

// EncodeDailyReport marshals a DailyReport to JSON bytes.
func EncodeDailyReport(report *DailyReport) ([]byte, error) {
	return json.Marshal(report)
}

// DecodeDailyReport unmarshals JSON bytes into a DailyReport.
func DecodeDailyReport(data []byte) (*DailyReport, error) {
	var report DailyReport
	if err := json.Unmarshal(data, &report); err != nil {
		return nil, err
	}
	return &report, nil
}
