package trading

import (
	"math"
	"sort"
)

// BandPoint is a single point on a PE/PB Band chart.
type BandPoint struct {
	Date  string  `json:"date"`
	Close float64 `json:"close"`
	Band1 float64 `json:"band_1"` // μ - 2σ
	Band2 float64 `json:"band_2"` // μ - 1σ
	Band3 float64 `json:"band_3"` // μ
	Band4 float64 `json:"band_4"` // μ + 1σ
	Band5 float64 `json:"band_5"` // μ + 2σ
}

// BandResult holds PE or PB Band computation results.
type BandResult struct {
	Symbol     string      `json:"symbol"`
	Metric     string      `json:"metric"`
	Current    float64     `json:"current"`
	Mean       float64     `json:"mean"`
	StdDev     float64     `json:"stddev"`
	Percentile float64     `json:"percentile"`
	Points     []BandPoint `json:"points"`
}

// ComputePriceBand calculates price band from OHLCV close data.
// Uses close prices to derive μ and σ, then produces 5 band channels.
func ComputePriceBand(symbol string, bars []OHLCVBar) *BandResult {
	if len(bars) == 0 {
		return &BandResult{Symbol: symbol, Metric: "price"}
	}

	// Collect close prices
	prices := make([]float64, len(bars))
	for i, b := range bars {
		prices[i] = b.Close
	}

	// Compute mean and std deviation
	n := float64(len(prices))
	sum := 0.0
	for _, p := range prices {
		sum += p
	}
	mean := sum / n

	variance := 0.0
	for _, p := range prices {
		d := p - mean
		variance += d * d
	}
	stddev := math.Sqrt(variance / n)

	latest := prices[len(prices)-1]
	percentile := float64(50)
	sorted := make([]float64, len(prices))
	copy(sorted, prices)
	sort.Float64s(sorted)
	count := 0
	for _, p := range sorted {
		if p <= latest {
			count++
		}
	}
	percentile = math.Round(float64(count)/n*10000) / 100

	// Generate band points (every Nth bar to reduce data size)
	step := 1
	if len(bars) > 250 {
		step = len(bars) / 250
	}

	points := make([]BandPoint, 0, len(bars)/step+1)
	for i := 0; i < len(bars); i += step {
		points = append(points, BandPoint{
			Date:  bars[i].Date,
			Close: bars[i].Close,
			Band1: mean - 2*stddev,
			Band2: mean - 1*stddev,
			Band3: mean,
			Band4: mean + 1*stddev,
			Band5: mean + 2*stddev,
		})
	}

	return &BandResult{
		Symbol:     symbol,
		Metric:     "price",
		Current:    latest,
		Mean:       math.Round(mean*100) / 100,
		StdDev:     math.Round(stddev*100) / 100,
		Percentile: percentile,
		Points:     points,
	}
}
