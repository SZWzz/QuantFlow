// Package backtest provides a multi-market historical backtesting engine.
// It reuses the trading package's OMS, OrderMatcher, and RiskPipeline
// for bar-by-bar simulation of historical data.
package backtest

import (
	"errors"
	"time"

	"quantflow/internal/trading"
)

// errNoData is returned when no OHLCV data is provided for backtesting.
var errNoData = errors.New("no OHLCV data provided")

// Config holds the parameters for a backtest run.
type Config struct {
	StartDate   time.Time `json:"start_date"`
	EndDate     time.Time `json:"end_date"`
	InitialCash float64   `json:"initial_cash"`
	Commission   float64   `json:"commission"`    // per-trade commission rate
	Slippage     float64   `json:"slippage"`      // slippage as fraction of price
	Benchmark    string    `json:"benchmark"`     // benchmark symbol for comparison
	RiskFreeRate float64   `json:"risk_free_rate"` // annual risk-free rate (default 0.02 = 2%)
}

// DefaultConfig returns sensible defaults for backtesting.
func DefaultConfig() Config {
	return Config{
		InitialCash: 1_000_000,
		Commission:  0.0003, // 万三
		Slippage:    0.001,  // 10 bps
		RiskFreeRate: 0.02,   // 2% annual risk-free rate
	}
}

// Portfolio tracks cash and positions during a backtest.
type Portfolio struct {
	Cash      float64
	Positions map[string]float64 // symbol → quantity
	AvgPrice  map[string]float64 // symbol → avg entry price
}

// NewPortfolio creates a portfolio with the given initial cash.
func NewPortfolio(initialCash float64) *Portfolio {
	return &Portfolio{
		Cash:      initialCash,
		Positions: make(map[string]float64),
		AvgPrice:  make(map[string]float64),
	}
}

// Equity returns total portfolio value at current market prices.
func (p *Portfolio) Equity(prices map[string]float64) float64 {
	value := p.Cash
	for sym, qty := range p.Positions {
		if price, ok := prices[sym]; ok {
			value += qty * price
		}
	}
	return value
}

// EquityPoint is a single point on the equity curve.
type EquityPoint struct {
	Date   string  `json:"date"`
	Equity float64 `json:"equity"`
	Cash   float64 `json:"cash"`
}

// TradeRecord is a simplified trade for backtest reporting.
type TradeRecord struct {
	Date     string  `json:"date"`
	Symbol   string  `json:"symbol"`
	Side     string  `json:"side"`
	Quantity float64 `json:"quantity"`
	Price    float64 `json:"price"`
	PnL      float64 `json:"pnl,omitempty"`
}

// Result contains the complete output of a backtest run.
type Result struct {
	Config      Config         `json:"config"`
	EquityCurve []EquityPoint  `json:"equity_curve"`
	Trades      []TradeRecord  `json:"trades"`
	Metrics     Metrics        `json:"metrics"`
}

// ToRiskConfig converts backtest Config to trading.RiskConfig.
func (c Config) ToRiskConfig() trading.RiskConfig {
	return trading.RiskConfig{
		MaxPositionPct: 0.25,
		StopLossPct:    0.05,
		TakeProfitPct:  0.15,
		MaxDrawdownPct: 0.20,
	}
}

// TradingDaysInRange returns estimated trading days between two dates.
func TradingDaysInRange(start, end time.Time) int {
	days := 0
	current := start
	for !current.After(end) {
		weekday := current.Weekday()
		if weekday != time.Saturday && weekday != time.Sunday {
			days++
		}
		current = current.AddDate(0, 0, 1)
	}
	return days
}
