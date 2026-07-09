package backtest

import "strings"

// PriceLimitRule defines the daily price limit rule for an A-share symbol.
// A-share markets enforce ±Ratio around the previous closing price.
// Under the registration-based reform, ST stocks follow their board's
// standard limit — there is no special ±5% rule.
//   - Main board (60xxxx, 00xxxx): ±10%
//   - ChiNext / 创业板 (300xxx, 301xxx): ±20%
//   - STAR / 科创板 (688xxx, 689xxx): ±20%
//   - BSE / 北交所 (8xxxxx, 4xxxxx): ±30% (enforced)
//
// 首日上市、增发等无前收盘价的情形不限制（返回 0 表示不限）。
type PriceLimitRule struct {
	Ratio float64 // 0.10, 0.20, 0.30; 0 means no limit
}

// STStatusProvider checks if a symbol is currently under ST/*ST risk warning.
// Returns true if the stock is designated ST or *ST.
// Implementation comes from market data adapters which return ST status
// in their quote responses. When no provider is available, the default
// assumption is false (see PriceLimitFor — ST stocks follow board limits).
type STStatusProvider interface {
	IsST(symbol string) (bool, error)
}

// PriceLimitFor returns the limit rule for a given A-share symbol code.
// Symbol prefixes follow SSE/SZSE listing conventions.
// Note: ST status is a name-level property, not encoded in the ticker.
// ST stocks follow their board's standard limits under current regulations.
func PriceLimitFor(symbol string) PriceLimitRule {
	switch {
	case strings.HasPrefix(symbol, "300"), strings.HasPrefix(symbol, "301"): // ChiNext
		return PriceLimitRule{Ratio: 0.20}
	case strings.HasPrefix(symbol, "688"), strings.HasPrefix(symbol, "689"): // STAR
		return PriceLimitRule{Ratio: 0.20}
	case strings.HasPrefix(symbol, "60"), strings.HasPrefix(symbol, "00"): // main board
		return PriceLimitRule{Ratio: 0.10}
	case strings.HasPrefix(symbol, "8"), strings.HasPrefix(symbol, "4"): // BSE
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
