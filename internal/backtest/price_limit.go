package backtest

import "strings"

// PriceLimitRule defines the daily price limit rule for an A-share symbol.
// A-share markets enforce ±Ratio around the previous closing price.
//   - Main board (60xxxx, 00xxxx): ±10%
//   - ChiNext / 创业板 (300xxx, 301xxx): ±20%
//   - STAR / 科创板 (688xxx, 689xxx): ±20%
//   - ST / *ST stocks: ±5% (needs Config.PriceLimitOverrides — ST status cannot
//     be inferred from symbol code alone; deferred to Phase B)
//   - BSE / 北交所 (8xxxxx, 4xxxxx): ±30% (enforced)
//
// 首日上市、增发等无前收盘价的情形不限制（返回 0 表示不限）。
type PriceLimitRule struct {
	Ratio float64 // 0.10, 0.20, 0.05; 0 means no limit
}

// PriceLimitFor returns the limit rule for a given A-share symbol code.
// Symbol prefixes follow SSE/SZSE listing conventions.
func PriceLimitFor(symbol string) PriceLimitRule {
	upper := strings.ToUpper(symbol)

	// ST / *ST detection (name-based would be more accurate, but code-based
	// fallback: we cannot know ST status from code alone, so default ST to 5%.
	// Callers may override via Config.PriceLimitOverrides.)
	switch {
	case strings.HasPrefix(upper, "300"), strings.HasPrefix(upper, "301"): // ChiNext
		return PriceLimitRule{Ratio: 0.20}
	case strings.HasPrefix(upper, "688"), strings.HasPrefix(upper, "689"): // STAR
		return PriceLimitRule{Ratio: 0.20}
	case strings.HasPrefix(upper, "60"), strings.HasPrefix(upper, "00"): // main board
		return PriceLimitRule{Ratio: 0.10}
	case strings.HasPrefix(upper, "8"), strings.HasPrefix(upper, "4"): // BSE
		return PriceLimitRule{Ratio: 0.30}
	default:
		return PriceLimitRule{Ratio: 0.10} // safe default
	}
}

// LimitUp returns the limit-up price for today given prevClose.
// Returns 0 if no limit applies (rule.Ratio == 0 or prevClose <= 0).
func (r PriceLimitRule) LimitUp(prevClose float64) float64 {
	if r.Ratio == 0 || prevClose <= 0 {
		return 0
	}
	return prevClose * (1 + r.Ratio)
}

// LimitDown returns the limit-down price for today given prevClose.
// Returns 0 if no limit applies (rule.Ratio == 0 or prevClose <= 0).
func (r PriceLimitRule) LimitDown(prevClose float64) float64 {
	if r.Ratio == 0 || prevClose <= 0 {
		return 0
	}
	return prevClose * (1 - r.Ratio)
}

// CanBuy reports whether a buy is allowed at the given price today.
// A-share rule: cannot BUY at or above limit-up (涨停封板买不进).
func (r PriceLimitRule) CanBuy(price, prevClose float64) bool {
	up := r.LimitUp(prevClose)
	if up <= 0 {
		return true // no limit
	}
	return price < up
}

// CanSell reports whether a sell is allowed at the given price today.
// A-share rule: cannot SELL at or below limit-down (跌停封板卖不出).
func (r PriceLimitRule) CanSell(price, prevClose float64) bool {
	down := r.LimitDown(prevClose)
	if down <= 0 {
		return true // no limit
	}
	return price > down
}
