package backtest

import (
	"context"
	"fmt"
	"quantflow/internal/trading"
	"sort"
	"time"
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

func (p *pdtTracker) dayTradesIn5Days(currentDate time.Time, tradingDates []time.Time) int {
	// Walk backwards through trading dates to find the boundary 5 trading days ago
	fiveTradingDaysAgo := currentDate
	daysBeforeCurrent := 0
	for i := len(tradingDates) - 1; i >= 0; i-- {
		if tradingDates[i].Equal(currentDate) {
			continue
		}
		if tradingDates[i].After(currentDate) {
			continue
		}
		daysBeforeCurrent++
		fiveTradingDaysAgo = tradingDates[i]
		if daysBeforeCurrent >= 5 {
			break
		}
	}

	count := 0
	for _, d := range p.trades {
		if !d.Before(fiveTradingDaysAgo) && !d.After(currentDate) {
			count++
		}
	}
	return count
}

func (p *pdtTracker) isPDT(currentDate time.Time, equity float64, tradingDates []time.Time) bool {
	return p.dayTradesIn5Days(currentDate, tradingDates) >= 4 && equity < 25000
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

	// Build sorted unique trading dates for PDT 5-business-day window calculation
	tradingDates := extractTradingDates(bars)

	portfolio := NewPortfolio(e.config.InitialCash)
	if err := e.oms.GetCashLedger().Deposit(e.config.InitialCash); err != nil {
		return nil, fmt.Errorf("deposit initial cash: %w", err)
	}
	var equityCurve []EquityPoint
	var tradeRecords []TradeRecord
	latestPrices := make(map[string]float64)

	// Track daily buys per symbol to detect day trades (buy + sell same day same symbol)
	dailyBuys := make(map[string]bool)
	var prevBar *trading.OHLCVBar

	for _, bar := range bars {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		e.oms.UpdateMarketPrice(bar.Symbol, bar.Close)
		latestPrices[bar.Symbol] = bar.Close

		// US is T+0 tradable (T+2 is cash settlement, not a sell lock):
		// clear the OMS T+1 lock every bar so same-day sells are allowed.
		// Without this, shares bought on any prior bar stay locked forever
		// and every sell is silently rejected ("T+1 lock: cannot sell").
		e.oms.ClearT1Lock()

		// Calculate current equity for PDT check
		currentEquity := portfolio.Equity(latestPrices)

		// 1. Check stop-loss/take-profit on existing positions
		// P0: Fill at bar.Close — stop was triggered at close, so open has already passed.
		if pos := e.oms.GetPosition(bar.Symbol); pos != nil && pos.Quantity > 0 {
			avgPrice := pos.AvgPrice // capture before FillOrder clears it
			posQty := pos.Quantity   // 同理：FillOrder 会原地扣减持仓
			if e.risk.CheckStopLoss(pos, bar.Close) {
				order, err := e.oms.PlaceOrder(bar.Symbol, trading.SideSell, trading.TypeMarket, "", posQty, 0)
				if err == nil {
					// Only record the trade when the fill actually succeeds —
					// recording a rejected fill produces phantom P&L.
					if _, fillErr := e.oms.FillOrder(order.ID, posQty, bar.Close); fillErr == nil {
						tradeRecords = append(tradeRecords, TradeRecord{
							Date: bar.Date, Symbol: bar.Symbol, Side: "sell",
							Quantity: posQty, Price: bar.Close,
							PnL: (bar.Close - avgPrice) * posQty,
						})
						if dailyBuys[bar.Symbol] {
							barDate, _ := time.Parse("2006-01-02", bar.Date)
							if !barDate.IsZero() {
								e.pdt.recordDayTrade(barDate)
							}
						}
						portfolio.Cash = e.oms.GetCashBalance()
						delete(portfolio.Positions, bar.Symbol)
						delete(portfolio.AvgPrice, bar.Symbol)
						goto recordEquityUS
					}
				}
			}
			if e.risk.CheckTakeProfit(pos, bar.Close) {
				order, err := e.oms.PlaceOrder(bar.Symbol, trading.SideSell, trading.TypeMarket, "", posQty, 0)
				if err == nil {
					if _, fillErr := e.oms.FillOrder(order.ID, posQty, bar.Close); fillErr == nil {
						tradeRecords = append(tradeRecords, TradeRecord{
							Date: bar.Date, Symbol: bar.Symbol, Side: "sell",
							Quantity: posQty, Price: bar.Close,
							PnL: (bar.Close - avgPrice) * posQty,
						})
						if dailyBuys[bar.Symbol] {
							barDate, _ := time.Parse("2006-01-02", bar.Date)
							if !barDate.IsZero() {
								e.pdt.recordDayTrade(barDate)
							}
						}
						portfolio.Cash = e.oms.GetCashBalance()
						delete(portfolio.Positions, bar.Symbol)
						delete(portfolio.AvgPrice, bar.Symbol)
						goto recordEquityUS
					}
				}
			}
		}

		// 2. Generate signal from strategy
		if strategy.SignalFunc != nil {
			signal := strategy.SignalFunc(bar.Open, prevBar, portfolio)
			if signal != nil && signal.Direction != "hold" {
				if signal.Direction == "buy" {
					// PDT check: block buy if PDT triggered and equity < $25k
					barDate, _ := time.Parse("2006-01-02", bar.Date)
					if !barDate.IsZero() && e.pdt.isPDT(barDate, currentEquity, tradingDates) {
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
		prevBar = &bar
	}

	metrics := ComputeMetrics(equityCurve, tradeRecords, e.config.RiskFreeRate)

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
		qty = 1 // US fractional shares: default 1 share
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
		Symbol:    bar.Symbol,
		Side:      trading.SideBuy,
		OrderType: trading.TypeMarket,
		Quantity:  qty,
		Price:     effectivePrice,
	}
	if err := e.risk.CheckOrder(mockOrder, pos, portfolioValue); err != nil {
		return
	}

	order, err := e.oms.PlaceOrder(bar.Symbol, trading.SideBuy, trading.TypeMarket, "", qty, 0)
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

	order, err := e.oms.PlaceOrder(bar.Symbol, trading.SideSell, trading.TypeMarket, "", qty, 0)
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

// extractTradingDates extracts unique sorted trading dates from OHLCV bars.
// Used by PDT tracker to compute 5-business-day windows correctly.
func extractTradingDates(bars []trading.OHLCVBar) []time.Time {
	seen := make(map[string]bool)
	var dates []time.Time
	for _, bar := range bars {
		if !seen[bar.Date] {
			seen[bar.Date] = true
			d, err := time.Parse("2006-01-02", bar.Date)
			if err == nil {
				dates = append(dates, d)
			}
		}
	}
	sort.Slice(dates, func(i, j int) bool { return dates[i].Before(dates[j]) })
	return dates
}
