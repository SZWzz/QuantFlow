# Phase 6: Frontend Panels — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build 7 new Vue 3 panels, 2 Pinia stores, and enhance OrderEntryPanel to complete the Phase 5 frontend integration.

**Architecture:** Independent panels following existing dark-theme patterns, direct Wails IPC calls to Go backend, Pinia stores for shared state (notifications, portfolio cache), ECharts for charts.

**Tech Stack:** Vue 3 Composition API + TypeScript + Pinia + ECharts + vue-echarts

---

## Task 1: Foundation — CSV Utility + Pinia Stores

**Files:**
- Create: `frontend/src/lib/export.ts`
- Create: `frontend/src/stores/portfolio.ts`
- Create: `frontend/src/stores/notify.ts`

### Step 1: Create CSV export utility

Create `frontend/src/lib/export.ts`:

```typescript
export function exportCSV(filename: string, headers: string[], rows: string[][]): void {
  const csvContent = [
    headers.join(','),
    ...rows.map(row => row.map(cell => `"${String(cell).replace(/"/g, '""')}"`).join(','))
  ].join('\n')
  const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8;' })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = filename
  link.click()
  URL.revokeObjectURL(url)
}
```

### Step 2: Create portfolioStore

Create `frontend/src/stores/portfolio.ts`:

```typescript
import { defineStore } from 'pinia'
import { ref } from 'vue'

export interface PortfolioSummary {
  total_value: number; cash_balance: number; market_value: number
  total_pnl: number; total_pnl_pct: number
}
export interface Allocation {
  by_market: Record<string, number>; by_sector: Record<string, number>; by_currency: Record<string, number>
}
export interface PositionDetail {
  symbol: string; quantity: number; avg_price: number; market_price: number
  pnl: number; pnl_pct: number; market: string; currency: string; cost_basis: number; alloc_pct: number
}

export const usePortfolioStore = defineStore('portfolio', () => {
  const summary = ref<PortfolioSummary | null>(null)
  const allocation = ref<Allocation | null>(null)
  const positions = ref<PositionDetail[]>([])

  async function fetchSummary() {
    try { summary.value = await (window as any).go.main.App.GetPortfolioSummary() }
    catch (e) { console.warn('GetPortfolioSummary not available:', e) }
  }
  async function fetchAllocation() {
    try { allocation.value = await (window as any).go.main.App.GetPortfolioAllocation() }
    catch (e) { console.warn('GetPortfolioAllocation not available:', e) }
  }
  async function fetchPositions() {
    try { positions.value = await (window as any).go.main.App.GetPositions() }
    catch (e) { console.warn('GetPositions not available:', e) }
  }

  let timer: ReturnType<typeof setInterval> | null = null
  function startAutoRefresh() {
    fetchSummary(); fetchAllocation(); fetchPositions()
    timer = setInterval(() => { fetchSummary(); fetchAllocation(); fetchPositions() }, 10000)
  }
  function stopAutoRefresh() { if (timer) { clearInterval(timer); timer = null } }
  return { summary, allocation, positions, fetchSummary, fetchAllocation, fetchPositions, startAutoRefresh, stopAutoRefresh }
})
```

### Step 3: Create notifyStore

Create `frontend/src/stores/notify.ts`:

```typescript
import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

export interface Notification {
  id: number; level: string; title: string; body: string; metadata: string; is_read: boolean; created_at: string
}

export const useNotifyStore = defineStore('notify', () => {
  const notifications = ref<Notification[]>([])
  const unreadCount = ref(0)
  const levelFilter = ref<string>('all')

  async function fetchNotifications(limit = 50, offset = 0) {
    try {
      const result = await (window as any).go.main.App.GetNotifications(limit, offset)
      if (result) {
        notifications.value = result
        unreadCount.value = result.filter((n: Notification) => !n.is_read).length
      }
    } catch (e) { console.warn('GetNotifications not available:', e) }
  }

  async function markRead(id: number) {
    try {
      await (window as any).go.main.App.MarkNotificationRead(id)
      const n = notifications.value.find(x => x.id === id)
      if (n && !n.is_read) { n.is_read = true; unreadCount.value-- }
    } catch (e) { console.warn('MarkNotificationRead not available:', e) }
  }

  async function markAllRead() {
    for (const n of notifications.value) { if (!n.is_read) { await markRead(n.id) } }
    unreadCount.value = 0
  }

  const filteredNotifications = computed(() => {
    if (levelFilter.value === 'all') return notifications.value
    return notifications.value.filter(n => n.level === levelFilter.value)
  })
  function setFilter(level: string) { levelFilter.value = level }
  return { notifications, unreadCount, levelFilter, filteredNotifications, fetchNotifications, markRead, markAllRead, setFilter }
})
```

### Step 4: Verify + Commit

```bash
cd /Volumes/etx/coding/rebuild/quantflow/frontend && npx vue-tsc --noEmit 2>&1 | head -5
git add frontend/src/lib/export.ts frontend/src/stores/portfolio.ts frontend/src/stores/notify.ts
git commit -m "feat(frontend): add CSV export utility, portfolioStore, and notifyStore"
```

---

## Task 2: PortfolioSummary Panel

**Files:**
- Create: `frontend/src/terminal/panels/PortfolioSummary.vue`

Create the panel with:
- 5 KPI cards row (Total Value, Cash, Market Value, Total P&L, P&L%)
- Left: ECharts equity curve (30d line area chart)
- Right: ECharts allocation pie chart (by market with CN=#d32f2f, HK=#1976d2, US=#388e3c, CRYPTO=#f57c00 colors)
- Bottom: positions table (Symbol, Market, Qty, Avg$, Mkt$, P&L, %, Alloc)
- Dark theme matching existing panels (#1a1a2e bg, #16213e cards, #0f2137 inputs)
- Mock data for development, auto-refresh every 10s

```bash
cd /Volumes/etx/coding/rebuild/quantflow/frontend && npx vue-tsc --noEmit 2>&1 | head -5
git add frontend/src/terminal/panels/PortfolioSummary.vue
git commit -m "feat(frontend): add PortfolioSummary panel with KPI cards, equity curve, and allocation pie"
```

---

## Task 3: PositionDetail Panel

**Files:**
- Create: `frontend/src/terminal/panels/PositionDetail.vue`

Create the panel with:
- Header: Symbol + market badge + currency
- 2x3 KPI grid (Quantity, Avg Price, Market Price, Market Value, P&L$, P&L%, Alloc%)
- ECharts price history chart (line + cost basis dashed reference line)
- P&L summary at bottom
- Accepts `params?.symbol` to pre-fill the symbol

```bash
cd /Volumes/etx/coding/rebuild/quantflow/frontend && npx vue-tsc --noEmit 2>&1 | head -5
git add frontend/src/terminal/panels/PositionDetail.vue
git commit -m "feat(frontend): add PositionDetail panel with price chart and KPI grid"
```

---

## Task 4: RiskDashboard Panel

**Files:**
- Create: `frontend/src/terminal/panels/RiskDashboard.vue`

Create the panel with:
- 2x3 KPI card grid (VaR95, CVaR95, MaxDrawdown, Sharpe, Sortino, AnnVol)
- Cards with left-border color indicator (VaR=#f0883e, Sharpe>1=#3fb950, MaxDD<-10=#f85149)
- ECharts drawdown curve (red fill area chart)
- Drawdown peak-to-trough date info

```bash
cd /Volumes/etx/coding/rebuild/quantflow/frontend && npx vue-tsc --noEmit 2>&1 | head -5
git add frontend/src/terminal/panels/RiskDashboard.vue
git commit -m "feat(frontend): add RiskDashboard with VaR/CVaR/MaxDD/Sharpe KPI cards and drawdown chart"
```

---

## Task 5: TradeHistory Panel

**Files:**
- Create: `frontend/src/terminal/panels/TradeHistory.vue`

Create the panel with:
- Filters: symbol search + Trades/Orders tab switch + status dropdown + CSV export button
- Trades table: Date, Symbol, Side(green/red), Qty, Price, Total, OrderID
- Orders table: Placed, Symbol, Side, Type, Qty/Filled, Price, Status (colored badges)
- CSV export using `exportCSV()` from lib/export.ts

```bash
cd /Volumes/etx/coding/rebuild/quantflow/frontend && npx vue-tsc --noEmit 2>&1 | head -5
git add frontend/src/terminal/panels/TradeHistory.vue
git commit -m "feat(frontend): add TradeHistory panel with trades/orders tabs, filters, and CSV export"
```

---

## Task 6: SchedulePanel

**Files:**
- Create: `frontend/src/terminal/panels/SchedulePanel.vue`

Create the panel with:
- Task list rows (Name, CronExpr in monospace, LastStatus ✅/❌, ON/OFF toggle, Delete)
- New Task modal with: Name input, Cron expression + 5 preset buttons (Every Hour, Daily 9:00, Weekdays 9:25, Every 5min, Weekly), Workflow ID input, Timeout input
- Cancel/Save modal actions

```bash
cd /Volumes/etx/coding/rebuild/quantflow/frontend && npx vue-tsc --noEmit 2>&1 | head -5
git add frontend/src/terminal/panels/SchedulePanel.vue
git commit -m "feat(frontend): add SchedulePanel with task CRUD, cron presets, and modal"
```

---

## Task 7: NotifyPanel

**Files:**
- Create: `frontend/src/terminal/panels/NotifyPanel.vue`

Create the panel with:
- Level filter bar (All/Trade/Warn/Error/Info) with active state
- Notification list with: level icon (💹⚠️❌ℹ️), title, body, timestamp
- Unread rows: bold white text + highlighted background
- Click to mark read, Mark All Read button, unread count badge

```bash
cd /Volumes/etx/coding/rebuild/quantflow/frontend && npx vue-tsc --noEmit 2>&1 | head -5
git add frontend/src/terminal/panels/NotifyPanel.vue
git commit -m "feat(frontend): add NotifyPanel with level filtering, read/unread states, and mark-all-read"
```

---

## Task 8: BrokerConfig Panel

**Files:**
- Create: `frontend/src/terminal/panels/BrokerConfig.vue`

Create the panel with:
- Broker dropdown (Binance / Futu)
- Binance: API Key (password), Secret Key (password), Testnet checkbox
- Futu: Host, Port, connection status dot (red/green)
- Test Connection + Save buttons

```bash
cd /Volumes/etx/coding/rebuild/quantflow/frontend && npx vue-tsc --noEmit 2>&1 | head -5
git add frontend/src/terminal/panels/BrokerConfig.vue
git commit -m "feat(frontend): add BrokerConfig panel with Binance/Futu configuration forms"
```

---

## Task 9: Registry Update + OrderEntryPanel Enhancement

**Files:**
- Modify: `frontend/src/terminal/panels/registry.ts`
- Modify: `frontend/src/terminal/panels/OrderEntryPanel.vue`

### Step 1: Read both files first, then edit

### Step 2: Add 7 registrations to registry.ts

```typescript
register('portfolio-summary', () => import('./PortfolioSummary.vue'))
register('position-detail', () => import('./PositionDetail.vue'))
register('risk-dashboard', () => import('./RiskDashboard.vue'))
register('trade-history', () => import('./TradeHistory.vue'))
register('schedule-panel', () => import('./SchedulePanel.vue'))
register('notify-panel', () => import('./NotifyPanel.vue'))
register('broker-config', () => import('./BrokerConfig.vue'))
```

### Step 3: Add broker selector to OrderEntryPanel.vue

In `<script setup>`: `const broker = ref<'paper' | 'binance' | 'futu'>('paper')`

In `<template>`, after the symbol input group: broker dropdown with paper/binance/futu options.

Update `placeOrder()`: call `(window as any).go.main.App.PlaceOrder(symbol, side, orderType, qty, price)`

### Step 4: Verify + Commit

```bash
cd /Volumes/etx/coding/rebuild/quantflow/frontend && npx vue-tsc --noEmit 2>&1 | tail -10
git add frontend/src/terminal/panels/registry.ts frontend/src/terminal/panels/OrderEntryPanel.vue
git commit -m "feat(frontend): register 7 new panels and add broker selector to OrderEntryPanel"
```

---

## Task 10: Final Verification + CHANGELOG

### Step 1: Full TypeScript check

```bash
cd /Volumes/etx/coding/rebuild/quantflow/frontend && npx vue-tsc --noEmit 2>&1
```

### Step 2: Go build + tests

```bash
cd /Volumes/etx/coding/rebuild/quantflow && go build ./... && go test ./... -count=1 2>&1 | tail -10
```

### Step 3: Update CHANGELOG

Add to `## [2026.6.17]`:

```markdown
#### Phase 6 — Frontend Panels + Stores
- [Frontend] 7 new panels: PortfolioSummary, PositionDetail, RiskDashboard, TradeHistory, SchedulePanel, NotifyPanel, BrokerConfig
- [Frontend] portfolioStore and notifyStore (Pinia) with auto-refresh
- [Frontend] Enhanced OrderEntryPanel with broker selector
- [Frontend] CSV export utility for trade data
- [Frontend] ECharts integration: equity curve, allocation pie, drawdown chart, price history
```

### Step 4: Commit

```bash
git add CHANGELOG.md
git commit -m "docs: add Phase 6 frontend panels changelog entries"
```
