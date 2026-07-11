# CI/CD 修复与测试稳定性 — 实施计划

> Spec: [2026-07-11-ci-test-stabilization.md](../../specs/2026-07-11-ci-test-stabilization.md)

---

### Task 1: 修复 CI Go 版本

**Files:**
- Modify: `.github/workflows/ci.yml`

**Step 1: 修改 go-version**

```yaml
- uses: actions/setup-go@v5
  with: { go-version: '1.25' }
```

**Step 2: Commit**

```bash
git add .github/workflows/ci.yml
git commit -m "fix(ci): update go-version from 1.22 to 1.25 (matching go.mod)"
```

---

### Task 2: 建立统一 Mock 层 + 修复 i18n

**Files:**
- New: `frontend/src/__tests__/mocks.ts`
- Modify: `frontend/vitest.setup.ts`

**Step 1: 创建统一 Mock 文件**

`frontend/src/__tests__/mocks.ts`:

```typescript
import { vi } from 'vitest'

export function mockWailsIPC() {
  const app = {
    SearchSymbols: vi.fn().mockResolvedValue({ data: [] }),
    GetQuote: vi.fn().mockResolvedValue(null),
    FetchOHLCV: vi.fn().mockResolvedValue([]),
    GetMinuteLine: vi.fn().mockResolvedValue([]),
    GetMarketOverview: vi.fn<[string?], any>().mockResolvedValue({
      indices: [{ name: '上证指数', last: 3000, changePct: 0.5 }],
      breadth: { advancers: 1500, decliners: 500 },
      sectors: [{ name: '科技', changePct: 1.2 }],
    }),
    GetIndustryRanks: vi.fn().mockResolvedValue([]),
    ListNodes: vi.fn().mockResolvedValue([]),
    GetNodePorts: vi.fn().mockResolvedValue({ inputs: [], outputs: [] }),
    GetAbnormalStocks: vi.fn().mockResolvedValue([]),
    GetRealtimeDepth: vi.fn().mockResolvedValue(null),
    GetFundingRate: vi.fn().mockResolvedValue([]),
    FetchBacktest: vi.fn().mockResolvedValue(null),
    GetPortfolioSummary: vi.fn().mockResolvedValue(null),
    GetOrders: vi.fn().mockResolvedValue({ orders: [], total: 0 }),
    GetTrades: vi.fn().mockResolvedValue({ trades: [], total: 0 }),
    GetEquityCurve: vi.fn().mockResolvedValue(null),
    CancelOrder: vi.fn().mockResolvedValue(true),
    GetBrokerStatus: vi.fn().mockResolvedValue([]),
    GetEvents: vi.fn().mockResolvedValue([]),
    DismissEvent: vi.fn().mockResolvedValue(true),
    ApproveEvent: vi.fn().mockResolvedValue(true),
    SearchSymbols: vi.fn().mockResolvedValue({ data: [] }),
    GetMinuteLineAsync: vi.fn().mockResolvedValue([]),
  }
  ;(window as any).go = { main: { App: app } }
  return app
}

export function mockWebSocket() {
  class MockWebSocket {
    readyState = WebSocket.OPEN
    send = vi.fn()
    close = vi.fn()
    addEventListener = vi.fn()
    removeEventListener = vi.fn()
  }
  vi.stubGlobal('WebSocket', MockWebSocket as any)
}

export function mockI18n() {
  vi.mock('vue-i18n', async () => {
    const actual = await vi.importActual('vue-i18n')
    return {
      ...(actual as any),
      useI18n: () => ({
        t: (key: string) => key.split('.').pop() || key,
        locale: { value: 'zh-CN' },
        availableLocales: ['zh-CN', 'en'],
      }),
    }
  })
  // Also mock the app's i18n instance
  vi.mock('@/lib/i18n', () => ({
    i18n: {
      global: {
        t: (key: string) => key.split('.').pop() || key,
      },
    },
  }))
}
```

**Step 2: 更新 vitest.setup.ts**

```typescript
import { vi } from 'vitest'
import { mockWailsIPC, mockWebSocket, mockI18n } from './src/__tests__/mocks'

// Apply global mocks
mockWailsIPC()
mockWebSocket()
mockI18n()

// Prevent Wails drag.js window reference after teardown
const originalSetTimeout = global.setTimeout
global.setTimeout = ((fn: any, ms: any, ...args: any[]) => {
  if (typeof fn === 'string' && fn.includes('window')) return 0
  if (fn.toString().includes('window')) return 0
  return originalSetTimeout(fn, ms, ...args)
}) as any

// Global mocks for test environment
global.ResizeObserver = class {
  observe() {}
  unobserve() {}
  disconnect() {}
} as any
```

**Step 3: Remove old mock code from vitest.setup.ts** (remove old `;(window as any).go = ...` block)

**Step 4: Update package.json to include mocks.ts path resolution**

Check `frontend/vite.config.ts` for alias config — mocks.ts needs `@/__tests__/mocks` alias to work.

**Step 5: Commit**

```bash
git add frontend/src/__tests__/mocks.ts frontend/vitest.setup.ts
git commit -m "test(frontend): unified mock layer for Wails IPC, WebSocket, and i18n"
```

---

### Task 3: 修复 Go flaky 测试

**Files:**
- Modify: `internal/market/poller_test.go`

**Step 1: Replace sleep with poll loop**

In `TestQuotePoller_FetchesAndPublishesData`, replace:
```go
time.Sleep(60 * time.Millisecond)
```
with polling loop:
```go
var msg *MarketMessage
var ok bool
for i := 0; i < 50; i++ {
    msg, ok = marketHub.GetLatest("market:quote:CN:600519")
    if ok {
        break
    }
    time.Sleep(10 * time.Millisecond)
}
```

**Step 2: Run flaky test 10 times to confirm fix**

```bash
for i in $(seq 1 10); do
  go test ./internal/market -count=1 -run TestQuotePoller_FetchesAndPublishesData -v 2>&1 | tail -3
done
```

**Step 3: Commit**

```bash
git add internal/market/poller_test.go
git commit -m "fix(market): replace sleep with poll loop in flaky TestQuotePoller test"
```

---

### Task 4: 修复 Store 测试

**Files:**
- Modify: `frontend/src/stores/data.test.ts`
- Modify: `frontend/src/stores/portfolio.test.ts`

**Step 1: Fix data.test.ts — the mock now returns proper MarketOverview**

The mock in `mocks.ts` already returns proper structure, so `data.test.ts:50` should pass. But the store's `fetchMarketOverview` method may handle the result differently. Check the store and adjust if needed.

**Step 2: Fix portfolio.test.ts — update mock expectations**

The portfolio store tests fail because `GetOrders` and `GetTrades` now return `{ orders: [], total: 0 }` format. Update test assertions to match.

**Step 3: Run store tests**

```bash
cd frontend && npx vitest run src/stores/
```

**Step 4: Commit**

```bash
git add frontend/src/stores/
git commit -m "test(frontend): fix store tests for unified mock layer"
```

---

### Task 5: 修复 Panel 测试

**Files:**
- All 22 failing panel test files

**Step 1: Fix i18n-dependent tests**

Many panel tests check for English text but i18n mock returns key suffix. Update assertions:
- `'Paper Trading'` → matches i18n key suffix
- `'Basket Summary'` → update  
- `'Execute Basket'` → update

**Step 2: Run panel tests and fix individually**

```bash
cd frontend && npx vitest run src/terminal/panels/__tests__/
```

**Step 3: Fix ActionCenterPanel.test.ts** — mock `GetEvents` to return proper data

**Step 4: Fix BrokerStatusPanel.test.ts** — broker cards rendered with i18n text

**Step 5: Fix remaining panels** — similar patterns

**Step 6: Commit**

```bash
git add frontend/src/terminal/panels/__tests__/
git commit -m "test(frontend): fix panel tests for unified mock + i18n"
```

---

### Task 6: CI 增强 + CHANGELOG

**Files:**
- Modify: `.github/workflows/ci.yml` — add `--bail=5` to vitest
- Modify: `CHANGELOG.md`

**Step 1: Add --bail=5 to CI**

```yaml
- run: cd frontend && npx vitest run --bail=5
```

**Step 2: Final verification**

```bash
cd frontend && npx vitest run
go test ./internal/market -count=1 -run TestQuotePoller -v
```

**Step 3: Update CHANGELOG + Commit**

```bash
git add .github/workflows/ci.yml CHANGELOG.md
git commit -m "chore: enhance CI with --bail=5, update CHANGELOG"
```
