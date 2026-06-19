package trading

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// OMS (Order Management System) manages orders, trades, and positions.
// It is safe for concurrent use.
type OMS struct {
	mu         sync.RWMutex
	orders     map[string]*Order
	trades     []*Trade
	positions  map[string]*Position
	broker     Broker
	orderCbs   []func(*Order) // notified on order state changes
	tradeCbs   []func(*Trade) // notified on new trades
}

// NewOMS creates a new Order Management System.
func NewOMS() *OMS {
	return &OMS{
		orders:    make(map[string]*Order),
		positions: make(map[string]*Position),
	}
}

// OnOrderUpdate registers a callback for order state changes.
// Callbacks are called synchronously under the OMS lock; keep them fast.
func (o *OMS) OnOrderUpdate(fn func(*Order)) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.orderCbs = append(o.orderCbs, fn)
}

// OnTradeUpdate registers a callback for new trades (fills).
func (o *OMS) OnTradeUpdate(fn func(*Trade)) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.tradeCbs = append(o.tradeCbs, fn)
}

// notifyOrder calls registered order callbacks (must be called under lock).
func (o *OMS) notifyOrder(order *Order) {
	for _, cb := range o.orderCbs {
		cb(order)
	}
}

// notifyTrade calls registered trade callbacks (must be called under lock).
func (o *OMS) notifyTrade(trade *Trade) {
	for _, cb := range o.tradeCbs {
		cb(trade)
	}
}

// PlaceOrder creates and registers a new order.
func (o *OMS) PlaceOrder(symbol string, side OrderSide, orderType OrderType, qty, price float64) (*Order, error) {
	if qty <= 0 {
		return nil, fmt.Errorf("quantity must be positive, got %f", qty)
	}
	if orderType == TypeLimit && price <= 0 {
		return nil, fmt.Errorf("limit order requires a positive price")
	}

	o.mu.Lock()
	defer o.mu.Unlock()

	order := &Order{
		ID:        uuid.New().String()[:8],
		Symbol:    symbol,
		Side:      side,
		OrderType: orderType,
		Quantity:  qty,
		Price:     price,
		Status:    StatusPending,
		PlacedAt:  time.Now(),
	}
	o.orders[order.ID] = order
	return order, nil
}

// CancelOrder cancels a pending order.
func (o *OMS) CancelOrder(orderID string) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	order, ok := o.orders[orderID]
	if !ok {
		return fmt.Errorf("order %q not found", orderID)
	}
	if order.Status != StatusPending && order.Status != StatusPartial {
		return fmt.Errorf("cannot cancel order in status %q", order.Status)
	}
	order.Status = StatusCancelled
	return nil
}

// FillOrder records a fill for an order. Returns error if the order is not fillable.
func (o *OMS) FillOrder(orderID string, fillQty, fillPrice float64) (*Trade, error) {
	o.mu.Lock()
	defer o.mu.Unlock()

	order, ok := o.orders[orderID]
	if !ok {
		return nil, fmt.Errorf("order %q not found", orderID)
	}
	if order.Status != StatusPending && order.Status != StatusPartial {
		return nil, fmt.Errorf("order %q in status %q cannot be filled", orderID, order.Status)
	}

	remainingQty := order.Quantity - order.FilledQty
	if fillQty > remainingQty {
		fillQty = remainingQty
	}

	// Update average fill price
	totalValue := order.FilledAvgPrice*order.FilledQty + fillPrice*fillQty
	order.FilledQty += fillQty
	order.FilledAvgPrice = totalValue / order.FilledQty

	if order.FilledQty >= order.Quantity {
		order.Status = StatusFilled
		now := time.Now()
		order.FilledAt = &now
	} else {
		order.Status = StatusPartial
	}

	trade := &Trade{
		ID:        uuid.New().String()[:8],
		OrderID:   orderID,
		Symbol:    order.Symbol,
		Side:      order.Side,
		Quantity:  fillQty,
		Price:     fillPrice,
		Timestamp: time.Now(),
	}
	o.trades = append(o.trades, trade)

	// Update position
	pos, ok := o.positions[order.Symbol]
	if !ok {
		pos = &Position{Symbol: order.Symbol}
		o.positions[order.Symbol] = pos
	}

	if order.Side == SideBuy {
		totalPosValue := pos.AvgPrice*pos.Quantity + fillPrice*fillQty
		pos.Quantity += fillQty
		if pos.Quantity > 0 {
			pos.AvgPrice = totalPosValue / pos.Quantity
		}
	} else {
		// Validate we have enough shares to sell (prevent negative positions).
		if fillQty > pos.Quantity {
			fillQty = pos.Quantity
		}
		if fillQty <= 0 {
			return nil, fmt.Errorf("fill %s: no position to sell for %s", order.ID, order.Symbol)
		}
		// Realize P&L for the sold portion.
		pos.PnL = (fillPrice - pos.AvgPrice) * fillQty
		if pos.AvgPrice > 0 {
			pos.PnLPct = (fillPrice - pos.AvgPrice) / pos.AvgPrice * 100
		}
		pos.Quantity -= fillQty
		// Reset AvgPrice when position is flat so stale prices don't affect future entries.
		if pos.Quantity == 0 {
			pos.AvgPrice = 0
		}
	}

	o.notifyTrade(trade)
	o.notifyOrder(order)
	return trade, nil
}

// GetOrder returns an order by ID.
func (o *OMS) GetOrder(orderID string) (*Order, bool) {
	o.mu.RLock()
	defer o.mu.RUnlock()
	order, ok := o.orders[orderID]
	return order, ok
}

// GetPosition returns the current position for a symbol.
func (o *OMS) GetPosition(symbol string) *Position {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.positions[symbol]
}

// GetAllPositions returns all current positions.
func (o *OMS) GetAllPositions() []*Position {
	o.mu.RLock()
	defer o.mu.RUnlock()
	result := make([]*Position, 0, len(o.positions))
	for _, p := range o.positions {
		result = append(result, p)
	}
	return result
}

// GetOrders returns all orders.
func (o *OMS) GetOrders() []*Order {
	o.mu.RLock()
	defer o.mu.RUnlock()
	result := make([]*Order, 0, len(o.orders))
	for _, order := range o.orders {
		result = append(result, order)
	}
	return result
}

// GetTrades returns all trades.
func (o *OMS) GetTrades() []*Trade {
	o.mu.RLock()
	defer o.mu.RUnlock()
	result := make([]*Trade, len(o.trades))
	copy(result, o.trades)
	return result
}

// UpdateMarketPrice updates the market price for a position and recalculates P&L.
func (o *OMS) UpdateMarketPrice(symbol string, marketPrice float64) {
	o.mu.Lock()
	defer o.mu.Unlock()

	pos, ok := o.positions[symbol]
	if !ok {
		return
	}
	pos.MarketPrice = marketPrice
	if pos.Quantity != 0 {
		pos.PnL = (marketPrice - pos.AvgPrice) * pos.Quantity
		if pos.AvgPrice > 0 {
			pos.PnLPct = (marketPrice - pos.AvgPrice) / pos.AvgPrice * 100
		}
	}
}

// SetBroker attaches a live broker to the OMS. Pass nil to return to paper mode.
func (o *OMS) SetBroker(b Broker) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.broker = b
}

// HasBroker reports whether a live broker is attached.
func (o *OMS) HasBroker() bool {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.broker != nil
}

// PlaceOrderLive places an order through the attached broker instead of paper.
func (o *OMS) PlaceOrderLive(ctx context.Context, symbol string, side OrderSide, orderType OrderType, qty, price, stopPrice float64) (*Order, error) {
	if qty <= 0 {
		return nil, fmt.Errorf("quantity must be positive, got %f", qty)
	}
	if orderType == TypeLimit && price <= 0 {
		return nil, fmt.Errorf("limit order requires a positive price")
	}

	o.mu.Lock()
	clientOrderID := uuid.New().String()
	order := &Order{
		ID:            clientOrderID[:8],
		ClientOrderID: clientOrderID,
		Symbol:        symbol,
		Side:          side,
		OrderType:     orderType,
		Quantity:      qty,
		Price:         price,
		StopPrice:     stopPrice,
		Status:        StatusPending,
		PlacedAt:      time.Now(),
	}
	o.orders[order.ID] = order
	br := o.broker
	o.mu.Unlock()

	result, err := br.SubmitOrder(ctx, order)
	if err != nil {
		o.mu.Lock()
		// Keep the order in the book with Rejected status so the user can see
		// what happened and retry. The clientOrderID serves as the idempotency
		// key — if the broker actually received the order but the response was
		// lost (timeout), the user can query/reconcile via the broker's API.
		order.Status = StatusRejected
		o.mu.Unlock()
		return order, fmt.Errorf("broker submit %s: %w", order.ClientOrderID, err)
	}

	o.mu.Lock()
	// Map local ID → broker ID for reconciliation. Keep the clientOrderID
	// so the user can trace the order across systems.
	delete(o.orders, order.ID)
	order.ID = result.BrokerOrderID
	o.orders[result.BrokerOrderID] = order
	order.Status = result.Status
	o.mu.Unlock()

	return order, nil
}
