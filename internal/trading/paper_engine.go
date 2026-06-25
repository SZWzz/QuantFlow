package trading

import (
	"fmt"
	"log/slog"
)

// PaperEngine executes trades against simulated (paper) market data.
// It uses the OrderMatcher to fill orders against each bar.
type PaperEngine struct {
	oms           *OMS
	matcher       *OrderMatcher
	riskPipeline  *RiskPipeline
	portfolioValue float64
}

// NewPaperEngine creates a new paper trading engine.
func NewPaperEngine(oms *OMS, riskConfig RiskConfig, initialCapital float64) *PaperEngine {
	return &PaperEngine{
		oms:            oms,
		matcher:        NewOrderMatcher(),
		riskPipeline:   NewRiskPipeline(riskConfig),
		portfolioValue: initialCapital,
	}
}

// OnBar processes a single OHLCV bar through the paper trading pipeline:
// 1. Check risk triggers (stop loss / take profit)
// 2. Match pending orders against the bar
// 3. Update positions with market prices
func (pe *PaperEngine) OnBar(bar OHLCVBar) []*Trade {
	var trades []*Trade

	// Check risk triggers for all positions
	for _, pos := range pe.oms.GetAllPositions() {
		if pos.Quantity == 0 {
			continue
		}

		// Stop loss / take profit generate market orders to close
		if pe.riskPipeline.CheckStopLoss(pos, bar.Close) {
			slog.Info("stop loss triggered", "symbol", pos.Symbol, "price", bar.Close, "pnl", pos.PnL)
			// Close position at market
			if pos.Quantity > 0 {
				order, err := pe.oms.PlaceOrder(pos.Symbol, SideSell, TypeMarket, pos.Quantity, 0)
				if err != nil { slog.Error("stop-loss place failed", "symbol", pos.Symbol, "error", err); continue }
				trade, err := pe.oms.FillOrder(order.ID, pos.Quantity, bar.Close)
				if err != nil { slog.Error("stop-loss fill failed", "symbol", pos.Symbol, "error", err); continue }
				if trade != nil { trades = append(trades, trade) }
			} else {
				order, err := pe.oms.PlaceOrder(pos.Symbol, SideBuy, TypeMarket, -pos.Quantity, 0)
				if err != nil { slog.Error("stop-loss place failed", "symbol", pos.Symbol, "error", err); continue }
				trade, err := pe.oms.FillOrder(order.ID, -pos.Quantity, bar.Close)
				if err != nil { slog.Error("stop-loss fill failed", "symbol", pos.Symbol, "error", err); continue }
				if trade != nil { trades = append(trades, trade) }
			}
			continue
		}

		if pe.riskPipeline.CheckTakeProfit(pos, bar.Close) {
			slog.Info("take profit triggered", "symbol", pos.Symbol, "price", bar.Close, "pnl", pos.PnL)
			if pos.Quantity > 0 {
				order, err := pe.oms.PlaceOrder(pos.Symbol, SideSell, TypeMarket, pos.Quantity, 0)
				if err != nil { slog.Error("take-profit place failed", "symbol", pos.Symbol, "error", err); continue }
				trade, err := pe.oms.FillOrder(order.ID, pos.Quantity, bar.Close)
				if err != nil { slog.Error("take-profit fill failed", "symbol", pos.Symbol, "error", err); continue }
				if trade != nil { trades = append(trades, trade) }
			} else {
				order, err := pe.oms.PlaceOrder(pos.Symbol, SideBuy, TypeMarket, -pos.Quantity, 0)
				if err != nil { slog.Error("take-profit place failed", "symbol", pos.Symbol, "error", err); continue }
				trade, err := pe.oms.FillOrder(order.ID, -pos.Quantity, bar.Close)
				if err != nil { slog.Error("take-profit fill failed", "symbol", pos.Symbol, "error", err); continue }
				if trade != nil { trades = append(trades, trade) }
			}
		}
	}

	// Match pending orders
	for _, order := range pe.oms.GetOrders() {
		if order.Status != StatusPending && order.Status != StatusPartial {
			continue
		}

		result := pe.matcher.Match(order, bar)
		if result.IsFillable && result.FilledQty > 0 {
			trade, err := pe.oms.FillOrder(order.ID, result.FilledQty, result.FillPrice)
			if err != nil {
				slog.Error("failed to fill order", "order", order.ID, "error", err)
				continue
			}
			if trade != nil {
				trades = append(trades, trade)
			}
		}
	}

	// Update market prices
	pe.oms.UpdateMarketPrice(bar.Symbol, bar.Close)

	return trades
}

// PlaceOrder places an order through the paper engine with risk checks.
func (pe *PaperEngine) PlaceOrder(symbol string, side OrderSide, orderType OrderType, qty, price float64) (*Order, error) {
	// Get current position for risk checks
	pos := pe.oms.GetPosition(symbol)

	// Create a temporary order for risk validation
	tempOrder := &Order{
		Symbol:    symbol,
		Side:      side,
		OrderType: orderType,
		Quantity:  qty,
		Price:     price,
	}

	if err := pe.riskPipeline.CheckOrder(tempOrder, pos, pe.portfolioValue); err != nil {
		return nil, fmt.Errorf("risk check failed: %w", err)
	}

	return pe.oms.PlaceOrder(symbol, side, orderType, qty, price)
}

// GetOMS returns the underlying OMS.
func (pe *PaperEngine) GetOMS() *OMS {
	return pe.oms
}

// GetPositions returns all current positions with P&L.
func (pe *PaperEngine) GetPositions() []*Position {
	return pe.oms.GetAllPositions()
}
