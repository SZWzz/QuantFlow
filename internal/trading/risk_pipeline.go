package trading

import "fmt"

// RiskConfig defines the risk parameters for the trading engine.
type RiskConfig struct {
	MaxPositionPct float64 // max single position as % of portfolio (0 = disabled)
	StopLossPct    float64 // stop loss % from entry price (0 = disabled)
	TakeProfitPct  float64 // take profit % from entry price (0 = disabled)
	MaxDrawdownPct float64 // max drawdown before suspending (0 = disabled)
}

// DefaultRiskConfig returns sensible default risk parameters.
func DefaultRiskConfig() RiskConfig {
	return RiskConfig{
		MaxPositionPct: 0.25,  // 25% per position
		StopLossPct:    0.05,  // 5% stop loss
		TakeProfitPct:  0.15,  // 15% take profit
		MaxDrawdownPct: 0.20,  // 20% max drawdown
	}
}

// RiskPipeline checks orders and positions against risk rules.
type RiskPipeline struct {
	config RiskConfig
}

// NewRiskPipeline creates a new RiskPipeline.
func NewRiskPipeline(config RiskConfig) *RiskPipeline {
	return &RiskPipeline{config: config}
}

// CheckOrder validates an order against risk rules.
// Returns nil if the order passes all checks.
func (r *RiskPipeline) CheckOrder(order *Order, position *Position, portfolioValue float64) error {
	if portfolioValue <= 0 {
		return fmt.Errorf("invalid portfolio value: %f", portfolioValue)
	}

	// Check max position size
	if r.config.MaxPositionPct > 0 {
		orderValue := order.Quantity * order.Price
		// For market orders (Price=0), use available price from position.
		if order.OrderType == TypeMarket && order.Price == 0 {
			if position != nil && position.MarketPrice > 0 {
				orderValue = order.Quantity * position.MarketPrice
			} else if position != nil && position.AvgPrice > 0 {
				orderValue = order.Quantity * position.AvgPrice
			}
			// If price is still unknowable, skip the check (backward-compatible).
			if orderValue == 0 {
				return nil
			}
		}
		positionPct := orderValue / portfolioValue
		if positionPct > r.config.MaxPositionPct {
			return fmt.Errorf("order value %.2f exceeds max position %.1f%% of portfolio",
				orderValue, r.config.MaxPositionPct*100)
		}
	}

	return nil
}

// CheckStopLoss checks if the current price triggers a stop loss for the position.
// Returns true if stop loss should be triggered.
// For long positions (qty>0): triggers when price drops below entry.
// For short positions (qty<0): triggers when price rises above entry.
func (r *RiskPipeline) CheckStopLoss(position *Position, currentPrice float64) bool {
	if r.config.StopLossPct <= 0 || position == nil || position.Quantity == 0 || position.AvgPrice == 0 {
		return false
	}

	pricePct := (currentPrice - position.AvgPrice) / position.AvgPrice
	if position.Quantity > 0 {
		// Long: loss when price goes down
		return pricePct <= -r.config.StopLossPct
	}
	// Short: loss when price goes up
	return pricePct >= r.config.StopLossPct
}

// CheckTakeProfit checks if the current price triggers a take profit for the position.
// For long positions (qty>0): triggers when price rises above entry.
// For short positions (qty<0): triggers when price drops below entry.
func (r *RiskPipeline) CheckTakeProfit(position *Position, currentPrice float64) bool {
	if r.config.TakeProfitPct <= 0 || position == nil || position.Quantity == 0 || position.AvgPrice == 0 {
		return false
	}

	pricePct := (currentPrice - position.AvgPrice) / position.AvgPrice
	if position.Quantity > 0 {
		// Long: profit when price goes up
		return pricePct >= r.config.TakeProfitPct
	}
	// Short: profit when price goes down
	return pricePct <= -r.config.TakeProfitPct
}
