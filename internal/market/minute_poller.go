package market

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"quantflow/internal/ws"
)

// MinutePoller periodically fetches minute ticks for subscribed symbols
// and broadcasts deltas to a ws.Hub under "market:minute:{symbol}" topics.
// It tracks the last push timestamp per symbol so only new ticks are sent.
type MinutePoller struct {
	wsHub    *ws.Hub
	fetcher  func(symbol string, sinceTimestamp int64) ([]MinuteTick, error)
	lastPush map[string]int64 // symbol → last push timestamp
	subs     map[string]bool
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
		subs:     make(map[string]bool),
		close:    make(chan struct{}),
		interval: 5 * time.Second,
	}
}

// Subscribe adds a symbol to the poll set.
func (p *MinutePoller) Subscribe(symbol string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.subs[symbol] {
		p.subs[symbol] = true
		slog.Info("minute_poller: subscribed", "symbol", symbol)
	}
}

// Unsubscribe removes a symbol.
func (p *MinutePoller) Unsubscribe(symbol string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.subs, symbol)
	delete(p.lastPush, symbol)
	slog.Info("minute_poller: unsubscribed", "symbol", symbol)
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
	if !IsTradingHours("CN") {
		return
	}

	p.mu.Lock()
	symbols := make([]string, 0, len(p.subs))
	for sym := range p.subs {
		symbols = append(symbols, sym)
	}
	p.mu.Unlock()

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
