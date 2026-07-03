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
	// Sample variance: use N-1 denominator for unbiased estimate
	denom := float64(nDays)
	if nDays > 1 {
		denom = float64(nDays - 1)
	}
	stdDaily := math.Sqrt(variance / denom)
	annualVol := stdDaily * math.Sqrt(252)

	// Sharpe ratio (arithmetic annualization - risk-free rate 2%)
	const riskFreeRate = 0.02
	sharpe := 0.0
	if annualVol > 0 {
		sharpe = (meanReturn*252 - riskFreeRate) / annualVol
	}

	// Sortino ratio (downside deviation, arithmetic annualized return)
	annualReturn := meanReturn * 252
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
			sortino = (annualReturn - riskFreeRate) / downStd
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
	winRate, profitFactor, closedTrades := computeTradeStats(trades)

	m := Metrics{
		TotalReturn:      totalReturn,
		CAGR:             cagr,
		MaxDrawdown:      maxDD,
		SharpeRatio:      sharpe,
		SortinoRatio:     sortino,
		CalmarRatio:      calmar,
		WinRate:          winRate,
		ProfitFactor:     profitFactor,
		TotalTrades:      closedTrades,
		AnnualVolatility: annualVol,
	}
	sanitizeMetrics(&m)
	return m
}

// sanitizeMetrics replaces NaN/Inf values with 0 so JSON serialization does not fail.
func sanitizeMetrics(m *Metrics) {
	if math.IsNaN(m.TotalReturn) || math.IsInf(m.TotalReturn, 0) { m.TotalReturn = 0 }
	if math.IsNaN(m.CAGR) || math.IsInf(m.CAGR, 0) { m.CAGR = 0 }
	if math.IsNaN(m.MaxDrawdown) || math.IsInf(m.MaxDrawdown, 0) { m.MaxDrawdown = 0 }
	if math.IsNaN(m.SharpeRatio) || math.IsInf(m.SharpeRatio, 0) { m.SharpeRatio = 0 }
	if math.IsNaN(m.SortinoRatio) || math.IsInf(m.SortinoRatio, 0) { m.SortinoRatio = 0 }
	if math.IsNaN(m.CalmarRatio) || math.IsInf(m.CalmarRatio, 0) { m.CalmarRatio = 0 }
	if math.IsNaN(m.WinRate) || math.IsInf(m.WinRate, 0) { m.WinRate = 0 }
	if math.IsNaN(m.ProfitFactor) || math.IsInf(m.ProfitFactor, 0) { m.ProfitFactor = 0 }
	if math.IsNaN(m.AnnualVolatility) || math.IsInf(m.AnnualVolatility, 0) { m.AnnualVolatility = 0 }
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

func computeTradeStats(trades []TradeRecord) (winRate, profitFactor float64, closedTrades int) {
	if len(trades) == 0 {
		return 0, 0, 0
	}

	var wins, losses int
	var grossProfit, grossLoss float64

	sorted := make([]TradeRecord, len(trades))
	copy(sorted, trades)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Date < sorted[j].Date })

	// Only sell trades carry PnL; buy trades have PnL=0 (omitted).
	// Wins+losses = number of closed positions = consistent denominator.
	for _, t := range sorted {
		if t.PnL > 0 {
			wins++
			grossProfit += t.PnL
		} else if t.PnL < 0 {
			losses++
			grossLoss += -t.PnL
		}
	}

	closedTrades = wins + losses
	if closedTrades > 0 {
		winRate = float64(wins) / float64(closedTrades)
	}
	if grossLoss > 0 {
		profitFactor = grossProfit / grossLoss
	} else if grossProfit > 0 {
		profitFactor = 999999.0
	}

	return winRate, profitFactor, closedTrades
}
