# Symbol Context 联动系统 — 设计文档

> **Status**: Design — 等待实施计划
> **Priority**: 🔴 高
> **Part of**: Terminal Mode 核心架构

## Motivation

QuantFlow 有 50 个面板，其中 19 个面板有 symbol 概念。当前只有 1 个面板（Watchlist）能发布 symbol、3 个面板能订阅——其余面板各自管理自己的 symbol，完全孤立。

Bloomberg 终端的核心竞争力之一就是 **Link Group** 系统：多个面板共享一个 symbol 上下文，在面板 A 切换股票 → 面板 B/C/D 自动跟随。这是彭博用户每天使用最多的交互模式。

## Design

### Core Concepts

```
┌─────────────────────────────────────────────────────┐
│                   Symbol Context                     │
│                                                     │
│  Link Group 1 (● Red)    Link Group 2 (● Green)     │
│  ┌─────────────────┐    ┌─────────────────┐         │
│  │ active: AAPL     │    │ active: 600519  │         │
│  │ ┌─────┐ ┌─────┐  │    │ ┌─────┐ ┌─────┐ │         │
│  │ │Watch│ │K线  │  │    │ │Watch│ │财报 │ │         │
│  │ │list │ │Chart│  │    │ │list │ │Panel│ │         │
│  │ └─────┘ └─────┘  │    │ └─────┘ └─────┘ │         │
│  └─────────────────┘    └─────────────────┘         │
│                                                     │
│  Unlinked Panel               Independent symbol    │
│  ┌─────────────────┐                               │
│  │ MarketOverview  │  (no symbol needed)            │
│  └─────────────────┘                               │
└─────────────────────────────────────────────────────┘
```

### 1. Link Group（联动组）

每个 Link Group 是一个独立的 symbol 上下文：

```typescript
interface LinkGroup {
  id: string          // "group-1", "group-2", "group-3", "group-4"
  color: string       // "#ef4444", "#22c55e", "#f59e0b", "#3b82f6"
  label: string       // "Red", "Green", "Amber", "Blue"
  activeSymbol: string | null
  symbolHistory: string[]  // 最近 10 个 symbol
}
```

默认 4 个 group，颜色编码对标 Bloomberg：
- Group 1: 🔴 Red (`#ef4444`)
- Group 2: 🟢 Green (`#22c55e`)  
- Group 3: 🟡 Amber (`#f59e0b`)
- Group 4: 🔵 Blue (`#3b82f6`)

### 2. Panel Link State（面板联动状态）

每个面板有三种链路状态：

| 状态 | 含义 | 视觉 |
|------|------|------|
| **Linked** | 跟随所属 Group 的 activeSymbol | 面板标题栏显示 Group 颜色圆点 |
| **Unlinked** | 保持自己的 symbol，不跟随也不发布 | 灰色圆点 |
| **Master** | 可以修改 Group 的 activeSymbol | 面板标题栏颜色圆点 + ★ 标记 |

### 3. Symbol Publishing（Symbol 发布）

**Publisher Panel** 改为 Group 的 activeSymbol：

```typescript
// terminalStore
function setGroupSymbol(groupId: string, symbol: string) {
  const group = linkGroups.value[groupId]
  group.activeSymbol = symbol.toUpperCase()
  group.symbolHistory.unshift(symbol)
  if (group.symbolHistory.length > 10) group.symbolHistory.pop()
  lastSymbolUpdate.value = Date.now()
}
```

**发布者面板**：Watchlist、QuoteDetail、StockResearch、OrderEntry、Financials
- 用户在这些面板中选择/输入 symbol → 自动发布到所在 Group

**订阅者面板**：Candlestick、Financials、Sentiment、PeerComparison、AnalystEstimates、InsiderTrading、MarketDepth、PositionDetail、Drawing、Distribution、SurfaceChart
- 监听所在 Group 的 activeSymbol → 自动更新数据

**无 symbol 面板**：MarketOverview、Heatmap、Geopolitics 等 — 不受影响

### 4. Panel Group Assignment

面板打开时自动分配 Group：

```typescript
function openPanel(panelId: string, params?: Record<string, any>) {
  const instanceId = `${panelId}-${Date.now()}`
  const groupId = params?.groupId || activeGroupId || 'group-1'
  // ...
  panelInstances.set(instanceId, { groupId, linked: true })
}
```

规则：
- 从 Welcome 点击打开 → `group-1`（默认）
- 从已有面板 [⊕] 打开 → 继承该面板的 Group
- Ctrl+K 搜索打开 → 当前"焦点 Group"（最后操作过的 Group）
- `params.groupId` 可显式指定

### 5. StatusBar（状态栏增强）

```
┌──────────────────────────────────────────────────────────────┐
│ ● Connected  │  ● AAPL 195.32  ● 600519 1850.00  ● CTCUSDT │
│              │  Group 1 Red    Group 2 Green    Group 3 Amber│
└──────────────────────────────────────────────────────────────┘
```

每个活跃的 Group 在 StatusBar 显示：颜色圆点 + symbol + 当前价格

### 6. Panel Header Indicator

每个有 symbol 的面板标题栏显示联动指示器：

```
┌─ ● AAPL ── Watchlist ──────── [✕] ─┐
│  Group 1 · Linked                   │
└─────────────────────────────────────┘
```

点击颜色圆点可切换：Linked → Unlinked → Master → Linked

### 7. Symbol Input Bar（可选增强）

在 StatusBar 上方增加一个 Symbol Input Bar（对标 Bloomberg 的命令行）：

```
┌──────────────────────────────────────────┐
│ ● AAPL ▼  │  Quick symbol change for     │
│           │  active Group                 │
└──────────────────────────────────────────┘
```

## Data Flow

```
┌──────────────┐    setGroupSymbol()    ┌──────────────────┐
│  Watchlist   │ ─────────────────────→ │  terminalStore    │
│  (Publisher) │                        │  linkGroups[gid]  │
└──────────────┘                        │  .activeSymbol    │
                                        └───────┬──────────┘
                                                │
                    ┌───────────────────────────┼──────────────────────┐
                    │ watch(activeSymbol)       │                      │
                    ▼                           ▼                      ▼
            ┌──────────────┐           ┌──────────────┐      ┌──────────────┐
            │ Candlestick  │           │  Financials  │      │  QuoteDetail │
            │ (Subscriber) │           │ (Subscriber) │      │ (Subscriber) │
            └──────────────┘           └──────────────┘      └──────────────┘
```

## Panel Classification

### Publishers（发布者 — 5 个）
用户可在面板中选择/输入 symbol，自动发布到所在 Group

| Panel | 发布方式 |
|-------|---------|
| WatchlistPanel | 点击 symbol 行 |
| QuoteDetailPanel | symbol 输入框 |
| StockResearchPanel | symbol 输入框 |
| OrderEntryPanel | symbol 输入框 |
| FinancialsPanel | symbol 输入框 |

### Subscribers（订阅者 — 13 个 + 5 publishers 本身也是 subscribers）
监听 Group activeSymbol 变化，自动更新

| Panel | 更新行为 |
|-------|---------|
| CandlestickPanel | 重新加载 K线数据 |
| QuoteDetailPanel | 刷新报价 |
| FinancialsPanel | 加载财务数据 |
| StockResearchPanel | 加载全维度研究 |
| SentimentPanel | 加载情绪分析 |
| PeerComparisonPanel | 加载同行对比 |
| AnalystEstimatesPanel | 加载分析师数据 |
| InsiderTradingPanel | 加载内部交易 |
| MarketDepthPanel | 加载盘口数据 |
| PositionDetail | 加载持仓详情 |
| DrawingPanel | 加载画线数据 |
| DistributionPanel | 重新计算收益分布 |
| SurfaceChartPanel | 重新生成曲面 |
| OrderEntryPanel | 预填 symbol |
| PredictionDashboardPanel | 按 symbol 筛选 |

### Neither（无关面板 — 32 个）
不需要 symbol，不受联动影响

## Files

### New（3 个）
- `frontend/src/stores/symbolContext.ts` — Symbol Context store (link groups, pub/sub)
- `frontend/src/terminal/SymbolBar.vue` — Symbol Input Bar 组件
- `frontend/src/terminal/__tests__/SymbolBar.test.ts` — 测试

### Modified（按优先级）

**Store 层（1 个）**
- `frontend/src/stores/terminal.ts` — 移除旧 activeSymbol，迁移到 symbolContext store

**核心面板 — 改为 Publisher（5 个）**
- WatchlistPanel.vue — 点击时调用 `symbolContext.setGroupSymbol(groupId, sym)`
- QuoteDetailPanel.vue — symbol 输入提交时发布
- StockResearchPanel.vue — symbol 输入提交时发布
- OrderEntryPanel.vue — symbol 输入提交时发布
- FinancialsPanel.vue — symbol 输入提交时发布

**核心面板 — 改为 Subscriber（10 个）**
- CandlestickPanel.vue
- SentimentPanel.vue
- PeerComparisonPanel.vue
- AnalystEstimatesPanel.vue
- InsiderTradingPanel.vue
- MarketDepthPanel.vue
- PositionDetail.vue
- DrawingPanel.vue
- DistributionPanel.vue
- SurfaceChartPanel.vue

**UI 组件（2 个）**
- StatusBar.vue — 显示所有 Group 的 symbol
- DockTab.vue — 面板标题栏显示 Group 颜色指示器

**注册（1 个）**
- registry.ts — 注册 SymbolBar 组件

## Acceptance Criteria

### Phase 1: Core System
- [ ] `symbolContextStore` 存在，含 4 个 LinkGroup + `setGroupSymbol()` + `getGroupSymbol()`
- [ ] 面板可用 `useSymbolContext().getGroupSymbol(groupId)` 读取当前 symbol
- [ ] 面板可用 `useSymbolContext().setGroupSymbol(groupId, symbol)` 发布 symbol
- [ ] 旧 `terminalStore.activeSymbol` 已迁移到 symbolContextStore

### Phase 2: Publisher Migration（5 个）
- [ ] WatchlistPanel 点击 symbol 发布到所在 Group
- [ ] QuoteDetailPanel symbol 输入发布到所在 Group
- [ ] StockResearchPanel symbol 输入发布到所在 Group
- [ ] OrderEntryPanel symbol 输入发布到所在 Group
- [ ] FinancialsPanel symbol 输入发布到所在 Group

### Phase 3: Subscriber Migration（10 个）
- [ ] 每个订阅者面板 watch 所在 Group 的 activeSymbol
- [ ] Symbol 变化时面板数据自动更新
- [ ] 面板标题栏显示 Group 颜色指示器

### Phase 4: UI Polish
- [ ] StatusBar 显示所有活跃 Group 的 symbol + 颜色
- [ ] SymbolBar 组件：快速切换 symbol
- [ ] 面板支持 Linked/Unlinked/Master 三种状态切换
- [ ] 现有 185 前端测试全通过
- [ ] `go vet ./...` 通过

## Risks / Trade-offs

- **向后兼容** — `props.params.symbol` 仍然有效（用于初始化，之后被 Group 覆盖）
- **性能** — 4 个 Group × 10+ subscriber，symbol 变化时可能有 10+ 面板同时刷新（每个面板有自己的防抖）
- **复杂度** — 比单一 `activeSymbol` 复杂，但这是 Bloomberg 标准，用户期望
- **Unlinked 面板** — 用户可能忘记某个面板 unlinked，导致数据"不对"。需要明显的视觉提示
