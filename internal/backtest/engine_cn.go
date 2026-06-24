package backtest

import (
	"context"
	"sort"

	"quantflow/internal/trading"
)

// CNEngine is the A-share backtesting engine with market-specific rules:
//   - T+1 settlement: shares bought today cannot be sold until tomorrow
//   - Price limits: ±10% (main board) or ±20% (ChiNext/STAR)
//   - Stamp duty: 0.05% on sell only (2024新政)
//   - Minimum lot: 100 shares, multiples of 100
//   - Commission: 0.03% (万三) default
type CNEngine struct {
	*Runner
	t1Lock    *t1Tracker         // tracks T+1 locked shares
	prevClose map[string]float64 // symbol → previous trading day close (for price limit)
}

// t1Tracker tracks shares locked by T+1 settlement.
type t1Tracker struct {
	locked map[string]float64 // symbol → locked quantity from today's buys
}

func newT1Tracker() *t1Tracker {
	return &t1Tracker{locked: make(map[string]float64)}
}

// NewCNEngine creates an A-share backtesting engine with default A-share config.
func NewCNEngine(config Config) *CNEngine {
	// Apply A-share specific defaults
	if config.Commission == 0 {
		config.Commission = 0.0003 // 万三佣金
	}
	if config.Slippage == 0 {
		config.Slippage = 0.001 // 10 bps
	}
	return &CNEngine{
		Runner:    NewRunner(config),
		t1Lock:    newT1Tracker(),
		prevClose: make(map[string]float64),
	}
}

// stampDuty returns the stamp duty for a sell trade (0.05% since 2024).
func stampDuty(tradeValue float64) float64 {
	return tradeValue * 0.0005
}

// Run executes the backtest with A-share market rules.
func (e *CNEngine) Run(ctx context.Context, strategy Strategy, bars []trading.OHLCVBar) (*Result, error) {
	if len(bars) == 0 {
		return nil, errNoData
	}

	sort.Slice(bars, func(i, j int) bool { return bars[i].Date < bars[j].Date })

	portfolio := NewPortfolio(e.config.InitialCash)
	var equityCurve []EquityPoint
	var tradeRecords []TradeRecord

	for _, bar := range bars {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		e.oms.UpdateMarketPrice(bar.Symbol, bar.Close)

		// Available sellable quantity = held - T+1 locked
		heldQty := portfolio.Positions[bar.Symbol]
		lockedQty := e.t1Lock.locked[bar.Symbol]
		availableQty := heldQty - lockedQty

		// 1. Check stop-loss/take-profit (only on available shares)
		if pos := e.oms.GetPosition(bar.Symbol); pos != nil && availableQty > 0 {
			if e.risk.CheckStopLoss(pos, bar.Close) || e.risk.CheckTakeProfit(pos, bar.Close) {
				// P0-2: 跌停封板时止损/止盈单无法成交（A 股无买盘）。
				// 所有卖出路径都必须经过涨跌停校验，含止损/止盈。
				rule := PriceLimitFor(bar.Symbol)
				if !rule.CanSell(bar.Close, e.prevClose[bar.Symbol]) {
					goto recordEquityCN
				}
				order, err := e.oms.PlaceOrder(bar.Symbol, trading.SideSell, trading.TypeMarket, availableQty, 0)
				if err == nil {
					e.oms.FillOrder(order.ID, availableQty, bar.Close)
					revenue := bar.Close*availableQty - stampDuty(bar.Close*availableQty) - bar.Close*availableQty*e.config.Commission
					pnl := revenue - pos.AvgPrice*availableQty
					portfolio.Cash += revenue

					newQty := heldQty - availableQty
					if newQty <= 0 {
						delete(portfolio.Positions, bar.Symbol)
						delete(portfolio.AvgPrice, bar.Symbol)
					} else {
						portfolio.Positions[bar.Symbol] = newQty
					}
					tradeRecords = append(tradeRecords, TradeRecord{
						Date: bar.Date, Symbol: bar.Symbol, Side: "sell",
						Quantity: availableQty, Price: bar.Close, PnL: pnl,
					})
				}
				goto recordEquityCN
			}
		}

		// 2. Generate signals
		if strategy.SignalFunc != nil {
			signal := strategy.SignalFunc(bar, portfolio)
			if signal != nil && signal.Direction != "hold" {
				if signal.Direction == "buy" {
					e.processCNBuySignal(bar, signal, portfolio, &tradeRecords)
				} else if signal.Direction == "sell" {
					if signal.Quantity <= 0 || signal.Quantity > availableQty {
						signal.Quantity = availableQty
					}
					if signal.Quantity > 0 {
						e.processCNSellSignal(bar, signal, portfolio, &tradeRecords)
					}
				}
			}
		}

	recordEquityCN:
		// Update prevClose for next day's price limit check
		e.prevClose[bar.Symbol] = bar.Close

		// Clear T+1 lock (shares bought yesterday are now sellable)
		e.t1Lock.locked = make(map[string]float64)

		// Record equity
		prices := map[string]float64{bar.Symbol: bar.Close}
		equityCurve = append(equityCurve, EquityPoint{
			Date:   bar.Date,
			Equity: portfolio.Equity(prices),
			Cash:   portfolio.Cash,
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

func (e *CNEngine) processCNBuySignal(bar trading.OHLCVBar, signal *trading.Signal, portfolio *Portfolio, trades *[]TradeRecord) {
	rule := PriceLimitFor(bar.Symbol)
	if !rule.CanBuy(bar.Close, e.prevClose[bar.Symbol]) {
		// 涨停封板，买不进
		return
	}
	qty := signal.Quantity
	if qty <= 0 {
		qty = 100
	}
	// Round to lot size (multiples of 100)
	qty = float64(int(qty/100)) * 100
	if qty <= 0 {
		return
	}

	effectivePrice := bar.Close * (1 + e.config.Slippage)
	cost := effectivePrice*qty + effectivePrice*qty*e.config.Commission

	if cost > portfolio.Cash {
		return
	}

	order, err := e.oms.PlaceOrder(bar.Symbol, trading.SideBuy, trading.TypeMarket, qty, 0)
	if err != nil {
		return
	}

	// Risk check
	pos := e.oms.GetPosition(bar.Symbol)
	portfolioValue := portfolio.Cash
	for sym, q := range portfolio.Positions {
		portfolioValue += q * bar.Close
		_ = sym
	}
	order.Price = effectivePrice
	if err := e.risk.CheckOrder(order, pos, portfolioValue); err != nil {
		e.oms.CancelOrder(order.ID)
		return
	}

	if _, err := e.oms.FillOrder(order.ID, qty, effectivePrice); err != nil {
		return
	}

	portfolio.Cash -= cost
	oldQty := portfolio.Positions[bar.Symbol]
	oldAvg := portfolio.AvgPrice[bar.Symbol]
	newQty := oldQty + qty
	portfolio.Positions[bar.Symbol] = newQty
	if newQty > 0 {
		portfolio.AvgPrice[bar.Symbol] = (oldQty*oldAvg + qty*effectivePrice) / newQty
	}

	// T+1 lock: shares bought today cannot be sold today
	e.t1Lock.locked[bar.Symbol] += qty

	*trades = append(*trades, TradeRecord{
		Date: bar.Date, Symbol: bar.Symbol, Side: "buy",
		Quantity: qty, Price: effectivePrice,
	})
}

func (e *CNEngine) processCNSellSignal(bar trading.OHLCVBar, signal *trading.Signal, portfolio *Portfolio, trades *[]TradeRecord) {
	rule := PriceLimitFor(bar.Symbol)
	if !rule.CanSell(bar.Close, e.prevClose[bar.Symbol]) {
		// 跌停封板，卖不出
		return
	}
	qty := signal.Quantity
	heldQty := portfolio.Positions[bar.Symbol]
	if qty > heldQty {
		qty = heldQty
	}

	effectivePrice := bar.Close * (1 - e.config.Slippage)
	revenue := effectivePrice*qty - stampDuty(effectivePrice*qty) - effectivePrice*qty*e.config.Commission

	order, err := e.oms.PlaceOrder(bar.Symbol, trading.SideSell, trading.TypeMarket, qty, 0)
	if err != nil {
		return
	}

	if _, err := e.oms.FillOrder(order.ID, qty, effectivePrice); err != nil {
		return
	}

	portfolio.Cash += revenue
	avgPrice := portfolio.AvgPrice[bar.Symbol]
	pnl := revenue - avgPrice*qty

	newQty := heldQty - qty
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
