# Frontend Architecture: Type-Safe Bridge, Lifecycle, i18n, DX

## Motivation

Phase 12 review identified several systemic frontend issues:

### Type Safety (1 critical, 1 major)
- **`window.go.main.App.*` used everywhere**: All 60+ panels access Go backend via untyped `(window as any).go?.main?.App?.GetQuote(...)`. Silent runtime failures when backend API changes.
- **Module-level state in multi-instance panels**: `BacktestPanel.vue:213` uses `let selectedRow` as module-level var, shared across all panel instances → cross-contamination.

### Lifecycle (2 major)
- **`useWebSocket` creates per-component connections**: No singleton/connection pooling. Each panel instance opens its own WS.
- **Stores lack request dedup + stale response guards**: `portfolio.fetchPositions` fires unconditionally; sequential calls can race.

### i18n (1 major)
- **BacktestPanel hardcodes Chinese**: ~30 labels in Chinese strings, i18n system exists but unused.

### DX/Maintainability (2 major)
- **CandlestickPanel 1129 lines**: Single file handles kline, minute chart, depth sidebar, drawing, crosshair, WebSocket, polling, trading hours, watchlist integration.
- **`noUncheckedIndexedAccess` not in tsconfig**: `arr[i]` returns `T`, not `T | undefined` — hidden undefined bugs.

## Design

### 1. Type-Safe Go Bridge (Phase 1: Audit + Wrappers)

**Step 1: Auto-generate wrapper types**
Create a Go reflection tool (or manual audit) that lists all exported Wails methods from `App` and generates TypeScript type declarations:

```typescript
// frontend/src/lib/wails.ts — typed bridge (already exists, expand)
export interface GoApp {
    GetQuote(symbol: string): Promise<QuoteSnapshot>
    FetchOHLCV(symbol: string, interval: string, start: string, end: string): Promise<OHLCVBar[]>
    // ... all 50+ methods
}

const app: GoApp = new Proxy({} as GoApp, {
    get(_, method: string) {
        return (...args: any[]) => {
            const goMethod = (window as any).go?.main?.App?.[method]
            if (!goMethod) throw new Error(`Go method ${method} not available`)
            return goMethod(...args)
        }
    }
})

export function useGoApp(): GoApp { return app }
```

**Step 2: Codemod all panels**
Replace all `(window as any).go?.main?.App?.getQuote(sym)` with `app.GetQuote(sym)`.

Automation: Write a jscodeshift or simple regex replace script, then manual review.

**Modified files:**
- `frontend/src/lib/wails.ts` — Create proxy + full type declarations
- All 60+ panel `.vue` and `.ts` files — Replace `window.go.*` with typed calls
- `frontend/src/stores/*.ts` — Same replacement

### 2. Fix Module-Level State in Panels

**BacktestPanel.vue:**
```typescript
// Before: let selectedRow: any = null  (module-level, shared across instances)
// After:
const selectedRow = ref<any>(null)  // instance-level, reactive
```

**CandlestickPanel.vue:**
```typescript
// Before: let loadSeq = 0  (module-level)
// After:
const loadSeq = ref(0)  // instance-level
```

**Modified files:**
- `frontend/src/terminal/panels/BacktestPanel.vue`
- `frontend/src/terminal/panels/CandlestickPanel.vue`
- Audit all other panels for module-level state (search: `let \w+ = ` at module scope)

### 3. WebSocket Singleton Connection Pool

```typescript
// frontend/src/lib/composables/useWebSocket.ts
class WebSocketManager {
    private connections = new Map<string, {
        ws: WebSocket
        subscribers: Set<{ topic: string; handler: (data: any) => void }>
    }>()
    
    subscribe(url: string, topic: string, handler: (data: any) => void): () => void {
        if (!this.connections.has(url)) {
            this.connections.set(url, {
                ws: new WebSocket(url + '?subscribe=' + topic),
                subscribers: new Set()
            })
        }
        const conn = this.connections.get(url)!
        conn.subscribers.add({ topic, handler })
        return () => conn.subscribers.delete({ topic, handler })
    }
}

// Singleton
export const wsManager = new WebSocketManager()
```

**Modified files:**
- `frontend/src/lib/composables/useWebSocket.ts` — Singleton pattern
- Panels using WebSocket — Use `wsManager.subscribe()` instead of new connections

### 4. Store Request Dedup + Stale Response Guard

```typescript
// frontend/src/stores/portfolio.ts
const pendingRequests = new Map<string, Promise<any>>()
const requestSeq = new Map<string, number>()

async function fetchPositions() {
    const key = 'fetchPositions'
    const seq = (requestSeq.get(key) || 0) + 1
    requestSeq.set(key, seq)
    
    // Dedup: reuse in-flight promise
    if (pendingRequests.has(key)) return pendingRequests.get(key)
    
    const promise = (async () => {
        try {
            const result = await app.GetPositions()
            if (requestSeq.get(key) === seq) {
                positions.value = result
            }
        } finally {
            pendingRequests.delete(key)
        }
    })()
    
    pendingRequests.set(key, promise)
    return promise
}
```

**Modified files:**
- `frontend/src/stores/portfolio.ts` — Add dedup + stale guards
- `frontend/src/stores/ml.ts` — Same
- `frontend/src/stores/research.ts` — Same
- `frontend/src/stores/data.ts` — Same (for fetchMarketOverview etc.)

### 5. BacktestPanel i18n

Replace all hardcoded Chinese strings with i18n keys:

```typescript
// Before: <div>回测历史</div>
// After: <div>{{ $t('backtest.history') }}</div>
```

Key additions needed:
```typescript
// frontend/src/lib/i18n/zh.ts
backtest: {
    history: '回测历史',
    start: '回测开始',
    end: '回测结束',
    strategy: '策略',
    symbol: '标的',
    // ... ~30 keys
}

// frontend/src/lib/i18n/en.ts
backtest: {
    history: 'Backtest History',
    start: 'Start',
    end: 'End',
    strategy: 'Strategy',
    symbol: 'Symbol',
    // ...
}
```

**Modified files:**
- `frontend/src/lib/i18n/zh.ts` — Add `backtest.*` keys
- `frontend/src/lib/i18n/en.ts` — Add `backtest.*` keys
- `frontend/src/terminal/panels/BacktestPanel.vue` — Replace strings with `$t()`

### 6. CandlestickPanel Decomposition

Split 1129-line file into:
- `CandlestickPanel.vue` — Orchestrator (selected symbol, mode switching, loading/error states) ~200 lines
- `KlineChartSection.vue` — K-line chart + drawing + crosshair ~300 lines
- `MinuteChartSection.vue` — Minute chart + depth sidebar ~200 lines
- `composables/useKlineData.ts` — OHLCV fetching, caching, polling ~150 lines
- `composables/useMinuteData.ts` — Minute data fetching, cyclic ~100 lines

**Modified files:**
- `frontend/src/terminal/panels/CandlestickPanel.vue` — Trim to orchestrator
- `frontend/src/terminal/panels/KlineChartSection.vue` — New
- `frontend/src/terminal/panels/MinuteChartSection.vue` — New
- `frontend/src/lib/composables/useKlineData.ts` — New
- `frontend/src/lib/composables/useMinuteData.ts` — New
- `frontend/src/terminal/panels/registry.ts` — Update imports

### 7. Enable noUncheckedIndexedAccess

```json
// tsconfig.json
"compilerOptions": {
    "noUncheckedIndexedAccess": true
}
```

This will surface ~100+ type errors. Fix them systematically:
- `arr[i]` → `arr[i]!` (if known safe) or `if (item = arr[i])` pattern
- Particularly common in panel table rendering, chart data access

**Modified files:**
- `frontend/tsconfig.json` — Enable flag
- `frontend/src/**/*.ts` — Fix type errors (estimate: 50-150 locations)

## Acceptance Criteria

- [ ] All `window.go.*` access goes through typed `useGoApp()` bridge; 0 bare `(window as any)` references
- [ ] BacktestPanel `selectedRow` is a `ref`, not module-level variable
- [ ] WebSocket connections: N panels sharing same URL → 1 connection (not N)
- [ ] Store fetch requests: duplicate simultaneous calls → 1 network request
- [ ] Stale response: slow response discarded if newer request completed first
- [ ] BacktestPanel: all labels use i18n (zh + en), 0 hardcoded Chinese strings
- [ ] CandlestickPanel <400 lines; split into 4+ files
- [ ] `noUncheckedIndexedAccess` enabled, 0 tsc errors
- [ ] All existing tests pass

## Risks / Trade-offs

- **Codemod risk**: Replacing `window.go.*` across 60+ files is high-touch. Use a two-phase approach: first add the proxy type, then migrate file-by-file over several days. Ship each batch independently.
- **noUncheckedIndexedAccess**: Will surface many pre-existing bugs. Budget 1-2 days for the fix cycle. Some may be subtle (e.g., `arr[i].property` where `i` could be out of bounds).
- **WebSocket singleton**: Changes connection lifecycle — currently each panel manages its own reconnection. Singleton must handle reconnection with backoff for all subscribers.
