# CandlestickPanel 性能优化与重构

## Motivation

`CandlestickPanel.vue` 当前 756 行，脚本占 80%，存在多处性能问题：

1. **P0 — `:key` 导致 VChart 完全销毁重建**：切换指标/间隔/代码时 ECharts 实例被 Vue 卸载重建，丢失缩放状态
2. **P0 — 指标无缓存**：每次 computed 重算都全量跑 O(n×period) 指标计算
3. **P1 — 30s 全量轮询**：分时模式下每 30s 重新拉取 250-450 天 OHLCV 数据
4. **P1 — `useChartTheme()` 每次评估触发 DOM 读取**：7 次 `getComputedStyle`，造成回流
5. **P1 — 分钟 KDJ 退化**：`kdj(prices, prices, prices)` 高=收=低，RSV 无效
6. **P1 — 缺少用户错误提示**：`console.error` 静默失败
7. **P2 — `(window as any).go?.main?.App` 模式重复 4 处**
8. **P2 — option 构建器代码重复**：分钟图与多日图 80% 相同
9. **P2 — `any` 类型泛滥**：数据解包/series/轴配置全部无类型保护
10. **P3 — 竞态条件**：symbol watch + 联动上下文同时触发可能旧数据覆盖新数据

## Design — 分三阶段实施

### Phase 1: 核心性能（P0 + 部分 P1）

#### 1.1 稳定 VChart key + option-only 更新

**问题**：`:key="\`${symbol}-${interval}-${topOverlay}-${bottomMode}\`"` 导致 ECharts 实例销毁重建。

**方案**：将 `<VChart>` 封装为 `KlineChart.vue` 组件，使用稳定 key（仅 `${symbol}`），通过 `watch` 调用 `chart.setOption(option, { notMerge: false })` 增量更新。

```
CandlestickPanel.vue
  └─ KlineChart.vue (new)
       └─ VChart (stable key: symbol)
```

`KlineChart.vue` 接口：

```vue
<script setup lang="ts">
// Props 只接受 option 对象和 symbol（用于 key）
defineProps<{
  option: ECBasicOption
  symbol: string
  loading?: boolean
}>()
</script>
<template>
  <VChart :key="`kc-${symbol}`" :option="option" autoresize />
</template>
```

`option` computed 属性仍留在 `CandlestickPanel.vue`（不抽取），但 key 不再包含 `interval/topOverlay/bottomMode`，切换这些选项时不会销毁 ECharts，只会通过 `setOption` 增量 diff。

分钟图和多日图也使用同样的 `KlineChart.vue` 组件，各自维护自己的 option computed。

#### 1.2 指标计算 memoization

**问题**：当前每次 `option` computed 重算都全量执行所有分支的指标函数。

**方案**：在 `useIndicators.ts` 中为每个指标函数添加 `WeakMap` 缓存：

```typescript
const memoCache = new WeakMap<object, Map<string, any>>()

function memoKey(data: number[], ...params: number[]): object {
  return { dataLen: data.length, data0: data[0], dataLast: data[data.length-1], params: params.join(',') }
}
```

缓存策略：
- key = `data.length + data[0] + data[data.length-1] + params`
- 用 `WeakMap<array, Map<string, result>>` 绑定到数据数组生命周期
- 数据不变时直接返回缓存；数据变（新 tick 追加）时清缓存

不在原函数上加缓存，而是包装为 `useMemoizedIndicators()` composable：

```typescript
function useMemoizedIndicators() {
  const cache = new Map<string, any>()
  function getCached(key: string, fn: () => any) {
    if (cache.has(key)) return cache.get(key)
    const result = fn()
    cache.set(key, result)
    return result
  }
  return { cache, getCached }
}
```

在 `CandlestickPanel.vue` 中：

```typescript
const ohlcvKey = computed(() => `${symbol.value}-${interval.value}-${ohlcvData.value.length}`)
const macdResult = computed(() => getCached(`macd-${ohlcvKey.value}`, () => macd(close)))
const kdjResult = computed(() => getCached(`kdj-${ohlcvKey.value}`, () => kdj(close, high, low)))
```

切换 `bottomMode`（例如 volume → macd）时，macdResult 已在缓存中，无需重算。

#### 1.3 useChartTheme 缓存

**问题**：每次 computed 评估都调用 `getComputedStyle` DOM 读取。

**方案**：改为在 `useChartTheme()` composable 内部使用 `ref` + 只读 computed，仅在 CSS 类/主题变化时更新。使用 `MutationObserver` 监听 `document.body` 的类变化。

```typescript
export function useChartTheme() {
  const theme = reactive({ textColor: '#...', axisColor: '#...', ... })
  function update() { /* getComputedStyle 7次 */ }
  update()
  const observer = new MutationObserver(() => update())
  onMounted(() => observer.observe(document.body, { attributes: true, attributeFilter: ['class'] }))
  onUnmounted(() => observer.disconnect())
  return toRefs(theme)
}
```

#### 1.4 分钟 KDJ 修复

**问题**：`kdj(prices, prices, prices)` 高=收=低，RSV 无效。

**方案**：分钟线数据为 `{ time, price, avg_price, volume }`，缺少 H/L。改为用 min/max 替代：

```typescript
// 分钟 KDJ: 用滚动 N 周期内的最高最低价替代 H/L
const minPrices = prices.map((_, i) => {
  const start = Math.max(0, i - 8)
  return Math.min(...prices.slice(start, i + 1))
})
const maxPrices = prices.map((_, i) => {
  const start = Math.max(0, i - 8)
  return Math.max(...prices.slice(start, i + 1))
})
const kd = kdj(prices, maxPrices, minPrices, 9, 3, 3)
```

### Phase 2: 数据加载优化（P1）

#### 2.1 增量 K 线轮询

**问题**：30s 全量拉取 250-450 根 OHLCV。

**方案**：在 `loadOHLCV()` 中区分首次加载和增量刷新：

```typescript
async function loadOHLCV(incremental = false) {
  if (incremental && ohlcvData.value.length > 0) {
    const lastDate = ohlcvData.value[ohlcvData.value.length - 1].date
    // 仅拉取 lastDate 之后的数据
    const newBars = await app.FetchOHLCV(market, symbol, interval, 'qfq', lastDate, endDate)
    if (newBars?.length) mergeIncremental(ohlcvData.value, newBars)
    return
  }
  // 全量加载...
}
```

轮询定时器调用 `loadOHLCV(true)`，首次加载调用 `loadOHLCV(false)`。

**mergeIncremental**：按 date 去重合并，覆盖同日期数据（修复可能的分笔）。

#### 2.2 错误提示

**问题**：数据加载失败时只在 console.error 输出，用户无感知。

**方案**：在面板 header 区域新增内联错误条（已有 `ErrorBoundary` 组件可用模式）

```typescript
const errorMsg = ref('')
// catch 块中:
errorMsg.value = 'K线数据加载失败: ' + (e.message || '未知错误')
setTimeout(() => { if (errorMsg.value === msg) errorMsg.value = '' }, 8000)
```

模板：

```html
<div v-if="errorMsg" class="err-toast">{{ errorMsg }}</div>
```

#### 2.3 竞态条件修复

**问题**：symbol watch + 联动上下文同时触发时可能旧数据覆盖新数据。

**方案**：引入 `loadSeq` 计数器，每次开始加载时递增，回调时校验：

```typescript
let loadSeq = 0
async function loadOHLCV() {
  const seq = ++loadSeq
  const data = await fetch(...)
  if (seq !== loadSeq) return // 被新请求废弃
  ohlcvData.value = data
}
```

同样的模式应用到 `loadMinuteLine` 和 `fetchMultiDayMinute`。

### Phase 3: 代码质量（P2）

#### 3.1 抽取 useWailsApp composable

```typescript
// frontend/src/lib/composables/useWailsApp.ts
export function useWailsApp() {
  const app = (window as any)?.go?.main?.App ?? null
  return app as {
    FetchOHLCV: (mkt: string, sym: string, interval: string, fq: string, start: string, end: string) => Promise<any[]>
    GetMinuteLine: (sym: string, since: number) => Promise<any[]>
    GetMultiDayMinute: (sym: string, days: number) => Promise<any[]>
    // ...
  } | null
}
```

#### 3.2 拆分 option 构建器

从 `CandlestickPanel.vue` 中提取三个纯函数：

```typescript
// lib/buildChartOption.ts
export function buildKlineOption(data, topOverlay, bottomMode, theme, indicators): ECBasicOption
export function buildMinuteOption(data, bottomMode, theme, indicators): ECBasicOption
export function buildMultiDayOption(data, theme): ECBasicOption
```

这三个函数共用网格/轴/工具提示等基础配置，通过 `mergeOption(baseConfig, specificConfig)` 合并。

#### 3.3 类型化

为 `FetchOHLCV` 返回数据定义接口：

```typescript
interface OHLCVBar {
  date: string
  open: number
  high: number
  low: number
  close: number
  volume: number
}

interface MinuteTick {
  time: string
  price: number
  avg_price: number
  volume: number
}
```

移除所有 `as any`、`(d: any)`、`let series: any[]`。

## New/Modified Files

### Phase 1

| File | Action | Description |
|------|--------|-------------|
| `frontend/src/terminal/components/panel/KlineChart.vue` | Create | VChart 封装，稳定 key |
| `frontend/src/terminal/panels/CandlestickPanel.vue` | Modify | 换用 KlineChart，加指标缓存，加 #loadSeq |
| `frontend/src/lib/composables/useIndicators.ts` | Modify | 加 memoization wrapper |
| `frontend/src/lib/composables/useChartTheme.ts` | Modify | MutationObserver 缓存 |

### Phase 2

| File | Action | Description |
|------|--------|-------------|
| `frontend/src/terminal/panels/CandlestickPanel.vue` | Modify | 增量轮询、错误提示、竞态修复 |

### Phase 3

| File | Action | Description |
|------|--------|-------------|
| `frontend/src/lib/composables/useWailsApp.ts` | Create | 类型化 Wails 桥接 |
| `frontend/src/lib/buildChartOption.ts` | Create | option 构建器纯函数 |
| `frontend/src/terminal/panels/CandlestickPanel.vue` | Modify | 接入 composable + 类型化 |

## Acceptance Criteria

Phase 1:
- [ ] 切换 topOverlay/bottomMode/interval 时 VChart 不重新创建（DOM 节点稳定）
- [ ] dataZoom 位置在切换指标后保持
- [ ] 切换指标后指标计算命中缓存（重复切换零重算）
- [ ] theme 变化时仍然更新图表颜色
- [ ] 分钟 KDJ 曲线正确波动（非平坦 50）

Phase 2:
- [ ] 首次加载 30s 轮询仅拉取增量数据（网络请求验证）
- [ ] 数据加载失败时面板显示内联错误提示
- [ ] 快速切换 symbol 时不会出现旧数据闪烁
- [ ] 分钟线轮询 5s 继续正常工作

Phase 3:
- [ ] `useWailsApp()` 返回类型化对象，无 `as any`
- [ ] `buildChartOption.ts` 三个纯函数覆盖全部 option 构建
- [ ] OHLCVBar/MinuteTick 接口无 `any`
- [ ] `CandlestickPanel.vue` 脚本部分 < 400 行

## Risks / Trade-offs

- **指标缓存占内存**：每个 symbol×interval 组合缓存一份指标结果。估算：100 symbol × 3 interval × 5KB ≈ 1.5MB，可接受
- **增量轮询依赖后端支持**：`FetchOHLCV` 需要支持 start_date 参数。当前 Go 端已支持 start/end 参数，可行
- **KlineChart 封装增加层级**：增加一个组件嵌套，但 ECharts 初始化成本远大于此
- **MutationObserver 兼容性**：Wails v3 基于 Chromium，完全支持
- **代码拆分后 import 链变长**：从 1 个文件变为 5+ 个文件，但每个文件职责更清晰
