package backtest

import (
	"context"
	"sort"

	"quantflow/internal/trading"
)

// HKEngine is the Hong Kong stock backtesting engine.
// HK market rules:
//   - Stamp duty: 0.13% total (0.1% stamp + 0.0027% SFC + 0.00015% FRC), charged on both buy & sell
//   - Exchange trading fee: 0.00565%
//   - SFC transaction levy: 0.00278%
//   - T+2 settlement (tagged but does not affect bar-by-bar simulation)
//   - Lot size: minimum trading unit, varies by stock (default 100 shares)
//   - No price limits
type HKEngine struct {
	*Runner
	stampDutyRate float64 // 印花税 + SFC交易征费 + 财汇局交易征费 = ~0.13%
	tradingFee    float64 // 联交所交易费 + 证监会交易征费
	lotSize       int     // 每手股数, varies by stock
}

// NewHKEngine creates a HK stock backtesting engine with default HK config.
func NewHKEngine(config Config) *HKEngine {
	if config.Commission == 0 {
		config.Commission = 0.0003 // 0.03% typical HK brokerage
	}
	return &HKEngine{
		Runner:        NewRunner(config),
		stampDutyRate: 0.0013,                  // 0.1% stamp + 0.0027% SFC + 0.00015% FRC = ~0.13%
		tradingFee:    0.0000565 + 0.0000278,   // Exchange + SFC levy
		lotSize:       100,                      // Default lot size
	}
}

// Run executes the backtest with HK market rules.
func (e *HKEngine) Run(ctx context.Context, strategy Strategy, bars []trading.OHLCVBar) (*Result, error) {
	if len(bars) == 0 {
		return nil, errNoData
	}

	sort.Slice(bars, func(i, j int) bool { return bars[i].Date < bars[j].Date })

	portfolio := NewPortfolio(e.config.InitialCash)
	e.oms.GetCashLedger().Deposit(e.config.InitialCash)
	var equityCurve []EquityPoint
	var tradeRecords []TradeRecord
	latestPrices := make(map[string]float64)

	// T+2 settlement tracker (informational, does not block trades in bar-by-bar simulation)
	// settlementQueue maps settlement date → unlocked quantity
	_ = make(map[string]float64) // reserved for future T+2 settlement enforcement

	for _, bar := range bars {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		e.oms.UpdateMarketPrice(bar.Symbol, bar.Close)
		latestPrices[bar.Symbol] = bar.Close

		// 1. Check stop-loss/take-profit on existing positions
		if pos := e.oms.GetPosition(bar.Symbol); pos != nil && pos.Quantity > 0 {
			if e.risk.CheckStopLoss(pos, bar.Close) || e.risk.CheckTakeProfit(pos, bar.Close) {
				order, err := e.oms.PlaceOrder(bar.Symbol, trading.SideSell, trading.TypeMarket, pos.Quantity, 0)
				if err == nil {
					e.oms.FillOrder(order.ID, pos.Quantity, bar.Open)
					revenue := bar.Open*pos.Quantity - e.stampDuty(bar.Open*pos.Quantity) - e.tradeFee(bar.Open*pos.Quantity) - bar.Open*pos.Quantity*e.config.Commission
					pnl := revenue - pos.AvgPrice*pos.Quantity
					tradeRecords = append(tradeRecords, TradeRecord{
						Date: bar.Date, Symbol: bar.Symbol, Side: "sell",
						Quantity: pos.Quantity, Price: bar.Open, PnL: pnl,
					})
				}
				portfolio.Cash = e.oms.GetCashBalance()
				delete(portfolio.Positions, bar.Symbol)
				delete(portfolio.AvgPrice, bar.Symbol)
				goto recordEquityHK
			}
		}

		// 2. Generate signals
		if strategy.SignalFunc != nil {
			signal := strategy.SignalFunc(bar, portfolio)
			if signal != nil && signal.Direction != "hold" {
				if signal.Direction == "buy" {
					e.processHKBuySignal(bar, signal, portfolio, &tradeRecords)
				} else if signal.Direction == "sell" {
					e.processHKSellSignal(bar, signal, portfolio, &tradeRecords)
				}
			}
		}

	recordEquityHK:
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

// stampDuty returns the total HK stamp duty (0.13%) for a trade.
// Charged on both buy and sell in HK market.
func (e *HKEngine) stampDuty(tradeValue float64) float64 {
	return tradeValue * e.stampDutyRate
}

// tradeFee returns the Exchange + SFC fees for a trade.
func (e *HKEngine) tradeFee(tradeValue float64) float64 {
	return tradeValue * e.tradingFee
}

func (e *HKEngine) processHKBuySignal(bar trading.OHLCVBar, signal *trading.Signal, portfolio *Portfolio, trades *[]TradeRecord) {
	qty := signal.Quantity
	if qty <= 0 {
		qty = float64(e.lotSize)
	}
	// Round to lot size
	qty = float64(int(qty/float64(e.lotSize))) * float64(e.lotSize)
	if qty <= 0 {
		return
	}

	effectivePrice := bar.Open * (1 + e.config.Slippage)
	// Buy cost includes commission + stamp duty + trading fee
	cost := effectivePrice*qty + effectivePrice*qty*e.config.Commission + e.stampDuty(effectivePrice*qty) + e.tradeFee(effectivePrice*qty)

	if cost > portfolio.Cash {
		return
	}

	order, err := e.oms.PlaceOrder(bar.Symbol, trading.SideBuy, trading.TypeMarket, qty, 0)
	if err != nil {
		return
	}

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

	portfolio.Cash = e.oms.GetCashBalance()
	oldQty := portfolio.Positions[bar.Symbol]
	oldAvg := portfolio.AvgPrice[bar.Symbol]
	newQty := oldQty + qty
	portfolio.Positions[bar.Symbol] = newQty
	if newQty > 0 {
		portfolio.AvgPrice[bar.Symbol] = (oldQty*oldAvg + qty*effectivePrice) / newQty
	}

	*trades = append(*trades, TradeRecord{
		Date: bar.Date, Symbol: bar.Symbol, Side: "buy",
		Quantity: qty, Price: effectivePrice,
	})
}

func (e *HKEngine) processHKSellSignal(bar trading.OHLCVBar, signal *trading.Signal, portfolio *Portfolio, trades *[]TradeRecord) {
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
	// Sell: revenue after commission + stamp duty + trading fee
	revenue := effectivePrice*qty - effectivePrice*qty*e.config.Commission - e.stampDuty(effectivePrice*qty) - e.tradeFee(effectivePrice*qty)

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

	*trades = append(*trades, TradeRecord{
		Date: bar.Date, Symbol: bar.Symbol, Side: "sell",
		Quantity: qty, Price: effectivePrice, PnL: pnl,
	})
}
