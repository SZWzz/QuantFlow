// Package wsconn manages external exchange WebSocket connections
// for real-time market data push. Each exchange adapter can optionally
// implement WSConnector to enable push-mode data delivery.
package wsconn

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"quantflow/internal/market"
)

// WSConnector is an optional interface that adapters can implement
// to provide real-time WebSocket data push.
type WSConnector interface {
	// ConnectWS establishes a WebSocket connection and begins pushing
	// data to the provided MarketDataHub.
	ConnectWS(ctx context.Context, hub *market.MarketDataHub) error

	// DisconnectWS closes the WebSocket connection.
	DisconnectWS() error

	// SupportsWS returns true if this adapter supports WebSocket push.
	SupportsWS() bool

	// ExchangeName returns the exchange identifier for logging/metrics.
	ExchangeName() string
}

// managedConn wraps an active WSConnector with its lifecycle state.
type managedConn struct {
	connector WSConnector
	cancel    context.CancelFunc
}

// Manager manages multiple exchange WebSocket connections with
// shared reconnection and heartbeat logic.
type Manager struct {
	mu    sync.Mutex
	conns map[string]*managedConn // exchange name → connection
	hub   *market.MarketDataHub
}

// NewManager creates a new wsconn Manager.
func NewManager(hub *market.MarketDataHub) *Manager {
	return &Manager{
		conns: make(map[string]*managedConn),
		hub:   hub,
	}
}

// Add establishes a WebSocket connection for the given connector.
// If the connector does not support WS, this is a no-op.
func (m *Manager) Add(ctx context.Context, connector WSConnector) error {
	if !connector.SupportsWS() {
		slog.Info("wsconn: adapter does not support WS, skipping", "exchange", connector.ExchangeName())
		return nil
	}

	m.mu.Lock()
	if _, exists := m.conns[connector.ExchangeName()]; exists {
		m.mu.Unlock()
		slog.Info("wsconn: connection already exists", "exchange", connector.ExchangeName())
		return nil
	}
	m.mu.Unlock()

	wsCtx, cancel := context.WithCancel(ctx)

	if err := connector.ConnectWS(wsCtx, m.hub); err != nil {
		cancel()
		return err
	}

	m.mu.Lock()
	m.conns[connector.ExchangeName()] = &managedConn{
		connector: connector,
		cancel:    cancel,
	}
	m.mu.Unlock()

	slog.Info("wsconn: connected", "exchange", connector.ExchangeName())
	return nil
}

// Remove disconnects and removes a connector by exchange name.
func (m *Manager) Remove(exchangeName string) {
	m.mu.Lock()
	mc, ok := m.conns[exchangeName]
	if !ok {
		m.mu.Unlock()
		return
	}
	delete(m.conns, exchangeName)
	m.mu.Unlock()

	mc.cancel()
	if err := mc.connector.DisconnectWS(); err != nil {
		slog.Warn("wsconn: error disconnecting", "exchange", exchangeName, "error", err)
	}
	slog.Info("wsconn: disconnected", "exchange", exchangeName)
}

// Stop disconnects all connectors.
func (m *Manager) Stop() {
	m.mu.Lock()
	conns := make([]*managedConn, 0, len(m.conns))
	names := make([]string, 0, len(m.conns))
	for name, mc := range m.conns {
		conns = append(conns, mc)
		names = append(names, name)
		delete(m.conns, name)
	}
	m.mu.Unlock()

	for i, mc := range conns {
		mc.cancel()
		if err := mc.connector.DisconnectWS(); err != nil {
			slog.Warn("wsconn: error disconnecting", "exchange", names[i], "error", err)
		}
	}
	slog.Info("wsconn: all connections stopped")
}

// IsActive returns true if the given exchange has an active WS connection.
func (m *Manager) IsActive(exchangeName string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.conns[exchangeName]
	return ok
}

// ActiveExchanges returns the names of exchanges with active WS connections.
func (m *Manager) ActiveExchanges() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	names := make([]string, 0, len(m.conns))
	for name := range m.conns {
		names = append(names, name)
	}
	return names
}

// backoff returns the next reconnect delay using exponential backoff.
// Delays: 1s → 2s → 4s → 8s → 16s → 30s (max).
func backoff(attempt int) time.Duration {
	delays := []time.Duration{1, 2, 4, 8, 16, 30}
	if attempt >= len(delays) {
		return 30 * time.Second
	}
	return delays[attempt] * time.Second
}
