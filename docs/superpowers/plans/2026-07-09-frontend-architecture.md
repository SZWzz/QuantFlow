# Frontend Architecture Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Type-safe Go bridge proxy, fix module-level panel state, WebSocket singleton, store request dedup + stale response guards, BacktestPanel i18n, CandlestickPanel decomposition, enable noUncheckedIndexedAccess.

**Architecture:** Incremental migration — add proxy type first, then codemod file-by-file, leaving existing `window.go.*` calls working during transition. BacktestPanel i18n is pure template substitution. CandlestickPanel split is re-export from new files.

**Tech Stack:** Vue 3.5+ TypeScript 5+, Pinia, vue-i18n, vitest

## Global Constraints

- `window.go.*` calls must keep working during migration (proxy returns same function)
- No global infrastructure changes (no new dependency, no build tool change)
- WebSocket singleton: backward compatible API (same return type: `() => void` unsubscribe)
- CandlestickPanel split: no visual or behavioral change

---

### Task 1: Type-Safe Go Bridge (Proxy + Wrappers)

**Files:**
- Modify: `frontend/src/lib/wails.ts`

- [ ] **Step 1: Write test**

```typescript
// frontend/src/lib/__tests__/wails.test.ts (augment existing)
import { describe, it, expect, vi, beforeEach } from 'vitest'

describe('useGoApp proxy', () => {
  beforeEach(() => {
    ;(window as any).go = {
      main: {
        App: {
          GetQuote: vi.fn().mockResolvedValue({ Symbol: 'AAPL', Last: 150 }),
        }
      }
    }
  })

  it('proxies method calls to window.go.main.App', async () => {
    const app = await import('@/lib/wails').then(m => m.useGoApp())
    const quote = await app.GetQuote('AAPL')
    expect(quote).toEqual({ Symbol: 'AAPL', Last: 150 })
  })

  it('throws when method is not available', async () => {
    delete (window as any).go
    const app = await import('@/lib/wails').then(m => m.useGoApp())
    await expect(app.GetQuote('AAPL')).rejects.toThrow('not available')
  })
})
```

- [ ] **Step 2: Add proxy + useGoApp()**

```typescript
// frontend/src/lib/wails.ts — add with other exports

export interface WailsApp {
  // Market Data
  GetQuote(symbol: string): Promise<QuoteSnapshot>
  FetchOHLCV(symbol: string, interval: string, startTime: string, endTime: string): Promise<OHLCVBar[]>
  // TODO: add remaining 50+ methods gradually
}

const _goApp = new Proxy({} as WailsApp, {
  get(_target: any, prop: string) {
    return (...args: any[]) => {
      const go = (window as any).go?.main?.App
      if (!go?.[prop]) throw new Error(`Go method ${prop} not available`)
      return go[prop](...args)
    }
  },
})

export function useGoApp(): WailsApp {
  return _goApp
}
```

- [ ] **Step 3: Export GoApp type from returned type**

Also export the proxy as `goApp` for direct import:

```typescript
export const goApp = useGoApp()
```

- [ ] **Step 4: Run test**

```bash
cd frontend && npx vitest run src/lib/__tests__/wails.test.ts
```
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add frontend/src/lib/wails.ts frontend/src/lib/__tests__/wails.test.ts
git commit -m "feat(frontend): add type-safe useGoApp() proxy for Wails bridge"
```

---

### Task 2: Fix Module-Level State in BacktestPanel + CandlestickPanel + Audit

**Files:**
- Modify: `frontend/src/terminal/panels/BacktestPanel.vue`
- Modify: `frontend/src/terminal/panels/CandlestickPanel.vue`
- Check: all panels (grep for `let \w+ = ` at module level)

- [ ] **Step 1: Fix BacktestPanel — selectedRow**

```typescript
// Search for: let selectedRow
// Current: let selectedRow: any = null  (module level)
// Fix:
const selectedRow = ref<any>(null)  // instance level
```

Regenerate all references to `selectedRow` → `selectedRow.value`.

- [ ] **Step 2: Fix CandlestickPanel — loadSeq**

```typescript
// Current: let loadSeq = 0  (module level)
// Fix:
const loadSeq = ref(0)  // instance level
```

Update references: `loadSeq++` → `loadSeq.value++`, `if (seq < loadSeq)` → `if (seq < loadSeq.value)`.

- [ ] **Step 3: Grep-check all other panels**

```bash
grep -rn "^let [a-zA-Z]" frontend/src/terminal/panels/*.vue | grep -v " let " | grep -v "ref\|shallowRef\|computed"
```

For any remaining module-level state that should be instance-level, convert to `ref()`.

- [ ] **Step 4: Run panel tests**

```bash cd frontend && npx vitest run src/terminal/panels/ --reporter=verbose
```
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add frontend/src/terminal/panels/BacktestPanel.vue frontend/src/terminal/panels/CandlestickPanel.vue
git commit -m "fix(frontend): convert module-level state to instance-level ref in BacktestPanel and CandlestickPanel"
```

---

### Task 3: WebSocket Singleton Connection Pool

**Files:**
- Modify: `frontend/src/lib/composables/useWebSocket.ts`

- [ ] **Step 1: Write test**

```typescript
// frontend/src/lib/composables/__tests__/useWebSocket.test.ts
import { describe, it, expect, vi, beforeEach } from 'vitest'

describe('WebSocketManager', () => {
  beforeEach(() => {
    vi.stubGlobal('WebSocket', vi.fn().mockImplementation(() => ({
      close: vi.fn(),
      send: vi.fn(),
      readyState: WebSocket.OPEN,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
    })))
  })

  it('reuses connection for same URL', async () => {
    const { wsManager } = await import('@/lib/composables/useWebSocket')
    const unsub1 = wsManager.subscribe('ws://test', 'topic1', vi.fn())
    const unsub2 = wsManager.subscribe('ws://test', 'topic2', vi.fn())
    expect(vi.mocked(window.WebSocket)).toHaveBeenCalledTimes(1)
    unsub1()
    unsub2()
  })

  it('creates separate connection for different URL', async () => {
    const { wsManager } = await import('@/lib/composables/useWebSocket')
    wsManager.subscribe('ws://a', 't1', vi.fn())
    wsManager.subscribe('ws://b', 't2', vi.fn())
    expect(vi.mocked(window.WebSocket)).toHaveBeenCalledTimes(2)
  })
})
```

- [ ] **Step 2: Implement WebSocketManager singleton**

```typescript
// frontend/src/lib/composables/useWebSocket.ts
type TopicHandler = (data: any) => void

interface WSConnection {
  ws: WebSocket
  subscribers: Map<string, Set<WSHandler>>
  reconnectTimer?: ReturnType<typeof setTimeout>
}

interface WSHandler {
  topic: string
  handler: TopicHandler
}

class WebSocketManager {
  private connections = new Map<string, WSConnection>()
  private pendingQueue = new Map<string, string[]>() // url → topics[]
  private reconnectBase = 1000  // 1s initial backoff
  private reconnectMax = 30000  // 30s max

  subscribe(url: string, topic: string, handler: TopicHandler): () => void {
    let conn = this.connections.get(url)
    if (!conn) {
      try {
        const ws = new WebSocket(url)
        conn = { ws, subscribers: new Set() }
        this.connections.set(url, conn)

        ws.addEventListener('message', (event) => {
          try {
            const msg = JSON.parse(event.data)
            if (msg.topic && conn) {
              for (const sub of conn.subscribers) {
                if (sub.topic === msg.topic) sub.handler(msg.data)
              }
            }
          } catch { /* ignore parse errors */ }
        })

        ws.addEventListener('close', () => {
          if (conn) this.reconnect(url, conn)
        })
      } catch {
        // If WebSocket fails, return no-op unsubscriber
        return () => {}
      }
    }

    const handlerEntry: WSHandler = { topic, handler }
    conn.subscribers.add(handlerEntry)

    return () => {
      const c = this.connections.get(url)
      if (c) {
        c.subscribers.delete(handlerEntry)
        if (c.subscribers.size === 0) {
          c.ws.close()
          this.connections.delete(url)
        }
      }
    }
  }

  private reconnect(url: string, conn: WSConnection) {
    if (conn.reconnectTimer) return
    const delay = Math.min(this.reconnectBase * 2 ** this.connections.size, this.reconnectMax)
    conn.reconnectTimer = window.setTimeout(() => {
      try {
        const ws = new WebSocket(url)
        conn.ws = ws
        // reattach listeners...
      } catch {}
      conn.reconnectTimer = undefined
    }, delay)
  }
}

export const wsManager = new WebSocketManager()
```

- [ ] **Step 3: Run test**

```bash
cd frontend && npx vitest run src/lib/composables/__tests__/useWebSocket.test.ts
```
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add frontend/src/lib/composables/useWebSocket.ts frontend/src/lib/composables/__tests__/useWebSocket.test.ts
git commit -m "feat(frontend): add WebSocketManager singleton — N panels sharing same URL use 1 connection"
```

---

### Task 4: Store Request Dedup + Stale Response Guard

**Files:**
- Modify: `frontend/src/stores/portfolio.ts`
- Modify: `frontend/src/stores/ml.ts`
- Modify: `frontend/src/stores/data.ts`

- [ ] **Step 1: Add dedup composable or utility**

```typescript
// frontend/src/lib/useRequestGuard.ts
import { ref, type Ref } from 'vue'

const pendingRequests = new Map<string, Promise<any>>()
const requestSeqs = new Map<string, number>()

export function useRequestGuard() {
  function execute<T>(key: string, fetcher: () => Promise<T>): Promise<T> {
    const seq = (requestSeqs.get(key) || 0) + 1
    requestSeqs.set(key, seq)

    // Dedup: reuse in-flight request
    if (pendingRequests.has(key)) {
      return pendingRequests.get(key)!
    }

    const promise = fetcher().finally(() => {
      if (requestSeqs.get(key) === seq) {
        pendingRequests.delete(key)
      }
    })
    pendingRequests.set(key, promise)
    return promise
  }

  function isStale(key: string, seq: number): boolean {
    return (requestSeqs.get(key) || 0) > seq
  }

  function getSeq(key: string): number {
    return requestSeqs.get(key) || 0
  }

  return { execute, isStale, getSeq }
}
```

- [ ] **Step 2: Apply to portfolioStore.fetchPositions**

```typescript
// frontend/src/stores/portfolio.ts
const { execute: dedup, isStale, getSeq } = useRequestGuard()

const fetchPositions = async () => {
  const key = 'fetchPositions'
  const seq = getSeq(key) + 1

  const result = await dedup(key, async () => {
    const app = (window as any).go?.main?.App
    if (!app?.GetPositions) return []
    return app.GetPositions()
  })

  if (!isStale(key, seq)) {
    positions.value = result
  }
}
```

- [ ] **Step 3: Apply to dataStore.fetchMarketOverview**

```typescript
// frontend/src/stores/data.ts
const { execute: dedup, isStale, getSeq } = useRequestGuard()

const fetchMarketOverview = async () => {
  const key = 'fetchMarketOverview'
  const seq = getSeq(key) + 1

  const result = await dedup(key, async () => {
    const app = (window as any).go?.main?.App
    if (!app?.GetMarketOverview) return { indices: [], sectors: [] }
    return app.GetMarketOverview()
  })

  if (!isStale(key, seq)) {
    marketOverview.value = result
  }
}
```

- [ ] **Step 4: Run store tests**

```bash
cd frontend && npx vitest run src/stores/
```
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add frontend/src/lib/composables/useRequestGuard.ts frontend/src/stores/portfolio.ts frontend/src/stores/data.ts frontend/src/stores/ml.ts
git commit -m "feat(frontend): add request dedup and stale response guard — useRequestGuard composable applied to portfolio, data, ml stores"
```

---

### Task 5: BacktestPanel i18n

**Files:**
- Modify: `frontend/src/terminal/panels/BacktestPanel.vue`
- Modify: `frontend/src/lib/i18n/zh.ts`
- Modify: `frontend/src/lib/i18n/en.ts`

- [ ] **Step 1: Add i18n keys**

```typescript
// frontend/src/lib/i18n/zh.ts
backtest: {
  title: '回测',
  history: '回测历史',
  detail: '回测详情',
  start: '回测开始',
  end: '回测结束',
  strategy: '策略',
  symbol: '标的',
  engine: '引擎',
  totalReturn: '总收益率',
  cagr: '年化收益',
  maxDrawdown: '最大回撤',
  sharpe: '夏普比率',
  sortino: '索提诺比率',
  winRate: '胜率',
  profitFactor: '盈亏比',
  totalTrades: '总交易次数',
  clearAll: '清空全部',
  delete: '删除',
  noData: '暂无回测记录',
  loading: '加载中...',
  error: '加载失败',
  tradeList: '交易记录',
  equityCurve: '净值曲线',
  ohlcvChart: 'K线图',
  backToHistory: '← 返回列表',
}

// frontend/src/lib/i18n/en.ts
backtest: {
  title: 'Backtest',
  history: 'Backtest History',
  detail: 'Backtest Details',
  start: 'Start',
  end: 'End',
  strategy: 'Strategy',
  symbol: 'Symbol',
  engine: 'Engine',
  totalReturn: 'Total Return',
  cagr: 'CAGR',
  maxDrawdown: 'Max Drawdown',
  sharpe: 'Sharpe',
  sortino: 'Sortino',
  winRate: 'Win Rate',
  profitFactor: 'Profit Factor',
  totalTrades: 'Total Trades',
  clearAll: 'Clear All',
  delete: 'Delete',
  noData: 'No backtest records',
  loading: 'Loading...',
  error: 'Failed to load',
  tradeList: 'Trades',
  equityCurve: 'Equity Curve',
  ohlcvChart: 'OHLCV Chart',
  backTo: '← Back',
}
```

- [ ] **Step 2: Replace hardcoded strings in BacktestPanel.vue**

Search for all hardcoded Chinese strings in `BacktestPanel.vue` and replace with `$t('backtest.xxx')`.

Example replacements:
- `回测历史` → `{{ $t('backtest.history') }}`
- `回测开始` → `{{ $t('backtest.start') }}`
- `策略` → `{{ $t('backtest.strategy') }}`
- `标的` → `{{ $t('backtest.symbol') }}`
- `总收益率` → `{{ $t('backtest.totalReturn') }}`

Estimate ~30 replacements. The BacktestPanel.vue is large — process systematically, starting with the template section.

- [ ] **Step 3: Verify no hardcoded Chinese remains**

```bash
grep -n "[\u4e00-\u9fff].*:" frontend/src/terminal/panels/BacktestPanel.vue | grep -v "i18n\|t("
```
If any remain, replace them.

- [ ] **Step 4: Commit**

```bash
git add frontend/src/terminal/panels/BacktestPanel.vue frontend/src/lib/i18n/zh.ts frontend/src/lib/i18n/en.ts
git commit -m "feat(i18n): full BacktestPanel i18n coverage — zh + en, >30 keys"
```

---

### Task 6: Enable noUncheckedIndexedAccess

**Files:**
- Modify: `frontend/tsconfig.json`
- Modify: ~50-150 `.ts`/`.vue` files

- [ ] **Step 1: Enable the flag in tsconfig**

```json
// tsconfig.json compilerOptions:
"noUncheckedIndexedAccess": true
```

- [ ] **Step 2: Run tsc to surface errors**

```bash
cd frontend && npx vue-tsc --noEmit 2>&1 | tee /tmp/tsc_errors.txt
wc -l /tmp/tsc_errors.txt
```

- [ ] **Step 3: Fix by category**

Most common fix patterns:
```typescript
// arr[i] → arr[i]! (if known safe)
// arr[i].property → arr[i]?.property (if might be undefined)
// arr[i] && arr[i].property → arr[i]?.property

// Object keys access:
// obj[key] → obj[key as keyof typeof obj] or obj[key]!

// For map access:
// map.get(k) → already returns T | undefined — that's fine
```

Use `--auto-fix` if possible, otherwise manual fixes by file. Target 0 errors.

- [ ] **Step 4: Final verification**

```bash
cd frontend && npx vue-tsc --noEmit
```
Expected: 0 errors

- [ ] **Step 5: Commit**

```bash
git add frontend/tsconfig.json
# WARNING: may touch many files; add them all
git commit -m "feat(frontend): enable noUncheckedIndexedAccess strictness and fix all resulting type errors"
```

---

### Task 7: Update CHANGELOG

- [ ] **Step 1: Update CHANGELOG.md**

```markdown
### Added
- [Frontend] Type-safe Go bridge proxy `useGoApp()` — typed method calls, no more `(window as any).go?.main?.App?.xxx`
- [Frontend] WebSocketManager singleton — N panels sharing same URL → 1 connection
- [Frontend] useRequestGuard composable — request dedup + stale response guard applied to portfolio, data, ml stores
- [i18n] BacktestPanel fully localized (zh + en, ~30 keys)

### Fixed
- [Frontend] BacktestPanel and CandlestickPanel module-level state moved to instance-level refs (cross-instance contamination)
- [Frontend] Enable noUncheckedIndexedAccess in tsconfig — eliminated ~100+ hidden undefined bugs
```

- [ ] **Step 2: Commit**

```bash
git add CHANGELOG.md
git commit -m "docs: update CHANGELOG for frontend architecture improvements"
```