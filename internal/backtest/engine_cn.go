package backtest

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"quantflow/internal/trading"
	"sort"
)

// ── Slippage Models ──────────────────────────────────────────────────────────

// SlippageModel applies execution friction to trade fills.
type SlippageModel interface {
	Apply(order trading.Order, bar trading.OHLCVBar) float64
}

// FixedSlippage applies a constant bps-based slippage.
// E.g. Bps=0.001 represents 0.1% of the bar's mid/close price.
type FixedSlippage struct{ Bps float64 }

func (s *FixedSlippage) Apply(order trading.Order, bar trading.OHLCVBar) float64 {
	if s.Bps <= 0 {
		return 0
	}
	return bar.Close * s.Bps
}

// QuadraticSlippage models price impact proportional to (orderQty / volume)².
// Formula: Base * (1 + impact²), where impact = VolRatio * qty / volume.
// Larger orders in thin markets incur disproportionately higher costs.
type QuadraticSlippage struct {
	Base     float64
	VolRatio float64
}

func (s *QuadraticSlippage) Apply(order trading.Order, bar trading.OHLCVBar) float64 {
	if bar.Volume <= 0 {
		return s.Base
	}
	impact := s.VolRatio * float64(order.Quantity) / bar.Volume
	// Quadratic impact: larger orders in thin markets cost more
	if impact > 0 {
		return s.Base * (1 + impact*impact)
	}
	return s.Base
}

// CNEngine is the A-share backtesting engine with market-specific rules:
//   - T+1 settlement: shares bought today cannot be sold until tomorrow
//   - Price limits: ±10% (main board) or ±20% (ChiNext/STAR)
//   - Stamp duty: 0.05% on sell only (2024新政)
//   - Minimum lot: 100 shares, multiples of 100
//   - Commission: 0.03% (万三) default
type CNEngine struct {
	*Runner
	prevClose     map[string]float64 // symbol → previous trading day close (for price limit)
	slippageModel SlippageModel      // execution slippage model
	stampDutyRate float64            // 印花税率，默认 0.0005 (万分之5，卖出)
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
		Runner:        NewRunner(config),
		prevClose:     make(map[string]float64),
		slippageModel: &FixedSlippage{Bps: 0.001},
		stampDutyRate: 0.0005,
	}
}

// stampDuty returns the stamp duty for a sell trade, rounded to 0.01 CNY (fen).
func (e *CNEngine) stampDuty(tradeValue float64) float64 {
	return math.Round(tradeValue*e.stampDutyRate*100) / 100
}

// Run executes the backtest with A-share market rules.
func (e *CNEngine) Run(ctx context.Context, strategy Strategy, bars []trading.OHLCVBar) (*Result, error) {
	if len(bars) == 0 {
		return nil, errNoData
	}

	sort.Slice(bars, func(i, j int) bool { return bars[i].Date < bars[j].Date })

	portfolio := NewPortfolio(e.config.InitialCash)
	if err := e.oms.GetCashLedger().Deposit(e.config.InitialCash); err != nil {
		return nil, fmt.Errorf("deposit initial cash: %w", err)
	}
	var equityCurve []EquityPoint
	var tradeRecords []TradeRecord
	latestPrices := make(map[string]float64)
	var lastDate string
	var prevBar *trading.OHLCVBar

	for _, bar := range bars {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		e.oms.UpdateMarketPrice(bar.Symbol, bar.Close)
		latestPrices[bar.Symbol] = bar.Close

		// Available sellable quantity = held - T+1 locked
		heldQty := portfolio.Positions[bar.Symbol]
		availableQty := e.oms.GetT1Available(bar.Symbol, heldQty)

		// 1. Check stop-loss/take-profit (only on available shares)
		// P0: Fill at bar.Close — stop was triggered at close, so open has already passed.
		if pos := e.oms.GetPosition(bar.Symbol); pos != nil && availableQty > 0 {
			if e.risk.CheckStopLoss(pos, bar.Close) || e.risk.CheckTakeProfit(pos, bar.Close) {
				rule := PriceLimitFor(bar.Symbol)
				if !rule.CanSell(bar.Close, e.prevClose[bar.Symbol]) {
					goto recordEquityCN
				}
				avgPrice := pos.AvgPrice // capture before FillOrder clears it
				order, err := e.oms.PlaceOrder(bar.Symbol, trading.SideSell, trading.TypeMarket, "", availableQty, 0)
				if err == nil {
					// Only record the trade when the fill actually succeeds —
					// recording a rejected fill produces phantom P&L.
					if _, fillErr := e.oms.FillOrder(order.ID, availableQty, bar.Close); fillErr == nil {
						portfolio.Cash = e.oms.GetCashBalance()
						revenue := bar.Close*availableQty - e.stampDuty(bar.Close*availableQty) - bar.Close*availableQty*e.config.Commission
						pnl := revenue - avgPrice*availableQty

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
				}
				goto recordEquityCN
			}
		}

		// 2. Generate signals
		if strategy.SignalFunc != nil {
			signal := strategy.SignalFunc(bar.Open, prevBar, portfolio)
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
		e.prevClose[bar.Symbol] = bar.Close

		if lastDate != "" && bar.Date != lastDate {
			e.oms.ClearT1Lock()
		}
		lastDate = bar.Date

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

func (e *CNEngine) processCNBuySignal(bar trading.OHLCVBar, signal *trading.Signal, portfolio *Portfolio, trades *[]TradeRecord) {
	qty := signal.Quantity
	if qty <= 0 {
		qty = 100
	}
	qty = float64(int(qty/100)) * 100
	if qty <= 0 {
		return
	}

	slippage := e.config.Slippage
	if e.slippageModel != nil {
		mockOrder := trading.Order{Symbol: bar.Symbol, Side: trading.SideBuy, Quantity: qty}
		slippage = e.slippageModel.Apply(mockOrder, bar) / bar.Close
	}
	effectivePrice := bar.Open * (1 + slippage)

	// P1: Limit check with effectivePrice, not bar.Close
	rule := PriceLimitFor(bar.Symbol)
	if !rule.CanBuy(effectivePrice, e.prevClose[bar.Symbol]) {
		return
	}

	cost := effectivePrice*qty + effectivePrice*qty*e.config.Commission
	if cost > portfolio.Cash {
		slog.Debug("buy signal skipped: insufficient cash", "symbol", bar.Symbol, "need", cost, "have", portfolio.Cash)
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
		slog.Debug("buy signal risk rejected", "symbol", bar.Symbol, "error", err)
		return
	}

	order, err := e.oms.PlaceOrder(bar.Symbol, trading.SideBuy, trading.TypeMarket, "", qty, 0)
	if err != nil {
		slog.Debug("buy signal order failed", "symbol", bar.Symbol, "error", err)
		return
	}

	if _, err := e.oms.FillOrder(order.ID, qty, effectivePrice); err != nil {
		slog.Debug("buy signal fill failed", "symbol", bar.Symbol, "error", err)
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

	// Compute slippage via model if available, fall back to config.Slippage
	slippage := e.config.Slippage
	if e.slippageModel != nil {
		mockOrder := trading.Order{Symbol: bar.Symbol, Side: trading.SideSell, Quantity: qty}
		slippage = e.slippageModel.Apply(mockOrder, bar) / bar.Close
	}
	effectivePrice := bar.Open * (1 - slippage)
	revenue := effectivePrice*qty - e.stampDuty(effectivePrice*qty) - effectivePrice*qty*e.config.Commission

	order, err := e.oms.PlaceOrder(bar.Symbol, trading.SideSell, trading.TypeMarket, "", qty, 0)
	if err != nil {
		slog.Debug("sell signal order failed", "symbol", bar.Symbol, "error", err)
		return
	}

	if _, err := e.oms.FillOrder(order.ID, qty, effectivePrice); err != nil {
		slog.Debug("sell signal fill failed", "symbol", bar.Symbol, "error", err)
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
