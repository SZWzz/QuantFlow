package trading

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// TradingCosts holds fee configuration for trade execution.
type TradingCosts struct {
	CommissionRate float64 // 佣金率，默认 0.00025 (万分之2.5)
	StampTaxRate   float64 // 印花税率，卖出 0.0005 (万分之5)，买入 0
	MinCommission  float64 // 最低佣金，默认 5 元
}

// PriceLimitConfig defines daily price limits for a symbol.
type PriceLimitConfig struct {
	PrevClose float64 // 昨日收盘价
	MaxPct    float64 // 0.10 for main board, 0.20 for STAR/ChiNext
}

// OMS (Order Management System) manages orders, trades, and positions.
// It is safe for concurrent use.
type OMS struct {
	mu        sync.RWMutex
	orders    map[string]*Order
	trades    []*Trade
	positions map[string]*Position
	broker    Broker
	orderCbs  []func(*Order) // notified on order state changes
	tradeCbs  []func(*Trade) // notified on new trades

	// P0-10: T+1 lock for A-share (today's buys can't be sold same day)
	t1Lock *T1Tracker

	// P0-2: Price limits (涨跌停)
	priceLimits map[string]PriceLimitConfig

	// P0-3: Trading costs (佣金 + 印花税)
	costConfig TradingCosts

	// P0-4: Cash ledger
	cashLedger *CashLedger

	// quoteCache maps symbol → stock name, populated by adapter quotes.
	quoteCache map[string]string
}

// NewOMS creates a new Order Management System.
func NewOMS() *OMS {
	return &OMS{
		orders:      make(map[string]*Order),
		positions:   make(map[string]*Position),
		t1Lock:      NewT1Tracker(),
		priceLimits: make(map[string]PriceLimitConfig),
		costConfig: TradingCosts{
			CommissionRate: 0.00025, // 万分之2.5
			StampTaxRate:   0.0005,  // 万分之5
			MinCommission:  5.0,     // 最低 5 元
		},
		cashLedger: NewCashLedger(),
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

// PlaceOrder creates and registers a new order. If brokerName is non-empty and
// not "paper", routes the order through the attached live broker.
func (o *OMS) PlaceOrder(symbol string, side OrderSide, orderType OrderType, brokerName string, qty, price float64) (*Order, error) {
	if qty <= 0 {
		return nil, fmt.Errorf("quantity must be positive, got %f", qty)
	}
	if orderType == TypeLimit && price <= 0 {
		return nil, fmt.Errorf("limit order requires a positive price")
	}

	// Route to live broker if one is attached and brokerName is specified.
	if brokerName != "" && brokerName != "paper" {
		o.mu.RLock()
		br := o.broker
		o.mu.RUnlock()
		if br == nil {
			return nil, fmt.Errorf("broker %q not attached", brokerName)
		}
		if br.Name() != brokerName {
			return nil, fmt.Errorf("broker %q not attached (active: %s)", brokerName, br.Name())
		}
		clientOrderID := uuid.New().String()
		order := &Order{
			ID:            clientOrderID[:12],
			ClientOrderID: clientOrderID,
			Symbol:        symbol,
			Side:          side,
			OrderType:     orderType,
			Quantity:      qty,
			Price:         price,
			Status:        StatusPending,
			PlacedAt:      time.Now(),
		}
		ctx := context.Background()
		result, err := br.SubmitOrder(ctx, order)
		if err != nil {
			order.Status = StatusRejected
			o.mu.Lock()
			o.orders[order.ID] = order
			o.mu.Unlock()
			return order, fmt.Errorf("broker submit: %w", err)
		}
		order.ID = result.BrokerOrderID
		order.Status = result.Status
		o.mu.Lock()
		o.orders[order.ID] = order
		o.mu.Unlock()
		return order, nil
	}

	// Paper trading path (existing logic).
	o.mu.Lock()
	defer o.mu.Unlock()

	order := &Order{
		ID:        uuid.New().String()[:12],
		Symbol:    symbol,
		Side:      side,
		OrderType: orderType,
		Quantity:  qty,
		Price:     price,
		Status:    StatusPending,
		PlacedAt:  time.Now(),
	}
	order.Name = o.getName(symbol)
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

	// P0-1 fix: for sell orders, clip fillQty to available position BEFORE updating
	// the order book. Previously the order book (FilledQty/FilledAvgPrice) was updated
	// with the unclipped fillQty, then fillQty was clipped inside the position block,
	// causing order.FilledQty > actual position change and P&L ledger mismatch.
	pos, ok := o.positions[order.Symbol]
	if order.Side == SideSell {
		if !ok || pos.Quantity <= 0 {
			return nil, fmt.Errorf("fill %s: no position to sell for %s", order.ID, order.Symbol)
		}
		if fillQty > pos.Quantity {
			fillQty = pos.Quantity
		}
		if fillQty <= 0 {
			return nil, fmt.Errorf("fill %s: no position to sell for %s", order.ID, order.Symbol)
		}

		// P0-10: T+1 lock — clip sell quantity to available (total - today's buys)
		avail := o.t1Lock.Available(order.Symbol, pos.Quantity)
		if avail <= 0 {
			return nil, fmt.Errorf("T+1 lock: cannot sell %s, all shares locked", order.Symbol)
		}
		if fillQty > avail {
			fillQty = avail
		}
	}
	if !ok {
		pos = &Position{Symbol: order.Symbol}
		o.positions[order.Symbol] = pos
	}

	// P0-2: Price limit validation (涨跌停)
	if err := o.CheckPriceLimit(order.Symbol, fillPrice); err != nil {
		return nil, err
	}

	// P0-3: Compute trading costs (佣金 + 印花税)
	tradeAmount := fillPrice * fillQty
	commission := tradeAmount * o.costConfig.CommissionRate
	if commission < o.costConfig.MinCommission {
		commission = o.costConfig.MinCommission
	}
	var stampTax float64
	if order.Side == SideSell {
		stampTax = tradeAmount * o.costConfig.StampTaxRate
	}

	// Update average fill price (fillQty is now the final, clipped value)
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
		ID:         uuid.New().String()[:12],
		OrderID:    orderID,
		Symbol:     order.Symbol,
		Side:       order.Side,
		Quantity:   fillQty,
		Price:      fillPrice,
		Commission: commission,
		StampTax:   stampTax,
		Timestamp:  time.Now(),
	}
	trade.Name = o.getName(order.Symbol)
	o.trades = append(o.trades, trade)

	// Update position (fillQty already clipped for sells above)
	if order.Side == SideBuy {
		totalPosValue := pos.AvgPrice*pos.Quantity + fillPrice*fillQty
		pos.Quantity += fillQty
		if pos.Quantity > 0 {
			pos.AvgPrice = totalPosValue / pos.Quantity
		}
		// P0-10: Lock T+1 shares from today's buy
		o.t1Lock.Lock(order.Symbol, fillQty)
	} else {
		// Realize P&L for the sold portion, deducting trading costs.
		realizedPnl := (fillPrice-pos.AvgPrice)*fillQty - commission - stampTax
		pos.PnL = realizedPnl
		pos.RealizedPnl += realizedPnl
		if pos.AvgPrice > 0 {
			pos.PnLPct = (fillPrice - pos.AvgPrice) / pos.AvgPrice * 100
		}
		pos.Quantity -= fillQty
		// Reset AvgPrice when position is flat so stale prices don't affect future entries.
		if pos.Quantity == 0 {
			pos.AvgPrice = 0
		}
	}

	// P0-4: Record transaction in cash ledger
	o.cashLedger.RecordTrade(order.Side, order.ID, tradeAmount, commission, stampTax)

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
	pos := o.positions[symbol]
	if pos != nil {
		pos.Name = o.getName(pos.Symbol)
	}
	return pos
}

// GetAllPositions returns all current positions.
func (o *OMS) GetAllPositions() []*Position {
	o.mu.RLock()
	defer o.mu.RUnlock()
	result := make([]*Position, 0, len(o.positions))
	for _, p := range o.positions {
		p.Name = o.getName(p.Symbol)
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
		pos.PnL = pos.RealizedPnl + (marketPrice-pos.AvgPrice)*pos.Quantity
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

// SetPriceLimit configures the daily price limit for a symbol.
// prevClose is yesterday's close, maxPct is the daily limit (e.g. 0.10 for main board).
func (o *OMS) SetPriceLimit(symbol string, prevClose, maxPct float64) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.priceLimits[symbol] = PriceLimitConfig{
		PrevClose: prevClose,
		MaxPct:    maxPct,
	}
}

// CheckPriceLimit validates whether fillPrice is within the symbol's price limits.
// Returns an error if price limits are configured and the price is out of bounds.
func (o *OMS) CheckPriceLimit(symbol string, fillPrice float64) error {
	cfg, ok := o.priceLimits[symbol]
	if !ok {
		return nil // no limit configured
	}
	low := cfg.PrevClose * (1 - cfg.MaxPct)
	high := cfg.PrevClose * (1 + cfg.MaxPct)
	if fillPrice < low || fillPrice > high {
		return fmt.Errorf("price limit: %s fillPrice %.2f outside [%.2f, %.2f]", symbol, fillPrice, low, high)
	}
	return nil
}

// GetCashBalance returns the current cash balance from the cash ledger.
func (o *OMS) GetCashBalance() float64 {
	return o.cashLedger.GetBalance()
}

// GetCashLedger returns the underlying CashLedger for external access.
func (o *OMS) GetCashLedger() *CashLedger {
	return o.cashLedger
}

// SetQuoteName stores the stock name for a symbol.
func (o *OMS) SetQuoteName(symbol, name string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.quoteCache == nil {
		o.quoteCache = make(map[string]string)
	}
	o.quoteCache[symbol] = name
}

// getName returns the cached stock name for a symbol. Returns empty if not cached.
// NOTE: Caller MUST hold o.mu (read or write lock). This method does NOT acquire
// the lock itself — doing so would deadlock when called from PlaceOrder/FillOrder
// which already hold the write lock (Go's sync.RWMutex is not reentrant).
func (o *OMS) getName(symbol string) string {
	return o.quoteCache[symbol]
}

// ResolveNames populates Name fields for positions that don't have it yet.
// Call this after quote data is available (e.g., via GetQuote).
func (o *OMS) ResolveNames() {
	o.mu.Lock()
	defer o.mu.Unlock()
	for _, p := range o.positions {
		if p.Name == "" {
			if name, ok := o.quoteCache[p.Symbol]; ok && name != "" {
				p.Name = name
			}
		}
	}
	for _, order := range o.orders {
		if order.Name == "" {
			if name, ok := o.quoteCache[order.Symbol]; ok && name != "" {
				order.Name = name
			}
		}
	}
	for _, trade := range o.trades {
		if trade.Name == "" {
			if name, ok := o.quoteCache[trade.Symbol]; ok && name != "" {
				trade.Name = name
			}
		}
	}
}

// ClearT1Lock clears all T+1 locks (called at end of trading day).
func (o *OMS) ClearT1Lock() {
	o.t1Lock.Clear()
}

// GetT1Available returns the number of shares available to sell (total - T+1 locked).
func (o *OMS) GetT1Available(symbol string, totalQty float64) float64 {
	return o.t1Lock.Available(symbol, totalQty)
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
		ID:            clientOrderID[:12],
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
