// Package brokers provides real broker adapter implementations.
package brokers

import (
	"context"
	"fmt"
	"sync"

	"quantflow/internal/trading"
)

// FutuConfig holds connection parameters for FutuOpenD.
type FutuConfig struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

// FutuBroker implements trading.Broker for Futu (FutuOpenD).
// This is a stub — actual FutuOpenD protocol implementation is pending.
type FutuBroker struct {
	cfg       FutuConfig
	connected bool
	mu        sync.RWMutex
	orderCbs  []func(*trading.Order)
	tradeCbs  []func(*trading.Trade)
}

// NewFutuBroker creates a new Futu broker with sensible defaults.
func NewFutuBroker(cfg FutuConfig) *FutuBroker {
	if cfg.Host == "" {
		cfg.Host = "localhost"
	}
	if cfg.Port == 0 {
		cfg.Port = 11111
	}
	return &FutuBroker{cfg: cfg}
}

// Name returns the broker identifier.
func (f *FutuBroker) Name() string { return "futu" }

// IsConnected returns whether the broker is currently connected.
func (f *FutuBroker) IsConnected() bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.connected
}

// Connect is a stub that returns an error — actual FutuOpenD connection is not yet implemented.
func (f *FutuBroker) Connect(ctx context.Context) error {
	return fmt.Errorf("futu broker: FutuOpenD connection not yet implemented — ensure FutuOpenD is running at %s:%d", f.cfg.Host, f.cfg.Port)
}

// Disconnect marks the broker as disconnected.
func (f *FutuBroker) Disconnect(ctx context.Context) error {
	f.connected = false
	return nil
}

// SubmitOrder is a stub — not yet implemented.
func (f *FutuBroker) SubmitOrder(ctx context.Context, order *trading.Order) (*trading.BrokerOrderResult, error) {
	return nil, fmt.Errorf("futu broker: not yet implemented")
}

// CancelOrder is a stub — not yet implemented.
func (f *FutuBroker) CancelOrder(ctx context.Context, orderID string) error {
	return fmt.Errorf("futu broker: not yet implemented")
}

// ModifyOrder is a stub — not yet implemented.
func (f *FutuBroker) ModifyOrder(ctx context.Context, orderID string, newPrice, newQty float64) error {
	return fmt.Errorf("futu broker: not yet implemented")
}

// GetOrders is a stub — not yet implemented.
func (f *FutuBroker) GetOrders(ctx context.Context) ([]*trading.Order, error) {
	return nil, fmt.Errorf("futu broker: not yet implemented")
}

// GetPositions is a stub — not yet implemented.
func (f *FutuBroker) GetPositions(ctx context.Context) ([]*trading.Position, error) {
	return nil, fmt.Errorf("futu broker: not yet implemented")
}

// GetAccount is a stub — not yet implemented.
func (f *FutuBroker) GetAccount(ctx context.Context) (*trading.AccountInfo, error) {
	return nil, fmt.Errorf("futu broker: not yet implemented")
}

// OnOrderUpdate registers a callback for order state changes.
func (f *FutuBroker) OnOrderUpdate(fn func(*trading.Order)) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.orderCbs = append(f.orderCbs, fn)
}

// OnTradeUpdate registers a callback for trade executions.
func (f *FutuBroker) OnTradeUpdate(fn func(*trading.Trade)) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.tradeCbs = append(f.tradeCbs, fn)
}
