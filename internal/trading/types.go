// Package trading provides the trading engine — bar-by-bar pipeline, OMS,
// paper trading, order matching, and risk management.
package trading

import "time"

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
}

// Trade is a filled execution record.
type Trade struct {
	ID        string    `json:"id"`
	OrderID   string    `json:"order_id"`
	Symbol    string    `json:"symbol"`
	Side      OrderSide `json:"side"`
	Quantity  float64   `json:"quantity"`
	Price     float64   `json:"price"`
	Timestamp time.Time `json:"timestamp"`
}

// Position represents a current holding.
type Position struct {
	Symbol      string  `json:"symbol"`
	Quantity    float64 `json:"quantity"`
	AvgPrice    float64 `json:"avg_price"`
	MarketPrice float64 `json:"market_price"`
	PnL         float64 `json:"pnl"`
	PnLPct      float64 `json:"pnl_pct"`
}

// OHLCVBar is a price bar used for order matching.
type OHLCVBar struct {
	Symbol string  `json:"symbol"`
	Date   string  `json:"date"`
	Open   float64 `json:"open"`
	High   float64 `json:"high"`
	Low    float64 `json:"low"`
	Close  float64 `json:"close"`
	Volume float64 `json:"volume"`
}
