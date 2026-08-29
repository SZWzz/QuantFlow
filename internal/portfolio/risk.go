package portfolio

import (
	"math"
	"sort"
)

// RiskMetrics bundles standard risk statistics computed from daily P&L history.
type RiskMetrics struct {
	Var95         float64 `json:"var_95"`
	CVaR95        float64 `json:"cvar_95"`
	MaxDrawdown   float64 `json:"max_drawdown"`
	MaxDDStart    string  `json:"max_dd_start"`
	MaxDDEnd      string  `json:"max_dd_end"`
	SharpeRatio   float64 `json:"sharpe_ratio"`
	SortinoRatio  float64 `json:"sortino_ratio"`
	CalmarRatio   float64 `json:"calmar_ratio"`
	TotalExposure float64 `json:"total_exposure"`
	Leverage      float64 `json:"leverage"`
	DailyVol      float64 `json:"daily_volatility"`
	AnnualVol     float64 `json:"annual_volatility"`
}

// ComputeMetrics calculates risk metrics from a series of daily P&L snapshots.
// dailyPnL is expected in reverse chronological order (most recent first).
func ComputeMetrics(dailyPnL []*DailyPnL, totalValue float64, riskFreeRate float64) *RiskMetrics {
	if len(dailyPnL) < 2 {
		return &RiskMetrics{TotalExposure: totalValue}
	}

	returns := make([]float64, len(dailyPnL)-1)
	for i := 1; i < len(dailyPnL); i++ {
		prev := dailyPnL[i].TotalValue
		curr := dailyPnL[i-1].TotalValue
		if prev > 0 {
			returns[i-1] = (curr - prev) / prev
		}
	}

	sorted := make([]float64, len(returns))
	copy(sorted, returns)
	sort.Float64s(sorted)

	var95Idx := int(float64(len(sorted)) * 0.05)
	if var95Idx >= len(sorted) {
		var95Idx = len(sorted) - 1
	}
	worst := sorted[:var95Idx+1]

	var95, cvar95 := 0.0, 0.0
	if len(worst) > 0 {
		var95 = worst[len(worst)-1]
		for _, r := range worst {
			cvar95 += r
		}
		cvar95 /= float64(len(worst))
	}

	mean := 0.0
	for _, r := range returns {
		mean += r
	}
	mean /= float64(len(returns))

	variance := 0.0
	for _, r := range returns {
		variance += (r - mean) * (r - mean)
	}
	// Sample variance: use N-1 denominator for unbiased estimate
	nSamples := float64(len(returns))
	if len(returns) > 1 {
		nSamples = float64(len(returns) - 1)
	}
	dailyVol := math.Sqrt(variance / nSamples)
	annualVol := dailyVol * math.Sqrt(252)

	sharpe := 0.0
	if annualVol > 0 {
		sharpe = ((mean * 252) - riskFreeRate) / annualVol
	}

	downVar, downN := 0.0, 0
	for _, r := range returns {
		if r < 0 {
			downVar += r * r
			downN++
		}
	}
	sortino := 0.0
	if downN > 0 {
		downDev := math.Sqrt(downVar/float64(downN)) * math.Sqrt(252)
		if downDev > 0 {
			sortino = ((mean * 252) - riskFreeRate) / downDev
		}
	}

	maxDD, ddStart, ddEnd := computeMaxDrawdown(dailyPnL)

	// Compute CAGR for Calmar ratio
	nYears := float64(len(returns)) / 252.0
	cagr := 0.0
	if nYears > 0 && len(dailyPnL) > 1 && dailyPnL[0].TotalValue > 0 && dailyPnL[len(dailyPnL)-1].TotalValue > 0 {
		cagr = math.Pow(dailyPnL[0].TotalValue/dailyPnL[len(dailyPnL)-1].TotalValue, 1.0/nYears) - 1.0
	}

	calmar := 0.0
	if maxDD > 0 {
		calmar = cagr / maxDD
	}

	return &RiskMetrics{
		Var95: math.Abs(var95) * totalValue, CVaR95: cvar95 * totalValue,
		MaxDrawdown: maxDD, MaxDDStart: ddStart, MaxDDEnd: ddEnd,
		SharpeRatio: sharpe, SortinoRatio: sortino, CalmarRatio: calmar,
		TotalExposure: totalValue, Leverage: 1.0,
		DailyVol: dailyVol, AnnualVol: annualVol,
	}
}

// computeMaxDrawdown finds the maximum peak-to-trough decline and its dates.
// dailyPnL is expected in reverse chronological order (most recent first).
func computeMaxDrawdown(dailyPnL []*DailyPnL) (float64, string, string) {
	if len(dailyPnL) < 2 {
		return 0, "", ""
	}
	peak := dailyPnL[len(dailyPnL)-1].TotalValue
	peakDate := dailyPnL[len(dailyPnL)-1].Date
	maxDD := 0.0
	ddStart, ddEnd := "", ""

	for i := len(dailyPnL) - 1; i >= 0; i-- {
		value := dailyPnL[i].TotalValue
		if value > peak {
			peak = value
			peakDate = dailyPnL[i].Date
		}
		dd := (peak - value) / peak
		if dd > maxDD {
			maxDD = dd
			ddStart = peakDate
			ddEnd = dailyPnL[i].Date
		}
	}
	return maxDD, ddStart, ddEnd
}
