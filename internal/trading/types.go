// Package trading provides the trading engine — bar-by-bar pipeline, OMS,
// paper trading, order matching, and risk management.
package trading

import (
	"time"

	"quantflow/internal/normalize"
)

// OrderSide represents the direction of an order.
type OrderSide string

const (
	SideBuy  OrderSide = "buy"
	SideSell OrderSide = "sell"
)

// OrderType represents the type of an order.
type OrderType string

const (
	TypeMarket OrderType = "market"
	TypeLimit  OrderType = "limit"
	TypeStop   OrderType = "stop"
)

// OrderStatus represents the lifecycle state of an order.
type OrderStatus string

const (
	StatusPending   OrderStatus = "pending"
	StatusPartial   OrderStatus = "partial"
	StatusFilled    OrderStatus = "filled"
	StatusCancelled OrderStatus = "cancelled"
	StatusRejected  OrderStatus = "rejected"
)

// Signal is a trading signal produced by strategy nodes.
type Signal struct {
	Symbol    string    `json:"symbol"`
	Direction string    `json:"direction"` // "buy" | "sell" | "hold"
	Quantity  float64   `json:"quantity"`
	Price     float64   `json:"price"` // 0 = market
	Reason    string    `json:"reason"`
	Timestamp time.Time `json:"timestamp"`
}

// TradingMode represents the current trading environment.
type TradingMode string

const (
	TradingModeInvalid TradingMode = ""
	TradingModePaper   TradingMode = "paper"
	TradingModeLive    TradingMode = "live"
)

// Valid returns true if the mode is a recognized value.
func (m TradingMode) Valid() bool {
	return m == TradingModePaper || m == TradingModeLive
}

// IsLive returns true if the mode is live trading.
func (m TradingMode) IsLive() bool { return m == TradingModeLive }

// SafetyCheck represents a single pre-flight check before going live.
type SafetyCheck struct {
	Name     string `json:"name"`
	OK       bool   `json:"ok"`
	Message  string `json:"message"`
	Blocking bool   `json:"blocking"` // failing blocking checks prevent mode switch
}

// SafetyReport is the result of pre-flight safety checks.
type SafetyReport struct {
	Checks   []SafetyCheck `json:"checks"`
	AllClear bool          `json:"all_clear"`
}

// Passed returns true if all blocking checks are OK.
func (r SafetyReport) Passed() bool {
	if len(r.Checks) == 0 {
		return false
	}
	for _, c := range r.Checks {
		if c.Blocking && !c.OK {
			return false
		}
	}
	return true
}

// Order represents an order in the OMS.
type Order struct {
	ID             string      `json:"id"`
	ClientOrderID  string      `json:"client_order_id"` // idempotency key for broker submission
	Symbol         string      `json:"symbol"`
	Side           OrderSide   `json:"side"`
	OrderType      OrderType   `json:"order_type"`
	Quantity       float64     `json:"quantity"`
	Price          float64     `json:"price"`          // 0 for market orders
	StopPrice      float64     `json:"stop_price"`     // for stop orders
	FilledQty      float64     `json:"filled_qty"`
	FilledAvgPrice float64     `json:"filled_avg_price"`
	Status         OrderStatus `json:"status"`
	PlacedAt       time.Time   `json:"placed_at"`
	FilledAt       *time.Time  `json:"filled_at,omitempty"`
	Name           string      `json:"name"`
}

// Trade is a filled execution record.
type Trade struct {
	ID         string    `json:"id"`
	OrderID    string    `json:"order_id"`
	Symbol     string    `json:"symbol"`
	Side       OrderSide `json:"side"`
	Quantity   float64   `json:"quantity"`
	Price      float64   `json:"price"`
	Commission float64   `json:"commission"`  // 佣金
	StampTax   float64   `json:"stamp_tax"`   // 印花税 (仅卖出)
	Timestamp  time.Time `json:"timestamp"`
	Name       string    `json:"name"` // 股票名称
}

// Position represents a current holding.
type Position struct {
	Symbol      string  `json:"symbol"`
	Quantity    float64 `json:"quantity"`
	AvgPrice    float64 `json:"avg_price"`
	MarketPrice float64 `json:"market_price"`
	PnL         float64 `json:"pnl"`          // Total P&L = RealizedPnl + unrealized
	PnLPct      float64 `json:"pnl_pct"`
	RealizedPnl float64 `json:"realized_pnl"` // Accumulated realized gains from closes
	Name        string  `json:"name"`
}

// OHLCVBar is an alias for normalize.OHLCVBar — the canonical OHLCV type.
type OHLCVBar = normalize.OHLCVBar
