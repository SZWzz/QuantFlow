// Package trading provides the trading engine — bar-by-bar pipeline, OMS,
// paper trading, order matching, risk management, and broker integration.
package trading

import "context"

// Broker abstracts a real brokerage connection. Implementations handle
// authentication, order submission, and position/account synchronization.
type Broker interface {
	Connect(ctx context.Context) error
	Disconnect(ctx context.Context) error
	IsConnected() bool
	Name() string

	SubmitOrder(ctx context.Context, order *Order) (*BrokerOrderResult, error)
	CancelOrder(ctx context.Context, orderID string) error
	ModifyOrder(ctx context.Context, orderID string, newPrice, newQty float64) error

	GetOrders(ctx context.Context) ([]*Order, error)
	GetPositions(ctx context.Context) ([]*Position, error)
	GetAccount(ctx context.Context) (*AccountInfo, error)

	OnOrderUpdate(func(order *Order))
	OnTradeUpdate(func(trade *Trade))
}

// AccountInfo holds broker account summary data.
type AccountInfo struct {
	BrokerName    string  `json:"broker_name"`
	TotalValue    float64 `json:"total_value"`
	CashBalance   float64 `json:"cash_balance"`
	MarginBalance float64 `json:"margin_balance"`
	BuyingPower   float64 `json:"buying_power"`
	Currency      string  `json:"currency"`
}

// BrokerOrderResult is returned by SubmitOrder with broker-specific details.
type BrokerOrderResult struct {
	BrokerOrderID string      `json:"broker_order_id"`
	Status        OrderStatus `json:"status"`
	Message       string      `json:"message"`
}
