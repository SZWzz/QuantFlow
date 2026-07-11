package market

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"quantflow/internal/ws"
)

// QuotePoller periodically fetches quotes for subscribed symbols and
// broadcasts them via MarketDataHub (cache) and ws.Hub (real-time push).
type QuotePoller struct {
	reg       *AdapterRegistry
	marketHub *MarketDataHub
	wsHub     *ws.Hub
	wsChecker WSCoverageChecker // optional: if set, symbols covered by WS are skipped

	mu       sync.RWMutex
	subs     map[string]bool
	close    chan struct{}
	running  bool
	interval time.Duration
}

// SetWSCoverageChecker sets an optional WS coverage checker.
// When set, symbols covered by an active WS connection will be skipped during polling.
func (p *QuotePoller) SetWSCoverageChecker(checker WSCoverageChecker) {
	p.wsChecker = checker
}

// NewQuotePoller creates a QuotePoller. Call Run() to start processing.
func NewQuotePoller(reg *AdapterRegistry, marketHub *MarketDataHub, wsHub *ws.Hub) *QuotePoller {
	return &QuotePoller{
		reg:       reg,
		marketHub: marketHub,
		wsHub:     wsHub,
		subs:      make(map[string]bool),
		close:     make(chan struct{}),
		interval:  5 * time.Second,
	}
}

func subscriberKey(market, symbol string) string { return market + ":" + symbol }

// Subscribe adds a symbol to the poll set. Idempotent.
func (p *QuotePoller) Subscribe(market, symbol string) {
	key := subscriberKey(market, symbol)
	p.mu.Lock()
	p.subs[key] = true
	p.mu.Unlock()
	slog.Debug("quote poller subscribed", "key", key)
}

// Unsubscribe removes a symbol from the poll set. Idempotent.
func (p *QuotePoller) Unsubscribe(market, symbol string) {
	key := subscriberKey(market, symbol)
	p.mu.Lock()
	delete(p.subs, key)
	p.mu.Unlock()
	slog.Debug("quote poller unsubscribed", "key", key)
}

// SubscriberCount returns the number of subscribed symbol keys.
func (p *QuotePoller) SubscriberCount() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.subs)
}

// Run starts the polling loop. Blocks until ctx is cancelled or Stop is called.
func (p *QuotePoller) Run(ctx context.Context) {
	p.mu.Lock()
	if p.running {
		p.mu.Unlock()
		return
	}
	p.running = true
	p.mu.Unlock()

	defer func() {
		if r := recover(); r != nil {
			slog.Error("quote poller panicked, restarting", "recover", r)
			p.mu.Lock()
			p.running = false
			p.mu.Unlock()
			time.Sleep(5 * time.Second)
			go p.Run(ctx)
		}
	}()
	slog.Info("quote poller started", "interval", p.interval)
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			p.Stop()
			return
		case <-p.close:
			return
		case <-ticker.C:
			p.pollOnce(ctx)
		}
	}
}

func (p *QuotePoller) pollOnce(ctx context.Context) {
	// Auto-detect active subscriptions from the WebSocket Hub so subscribers
	// are picked up dynamically without explicit Subscribe calls.
	topics := p.wsHub.GetTopics()
	quotePrefix := "market:quote:"
	seen := make(map[string]bool)
	for _, topic := range topics {
		if len(topic) > len(quotePrefix) && topic[:len(quotePrefix)] == quotePrefix {
			key := topic[len(quotePrefix):] // e.g. "CN:000001.SH"
			if key != "" && !seen[key] {
				seen[key] = true
			}
		}
	}

	for key := range seen {
		market, symbol := splitSubscriberKey(key)
		if market == "" || symbol == "" {
			continue
		}

			// Skip HTTP poll if a WS connection covers this symbol's exchange
			if p.wsChecker != nil && p.isCoveredByWS(market) {
				continue
			}

		quote, adapter, err := p.reg.FetchQuoteWithFallback(ctx, market, symbol)
		if err != nil {
			slog.Debug("quote poller fetch failed", "key", key, "error", err)
			continue
		}

		topic := "market:quote:" + key
		slog.Debug("quote poller publishing", "topic", topic, "price", quote.Last, "adapter", adapter)

		p.marketHub.Publish(topic, quote)
		p.wsHub.Broadcast(topic, quote)
	}
}

// Stop halts the polling loop.
func (p *QuotePoller) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.running {
		return
	}
	p.running = false
	select {
	case p.close <- struct{}{}:
	default:
	}
	slog.Info("quote poller stopped")
}

// isCoveredByWS returns true if the market has an active WS connection.
func (p *QuotePoller) isCoveredByWS(market string) bool {
	if market != "CRYPTO" {
		return false // only crypto exchanges support WS push currently
	}
	return len(p.wsChecker.ActiveExchanges()) > 0
}

func splitSubscriberKey(key string) (string, string) {
	for i := 0; i < len(key); i++ {
		if key[i] == ':' {
			return key[:i], key[i+1:]
		}
	}
	return "", ""
}
