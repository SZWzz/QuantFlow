package research

import (
	"math"
	"sort"

	"quantflow/internal/market/adapters"
)

// CBAnalysisResult is the output of convertible bond analysis.
type CBAnalysisResult struct {
	Quote          adapters.CBQuote `json:"quote"`
	DualLowScore   float64          `json:"dual_low_score"`   // 双低评分 (price + premium)
	FairValue      float64          `json:"fair_value"`       // 估算公允价值
	ValueGap       float64          `json:"value_gap"`        // 公允价值 - 市价 (正=低估)
	ValueGapPct    float64          `json:"value_gap_pct"`    // 折价率 (%)
	PutProbability float64          `json:"put_probability"`  // 下修概率 (0-1)
	IsCallRisk     bool             `json:"is_call_risk"`     // 强赎风险
	IsPutOpportunity bool           `json:"is_put_opportunity"` // 回售套利机会
}

// CBAnalyzer performs convertible bond analysis.
type CBAnalyzer struct {
	RiskFreeRate float64 // annual risk-free rate for bond valuation
}

// NewCBAnalyzer creates a CB analyzer with default 2% risk-free rate.
func NewCBAnalyzer() *CBAnalyzer {
	return &CBAnalyzer{RiskFreeRate: 0.02}
}

// Analyze computes full analysis for a single CB quote.
func (a *CBAnalyzer) Analyze(q adapters.CBQuote) CBAnalysisResult {
	result := CBAnalysisResult{
		Quote:        q,
		DualLowScore: q.DualLowScore(),
	}

	// Fair value = bond floor + option value
	bondFloor := a.computeBondFloor(q)
	optionValue := a.computeOptionValue(q)
	result.FairValue = bondFloor + optionValue

	if q.Price > 0 {
		result.ValueGap = result.FairValue - q.Price
		result.ValueGapPct = (result.FairValue/q.Price - 1) * 100
	}

	// Put probability (heuristic based on price vs put trigger)
	result.PutProbability = a.estimatePutProbability(q)

	// Call risk: price near or above call trigger
	result.IsCallRisk = q.CallPrice > 0 && q.Price >= q.CallPrice*0.9

	// Put arbitrage opportunity
	result.IsPutOpportunity = q.PutConvertPrice > 0 && q.Price < q.PutConvertPrice

	return result
}

// DualLowRank ranks CBs by dual-low score (ascending, lower is better).
func (a *CBAnalyzer) DualLowRank(quotes []adapters.CBQuote) []CBAnalysisResult {
	results := make([]CBAnalysisResult, 0, len(quotes))
	for _, q := range quotes {
		// Only include CBs with valid price and premium data
		if q.Price > 0 && q.StockPrice > 0 && q.ConversionPrice > 0 {
			results = append(results, a.Analyze(q))
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].DualLowScore < results[j].DualLowScore
	})

	return results
}

// Screen filters CBs by criteria and returns ranked results.
func (a *CBAnalyzer) Screen(quotes []adapters.CBQuote, maxPrice, maxPremiumRate float64, minYears float64) []CBAnalysisResult {
	filtered := make([]adapters.CBQuote, 0)
	for _, q := range quotes {
		if maxPrice > 0 && q.Price > maxPrice {
			continue
		}
		if maxPremiumRate > 0 && q.PremiumRate > maxPremiumRate {
			continue
		}
		// minYears filtering would require date parsing; skip for now
		_ = minYears
		filtered = append(filtered, q)
	}
	return a.DualLowRank(filtered)
}

// computeBondFloor estimates the pure bond value using discounted cash flows.
func (a *CBAnalyzer) computeBondFloor(q adapters.CBQuote) float64 {
	if q.BondValue > 0 {
		return q.BondValue // use EastMoney's computed value if available
	}
	// Fallback: simple estimate assuming par=100, coupon=2%, remaining=3yr
	return 100.0 // par value floor
}

// computeOptionValue estimates the embedded call option value using simplified BS.
func (a *CBAnalyzer) computeOptionValue(q adapters.CBQuote) float64 {
	if q.StockPrice <= 0 || q.ConversionPrice <= 0 {
		return 0
	}

	// Simplified Black-Scholes call option
	S := q.StockPrice
	K := q.ConversionPrice

	// Estimate remaining time (default 3 years if no maturity date)
	T := 3.0 // years
	r := a.RiskFreeRate
	sigma := 0.30 // assumed 30% volatility for A-share stocks

	d1 := (math.Log(S/K) + (r+sigma*sigma/2)*T) / (sigma * math.Sqrt(T))
	d2 := d1 - sigma*math.Sqrt(T)

	// Standard normal CDF approximation
	callValue := S*normalCDF(d1) - K*math.Exp(-r*T)*normalCDF(d2)
	if callValue < 0 {
		callValue = 0
	}

	// Conversion ratio: 100 / conversion_price shares per bond
	conversionRatio := 100.0 / q.ConversionPrice
	return callValue * conversionRatio
}

// estimatePutProbability estimates the probability of a downward conversion price adjustment.
func (a *CBAnalyzer) estimatePutProbability(q adapters.CBQuote) float64 {
	if q.StockPrice <= 0 || q.PutPrice <= 0 {
		return 0
	}
	// Put trigger typically: stock < put_trigger_price for N days
	// Heuristic: closer the stock is to the put trigger, higher the probability
	if q.StockPrice >= q.PutPrice {
		return 0.1 // above trigger, low probability
	}
	ratio := 1 - q.StockPrice/q.PutPrice
	return math.Min(ratio*2, 0.8) // max 80% probability
}

// normalCDF is a polynomial approximation of the standard normal CDF.
func normalCDF(x float64) float64 {
	return 0.5 * math.Erfc(-x/math.Sqrt2)
}
