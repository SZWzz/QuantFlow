package market

import (
	"context"
	"log/slog"
	"quantflow/internal/ws"
	"sync"
	"time"
)

// MinutePoller periodically fetches minute ticks for symbols that have
// active WebSocket subscribers (via market:minute:* topics) and broadcasts
// deltas to a ws.Hub.  Subscriptions are derived automatically from the Hub
// — no explicit Subscribe/Unsubscribe needed.
type MinutePoller struct {
	wsHub    *ws.Hub
	fetcher  func(symbol string, sinceTimestamp int64) ([]MinuteTick, error)
	lastPush map[string]int64 // symbol → last push timestamp
	mu       sync.Mutex
	close    chan struct{}
	running  bool
	interval time.Duration
}

// NewMinutePoller creates a MinutePoller. fetcher should call GetMinuteLine.
func NewMinutePoller(wsHub *ws.Hub, fetcher func(string, int64) ([]MinuteTick, error)) *MinutePoller {
	return &MinutePoller{
		wsHub:    wsHub,
		fetcher:  fetcher,
		lastPush: make(map[string]int64),
		close:    make(chan struct{}),
		interval: 5 * time.Second,
	}
}

// activeSymbols returns symbols currently subscribed via WS.
func (p *MinutePoller) activeSymbols() []string {
	topics := p.wsHub.GetTopics()
	seen := make(map[string]bool)
	var symbols []string
	prefix := "market:minute:"
	for _, topic := range topics {
		if len(topic) > len(prefix) && topic[:len(prefix)] == prefix {
			sym := topic[len(prefix):]
			if !seen[sym] && sym != "" {
				seen[sym] = true
				symbols = append(symbols, sym)
			}
		}
	}
	return symbols
}

// Run starts the polling loop. Blocks until ctx is cancelled or Stop is called.
func (p *MinutePoller) Run(ctx context.Context) {
	p.mu.Lock()
	if p.running {
		p.mu.Unlock()
		return
	}
	p.running = true
	p.mu.Unlock()

	defer func() {
		if r := recover(); r != nil {
			slog.Error("minute poller panicked, restarting", "recover", r)
			p.mu.Lock()
			p.running = false
			p.mu.Unlock()
			time.Sleep(5 * time.Second)
			go p.Run(ctx)
		}
	}()

	slog.Info("minute poller started", "interval", p.interval)
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
			p.pollOnce()
		}
	}
}

func (p *MinutePoller) pollOnce() {
	// Poll when any supported market is in trading hours
	if !IsTradingHours("CN") && !IsTradingHours("HK") {
		return
	}

	symbols := p.activeSymbols()
	for _, sym := range symbols {
		p.mu.Lock()
		since := p.lastPush[sym]
		p.mu.Unlock()

		ticks, err := p.fetcher(sym, since)
		if err != nil {
			continue
		}

		if len(ticks) == 0 {
			continue
		}

		// Update last push timestamp based on the last tick
		p.mu.Lock()
		p.lastPush[sym] = time.Now().Unix()
		p.mu.Unlock()

		topic := "market:minute:" + sym
		p.wsHub.Broadcast(topic, ticks)
	}
}

// Stop halts the polling loop.
func (p *MinutePoller) Stop() {
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
	slog.Info("minute poller stopped")
}
