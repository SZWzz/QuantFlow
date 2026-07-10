# WebSocket 实时数据迁移 — Phase 1 实施计划

> Spec: [2026-07-09-websocket-migration.md](../../specs/2026-07-09-websocket-migration.md)

**Goal:** CandlestickPanel 分时图 + TickerBar 从轮询迁移到 WebSocket 实时推送

**Architecture:**
```
Go QuotePoller  ──Publish──→  ws.Hub  ──WebSocket──→  useRealtimeData hook
   (扩展)                       (新增 topic)              (新增通用 hook)
                                                    ↓
                              CandlestickPanel       TickerBar
                              (替换 5s 轮询)          (替换 10s 轮询)
```

---

### Task 1: 通用 useRealtimeData Hook

**Files:**
- New: `frontend/src/lib/composables/useRealtimeData.ts`

- [ ] **Step 1: Create useRealtimeData.ts**

```typescript
// frontend/src/lib/composables/useRealtimeData.ts
import { onMounted, onUnmounted, type Ref, isRef } from 'vue'
import { useWebSocket } from '@/lib/composables/useWebSocket'

export function useRealtimeData<T = any>(
  topics: string[] | Ref<string[]> | (() => string[]),
  handler: (topic: string, data: T) => void,
) {
  const ws = useWebSocket()
  const wsUrl = `${location.protocol === 'https:' ? 'wss:' : 'ws:'}//${location.host}/ws/market`

  function connect() {
    const t = typeof topics === 'function' ? topics() : isRef(topics) ? topics.value : topics
    if (t.length) ws.connect(wsUrl, t)
  }

  ws.onMessage('*', (msg: any) => {
    handler(msg.topic, msg.data)
  })

  onMounted(() => connect())

  onUnmounted(() => ws.disconnect())

  return { resubscribe: connect }
}
```

- [ ] **Step 2: Commit**

```bash
git add frontend/src/lib/composables/useRealtimeData.ts
git commit -m "feat(frontend): add useRealtimeData composable for WebSocket subscriptions"
```

---

### Task 2: Go MinutePoller — 分时增量推送

**Files:**
- New: `internal/market/minute_poller.go`
- Modify: `app_startup.go`
- Modify: `app.go`

- [ ] **Step 1: Read existing QuotePoller for reference pattern**

```bash
rg -n "QuotePoller\|type.*Poller" internal/market/
```

- [ ] **Step 2: Create MinutePoller**

```go
// internal/market/minute_poller.go
package market

import (
    "context"
    "log/slog"
    "sync"
    "time"
)

// MinutePoller periodically fetches minute ticks for subscribed symbols
// and publishes deltas to a ws.Hub.
type MinutePoller struct {
    hub       wsHub
    fetcher   func(symbol string, sinceTimestamp int64) ([]MinuteTick, error)
    symbols   map[string]int64 // symbol → last push timestamp
    mu        sync.Mutex
    ctx       context.Context
    cancel    context.CancelFunc
    interval  time.Duration
}

// wsHub is the subset of ws.Hub that MinutePoller needs.
type wsHub interface {
    Broadcast(topic string, data any)
}

// NewMinutePoller creates a MinutePoller. fetcher should call GetMinuteLine.
func NewMinutePoller(hub wsHub, fetcher func(string, int64) ([]MinuteTick, error)) *MinutePoller {
    ctx, cancel := context.WithCancel(context.Background())
    return &MinutePoller{
        hub:      hub,
        fetcher:   fetcher,
        symbols:  make(map[string]int64),
        ctx:      ctx,
        cancel:   cancel,
        interval: 5 * time.Second,
    }
}

// Subscribe adds a symbol to the polling list.
func (p *MinutePoller) Subscribe(symbol string) {
    p.mu.Lock()
    defer p.mu.Unlock()
    if _, ok := p.symbols[symbol]; !ok {
        p.symbols[symbol] = 0
        slog.Info("minute_poller: subscribed", "symbol", symbol)
    }
}

// Unsubscribe removes a symbol.
func (p *MinutePoller) Unsubscribe(symbol string) {
    p.mu.Lock()
    defer p.mu.Unlock()
    delete(p.symbols, symbol)
}

// Start begins the polling loop.
func (p *MinutePoller) Start() {
    go p.loop()
}

// Stop shuts down the poller.
func (p *MinutePoller) Stop() {
    p.cancel()
}

func (p *MinutePoller) loop() {
    ticker := time.NewTicker(p.interval)
    defer ticker.Stop()
    for {
        select {
        case <-p.ctx.Done():
            return
        case <-ticker.C:
            p.poll()
        }
    }
}

func (p *MinutePoller) poll() {
    if !IsTradingHours("CN") {
        return // skip outside trading hours
    }

    p.mu.Lock()
    symbols := make([]string, 0, len(p.symbols))
    for sym := range p.symbols {
        symbols = append(symbols, sym)
    }
    p.mu.Unlock()

    for _, sym := range symbols {
        p.mu.Lock()
        since := p.symbols[sym]
        p.mu.Unlock()

        ticks, err := p.fetcher(sym, since)
        if err != nil {
            continue
        }

        if len(ticks) == 0 {
            continue
        }

        // Update last timestamp from the last tick
        p.mu.Lock()
        p.symbols[sym] = time.Now().Unix()
        p.mu.Unlock()

        // Publish deltas to WebSocket
        topic := "market:minute:" + sym
        p.hub.Broadcast(topic, ticks)
    }
}
```

- [ ] **Step 3: Wire MinutePoller in app_startup.go**

In `ServiceStartup()`, after creating `wsHub`:

```go
// Start minute data poller for CandlestickPanel real-time updates
a.minutePoller = market.NewMinutePoller(
    a.wsHub,
    func(symbol string, since int64) ([]market.MinuteTick, error) {
        ctx, cancel := market.RequestCtx()
        defer cancel()
        ticks, _, err := a.GetMinuteLine(ctx, symbol, since)
        return ticks, err
    },
)
a.minutePoller.Start()
```

- [ ] **Step 4: Add minutePoller field to App struct**

In `app.go`:
```go
type App struct {
    // ... existing fields ...
    minutePoller *market.MinutePoller
}
```

- [ ] **Step 5: Add shutdown in app_shutdown.go**

```go
if a.minutePoller != nil {
    a.minutePoller.Stop()
}
```

- [ ] **Step 6: Verify compilation**

```bash
go vet ./...
```

- [ ] **Step 7: Commit**

---

### Task 3: CandlestickPanel 分时图 → WebSocket

**Files:**
- Modify: `frontend/src/terminal/panels/CandlestickPanel.vue`

- [ ] **Step 1: Read current minute polling code**

```bash
grep -n "loadMinute\|minuteLoading\|setInterval\|5.*1000\|minuteTicks" frontend/src/terminal/panels/CandlestickPanel.vue | head -20
```

- [ ] **Step 2: Replace polling with useRealtimeData**

Key changes in CandlestickPanel.vue `<script setup>`:

```typescript
// Add import
import { useRealtimeData } from '@/lib/composables/useRealtimeData'

// Replace polling timer + loadMinuteLine() with:
const { resubscribe } = useRealtimeData<MinuteTick[]>(
  () => [`market:minute:${symbol.value}`],
  (topic, ticks) => {
    if (!Array.isArray(ticks) || ticks.length === 0) return
    // Merge incremental ticks (same logic as before)
    const existing = new Map(minuteTicks.value.map(t => [t.time, t]))
    for (const t of ticks) {
      existing.set(t.time, t)
    }
    minuteTicks.value = Array.from(existing.values())
      .sort((a, b) => a.time.localeCompare(b.time))

    if (prevClose.value === 0 && minuteTicks.value.length > 0) {
      prevClose.value = minuteTicks.value[0].price
    }
  }
)

// Re-subscribe when symbol changes
watch(symbol, () => resubscribe())

// Remove: setInterval timer, loadMinuteLine(), loadSeq, minuteLoading
// Keep: initial load (fetch historical minute data for today)
```

- [ ] **Step 3: Keep initial load for historical data**

The WS only pushes INCREMENTAL updates. On symbol change, still need to load the full day's minute data:

```typescript
// Initial full load via IPC (not WS — WS only pushes deltas)
async function loadInitialMinute() {
  const result = await dataStore.fetchMinuteLine(symbol.value, 0)
  const ticks = Array.isArray(result) ? result[0] : result
  if (Array.isArray(ticks)) {
    minuteTicks.value = ticks
    if (prevClose.value === 0 && ticks.length > 0) {
      prevClose.value = ticks[0].price
    }
  }
}

watch(symbol, () => {
  loadInitialMinute()
  resubscribe()
})

onMounted(() => {
  loadInitialMinute()
  // WS subscription handled by useRealtimeData
})
```

- [ ] **Step 4: Run tests**

```bash
cd frontend && npx vitest run src/terminal/panels/__tests__/CandlestickPanel.test.ts 2>&1
```

- [ ] **Step 5: TypeScript check**

```bash
cd frontend && npx vue-tsc --noEmit 2>&1 | grep -i "CandlestickPanel"
```

- [ ] **Step 6: Commit**

---

### Task 4: Go TickerPoller — 滚动报价批量推送

**Files:**
- New: `internal/market/ticker_poller.go`
- Modify: `app_startup.go`
- Modify: `app.go`

- [ ] **Step 1: Create TickerPoller**

```go
// internal/market/ticker_poller.go
package market

// TickerPoller batch-fetches quotes for a fixed list of representative
// symbols per market and pushes them to ws.Hub under "market:ticker:{market}".
type TickerPoller struct {
    hub      wsHub
    registry *AdapterRegistry
    markets  []string
    symbols  map[string][]string // market → symbols
    mu       sync.Mutex
    ctx      context.Context
    cancel   context.CancelFunc
}

func NewTickerPoller(hub wsHub, reg *AdapterRegistry) *TickerPoller {
    ctx, cancel := context.WithCancel(context.Background())
    return &TickerPoller{
        hub:      hub,
        registry: reg,
        markets:  []string{"CN", "HK", "US"},
        symbols: map[string][]string{
            "CN": {"000001.SH", "399001.SZ", "399006.SZ"},
            "HK": {"00700.HK", "09988.HK", "00388.HK"},
            "US": {"AAPL", "TSLA", "NVDA"},
        },
        ctx:    ctx,
        cancel: cancel,
    }
}

// Methods: Start, Stop, poll → fetch quotes via registry.FetchQuoteWithFallback,
// aggregate into []QuoteSnapshot, Broadcast to "market:ticker:{mkt}"
// ... (similar loop pattern as MinutePoller, 3s interval)
```

- [ ] **Step 2: Wire TickerPoller + commit**

---

### Task 5: TickerBar → WebSocket

**Files:**
- Modify: `frontend/src/terminal/components/TickerBar.vue`

- [ ] **Step 1: Replace polling with useRealtimeData**

```typescript
import { useRealtimeData } from '@/lib/composables/useRealtimeData'

const tickerQuotes = ref<Record<string, QuoteData>>({})

useRealtimeData(
  ['market:ticker:CN', 'market:ticker:HK', 'market:ticker:US'],
  (topic, data) => {
    // data is []QuoteSnapshot — merge into tickerQuotes
    for (const q of data as any[]) {
      tickerQuotes.value[q.symbol ?? q.code] = q
    }
  }
)

// Remove: setInterval timer
```

- [ ] **Step 2: Run tests + commit**

---

### Task 6: Update CHANGELOG

- [ ] Add entries under `[2026.7.9]` → `### Changed`

```markdown
- [Frontend] CandlestickPanel 分时图从 5s 轮询迁移到 WebSocket 实时推送（useRealtimeData hook）
- [Frontend] TickerBar 从定时轮询迁移到 WebSocket 批量推送
- [Market] 新增 MinutePoller（分时增量推送 ws.Hub）
- [Market] 新增 TickerPoller（滚动报价批量推送 ws.Hub）
- [Frontend] 新增 useRealtimeData 通用 WebSocket hook
```

- [ ] Commit

---

### Task 7: 清理 MarketOverview 调试代码

- [ ] Remove `console.log` debug statements (noting they're already cleaned up)
- [ ] Remove Python `/tmp/quantflow_minute_debug.json` dump
- [ ] Commit

---

## Revision Notes

- Phase 1 focuses on the two highest-impact panels: CandlestickPanel (5s polling) and TickerBar (always visible)
- MinutePoller reuses existing `GetMinuteLine(symbol, sinceTimestamp)` for incremental fetching — no new adapter code needed
- TickerBar symbols are hardcoded representative tickers per market; future enhancement: dynamic hot-list
- MarketOverview was migrated to WS in a prior commit; this plan covers remaining P0 panels
