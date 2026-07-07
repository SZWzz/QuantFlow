// Package backtest provides multi-market backtesting engines with
// market-specific rules (T+1, price limits, PDT, stamp duty) and
// configurable slippage/commission models.
package backtest

import (
	"context"
	"fmt"
	"sort"

	"quantflow/internal/trading"
)

// Strategy defines the signal logic for a backtest.
type Strategy struct {
	ID         string
	Name       string
	// SignalFunc receives the current open price and the previous completed bar.
	// It does NOT receive the current bar's Close/High/Low to prevent look-ahead bias.
	SignalFunc func(openPrice float64, prevBar *trading.OHLCVBar, portfolio *Portfolio) *trading.Signal
	RiskConfig trading.RiskConfig
}

// Runner executes a bar-by-bar backtest using historical OHLCV data.
type Runner struct {
	config  Config
	oms     *trading.OMS
	matcher *trading.OrderMatcher
	risk    *trading.RiskPipeline
}

// NewRunner creates a new backtest runner with the given configuration.
func NewRunner(config Config) *Runner {
	return &Runner{
		config:  config,
		oms:     trading.NewOMS(),
		matcher: trading.NewOrderMatcher(),
		risk:    trading.NewRiskPipeline(config.ToRiskConfig()),
	}
}

// OMS returns the Order Management System for external access.
func (r *Runner) OMS() *trading.OMS {
	return r.oms
}

// Run executes the backtest strategy against historical OHLCV bars.
// Bars must be pre-sorted by date. The runner processes each bar sequentially,
// generating signals, applying risk checks, matching orders, and tracking equity.
func (r *Runner) Run(ctx context.Context, strategy Strategy, bars []trading.OHLCVBar) (*Result, error) {
	if len(bars) == 0 {
		return nil, fmt.Errorf("no OHLCV data provided")
	}

	// Sort bars by date
	sort.Slice(bars, func(i, j int) bool { return bars[i].Date < bars[j].Date })

	portfolio := NewPortfolio(r.config.InitialCash)
	r.oms.GetCashLedger().Deposit(r.config.InitialCash)
	var equityCurve []EquityPoint
	var tradeRecords []TradeRecord
	latestPrices := make(map[string]float64)

	var prevBar *trading.OHLCVBar
	for _, bar := range bars {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// Update market prices in OMS for current positions
		r.oms.UpdateMarketPrice(bar.Symbol, bar.Close)
		latestPrices[bar.Symbol] = bar.Close

		// 1. Check stop-loss/take-profit on existing positions
		// P0: Fill at bar.Close (stop was triggered at close, so open has already passed).
		if pos := r.oms.GetPosition(bar.Symbol); pos != nil && pos.Quantity > 0 {
			avgPrice := pos.AvgPrice // capture before FillOrder clears it
			if r.risk.CheckStopLoss(pos, bar.Close) {
				order, err := r.oms.PlaceOrder(bar.Symbol, trading.SideSell, trading.TypeMarket, pos.Quantity, 0)
				if err == nil {
					r.oms.FillOrder(order.ID, pos.Quantity, bar.Close)
					tradeRecords = append(tradeRecords, TradeRecord{
						Date: bar.Date, Symbol: bar.Symbol, Side: "sell",
						Quantity: pos.Quantity, Price: bar.Close,
						PnL:      (bar.Close - avgPrice) * pos.Quantity,
					})
				}
				portfolio.Cash = r.oms.GetCashBalance()
				delete(portfolio.Positions, bar.Symbol)
				delete(portfolio.AvgPrice, bar.Symbol)
				goto recordEquity
			}
			if r.risk.CheckTakeProfit(pos, bar.Close) {
				order, err := r.oms.PlaceOrder(bar.Symbol, trading.SideSell, trading.TypeMarket, pos.Quantity, 0)
				if err == nil {
					r.oms.FillOrder(order.ID, pos.Quantity, bar.Close)
					tradeRecords = append(tradeRecords, TradeRecord{
						Date: bar.Date, Symbol: bar.Symbol, Side: "sell",
						Quantity: pos.Quantity, Price: bar.Close,
						PnL:      (bar.Close - avgPrice) * pos.Quantity,
					})
				}
				portfolio.Cash = r.oms.GetCashBalance()
				delete(portfolio.Positions, bar.Symbol)
				delete(portfolio.AvgPrice, bar.Symbol)
				goto recordEquity
			}
		}

		// 2. Generate signal from strategy (only prevBar, not current bar's OHLC)
		if strategy.SignalFunc != nil {
			signal := strategy.SignalFunc(bar.Open, prevBar, portfolio)
			if signal != nil && signal.Direction != "hold" {
				if signal.Direction == "buy" {
					r.processBuySignal(bar, signal, portfolio, &tradeRecords)
				} else if signal.Direction == "sell" {
					r.processSellSignal(bar, signal, portfolio, &tradeRecords)
				}
			}
		}

	recordEquity:
		// 3. Record daily equity
		equityCurve = append(equityCurve, EquityPoint{
			Date:   bar.Date,
			Equity: portfolio.Equity(latestPrices),
			Cash:   r.oms.GetCashBalance(),
		})
		prevBar = &bar
	}

	metrics := ComputeMetrics(equityCurve, tradeRecords)

	return &Result{
		Config:      r.config,
		EquityCurve: equityCurve,
		Trades:      tradeRecords,
		Metrics:     metrics,
	}, nil
}

func (r *Runner) processBuySignal(bar trading.OHLCVBar, signal *trading.Signal, portfolio *Portfolio, trades *[]TradeRecord) {
	qty := signal.Quantity
	if qty <= 0 {
		qty = 100
	}

	effectivePrice := bar.Open * (1 + r.config.Slippage)
	cost := effectivePrice*qty + effectivePrice*qty*r.config.Commission

	if cost > portfolio.Cash {
		return
	}

	// P0: Risk check before PlaceOrder, not after
	pos := r.oms.GetPosition(bar.Symbol)
	portfolioValue := portfolio.Cash
	for sym, q := range portfolio.Positions {
		portfolioValue += q * bar.Close
		_ = sym
	}
	mockOrder := &trading.Order{
		Symbol:   bar.Symbol,
		Side:     trading.SideBuy,
		OrderType: trading.TypeMarket,
		Quantity: qty,
		Price:    effectivePrice,
	}
	if err := r.risk.CheckOrder(mockOrder, pos, portfolioValue); err != nil {
		return
	}

	order, err := r.oms.PlaceOrder(bar.Symbol, trading.SideBuy, trading.TypeMarket, qty, 0)
	if err != nil {
		return
	}

	if _, err := r.oms.FillOrder(order.ID, qty, effectivePrice); err != nil {
		return
	}

	portfolio.Cash = r.oms.GetCashBalance()
	oldQty := portfolio.Positions[bar.Symbol]
	oldAvg := portfolio.AvgPrice[bar.Symbol]
	newQty := oldQty + qty
	portfolio.Positions[bar.Symbol] = newQty
	if newQty > 0 {
		// P1: Include commission in cost basis
		totalCost := qty*effectivePrice + qty*effectivePrice*r.config.Commission
		portfolio.AvgPrice[bar.Symbol] = (oldQty*oldAvg + totalCost) / newQty
	}

	*trades = append(*trades, TradeRecord{
		Date: bar.Date, Symbol: bar.Symbol, Side: "buy",
		Quantity: qty, Price: effectivePrice,
	})
}

func (r *Runner) processSellSignal(bar trading.OHLCVBar, signal *trading.Signal, portfolio *Portfolio, trades *[]TradeRecord) {
	qty := signal.Quantity
	heldQty := portfolio.Positions[bar.Symbol]
	if qty <= 0 {
		qty = heldQty // Sell all
	}
	if qty > heldQty {
		qty = heldQty
	}
	if qty <= 0 {
		return
	}

	effectivePrice := bar.Open * (1 - r.config.Slippage)
	revenue := effectivePrice*qty - effectivePrice*qty*r.config.Commission

	order, err := r.oms.PlaceOrder(bar.Symbol, trading.SideSell, trading.TypeMarket, qty, 0)
	if err != nil {
		return
	}

	if _, err := r.oms.FillOrder(order.ID, qty, effectivePrice); err != nil {
		return
	}

	portfolio.Cash = r.oms.GetCashBalance()
	oldQty := portfolio.Positions[bar.Symbol]
	avgPrice := portfolio.AvgPrice[bar.Symbol]
	newQty := oldQty - qty
	pnl := revenue - avgPrice*qty

	if newQty <= 0 {
		delete(portfolio.Positions, bar.Symbol)
		delete(portfolio.AvgPrice, bar.Symbol)
	} else {
		portfolio.Positions[bar.Symbol] = newQty
	}

	*trades = append(*trades, TradeRecord{
		Date: bar.Date, Symbol: bar.Symbol, Side: "sell",
		Quantity: qty, Price: effectivePrice, PnL: pnl,
	})
}
