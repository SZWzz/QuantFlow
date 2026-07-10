# QuantFlow 性能优化设计

## Motivation

基于代码排查发现的四个性能瓶颈，系统性提升用户体验：

1. MarketOverview 无自动刷新——用户必须手动点刷新按钮
2. OHLCV 日线首次拉取 25 年全量数据——切换股票首次加载过重
3. 首屏 Bundle 体积过大——vendor-chart 1MB + AIChatPanel 1MB
4. 分时图逻辑在两个面板中重复实现——维护成本高、行为不一致

## Design

### 1. MarketOverview 智能轮询

**触发条件**：交易时段内每 30s 调 `dataStore.fetchMarketOverview(market)`。复用 CandlestickPanel 中已有的 `isTradingHours()` 函数，提取到 `@/lib/wails` 作为公共工具。

**休市行为**：非交易时段停止轮询，切换到交易时段自动恢复。

**WS 补充**：现有 WebSocket 继续推送 `last`/`changePct` 做秒级卡片数字更新，轮询负责刷新完整 overview 数据（indices + breadth + sectors）。

**改动范围**：
- `frontend/src/terminal/panels/MarketOverviewPanel.vue`：添加 timer + 交易时段检测

**Go 侧不变**：overviewCache 已有 60s TTL，30s 轮询足够及时。

### 2. OHLCV 渐进加载

**首次加载**：`lookbackDays` 从 9125（~25 年）缩减为 365（1 年），覆盖默认 250 根 K 线窗口。

**扩展加载**：监听 ECharts dataZoom 事件，当可视范围起点 `start < 5%` 时，向前再拉 365 天日线，通过已有的 `mergeMap` 合并去重。

**改动范围**：
- `frontend/src/terminal/panels/CandlestickPanel.vue`：修改 lookbackDays + 添加 dataZoom 扩展 handler

**Go 侧不变**：OHLCVCache 已有 LRU + SQLite 缓存，扩展请求间隔大。
**首屏加载量减少 96%**（365 / 9125）。

### 3. 首屏 Bundle 懒加载

**方式**：`defineAsyncComponent` 替换直接 import。

**目标模块**：
- `AIChatPanel.vue` (~1MB)
- `ModelRegistryPanel.vue`
- `AuditPanel.vue`
- `BacktestPanel.vue`
- `WorkflowMode.vue`

**改动范围**：
- `frontend/src/terminal/TerminalMode.vue`：面板注册改为 async 组件
- 路由入口同理

**预期收益**：首屏 JS 减少 ~1.2MB（~35%）。

### 4. 分时图 Composable 去重

**新增**：`frontend/src/lib/composables/useMinuteChart.ts`

封装以下状态和逻辑：
```typescript
function useMinuteChart(symbol: Ref<string>, prevClose: Ref<number>, opts?: {
  polling?: boolean        // default false, CandlestickPanel passes true
  pollingInterval?: number // default 5000
  bottomMode?: Ref<string> // default 'volume'
}) {
  // State
  const minuteTicks = shallowRef<MinuteTick[]>([])
  const minuteLoading = ref(false)

  // Methods
  async function loadMinuteLine() { /* IPC call, handles incremental merge */ }
  function startPolling() { /* 5s interval if polling=true */ }
  function stopPolling() { /* cleanup */ }

  // Computed: minute chart option via buildMinuteOption
  const minuteOption = computed(() => buildMinuteOption(
    minuteTicks.value, prevClose.value, bottomMode, theme, cache, symbol.value
  ))

  return { minuteTicks, minuteLoading, loadMinuteLine, startPolling, stopPolling, minuteOption }
}
```

`prevClose` 作为参数传入（由调用方从 quoteData/idx 中取，保持数据源可控）。`polling` 开关控制是否需要轮询（MarketOverview 不需要，Candlestick 需要）。

**消费方**：
- `MarketOverviewPanel.vue`：删除内联 minuteTicks/prevClose/loadMinuteLine/minuteOption，改用 composable
- `CandlestickPanel.vue`：同上

**附带收益**：MarketOverviewPanel 的分钟图现在走 `buildMinuteOption`，不再手写简化版 ECharts option，工具提示/均价线/面积填充与 CandlestickPanel 完全一致。

## Data Flow

```
MarketOverviewPanel              CandlestickPanel
     │                                  │
     ├─ useMinuteChart(symbol) ─────────┤  ← 同一 composable
     │  ├─ loadMinuteLine() → Go IPC    │
     │  ├─ minuteTicks                  │
     │  ├─ prevClose (from quoteData)   │
     │  └─ minuteOption (buildMinute)   │
     │                                  │
     ├─ fetchMarketOverview 30s poll ───┤  ← 仅 MarketOverview
     │                                  │
     └─ dataZoom → expand 365d ────────┘  ← 仅 Candlestick
```

## Acceptance Criteria

1. MarketOverview 在交易时段自动 30s 刷新，休市停止，切换回来恢复
2. OHLCV 日线首次加载 ≤ 365 根 bar，dataZoom 拖到边界自动扩展
3. 首屏 JS bundle < 2MB（不含 chart vendor）
4. MarketOverviewPanel 和 CandlestickPanel 的分钟图行为一致（均线、均价线、涨跌颜色、tooltip）
5. `useMinuteChart` composable 有单元测试覆盖核心逻辑

## Risks / Trade-offs

- **OHLCV 扩展加载**：dataZoom 事件触发频率高，需要 throttle（500ms）+ 去重 guard 防止重复请求
- **懒加载**：面板首次打开时有短暂加载延迟，需确保 LoadingState 组件覆盖
- **30s 轮询**：对于 CN 市场 4 小时交易时段，共 ~480 次请求/天，量级可接受
- **composable 重构**：两个面板的使用场景略有差异（MarketOverview 不轮询分时图，Candlestick 5s 轮询），需要参数化控制

## Execution Order

1 → 4 → 2 → 3（低风险先行，每步独立可验证）
