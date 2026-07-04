package backtest

import (
	"context"
	"sort"
	"time"

	"quantflow/internal/trading"
)

// ── PDT (Pattern Day Trader) Tracker ───────────────────────────────────────

// pdtTracker implements a 5-day rolling window for day trade counting.
// PDT rule: >= 4 day trades in 5 business days triggers PDT restriction
// when account equity < $25,000.
type pdtTracker struct {
	trades []time.Time // day trade dates
}

func newPDTTracker() *pdtTracker {
	return &pdtTracker{
		trades: make([]time.Time, 0),
	}
}

func (p *pdtTracker) recordDayTrade(date time.Time) {
	p.trades = append(p.trades, date)
}

func (p *pdtTracker) dayTradesIn5Days(currentDate time.Time) int {
	count := 0
	fiveDaysAgo := currentDate.AddDate(0, 0, -5)
	for _, d := range p.trades {
		if d.After(fiveDaysAgo) && !d.After(currentDate) {
			count++
		}
	}
	return count
}

func (p *pdtTracker) isPDT(currentDate time.Time, equity float64) bool {
	return p.dayTradesIn5Days(currentDate) >= 4 && equity < 25000
}

// ── US Engine ──────────────────────────────────────────────────────────────

// USEngine is the US stock backtesting engine.
// US market rules (simpler than A-shares):
//   - T+2 settlement (irrelevant for bar-by-bar simulation)
//   - No price limits
//   - PDT rule: pattern day trader check (>=4 day trades in 5 days with <$25k equity)
//   - Fractional shares: no lot size restriction
//   - No stamp duty
type USEngine struct {
	*Runner
	pdt *pdtTracker
}

// NewUSEngine creates a US stock backtesting engine with default US config.
func NewUSEngine(config Config) *USEngine {
	if config.Commission == 0 {
		config.Commission = 0.001 // 0.1% typical US commission
	}
	return &USEngine{
		Runner: NewRunner(config),
		pdt:    newPDTTracker(),
	}
}

// Run executes the backtest with US market rules and PDT enforcement.
// PDT blocks intraday buy orders when triggered and equity < $25,000.
func (e *USEngine) Run(ctx context.Context, strategy Strategy, bars []trading.OHLCVBar) (*Result, error) {
	if len(bars) == 0 {
		return nil, errNoData
	}

	sort.Slice(bars, func(i, j int) bool { return bars[i].Date < bars[j].Date })

	portfolio := NewPortfolio(e.config.InitialCash)
	e.oms.GetCashLedger().Deposit(e.config.InitialCash)
	var equityCurve []EquityPoint
	var tradeRecords []TradeRecord
	latestPrices := make(map[string]float64)

	// Track daily buys per symbol to detect day trades (buy + sell same day same symbol)
	dailyBuys := make(map[string]bool)

	for _, bar := range bars {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		e.oms.UpdateMarketPrice(bar.Symbol, bar.Close)
		latestPrices[bar.Symbol] = bar.Close

		// Calculate current equity for PDT check
		currentEquity := portfolio.Equity(latestPrices)

		// 1. Check stop-loss/take-profit on existing positions
		// P0: Fill at bar.Close — stop was triggered at close, so open has already passed.
		if pos := e.oms.GetPosition(bar.Symbol); pos != nil && pos.Quantity > 0 {
			if e.risk.CheckStopLoss(pos, bar.Close) {
				order, err := e.oms.PlaceOrder(bar.Symbol, trading.SideSell, trading.TypeMarket, pos.Quantity, 0)
				if err == nil {
					e.oms.FillOrder(order.ID, pos.Quantity, bar.Close)
					tradeRecords = append(tradeRecords, TradeRecord{
						Date: bar.Date, Symbol: bar.Symbol, Side: "sell",
						Quantity: pos.Quantity, Price: bar.Close,
					})
					if dailyBuys[bar.Symbol] {
						barDate, _ := time.Parse("2006-01-02", bar.Date)
						if !barDate.IsZero() {
							e.pdt.recordDayTrade(barDate)
						}
					}
				}
				portfolio.Cash = e.oms.GetCashBalance()
				delete(portfolio.Positions, bar.Symbol)
				delete(portfolio.AvgPrice, bar.Symbol)
				goto recordEquityUS
			}
			if e.risk.CheckTakeProfit(pos, bar.Close) {
				order, err := e.oms.PlaceOrder(bar.Symbol, trading.SideSell, trading.TypeMarket, pos.Quantity, 0)
				if err == nil {
					e.oms.FillOrder(order.ID, pos.Quantity, bar.Close)
					tradeRecords = append(tradeRecords, TradeRecord{
						Date: bar.Date, Symbol: bar.Symbol, Side: "sell",
						Quantity: pos.Quantity, Price: bar.Close,
						PnL:      (bar.Close - pos.AvgPrice) * pos.Quantity,
					})
					if dailyBuys[bar.Symbol] {
						barDate, _ := time.Parse("2006-01-02", bar.Date)
						if !barDate.IsZero() {
							e.pdt.recordDayTrade(barDate)
						}
					}
				}
				portfolio.Cash = e.oms.GetCashBalance()
				delete(portfolio.Positions, bar.Symbol)
				delete(portfolio.AvgPrice, bar.Symbol)
				goto recordEquityUS
			}
		}

		// 2. Generate signal from strategy
		if strategy.SignalFunc != nil {
			signal := strategy.SignalFunc(bar, portfolio)
			if signal != nil && signal.Direction != "hold" {
				if signal.Direction == "buy" {
					// PDT check: block buy if PDT triggered and equity < $25k
					barDate, _ := time.Parse("2006-01-02", bar.Date)
					if !barDate.IsZero() && e.pdt.isPDT(barDate, currentEquity) {
						goto recordEquityUS
					}
					e.processUSBuySignal(bar, signal, portfolio, &tradeRecords, &dailyBuys)
				} else if signal.Direction == "sell" {
					e.processUSSellSignal(bar, signal, portfolio, &tradeRecords, &dailyBuys)
				}
			}
		}

	recordEquityUS:
		// Reset daily buys tracking at end of day
		dailyBuys = make(map[string]bool)

		// 3. Record daily equity
		equityCurve = append(equityCurve, EquityPoint{
			Date:   bar.Date,
			Equity: portfolio.Equity(latestPrices),
			Cash:   e.oms.GetCashBalance(),
		})
	}

	metrics := ComputeMetrics(equityCurve, tradeRecords)

	return &Result{
		Config:      e.config,
		EquityCurve: equityCurve,
		Trades:      tradeRecords,
		Metrics:     metrics,
	}, nil
}

func (e *USEngine) processUSBuySignal(bar trading.OHLCVBar, signal *trading.Signal, portfolio *Portfolio, trades *[]TradeRecord, dailyBuys *map[string]bool) {
	qty := signal.Quantity
	if qty <= 0 {
		qty = 100
	}

	effectivePrice := bar.Open * (1 + e.config.Slippage)
	cost := effectivePrice*qty + effectivePrice*qty*e.config.Commission

	if cost > portfolio.Cash {
		return
	}

	// P0: Risk check before PlaceOrder
	pos := e.oms.GetPosition(bar.Symbol)
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
	if err := e.risk.CheckOrder(mockOrder, pos, portfolioValue); err != nil {
		return
	}

	order, err := e.oms.PlaceOrder(bar.Symbol, trading.SideBuy, trading.TypeMarket, qty, 0)
	if err != nil {
		return
	}

	if _, err := e.oms.FillOrder(order.ID, qty, effectivePrice); err != nil {
		return
	}

	portfolio.Cash = e.oms.GetCashBalance()
	oldQty := portfolio.Positions[bar.Symbol]
	oldAvg := portfolio.AvgPrice[bar.Symbol]
	newQty := oldQty + qty
	portfolio.Positions[bar.Symbol] = newQty
	if newQty > 0 {
		// P1: Include commission in cost basis
		totalCost := qty*effectivePrice + qty*effectivePrice*e.config.Commission
		portfolio.AvgPrice[bar.Symbol] = (oldQty*oldAvg + totalCost) / newQty
	}

	(*dailyBuys)[bar.Symbol] = true

	*trades = append(*trades, TradeRecord{
		Date: bar.Date, Symbol: bar.Symbol, Side: "buy",
		Quantity: qty, Price: effectivePrice,
	})
}

func (e *USEngine) processUSSellSignal(bar trading.OHLCVBar, signal *trading.Signal, portfolio *Portfolio, trades *[]TradeRecord, dailyBuys *map[string]bool) {
	qty := signal.Quantity
	heldQty := portfolio.Positions[bar.Symbol]
	if qty <= 0 {
		qty = heldQty
	}
	if qty > heldQty {
		qty = heldQty
	}
	if qty <= 0 {
		return
	}

	effectivePrice := bar.Open * (1 - e.config.Slippage)
	revenue := effectivePrice*qty - effectivePrice*qty*e.config.Commission

	order, err := e.oms.PlaceOrder(bar.Symbol, trading.SideSell, trading.TypeMarket, qty, 0)
	if err != nil {
		return
	}

	if _, err := e.oms.FillOrder(order.ID, qty, effectivePrice); err != nil {
		return
	}

	portfolio.Cash = e.oms.GetCashBalance()
	avgPrice := portfolio.AvgPrice[bar.Symbol]
	pnl := revenue - avgPrice*qty

	newQty := heldQty - qty
	if newQty <= 0 {
		delete(portfolio.Positions, bar.Symbol)
		delete(portfolio.AvgPrice, bar.Symbol)
	} else {
		portfolio.Positions[bar.Symbol] = newQty
	}

	// Record day trade if we bought the same symbol earlier today
	if (*dailyBuys)[bar.Symbol] {
		barDate, _ := time.Parse("2006-01-02", bar.Date)
		if !barDate.IsZero() {
			e.pdt.recordDayTrade(barDate)
		}
	}

	*trades = append(*trades, TradeRecord{
		Date: bar.Date, Symbol: bar.Symbol, Side: "sell",
		Quantity: qty, Price: effectivePrice, PnL: pnl,
	})
}

// RunWithPDT executes the backtest with PDT enforcement (same as Run in v2).
// The base Run() already includes PDT tracking and blocking.
func (e *USEngine) RunWithPDT(ctx context.Context, strategy Strategy, bars []trading.OHLCVBar) (*Result, error) {
	return e.Run(ctx, strategy, bars)
}
