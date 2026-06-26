# 前端质量全面整改 — 实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 消灭所有静默吞错、缺失 loading/空状态、Store 绕过、localStorage 混乱，建立统一数据获取标准。

**Architecture:** 新建 `useDataFetch` composable 统一 `{ data, loading, error }` 三元组，逐批接入面板和 Store。分 3 个 Phase：Phase 1 止血（1 个新文件 + 6 个面板），Phase 2 规范化（32 个面板），Phase 3 重构（10 个 Store + 5 个 BUG 修复）。

**Tech Stack:** Vue 3 Composition API, Pinia, TypeScript, Wails v3 IPC

## Global Constraints

- 所有数据获取统一走 `useDataFetch` composable 或至少加 `console.error`
- 模板标准：`loading → error → empty → data` 四态覆盖
- `catch {}` / `catch { /* silent */ }` 一律改为有日志
- 面板不再绕过 Store 直接调 Go IPC
- npm run build 每次改完后通过

---

### Task 1: 创建 useDataFetch composable

**Files:**
- Create: `frontend/src/lib/composables/useDataFetch.ts`

**Interfaces:**
- Produces: `useDataFetch<T>(fetcher: () => Promise<T>)` → `{ data, loading, error, execute }`

- [ ] **Step 1: 创建文件**

```typescript
// frontend/src/lib/composables/useDataFetch.ts
import { ref } from 'vue'

/**
 * Universal data fetching composable with loading/error/data triad.
 *
 * Usage:
 *   const { data, loading, error, execute } = useDataFetch(() => app.GetQuote(symbol))
 *   onMounted(() => execute())
 */
export function useDataFetch<T>(fetcher: () => Promise<T>) {
  const data = ref<T | null>(null) as ReturnType<typeof ref<T | null>>
  const loading = ref(false)
  const error = ref<string | null>(null)

  async function execute() {
    loading.value = true
    error.value = null
    try {
      data.value = await fetcher()
    } catch (e) {
      const msg = e instanceof Error ? e.message : String(e)
      error.value = msg
      console.error('[useDataFetch]', msg)
    } finally {
      loading.value = false
    }
  }

  return { data, loading, error, execute }
}
```

- [ ] **Step 2: 构建验证**

```bash
cd /Volumes/etx/coding/rebuild/quantflow/frontend && npm run build -q 2>&1 | tail -1
```

预期: `✓ built in X.XXs`

- [ ] **Step 3: Commit**

```bash
git add frontend/src/lib/composables/useDataFetch.ts
git commit -m "feat: add useDataFetch composable with loading/error/data triad"
```

---

### Task 2: 修复 SystemMonitorPanel（Critical — 空 catch + 无 loading）

**Files:**
- Modify: `frontend/src/terminal/panels/SystemMonitorPanel.vue`

**Interfaces:**
- Consumes: `useDataFetch` from Task 1
- Changes: ref-based stats → useDataFetch, template 加 loading/error/empty

- [ ] **Step 1: 替换数据获取逻辑**

将 `onMounted` + `setInterval` 中手动的 stats ref 替换为 `useDataFetch`：

```typescript
import { useDataFetch } from '@/lib/composables/useDataFetch'

const statsFetcher = useDataFetch(async () => {
  return await (window as any).go.main.App.GetSystemStats()
})
```

替换轮询逻辑：

```typescript
onMounted(() => {
  statsFetcher.execute()
  timer = setInterval(() => statsFetcher.execute(), 5000)
})
onUnmounted(() => clearInterval(timer))
```

删除旧的 `catch {}` 块。

- [ ] **Step 2: 更新模板加 loading/error/empty 状态**

在模板数据区域前添加：

```html
<div v-if="statsFetcher.loading.value" class="stat-loading">...</div>
<div v-else-if="statsFetcher.error.value" class="stat-error">错误: {{ statsFetcher.error.value }}</div>
<div v-else-if="!statsFetcher.data.value" class="stat-empty">--</div>
<div v-else class="stats-grid">
  <!-- existing stat items using statsFetcher.data.value -->
</div>
```

- [ ] **Step 3: 构建验证**

```bash
cd /Volumes/etx/coding/rebuild/quantflow/frontend && npm run build -q 2>&1 | tail -1
```

- [ ] **Step 4: Commit**

```bash
git add frontend/src/terminal/panels/SystemMonitorPanel.vue
git commit -m "fix: SystemMonitorPanel — useDataFetch + loading/error states"
```

---

### Task 3: 修复 TickerTapePanel + CryptoOverviewPanel（Critical — 无 loading + silent catch）

**Files:**
- Modify: `frontend/src/terminal/panels/TickerTapePanel.vue`
- Modify: `frontend/src/terminal/panels/CryptoOverviewPanel.vue`

**Interfaces:**
- Consumes: `useDataFetch` from Task 1

- [ ] **Step 1: TickerTapePanel — 接入 useDataFetch**

```typescript
// 替换 onMounted 中逐个获取行情的逻辑
import { useDataFetch } from '@/lib/composables/useDataFetch'

const quotesFetcher = useDataFetch(async () => {
  const symbols = ['000001', '600519', '300750', '002594', '601318', '600036', '000858', '601857']
  const results: Record<string, any> = {}
  for (const sym of symbols) {
    try {
      results[sym] = await (window as any).go.main.App.GetQuote('CN', sym)
    } catch { /* skip failed symbols */ }
  }
  return results
})

onMounted(() => quotesFetcher.execute())
```

模板底部添加：

```html
<Marquee v-if="quotesFetcher.loading.value" class="ticker-loading">加载行情中...</Marquee>
<div v-else-if="quotesFetcher.data.value" class="ticker-wrap">
  <!-- existing marquee content using quotesFetcher.data.value -->
</div>
```

- [ ] **Step 2: CryptoOverviewPanel — 接入 useDataFetch**

```typescript
import { useDataFetch } from '@/lib/composables/useDataFetch'

const cryptoFetcher = useDataFetch(async () => {
  return await (window as any).go.main.App.GetCryptoOverview()
})

onMounted(() => cryptoFetcher.execute())
```

删除旧的 `catch { /* silent */ }` 块。

模板标准四态：

```html
<div v-if="cryptoFetcher.loading.value">加载中...</div>
<div v-else-if="cryptoFetcher.error.value">错误: {{ cryptoFetcher.error.value }}</div>
<div v-else-if="!cryptoFetcher.data.value?.length">暂无数据</div>
<div v-else><!-- existing table --></div>
```

- [ ] **Step 3: 构建验证**

```bash
cd /Volumes/etx/coding/rebuild/quantflow/frontend && npm run build -q 2>&1 | tail -1
```

- [ ] **Step 4: Commit**

```bash
git add frontend/src/terminal/panels/TickerTapePanel.vue frontend/src/terminal/panels/CryptoOverviewPanel.vue
git commit -m "fix: TickerTape + CryptoOverview — useDataFetch + loading/error/empty states"
```

---

### Task 4: 修复静默 catch 的面板（MarketDepth + PredictionMarket + Candlestick 分钟 + Watchlist localStorage）

**Files:**
- Modify: `frontend/src/terminal/panels/MarketDepthPanel.vue`
- Modify: `frontend/src/terminal/panels/PredictionMarketPanel.vue`
- Modify: `frontend/src/terminal/panels/CandlestickPanel.vue`
- Modify: `frontend/src/terminal/panels/WatchlistPanel.vue`

- [ ] **Step 1: 替换所有 `// silent` / `/* silent */` catch 块**

**MarketDepthPanel.vue** — 找到 `} catch { // silent }`，替换为：

```typescript
} catch (e) {
  console.error('[MarketDepth] fetch failed:', e)
}
```

**PredictionMarketPanel.vue** — 找到 `} catch { /* no signal available */ }`，替换为：

```typescript
} catch (e) {
  console.error('[PredictionMarket] fetch failed:', e)
}
```

**CandlestickPanel.vue** — 找到 `} catch { // silent }`（minute 部分），替换为：

```typescript
} catch (e) {
  console.error('[Candlestick] minute fetch failed:', e)
}
```

**WatchlistPanel.vue** — 找到 localStorage JSON.parse 的 `} catch {}`，替换为：

```typescript
} catch (e) {
  console.error('[Watchlist] localStorage parse failed:', e)
}
```

- [ ] **Step 2: 构建验证**

```bash
cd /Volumes/etx/coding/rebuild/quantflow/frontend && npm run build -q 2>&1 | tail -1
```

- [ ] **Step 3: Commit**

```bash
git add frontend/src/terminal/panels/MarketDepthPanel.vue frontend/src/terminal/panels/PredictionMarketPanel.vue frontend/src/terminal/panels/CandlestickPanel.vue frontend/src/terminal/panels/WatchlistPanel.vue
git commit -m "fix: replace silent catches with console.error in 4 panels"
```

---

### Task 5: 修复 14 个静默重置无日志的面板

**Files:** SatellitePanel, GeopoliticsPanel, GovDataPanel, NewsPanel, SchedulePanel, PositionPanel, DrawingPanel, BrokerStatusPanel, SurfaceChartPanel（共 9 个文件，14 处 catch）

- [ ] **Step 1: 批量替换 catch 块**

以下面板的每个 catch 块都加入 `console.error`：

| 文件 | catch 行数 | 改动 |
|------|-----------|------|
| `SatellitePanel.vue` | 2 处 | `data.value = []` → `console.error('[Satellite] ...'); data.value = []` |
| `GeopoliticsPanel.vue` | 2 处 | 同上模式 |
| `GovDataPanel.vue` | 3 处 | 同上模式 |
| `NewsPanel.vue` | 1 处 | 同上模式 |
| `SchedulePanel.vue` | 1 处 | 同上模式 |
| `PositionPanel.vue` | 1 处 | 同上模式 |
| `DrawingPanel.vue` | 1 处 | 同上模式 |
| `BrokerStatusPanel.vue` | 1 处 | 同上模式 |
| `SurfaceChartPanel.vue` | 1 处 | 同上模式 |

每个 catch 统一格式：

```typescript
} catch (e) {
  console.error('[PanelName] method failed:', e)
  data.value = null  // or [] or as appropriate
}
```

- [ ] **Step 2: 构建验证**

```bash
cd /Volumes/etx/coding/rebuild/quantflow/frontend && npm run build -q 2>&1 | tail -1
```

- [ ] **Step 3: Commit**

```bash
git add frontend/src/terminal/panels/SatellitePanel.vue frontend/src/terminal/panels/GeopoliticsPanel.vue frontend/src/terminal/panels/GovDataPanel.vue frontend/src/terminal/panels/NewsPanel.vue frontend/src/terminal/panels/SchedulePanel.vue frontend/src/terminal/panels/PositionPanel.vue frontend/src/terminal/panels/DrawingPanel.vue frontend/src/terminal/panels/BrokerStatusPanel.vue frontend/src/terminal/panels/SurfaceChartPanel.vue
git commit -m "fix: add console.error to 14 silent catch blocks across 9 panels"
```

---

### Task 6: 修复 7 个内容区缺 loading 的面板

**Files:** StockResearchPanel, CongressTradingPanel, AnalystEstimatesPanel, FinancialsPanel, InsiderTradingPanel, PeerComparisonPanel, ModelRegistryPanel

- [ ] **Step 1: 统一加内容区 loading 指示器**

每个面板在数据区域前加：

```html
<div v-if="store.loading" class="loading-placeholder">
  {{ $t('common.loading') }}
</div>
```

以 `StockResearchPanel.vue` 为例，在现有内容前插入：

```html
<div v-if="store.loading" class="chart-fallback">{{ $t('common.loading') }}</div>
<template v-else-if="store.error">
  <div class="chart-fallback">错误: {{ store.error }}</div>
</template>
<template v-else>
  <!-- existing content -->
</template>
```

其余 6 个面板同样模式。

- [ ] **Step 2: 构建验证**

```bash
cd /Volumes/etx/coding/rebuild/quantflow/frontend && npm run build -q 2>&1 | tail -1
```

- [ ] **Step 3: Commit**

```bash
git add frontend/src/terminal/panels/StockResearchPanel.vue frontend/src/terminal/panels/CongressTradingPanel.vue frontend/src/terminal/panels/AnalystEstimatesPanel.vue frontend/src/terminal/panels/FinancialsPanel.vue frontend/src/terminal/panels/InsiderTradingPanel.vue frontend/src/terminal/panels/PeerComparisonPanel.vue frontend/src/terminal/panels/ModelRegistryPanel.vue
git commit -m "fix: add loading indicators to 7 panel content areas"
```

---

### Task 7: 修复 5 个缺空状态的面板

**Files:** TickerTapePanel, CryptoOverviewPanel, SystemMonitorPanel, ModelRegistryPanel, TradeHistory

- [ ] **Step 1: 加空状态消息**

```html
<!-- TickerTapePanel — 无行情数据时 -->
<div v-if="!quotesFetcher.data.value" class="empty-state">{{ $t('common.no_data') }}</div>

<!-- CryptoOverviewPanel — 已在 Task 3 修复，确认空状态 -->
<!-- SystemMonitorPanel — 已在 Task 2 修复，确认空状态 -->
<!-- ModelRegistryPanel — 表格下方 -->
<tr v-if="filteredModels.length === 0">
  <td colspan="5" class="empty-row">{{ $t('common.no_data') }}</td>
</tr>

<!-- TradeHistory — 替换 `--` -->
<td v-if="!item" colspan="7" class="empty-row">{{ $t('common.no_data') }}</td>
```

- [ ] **Step 2: 构建验证**

```bash
cd /Volumes/etx/coding/rebuild/quantflow/frontend && npm run build -q 2>&1 | tail -1
```

- [ ] **Step 3: Commit**

```bash
git add frontend/src/terminal/panels/TickerTapePanel.vue frontend/src/terminal/panels/CryptoOverviewPanel.vue frontend/src/terminal/panels/SystemMonitorPanel.vue frontend/src/terminal/panels/ModelRegistryPanel.vue frontend/src/terminal/panels/TradeHistory.vue
git commit -m "fix: add empty state messages to 5 panels"
```

---

### Task 8: 修复 6 个绕过 Store 的面板

**Files:** PositionPanel, QuoteDetailPanel, CandlestickPanel, WatchlistPanel, BrokerStatusPanel, SchedulePanel

- [ ] **Step 1: 改为走 Store 方法**

**PositionPanel.vue** — 替换直接调用为 store 方法：

```typescript
// Before:
const result = await (window as any).go.main.App.GetPositions()
// After:
import { usePortfolioStore } from '@/stores/portfolio'
const posStore = usePortfolioStore()
posStore.fetchPositions()
const positions = computed(() => posStore.positions)
```

**QuoteDetailPanel.vue** — 改为走 data store：

```typescript
// Before:
const result = await (window as any).go.main.App.GetQuote(market, symbol)
// After:
import { useDataStore } from '@/stores/data'
const dataStore = useDataStore()
dataStore.fetchQuote(market, symbol)
```

**WatchlistPanel.vue** — 走 data store 的行情批量方法。

**BrokerStatusPanel.vue** — 走 portfolio store。

**SchedulePanel.vue** — 走 notify store 或 terminal store 的 schedule 方法。

**CandlestickPanel.vue** — OHLCV 部分走 data store。

- [ ] **Step 2: 构建验证**

```bash
cd /Volumes/etx/coding/rebuild/quantflow/frontend && npm run build -q 2>&1 | tail -1
```

- [ ] **Step 3: Commit**

```bash
git add frontend/src/terminal/panels/PositionPanel.vue frontend/src/terminal/panels/QuoteDetailPanel.vue frontend/src/terminal/panels/CandlestickPanel.vue frontend/src/terminal/panels/WatchlistPanel.vue frontend/src/terminal/panels/BrokerStatusPanel.vue frontend/src/terminal/panels/SchedulePanel.vue
git commit -m "fix: route 6 panels through stores instead of direct IPC"
```

---

### Task 9: Store 层 — 统一加 error 状态 + IPC 风格统一

**Files:**
- Modify: `frontend/src/stores/data.ts`, `ml.ts`, `notify.ts`, `portfolio.ts`, `research.ts`

- [ ] **Step 1: 每个 Store 加 error ref**

为每个异步方法添加独立的 error 状态：

```typescript
// data.ts
const error = ref<string | null>(null)
// 每个 fetch 方法中:
error.value = null
try {
  // ... fetch ...
} catch (e) {
  error.value = e instanceof Error ? e.message : String(e)
  console.error('[DataStore]', e)
}
```

`ml.ts`, `notify.ts`, `portfolio.ts`, `research.ts` 同样模式。

- [ ] **Step 2: 统一 IPC 调用风格**

所有 Store 统一为风格 B（可选链检查）：

```typescript
const app = (window as any).go?.main?.App
if (!app) { error.value = 'Wails bridge not available'; return }
```

删除风格 A（直接裸调用）和风格 C（checkBridge 预检）的不一致使用。

- [ ] **Step 3: 构建验证**

```bash
cd /Volumes/etx/coding/rebuild/quantflow/frontend && npm run build -q 2>&1 | tail -1
```

- [ ] **Step 4: Commit**

```bash
git add frontend/src/stores/data.ts frontend/src/stores/ml.ts frontend/src/stores/notify.ts frontend/src/stores/portfolio.ts frontend/src/stores/research.ts
git commit -m "fix: add error state to all stores + unify IPC calling style"
```

---

### Task 10: Store 层 — 修复 5 个 BUG

**Files:**
- Modify: `frontend/src/stores/symbolContext.ts` — panelGroups 泄漏
- Modify: `frontend/src/stores/terminal.ts` — closeTab + instanceId + 布局持久化
- Modify: `frontend/src/stores/session.ts` + `frontend/src/lib/theme.ts` — localStorage 重叠
- Modify: `frontend/src/stores/settings.ts` — 类型安全

- [ ] **Step 1: 修复 symbolContext panelGroups 泄漏**

```typescript
// symbolContext.ts — 添加清理函数
function releasePanelGroup(panelId: string) {
  delete panelGroups[panelId]
}
// 面板 onUnmounted 中调用:
// const ctx = useSymbolContext()
// onUnmounted(() => ctx.releasePanelGroup(props.panelId))
```

- [ ] **Step 2: 合并 theme/session localStorage**

删除 `theme.ts` 中独立的 localStorage 读写，改为从 `session.ts` 读取。theme.ts 变成纯计算函数：

```typescript
// theme.ts
export function useThemeStore() {
  const session = useSessionStore()
  const themeName = computed(() => session.theme)
  // ... apply logic ...
}
```

删除 `App.vue` 中 2s 轮询 `setInterval(() => t.apply(), 2000)`。

- [ ] **Step 3: 修复 terminal store**

```typescript
// closeTab — 改为从指定 leaf 开始搜索
function closeTab(leafId: string, tabId: string) {
  function removeFrom(n: DockLayoutTree): boolean {
    if (n.id === leafId && n.type === 'tab') { /* only search this leaf */ }
    // ...
  }
}

// instanceId — 改用 crypto.randomUUID()
const instanceId = `${panelId}-${crypto.randomUUID().slice(0, 6)}`

// 布局持久化
function persistLayout() {
  localStorage.setItem('quantflow-layout', JSON.stringify(layout))
}
// watch layout changes → persistLayout()
// on store init → try restore from localStorage
```

- [ ] **Step 4: settings 类型安全**

```typescript
function update<K extends keyof SettingsState>(key: K, value: SettingsState[K]) {
  settings.value[key] = value
  save()
}
```

- [ ] **Step 5: 构建验证**

```bash
cd /Volumes/etx/coding/rebuild/quantflow/frontend && npm run build -q 2>&1 | tail -1
```

- [ ] **Step 6: Commit**

```bash
git add frontend/src/stores/symbolContext.ts frontend/src/stores/terminal.ts frontend/src/stores/session.ts frontend/src/lib/theme.ts frontend/src/App.vue frontend/src/stores/settings.ts
git commit -m "fix: 5 store bugs — panelGroups leak, localStorage overlap, closeTab, layout persist, type safety"
```

---

### Task 11: 全栈打包验证

- [ ] **Step 1: 前端构建**

```bash
cd /Volumes/etx/coding/rebuild/quantflow/frontend && npm run build -q 2>&1 | tail -1
```

预期: `✓ built in X.XXs`

- [ ] **Step 2: Go 构建**

```bash
cd /Volumes/etx/coding/rebuild/quantflow && go build -o build/quantflow . 2>&1 | grep -v "ld: warning"
```

预期: 无错误

- [ ] **Step 3: Commit**

```bash
git add -A
git commit -m "feat: complete frontend quality redesign — Phase 1-3"
```
