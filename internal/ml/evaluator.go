package ml

import "math"

// ComputeIC calculates the Pearson (rank) Information Coefficient between predictions and actuals.
func ComputeIC(predictions, actuals []float64) float64 {
	n := float64(len(predictions))
	if n < 3 {
		return 0
	}
	var sumP, sumA, sumPP, sumAA, sumPA float64
	for i := range predictions {
		sumP += predictions[i]
		sumA += actuals[i]
		sumPP += predictions[i] * predictions[i]
		sumAA += actuals[i] * actuals[i]
		sumPA += predictions[i] * actuals[i]
	}
	denom := math.Sqrt((n*sumPP - sumP*sumP) * (n*sumAA - sumA*sumA))
	if denom == 0 {
		return 0
	}
	ic := (n*sumPA - sumP*sumA) / denom
	if math.IsNaN(ic) {
		return 0
	}
	return ic
}

// ComputeIR calculates Information Ratio from an IC series.
func ComputeIR(icSeries []float64) float64 {
	if len(icSeries) < 2 {
		return 0
	}
	mean := 0.0
	for _, ic := range icSeries {
		mean += ic
	}
	mean /= float64(len(icSeries))
	std := 0.0
	for _, ic := range icSeries {
		std += (ic - mean) * (ic - mean)
	}
	std = math.Sqrt(std / float64(len(icSeries)))
	if std == 0 {
		return 0
	}
	return mean / std
}

// ComputeSharpe calculates annualized Sharpe ratio from a series of returns.
func ComputeSharpe(returns []float64) float64 {
	if len(returns) < 2 {
		return 0
	}
	mean := 0.0
	for _, r := range returns {
		mean += r
	}
	mean /= float64(len(returns))
	std := 0.0
	for _, r := range returns {
		std += (r - mean) * (r - mean)
	}
	std = math.Sqrt(std / float64(len(returns)))
	if std == 0 {
		return 0
	}
	// Annualize: assume daily returns, sqrt(252)
	return (mean / std) * math.Sqrt(252)
}
