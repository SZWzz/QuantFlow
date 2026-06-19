package backtest

import (
	"math"
	"sort"
)

// Metrics contains standard performance metrics for a backtest.
type Metrics struct {
	TotalReturn      float64 `json:"total_return"`
	CAGR             float64 `json:"cagr"`
	MaxDrawdown      float64 `json:"max_drawdown"`
	SharpeRatio      float64 `json:"sharpe_ratio"`
	SortinoRatio     float64 `json:"sortino_ratio"`
	CalmarRatio      float64 `json:"calmar_ratio"`
	WinRate          float64 `json:"win_rate"`
	ProfitFactor     float64 `json:"profit_factor"`
	TotalTrades      int     `json:"total_trades"`
	AnnualVolatility float64 `json:"annual_volatility"`
}

// ComputeMetrics calculates all performance metrics from equity curve and trades.
func ComputeMetrics(equityCurve []EquityPoint, trades []TradeRecord) Metrics {
	if len(equityCurve) < 2 {
		return Metrics{}
	}

	initialEquity := equityCurve[0].Equity
	finalEquity := equityCurve[len(equityCurve)-1].Equity

	// Total return
	totalReturn := (finalEquity - initialEquity) / initialEquity

	// Daily returns for Sharpe/Sortino/Volatility
	dailyReturns := make([]float64, 0, len(equityCurve)-1)
	for i := 1; i < len(equityCurve); i++ {
		if equityCurve[i-1].Equity > 0 {
			r := (equityCurve[i].Equity - equityCurve[i-1].Equity) / equityCurve[i-1].Equity
			dailyReturns = append(dailyReturns, r)
		}
	}

	nDays := len(dailyReturns)
	if nDays == 0 {
		return Metrics{TotalReturn: totalReturn}
	}

	// CAGR: (final/initial)^(252/ndays) - 1
	nYears := float64(nDays) / 252.0
	cagr := math.Pow(finalEquity/initialEquity, 1.0/nYears) - 1.0

	// Annual volatility
	meanReturn := mean(dailyReturns)
	variance := 0.0
	for _, r := range dailyReturns {
		variance += (r - meanReturn) * (r - meanReturn)
	}
	stdDaily := math.Sqrt(variance / float64(nDays))
	annualVol := stdDaily * math.Sqrt(252)

	// Sharpe ratio (assuming 0 risk-free rate for simplicity)
	sharpe := 0.0
	if annualVol > 0 {
		sharpe = cagr / annualVol
	}

	// Sortino ratio (downside deviation)
	downsideVariance := 0.0
	downCount := 0
	for _, r := range dailyReturns {
		if r < 0 {
			downsideVariance += r * r
			downCount++
		}
	}
	sortino := 0.0
	if downCount > 0 {
		downStd := math.Sqrt(downsideVariance / float64(downCount)) * math.Sqrt(252)
		if downStd > 0 {
			sortino = cagr / downStd
		}
	}

	// Max drawdown
	maxDD := computeMaxDrawdown(equityCurve)

	// Calmar ratio
	calmar := 0.0
	if maxDD < 0 {
		calmar = cagr / (-maxDD)
	}

	// Trade statistics
	winRate, profitFactor := computeTradeStats(trades)

	return Metrics{
		TotalReturn:      totalReturn,
		CAGR:             cagr,
		MaxDrawdown:      maxDD,
		SharpeRatio:      sharpe,
		SortinoRatio:     sortino,
		CalmarRatio:      calmar,
		WinRate:          winRate,
		ProfitFactor:     profitFactor,
		TotalTrades:      len(trades),
		AnnualVolatility: annualVol,
	}
}

func mean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func computeMaxDrawdown(equityCurve []EquityPoint) float64 {
	if len(equityCurve) == 0 {
		return 0
	}
	peak := equityCurve[0].Equity
	maxDD := 0.0
	for _, p := range equityCurve {
		if p.Equity > peak {
			peak = p.Equity
		}
		dd := (p.Equity - peak) / peak
		if dd < maxDD {
			maxDD = dd
		}
	}
	return maxDD
}

func computeTradeStats(trades []TradeRecord) (winRate, profitFactor float64) {
	if len(trades) == 0 {
		return 0, 0
	}

	// Group trades by symbol to compute P&L per round trip
	// Simple approach: just count profitable vs losing trades
	var wins, losses int
	var grossProfit, grossLoss float64

	// Sort by date
	sorted := make([]TradeRecord, len(trades))
	copy(sorted, trades)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Date < sorted[j].Date })

	for _, t := range sorted {
		if t.PnL > 0 {
			wins++
			grossProfit += t.PnL
		} else if t.PnL < 0 {
			losses++
			grossLoss += -t.PnL
		}
	}

	total := wins + losses
	if total > 0 {
		winRate = float64(wins) / float64(total)
	}
	if grossLoss > 0 {
		profitFactor = grossProfit / grossLoss
	} else if grossProfit > 0 {
		profitFactor = math.Inf(1)
	}

	return winRate, profitFactor
}
