package trading

import "math"

// OrderMatcher simulates fills of orders against OHLCV bars.
// Market orders fill at the bar's Open price.
// Limit buy orders fill if bar.Low <= limit price.
// Limit sell orders fill if bar.High >= limit price.
// Stop buy orders trigger if bar.High >= stop price.
// Stop sell orders trigger if bar.Low <= stop price.
type OrderMatcher struct{}

// NewOrderMatcher creates a new OrderMatcher.
func NewOrderMatcher() *OrderMatcher {
	return &OrderMatcher{}
}

// MatchResult is the result of matching an order against a bar.
type MatchResult struct {
	FilledQty  float64
	FillPrice  float64
	IsFillable bool
}

// Match checks if an order would fill against a given bar.
// Returns the quantity that would fill and at what price.
func (m *OrderMatcher) Match(order *Order, bar OHLCVBar) MatchResult {
	remaining := order.Quantity - order.FilledQty
	if remaining <= 0 {
		return MatchResult{}
	}

	switch order.OrderType {
	case TypeMarket:
		return MatchResult{
			FilledQty:  remaining,
			FillPrice:  bar.Open,
			IsFillable: true,
		}

	case TypeLimit:
		if order.Side == SideBuy {
			if bar.Low <= order.Price {
				// Fill at the better of limit price or bar.Open
				fillPrice := math.Min(order.Price, bar.Open)
				return MatchResult{
					FilledQty:  remaining,
					FillPrice:  fillPrice,
					IsFillable: true,
				}
			}
		} else {
			if bar.High >= order.Price {
				fillPrice := math.Max(order.Price, bar.Open)
				return MatchResult{
					FilledQty:  remaining,
					FillPrice:  fillPrice,
					IsFillable: true,
				}
			}
		}

	case TypeStop:
		if order.Side == SideBuy {
			if bar.High >= order.StopPrice {
				// Stop buy: fill at the stop price or worse (bar.Open/High)
				return MatchResult{
					FilledQty:  remaining,
					FillPrice:  math.Max(order.StopPrice, bar.Open),
					IsFillable: true,
				}
			}
		} else {
			if bar.Low <= order.StopPrice {
				return MatchResult{
					FilledQty:  remaining,
					FillPrice:  math.Min(order.StopPrice, bar.Open),
					IsFillable: true,
				}
			}
		}
	}

	return MatchResult{}
}
