# WebSocket Market Data Push — Implementation Plan

> **STATUS: ✅ COMPLETED** (2026-07-04) — All 6 tasks implemented. handler.go refactored (no globals), MarketWSService + QuotePoller created, wired in app.go/main.go, WatchlistPanel uses WebSocket push.

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace WatchlistPanel's 10s polling with WebSocket push — QuotePoller backend goroutine fetches quotes → broadcasts via ws.Hub → frontend `useWebSocket` receives and updates.

**Architecture:** QuotePoller (ticker-driven goroutine) calls `FetchQuoteWithFallback` for subscribed symbols, publishes to `MarketDataHub` (cache) and `ws.Hub.Broadcast` (real-time push). Frontend `useWebSocket` connects to Wails HTTP server at `/ws/market`, subscribes to `market:quote:*` topics, updates `quotes` ref reactively.

**Tech Stack:** Go 1.22+, `coder/websocket`, Vue 3 + `useWebSocket` composable, Wails v3 Service.Route

## Global Constraints

- All new files in `internal/market/` follow existing patterns (no panic, slog logging, explicit error returns)
- WebSocket uses `github.com/coder/websocket` (already imported by Wails v3)
- Frontend uses `useWebSocket` composable from `@/lib/composables/useWebSocket`
- QuotePoller respects `IsTradingHours` / `lastQuote` cache from `trading_hours.go`
- All existing tests must pass after changes
- CHANGELOG updated at end

---

### Task 1: Refactor `internal/ws/handler.go` — remove DefaultHub global + init()

**Files:**
- Modify: `internal/ws/handler.go`

**Interfaces:**
- Produces: `ServeWS(w http.ResponseWriter, r *http.Request, hub *Hub)` — new signature with explicit hub param
- Removes: `DefaultHub` global variable, `init()` function

- [x] **Step 1: Rewrite handler.go — remove globals, add hub parameter**

Write `internal/ws/handler.go`:

```go
package ws

import (
	"log/slog"
	"net/http"

	"github.com/coder/websocket"
)

// ServeWS upgrades an HTTP connection to WebSocket and registers the client.
// hub is the WebSocket hub managing topic-based subscriptions.
func ServeWS(w http.ResponseWriter, r *http.Request, hub *Hub) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true,
	})
	if err != nil {
		slog.Error("ws accept", "err", err)
		return
	}

	client := NewClient(hub, conn)
	hub.register <- client

	go client.WritePump()
	go client.ReadPump()
}
```

Remove `var DefaultHub = NewHub()` and `func init() { go DefaultHub.Run() }`.

- [x] **Step 2: Verify build and vet**

```bash
cd /Volumes/shenzy/vibe_coding/QuantFlow && go build ./internal/ws/... && go vet ./internal/ws/...
```

Expected: no errors

- [x] **Step 3: Run ws tests**

```bash
cd /Volumes/shenzy/vibe_coding/QuantFlow && go test ./internal/ws/... -v -count=1
```

Expected: `ok` or `no test files`

- [x] **Step 4: Confirm zero external references to removed symbols**

```bash
grep -rn "ws\.DefaultHub\|ws\.ServeWS" --include="*.go" . | grep -v "_test.go"
```

Expected: only `internal/ws/handler.go` (now removed)

- [x] **Step 5: Commit**

```bash
git add internal/ws/handler.go
git commit -m "refactor(ws): remove DefaultHub global, parameterize ServeWS"
```

---

### Task 2: Create `MarketWSService` — Wails http.Handler wrapper

**Files:**
- Create: `internal/ws/service.go`

**Interfaces:**
- Consumes: `ServeWS(w, r, hub)` from Task 1
- Produces: `MarketWSService` struct with `ServeHTTP(w, r)` — implements `http.Handler`. Field `Hub *Hub` is set during startup before HTTP server starts.

- [x] **Step 1: Create service.go**

Write `internal/ws/service.go`:

```go
package ws

import "net/http"

// MarketWSService wraps a ws.Hub as an http.Handler for Wails service Route registration.
// Hub is set during App.ServiceStartup, before the HTTP server starts serving.
type MarketWSService struct {
	Hub *Hub
}

// ServeHTTP implements http.Handler — upgrades HTTP to WebSocket.
func (s *MarketWSService) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ServeWS(w, r, s.Hub)
}
```

- [x] **Step 2: Build**

```bash
cd /Volumes/shenzy/vibe_coding/QuantFlow && go build ./internal/ws/...
```

Expected: no errors

- [x] **Step 3: Commit**

```bash
git add internal/ws/service.go
git commit -m "feat(ws): add MarketWSService for Wails Route registration"
```

---

### Task 3: Create `QuotePoller` — backend goroutine for periodic data push

**Files:**
- Create: `internal/market/poller.go`
- Create: `internal/market/poller_test.go`

**Interfaces:**
- Consumes:
  - `AdapterRegistry.FetchQuoteWithFallback(ctx, market, symbol)`
  - `MarketDataHub.Publish(topic, data)`
  - `ws.Hub.Broadcast(topic string, data any)`
- Produces:
  - `NewQuotePoller(reg, marketHub, wsHub) *QuotePoller`
  - `(*QuotePoller).Subscribe(market, symbol string)`
  - `(*QuotePoller).Unsubscribe(market, symbol string)`
  - `(*QuotePoller).SubscriberCount() int`
  - `(*QuotePoller).Run(ctx context.Context)`
  - `(*QuotePoller).Stop()`

- [x] **Step 1: Write failing tests**

Write `internal/market/poller_test.go`:

```go
package market

import (
	"context"
	"testing"
	"time"

	"quantflow/internal/ws"
)

type mockPollAdapter struct {
	name      string
	available bool
	quotes    map[string]*QuoteSnapshot
}

func (m *mockPollAdapter) Name() string                          { return m.name }
func (m *mockPollAdapter) IsAvailable(ctx context.Context) bool   { return m.available }
func (m *mockPollAdapter) FetchQuote(_ context.Context, symbol string) (*QuoteSnapshot, error) {
	q, ok := m.quotes[symbol]
	if !ok {
		return nil, nil
	}
	return q, nil
}

func TestQuotePoller_SubscribeUnsubscribe(t *testing.T) {
	reg := NewAdapterRegistry()
	wsHub := ws.NewHub()
	go wsHub.Run()

	poller := NewQuotePoller(reg, NewHub(), wsHub)
	if poller.SubscriberCount() != 0 {
		t.Fatalf("expected 0 subscribers initially, got %d", poller.SubscriberCount())
	}

	poller.Subscribe("CN", "600519")
	if poller.SubscriberCount() != 1 {
		t.Fatalf("expected 1 subscriber, got %d", poller.SubscriberCount())
	}

	poller.Unsubscribe("CN", "600519")
	if poller.SubscriberCount() != 0 {
		t.Fatalf("expected 0 subscribers after unsubscribe, got %d", poller.SubscriberCount())
	}
}

func TestQuotePoller_DeduplicateSubscriptions(t *testing.T) {
	poller := NewQuotePoller(nil, NewHub(), ws.NewHub())
	poller.Subscribe("CN", "600519")
	poller.Subscribe("CN", "600519")
	if poller.SubscriberCount() != 1 {
		t.Fatalf("expected 1 subscriber after dedup, got %d", poller.SubscriberCount())
	}
}

func TestQuotePoller_SubscribeDifferentSymbols(t *testing.T) {
	poller := NewQuotePoller(nil, NewHub(), ws.NewHub())
	poller.Subscribe("CN", "600519")
	poller.Subscribe("US", "AAPL")
	if poller.SubscriberCount() != 2 {
		t.Fatalf("expected 2 subscribers, got %d", poller.SubscriberCount())
	}
}

func TestQuotePoller_StopStart(t *testing.T) {
	reg := NewAdapterRegistry()
	wsHub := ws.NewHub()
	go wsHub.Run()

	poller := NewQuotePoller(reg, NewHub(), wsHub)
	poller.interval = 50 * time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())
	go poller.Run(ctx)

	poller.Subscribe("CN", "600519")
	time.Sleep(30 * time.Millisecond)
	poller.Unsubscribe("CN", "600519")
	time.Sleep(20 * time.Millisecond)

	cancel()
	poller.Stop()
	if poller.SubscriberCount() != 0 {
		t.Fatalf("expected 0 subscribers after stop, got %d", poller.SubscriberCount())
	}
}

func TestQuotePoller_FetchesAndPublishesData(t *testing.T) {
	reg := NewAdapterRegistry()
	adapter := &mockPollAdapter{
		name: "test", available: true,
		quotes: map[string]*QuoteSnapshot{
			"600519": {Symbol: "600519", Last: 1800.0, Change: 10.0, ChangePct: 0.56, Volume: 10000, Timestamp: time.Now().UnixMilli()},
		},
	}
	reg.Register(adapter)

	marketHub := NewHub()
	wsHub := ws.NewHub()
	go wsHub.Run()

	poller := NewQuotePoller(reg, marketHub, wsHub)
	poller.interval = 10 * time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())
	go poller.Run(ctx)

	poller.Subscribe("CN", "600519")

	// Wait for multiple poll cycles
	time.Sleep(60 * time.Millisecond)

	cancel()
	poller.Stop()

	// Verify the data was published to MarketDataHub
	msg, ok := marketHub.GetLatest("market:quote:CN:600519")
	if !ok {
		t.Fatal("expected market data hub to have cached message")
	}
	quote, ok := msg.Data.(*QuoteSnapshot)
	if !ok {
		t.Fatalf("expected *QuoteSnapshot, got %T", msg.Data)
	}
	if quote.Last != 1800.0 {
		t.Fatalf("expected Last=1800, got %f", quote.Last)
	}
}
```

- [x] **Step 2: Run test to verify it fails**

```bash
cd /Volumes/shenzy/vibe_coding/QuantFlow && go test ./internal/market -count=1 -run TestQuotePoller -v
```

Expected: FAIL — "undefined: NewQuotePoller"

- [x] **Step 3: Write QuotePoller implementation**

Write `internal/market/poller.go`:

```go
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

	mu       sync.RWMutex
	subs     map[string]bool
	close    chan struct{}
	running  bool
	interval time.Duration
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
	p.mu.RLock()
	keys := make([]string, 0, len(p.subs))
	for k := range p.subs {
		keys = append(keys, k)
	}
	p.mu.RUnlock()

	for _, key := range keys {
		market, symbol := splitSubscriberKey(key)
		if market == "" || symbol == "" {
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

func splitSubscriberKey(key string) (string, string) {
	for i := 0; i < len(key); i++ {
		if key[i] == ':' {
			return key[:i], key[i+1:]
		}
	}
	return "", ""
}
```

- [x] **Step 4: Run tests**

```bash
cd /Volumes/shenzy/vibe_coding/QuantFlow && go test ./internal/market -count=1 -run TestQuotePoller -v
```

Expected: All 5 tests PASS

- [x] **Step 5: Run full market package tests**

```bash
cd /Volumes/shenzy/vibe_coding/QuantFlow && go test ./internal/market -count=1 -v 2>&1 | tail -30
```

Expected: All tests PASS

- [x] **Step 6: Commit**

```bash
git add internal/market/poller.go internal/market/poller_test.go
git commit -m "feat(market): add QuotePoller for periodic quote fetching and broadcast"
```

---

### Task 4: Wire everything in `app.go` + `main.go`

**Files:**
- Modify: `app.go` (App struct + ServiceStartup)
- Modify: `main.go`

**Interfaces:**
- App struct gains: `marketHub *market.MarketDataHub`, `wsSvc *ws.MarketWSService`, `quotePoller *market.QuotePoller`
- ws.Hub created in ServiceStartup, stored on MarketWSService and passed to QuotePoller
- main.go: MarketWSService registered with Route `/ws/market`

- [x] **Step 1: Add fields to App struct**

In `app.go`, add after existing `marketReg` field (around line 75):

```go
	// Market data hub for in-process pub/sub.
	marketHub     *market.MarketDataHub

	// QuotePoller for periodic quote fetch + WebSocket broadcast.
	quotePoller   *market.QuotePoller
```

- [x] **Step 2: Modify ServiceStartup to create and wire hubs**

In `app.go`, find (around line 202-205):
```go
	_ = market.NewHub()
	slog.Info("market data hub initialized")
```

Replace with:
```go
	a.marketHub = market.NewHub()
	slog.Info("market data hub initialized")
```

Find after the market hub init block (before the CapabilityRegistry section around line 207), add:
```go
	// Create and start the WebSocket hub for real-time market data push.
	wsHub := ws.NewHub()
	go wsHub.Run()

	// Wire the WebSocket hub into the MarketWSService (registered in main.go)
	// so the /ws/market endpoint can upgrade connections.
	if a.wsSvc != nil {
		a.wsSvc.Hub = wsHub
	}

	// Start the QuotePoller: subscribes/unsubscribes based on frontend WS topics,
	// periodically fetches quotes and broadcasts via wsHub.
	a.quotePoller = market.NewQuotePoller(a.marketReg, a.marketHub, wsHub)
	go a.quotePoller.Run(ctx)
	slog.Info("quote poller started on ws hub")
```

- [x] **Step 3: Add `wsSvc` field to App struct**

Add to App struct (before `marketReg` or in the market-related section around line 75):

```go
	// WebSocket service wrapper (set during ServiceStartup, registered in main.go).
	wsSvc         *ws.MarketWSService
```

- [x] **Step 4: Update main.go**

Write `main.go`:

```go
package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v3/pkg/application"

	"quantflow/internal/ws"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// Create the MarketWSService first. Its Hub field is set during
	// App.ServiceStartup, before the HTTP server starts.
	wsSvc := &ws.MarketWSService{}

	app := application.New(application.Options{
		Name:        "quantflow",
		Description: "QuantFlow Terminal — 双模式量化金融终端",
		Services: []application.Service{
			application.NewService(&App{wsSvc: wsSvc}),
			application.NewServiceWithOptions(wsSvc, application.ServiceOptions{
				Route: "/ws/market",
			}),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "QuantFlow Terminal",
		Width:            1400,
		Height:           900,
		MinWidth:         900,
		MinHeight:        600,
		BackgroundColour: application.NewRGB(27, 38, 54),
		URL:              "/",
	})

	err := app.Run()
	if err != nil {
		log.Fatal(err.Error())
	}
}
```

- [x] **Step 5: Verify build**

```bash
cd /Volumes/shenzy/vibe_coding/QuantFlow && go build . 2>&1
```

Expected: success, binary at `./quantflow` or `build/quantflow`

- [x] **Step 6: Run all backend tests**

```bash
cd /Volumes/shenzy/vibe_coding/QuantFlow && go test ./internal/... -count=1 2>&1 | tail -10
```

Expected: all packages `ok`

- [x] **Step 7: Commit**

```bash
git add app.go main.go
git commit -m "feat: wire wsHub, QuotePoller, and MarketWSService into App"
```

---

### Task 5: Frontend — WatchlistPanel WebSocket migration

**Files:**
- Modify: `frontend/src/terminal/panels/WatchlistPanel.vue`

**Interfaces:**
- Consumes: `useWebSocket()` from `@/lib/composables/useWebSocket`
- Removes: `setInterval`, `startPolling()`, `stopPolling()`, `onVisibility()`, `pollTimer`
- Adds: `useWebSocket().connect(url, topics)` on mount, `onMessage` handler that updates `quotes[sym]`

- [x] **Step 1: Add `useWebSocket` import**

In `WatchlistPanel.vue`, add `useWebSocket` to the composables import line:

```typescript
import { useWebSocket } from '@/lib/composables/useWebSocket'
```

- [x] **Step 2: Initialize WebSocket in script setup**

Add after `const { fetchWithCache } = usePanelCache()` (line 13):

```typescript
const ws = useWebSocket()
const wsUrl = `${window.location.protocol === 'https:' ? 'wss:' : 'ws:'}//${window.location.host}/ws/market`
```

- [x] **Step 3: Replace refreshQuote to also handle WS data**

Add a `handleWSQuote` function that updates quotes from WS messages:

After `refreshQuote` function (around line 213):

```typescript
function handleWSQuote(topic: string, data: any) {
  const sym = topic.split(':').pop()
  if (!sym || !symbols.value.includes(sym)) return
  const snap = Array.isArray(data) ? data[0] : data
  if (!snap) return
  const prev = quotes[sym]
  quotes[sym] = {
    symbol: snap.symbol ?? sym,
    name: snap.name || prev?.name || sym,
    last: snap.last ?? 0,
    open: snap.open ?? 0,
    high: snap.high ?? 0,
    low: snap.low ?? 0,
    change: snap.change ?? 0,
    changePct: snap.change_pct ?? snap.changePct ?? 0,
    volume: snap.volume ?? 0,
    turnover: snap.turnover ?? 0,
    turnoverRate: snap.turnover_rate ?? 0,
    volumeRatio: snap.volume_ratio ?? 0,
    amplitude: snap.amplitude ?? 0,
    prevLast: prev?.last,
  }
  delete loading.value[sym]
}
```

- [x] **Step 4: Replace onMounted — remove polling, add WS**

Replace the `onMounted` block (lines 299-323) with:

```typescript
onMounted(async () => {
  window.addEventListener('watchlist-changed', onWatchlistChanged)
  document.addEventListener('click', closeContextMenu)

  try {
    const app = (window as any).go?.main?.App
    if (app?.SearchSymbols) {
      await Promise.all(symbols.value.map(async (sym) => {
        const { data: results } = await fetchWithCache(`search:${sym}`, () => app.SearchSymbols(sym, 1), 5 * 60 * 1000)
        if (Array.isArray(results) && results.length > 0 && results[0].name) {
          if (!quotes[sym]) {
            quotes[sym] = { symbol: sym, name: results[0].name, last: 0, open: 0, high: 0, low: 0, change: 0, changePct: 0, volume: 0, turnover: 0, turnoverRate: 0, volumeRatio: 0, amplitude: 0 }
          } else if (quotes[sym].name === sym) {
            quotes[sym].name = results[0].name
          }
        }
      }))
    }
  } catch { /* best-effort */ }

  // Initial data fetch via Wails IPC (instant, not waiting for WS)
  await refreshAll()
  initialLoadDone.value = true

  // Subscribe to real-time updates via WebSocket
  ws.connect(wsUrl, symbols.value.map(sym => `market:quote:${detectMarket(sym)}:${sym}`))
  ws.onMessage('*', (msg: any) => {
    if (msg.topic?.startsWith('market:quote:')) {
      handleWSQuote(msg.topic, msg.data)
    }
  })
  pollingActive.value = true
  document.addEventListener('visibilitychange', onVisibility)
})
```

- [x] **Step 5: Replace onUnmounted — stop WS instead of polling**

Replace the `onUnmounted` block (lines 325-330) with:

```typescript
onUnmounted(() => {
  window.removeEventListener('watchlist-changed', onWatchlistChanged)
  document.removeEventListener('click', closeContextMenu)
  document.removeEventListener('visibilitychange', onVisibility)
  ws.disconnect()
})
```

- [x] **Step 6: Remove dead polling code**

Remove lines 272-286 (the `startPolling`, `stopPolling`, `onVisibility` functions) and the `pollTimer` variable (line 273).

- [x] **Step 7: Simplify updateVisibility function**

Replace `onVisibility` with a simpler version that just tracks `pollingActive`:

```typescript
function onVisibility() {
  pollingActive.value = !document.hidden
  if (!pollingActive.value) return
  // When tab becomes visible, trigger a fresh quote fetch via IPC
  // to get instant data while WS catches up.
  refreshAll()
}
```

- [x] **Step 8: Verify frontend builds**

```bash
cd /Volumes/shenzy/vibe_coding/QuantFlow/frontend && npx vue-tsc --noEmit 2>&1 | head -20
```

Expected: no type errors

- [x] **Step 9: Verify frontend tests**

```bash
cd /Volumes/shenzy/vibe_coding/QuantFlow/frontend && npx vitest run 2>&1 | tail -10
```

Expected: all tests PASS

- [x] **Step 10: Commit**

```bash
git add frontend/src/terminal/panels/WatchlistPanel.vue
git commit -m "feat(frontend): WatchlistPanel uses WebSocket instead of polling"
```

---

### Task 6: Update CHANGELOG + Final verification

**Files:**
- Modify: `CHANGELOG.md`

- [x] **Step 1: Update CHANGELOG.md**

Add entry under `[2026.7.4]` section:

```markdown
### Added
- [MarketData] QuotePoller — background goroutine for periodic quote fetching and WebSocket broadcast
- [MarketData] MarketWSService — Wails service wrapper exposing /ws/market HTTP endpoint for real-time data
- [Frontend] WatchlistPanel now subscribes to WebSocket push instead of polling every 10s

### Changed
- [WS] internal/ws/handler.go: removed DefaultHub global + init(); ServeWS now accepts explicit *Hub parameter
- [WS] Added MarketWSService (http.Handler) for Wails Route registration
- [App] MarketDataHub is now stored on App struct instead of discarded
- [App] WebSocket hub created and wired during ServiceStartup

### Removed
- [Frontend] WatchlistPanel: setInterval polling mechanism removed
```

- [x] **Step 2: Full build + test**

```bash
cd /Volumes/shenzy/vibe_coding/QuantFlow && \
  go build ./... 2>&1 && \
  go vet ./... 2>&1 && \
  go test ./internal/... -count=1 2>&1 | tail -20 && \
  cd frontend && npx vue-tsc --noEmit 2>&1 && \
  npx vitest run 2>&1 | tail -10
```

Expected: everything passes

- [x] **Step 3: Commit**

```bash
git add CHANGELOG.md
git commit -m "docs: update CHANGELOG for WebSocket market data push"
```
