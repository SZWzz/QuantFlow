// analytics.go — portfolio-level analytical computations.
// Correlation matrix, return distribution histogram, volatility surface.
package portfolio

import "math"

// CorrelationMatrix computes the Pearson correlation matrix for a set of return series.
// Input: map[symbol] → daily log returns. Output: map[symbol] → map[symbol] → correlation.
func CorrelationMatrix(returns map[string][]float64) map[string]map[string]float64 {
	symbols := make([]string, 0, len(returns))
	for s := range returns {
		symbols = append(symbols, s)
	}
	result := make(map[string]map[string]float64, len(symbols))
	for i, si := range symbols {
		result[si] = make(map[string]float64, len(symbols))
		for j, sj := range symbols {
			if i == j {
				result[si][sj] = 1.0
				continue
			}
			ri, rj := returns[si], returns[sj]
			n := min(len(ri), len(rj))
			if n < 2 {
				result[si][sj] = 0
				continue
			}
			meanI, meanJ := 0.0, 0.0
			for k := 0; k < n; k++ {
				meanI += ri[k]
				meanJ += rj[k]
			}
			meanI /= float64(n)
			meanJ /= float64(n)
			cov, varI, varJ := 0.0, 0.0, 0.0
			for k := 0; k < n; k++ {
				di := ri[k] - meanI
				dj := rj[k] - meanJ
				cov += di * dj
				varI += di * di
				varJ += dj * dj
			}
			if varI == 0 || varJ == 0 {
				result[si][sj] = 0
				continue
			}
			result[si][sj] = cov / math.Sqrt(varI*varJ)
		}
	}
	return result
}

// ReturnDistribution builds a histogram of returns into numBins equal-width bins.
// Returns bins (midpoints) and counts.
func ReturnDistribution(returns []float64, numBins int) (bins []float64, counts []float64) {
	if len(returns) == 0 || numBins <= 0 {
		return nil, nil
	}
	minR, maxR := returns[0], returns[0]
	for _, r := range returns {
		if r < minR {
			minR = r
		}
		if r > maxR {
			maxR = r
		}
	}
	if minR == maxR {
		bins = []float64{minR}
		counts = []float64{float64(len(returns))}
		return
	}
	width := (maxR - minR) / float64(numBins)
	if width <= 0 {
		return nil, nil
	}
	bins = make([]float64, numBins)
	counts = make([]float64, numBins)
	for i := 0; i < numBins; i++ {
		bins[i] = minR + float64(i)*width + width/2
	}
	for _, r := range returns {
		idx := int((r - minR) / width)
		if idx >= numBins {
			idx = numBins - 1
		}
		if idx < 0 {
			idx = 0
		}
		counts[idx]++
	}
	return bins, counts
}

// VolatilitySurface computes historical volatility at different window sizes.
// Returns [][]float64 where each row is [windowDays, annualizedVolatility].
// If there is insufficient data for a window, that window is skipped.
func VolatilitySurface(returns []float64, windows []int) [][]float64 {
	result := make([][]float64, 0, len(windows))
	for _, w := range windows {
		if w > len(returns) {
			continue
		}
		vols := make([]float64, 0)
		for i := w; i <= len(returns); i++ {
			slice := returns[i-w : i]
			mean := 0.0
			for _, r := range slice {
				mean += r
			}
			mean /= float64(w)
			variance := 0.0
			for _, r := range slice {
				d := r - mean
				variance += d * d
			}
			vols = append(vols, math.Sqrt(variance/float64(w))*math.Sqrt(252))
		}
		avgVol := 0.0
		for _, v := range vols {
			avgVol += v
		}
		if len(vols) > 0 {
			avgVol /= float64(len(vols))
		}
		result = append(result, []float64{float64(w), avgVol})
	}
	return result
}
