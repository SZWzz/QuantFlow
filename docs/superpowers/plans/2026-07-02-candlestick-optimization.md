# K 线图综合优化 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 CandlestickPanel 从基础 K 线图升级为对标同花顺的专业终端，同时内聚 DrawingPanel 画线能力后删除独立面板。

**Architecture:** 不替换 ECharts，通过 Canvas overlay 补充画线/十字光标交互。WebSocket 推送实时数据（降级回轮询）。画线数据存储在 localStorage，坐标从像素改为数据空间 (dataIndex, price) 以支持缩放平移。新增 Go 包 `internal/ws/` 处理 WS hub。

**Tech Stack:** Go `golang.org/x/net/websocket`, ECharts 5 (vue-echarts), Canvas 2D API

## Global Constraints

- 所有新功能增量添加，不破坏现有 K 线/分时/多日分时功能
- 画线坐标改为数据空间 (dataIndex, price)，localStorage key 用 `drawings-v2/{symbol}` 避免与旧数据冲突
- 十字光标在 Canvas overlay 自绘，禁用 ECharts 默认十字线 tooltip
- WS 断连后自动降级轮询（30s K 线 / 5s 分时）
- Go WS hub 不依赖外部库（用 `net/http` + `golang.org/x/net/websocket`）
- 每个 task 结束必须运行 `go vet ./...`、`vue-tsc --noEmit`、`vitest run`

---

### Task 1: InfoBar 组件 + GetQuote 字段扩展

**Files:**
- Modify: `frontend/src/terminal/components/panel/InfoBar.vue` (Create)
- Modify: `frontend/src/terminal/panels/CandlestickPanel.vue`
- Modify: `app/app.go` (GetQuote 返回扩展)
- Modify: `frontend/src/lib/composables/useWailsApp.ts` (QuoteData 类型扩展)

**Interfaces:**
- Consumes: `GetQuote(symbol) → QuoteData` (扩展字段)
- Produces: `InfoBar.vue` 组件接受 `quote: QuoteData` prop

- [ ] **Step 1: 扩展 Go QuoteData 结构体**

`app/app.go` 中找到 `QuoteData` 结构体，增加字段。先用 Grep 找精确位置。

```bash
grep -n "type QuoteData struct" app/app.go
```

读取该结构体后，增加字段：

```go
type QuoteData struct {
    Symbol        string  `json:"symbol"`
    Price         float64 `json:"price"`
    Change        float64 `json:"change"`
    ChangePercent float64 `json:"change_percent"`
    High          float64 `json:"high"`
    Low           float64 `json:"low"`
    Open          float64 `json:"open"`
    Volume        float64 `json:"volume"`
    Amount        float64 `json:"amount"`
    Bid           float64 `json:"bid"`
    Ask           float64 `json:"ask"`
    // 新增
    TurnoverRate  float64 `json:"turnover_rate"`
    VolumeRatio   float64 `json:"volume_ratio"`
    Amplitude     float64 `json:"amplitude"`
    AvgPrice      float64 `json:"avg_price"`
    InsideVolume  float64 `json:"inside_volume"`
    OutsideVolume float64 `json:"outside_volume"`
    MarketCap     float64 `json:"market_cap"`
    PeRatio       float64 `json:"pe_ratio"`
    LimitUp       float64 `json:"limit_up"`
    LimitDown     float64 `json:"limit_down"`
}
```

- [ ] **Step 2: 扩展前端 QuoteData 类型**

`frontend/src/lib/composables/useWailsApp.ts` 中找到 `QuoteData` 类型（或查找相关类型），加上同样字段。

- [ ] **Step 3: 创建 InfoBar.vue 组件**

```vue
<script setup lang="ts">
import type { QuoteData } from '@/lib/composables/useWailsApp'
import { computed } from 'vue'
import { marketChangeColor } from '@/lib/composables/useMarketColors'
import { useI18n } from 'vue-i18n'

const props = defineProps<{
  quote: QuoteData | null
  symbol: string
  name: string
}>()

const { t } = useI18n()

const changeColor = computed(() => {
  if (!props.quote) return 'var(--color-text-primary)'
  return marketChangeColor(props.quote.change)
})
</script>

<template>
  <div class="info-bar">
    <span class="symbol">{{ symbol }}</span>
    <span class="name">{{ name }}</span>
    <span class="price" :style="{ color: changeColor }">
      {{ quote?.price?.toFixed(2) ?? '--' }}
    </span>
    <span class="change" :style="{ color: changeColor }">
      {{ quote?.change ?? '--' }}
    </span>
    <span class="change-pct" :style="{ color: changeColor }">
      {{ quote?.change_percent != null ? (quote.change_percent >= 0 ? '+' : '') + quote.change_percent.toFixed(2) + '%' : '--' }}
    </span>
    <span class="sep">|</span>
    <span class="stat">{{ t('kline.turnover') }} {{ quote?.turnover_rate != null ? quote.turnover_rate.toFixed(2) + '%' : '--' }}</span>
    <span class="stat">{{ t('kline.volume_ratio') }} {{ quote?.volume_ratio?.toFixed(2) ?? '--' }}</span>
    <span class="stat">{{ t('kline.amplitude') }} {{ quote?.amplitude?.toFixed(2) ?? '--' }}%</span>
    <span class="sep">|</span>
    <span class="stat">{{ t('kline.avg_price') }} {{ quote?.avg_price?.toFixed(2) ?? '--' }}</span>
    <span class="stat">{{ t('kline.inside') }} {{ formatVolume(quote?.inside_volume) }}</span>
    <span class="stat">{{ t('kline.outside') }} {{ formatVolume(quote?.outside_volume) }}</span>
    <span class="stat">{{ t('kline.market_cap') }} {{ formatMarketCap(quote?.market_cap) }}</span>
    <span class="stat">{{ t('kline.pe') }} {{ quote?.pe_ratio?.toFixed(1) ?? '--' }}</span>
  </div>
</template>

<style scoped>
.info-bar {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 4px 12px;
  font-size: 12px;
  font-variant-numeric: tabular-nums;
  background: var(--color-bg-elevated);
  border-bottom: 1px solid var(--color-border-subtle);
  overflow-x: auto;
  white-space: nowrap;
  min-height: 28px;
}
.symbol { font-weight: 700; color: var(--color-text-primary); }
.name { color: var(--color-text-tertiary); font-size: 11px; }
.price { font-weight: 700; font-size: 14px; }
.change, .change-pct { font-size: 13px; }
.sep { color: var(--color-border-subtle); }
.stat { color: var(--color-text-secondary); }
</style>
```

添加 i18n 键。读取 `frontend/src/lib/i18n/zh.ts` 和 `en.ts`，追加：

```ts
// zh.ts
kline: {
  // ... 现有
  turnover: '换手',
  volume_ratio: '量比',
  amplitude: '振幅',
  avg_price: '均价',
  inside: '内盘',
  outside: '外盘',
  market_cap: '流通市值',
  pe: '市盈率',
}

// en.ts
kline: {
  // ... 现有
  turnover: 'Turnover',
  volume_ratio: 'Vol Ratio',
  amplitude: 'Amplitude',
  avg_price: 'Avg Price',
  inside: 'Inside',
  outside: 'Outside',
  market_cap: 'Mkt Cap',
  pe: 'P/E',
}
```

- [ ] **Step 4: InfoBar 工具函数**

在 `InfoBar.vue` `script setup` 中添加：

```ts
function formatVolume(v?: number): string {
  if (v == null) return '--'
  if (v >= 10000) return (v / 10000).toFixed(1) + '万'
  return v.toFixed(0)
}
function formatMarketCap(v?: number): string {
  if (v == null) return '--'
  if (v >= 100000000) return (v / 100000000).toFixed(2) + '亿'
  if (v >= 10000) return (v / 10000).toFixed(2) + '万'
  return v.toFixed(0)
}
```

- [ ] **Step 5: 在 CandlestickPanel 中集成 InfoBar**

在 `CandlestickPanel.vue` 模板中找到 `<KlineChart>` 上方，在图表之前加入 InfoBar。

```vue
<InfoBar
  :quote="quoteData"
  :symbol="symbol"
  :name="name ?? ''"
/>
```

在 script 中添加引用和加载逻辑：

```ts
import InfoBar from '@/terminal/components/panel/InfoBar.vue'

const quoteData = ref<QuoteData | null>(null)
const { getQuote } = useWailsApp()

async function loadQuote() {
  try {
    quoteData.value = await getQuote(symbol.value)
  } catch (e) {
    console.error('[Candlestick] loadQuote:', e)
  }
}

// 在 symbol watch 中调用
watch(symbol, () => {
  loadQuote()
  // ... 已有逻辑
})

// 每 30s 轮询报价
let quoteTimer: ReturnType<typeof setInterval>
onMounted(() => {
  loadQuote()
  quoteTimer = setInterval(loadQuote, 30000)
})
onUnmounted(() => {
  clearInterval(quoteTimer)
})
```

- [ ] **Step 6: TypeScript 类型检查**

```bash
cd frontend && npx vue-tsc --noEmit
```

- [ ] **Step 7: 提交**

```bash
git add -A
git commit -m "feat: add InfoBar component and extend GetQuote fields"
```

---

### Task 2: 指标扩展 (SAR/EMA/CCI/OBV) + buildChartOption 集成

**Files:**
- Modify: `frontend/src/lib/composables/useIndicators.ts`
- Modify: `frontend/src/lib/buildChartOption.ts`

**Interfaces:**
- Consumes: `useIndicators.sar(high, low, close, accel, maxAccel)`, `.ema(close, period)`, `.cci(high, low, close, period)`, `.obv(close, volume)`
- Produces: `buildKlineOption()` 支持 `topOverlay: 'sar' | 'ema'` 和 `bottomMode: 'cci' | 'obv'`

- [ ] **Step 1: 读取当前 useIndicators.ts**

```bash
cat frontend/src/lib/composables/useIndicators.ts
```

- [ ] **Step 2: 新增 SAR 函数**

```ts
export function sar(high: number[], low: number[], close: number[], acceleration = 0.02, maxAcceleration = 0.2): (number | null)[] {
  const result: (number | null)[] = []
  if (high.length < 2) return high.map(() => null)

  let isLong = true
  let af = acceleration
  let ep = low[0]
  let sarVal = high[0]

  for (let i = 1; i < high.length; i++) {
    if (isLong) {
      sarVal = sarVal + af * (ep - sarVal)
      if (sarVal > low[i]) {
        isLong = false
        af = acceleration
        sarVal = ep = high[i]
        result.push(null)
        continue
      }
      if (high[i] > ep) {
        ep = high[i]
        af = Math.min(af + acceleration, maxAcceleration)
      }
    } else {
      sarVal = sarVal + af * (ep - sarVal)
      if (sarVal < high[i]) {
        isLong = true
        af = acceleration
        sarVal = ep = low[i]
        result.push(null)
        continue
      }
      if (low[i] < ep) {
        ep = low[i]
        af = Math.min(af + acceleration, maxAcceleration)
      }
    }
    result.push(sarVal)
  }

  // First few NaN
  for (let i = result.length; i < high.length; i++) {
    result.unshift(null)
  }

  return result
}
```

- [ ] **Step 3: 新增 EMA 函数**

```ts
export function ema(data: number[], period: number): (number | null)[] {
  const result: (number | null)[] = []
  if (data.length < period) return data.map(() => null)

  const multiplier = 2 / (period + 1)
  let emaVal = data.slice(0, period).reduce((a, b) => a + b, 0) / period

  for (let i = 0; i < data.length; i++) {
    if (i < period - 1) {
      result.push(null)
    } else if (i === period - 1) {
      result.push(emaVal)
    } else {
      emaVal = (data[i] - emaVal) * multiplier + emaVal
      result.push(emaVal)
    }
  }
  return result
}
```

- [ ] **Step 4: 新增 CCI 函数**

```ts
export function cci(high: number[], low: number[], close: number[], period = 20): (number | null)[] {
  const tp = high.map((h, i) => (h + low[i] + close[i]) / 3)
  const result: (number | null)[] = []

  for (let i = 0; i < tp.length; i++) {
    if (i < period - 1) {
      result.push(null)
      continue
    }
    const slice = tp.slice(i - period + 1, i + 1)
    const mean = slice.reduce((a, b) => a + b, 0) / period
    const mad = slice.reduce((sum, v) => sum + Math.abs(v - mean), 0) / period
    result.push(mad === 0 ? 0 : (tp[i] - mean) / (0.015 * mad))
  }
  return result
}
```

- [ ] **Step 5: 新增 OBV 函数**

```ts
export function obv(close: number[], volume: number[]): number[] {
  const result: number[] = [volume[0]]
  for (let i = 1; i < close.length; i++) {
    if (close[i] > close[i - 1]) {
      result.push(result[i - 1] + volume[i])
    } else if (close[i] < close[i - 1]) {
      result.push(result[i - 1] - volume[i])
    } else {
      result.push(result[i - 1])
    }
  }
  return result
}
```

- [ ] **Step 6: 在 useIndicators export 中暴露新函数**

确保 `createIndicatorCache` 或 `export` 包括新函数。

- [ ] **Step 7: 更新 buildChartOption.ts**

读取 `frontend/src/lib/buildChartOption.ts`，找到 `buildKlineOption` 函数中的 overlay 和 bottomMode 处理逻辑。

在 `topOverlay` 处理中添加 `'sar'` 和 `'ema'` 分支：

```ts
// 在 topOverlay switch/case 中
if (topOverlay === 'sar') {
  const sarData = indicators.sar(high, low, close)
  series.push({
    type: 'scatter' as const,
    name: 'SAR',
    data: sarData.map((v, i) => v != null ? [i, v] : null).filter(Boolean),
    symbol: 'circle',
    symbolSize: 4,
    itemStyle: { color: '#fbbf24' },
    xAxisIndex: 0,
    yAxisIndex: 0,
  })
}
if (topOverlay === 'ema') {
  const ema12 = indicators.ema(close, 12)
  const ema26 = indicators.ema(close, 26)
  series.push({
    type: 'line' as const,
    name: 'EMA12',
    data: ema12.map((v, i) => v != null ? [i, v] : null),
    smooth: true,
    lineStyle: { width: 1 },
    itemStyle: { color: '#22d3ee' },
    xAxisIndex: 0, yAxisIndex: 0,
  })
  series.push({
    type: 'line' as const,
    name: 'EMA26',
    data: ema26.map((v, i) => v != null ? [i, v] : null),
    smooth: true,
    lineStyle: { width: 1 },
    itemStyle: { color: '#f472b6' },
    xAxisIndex: 0, yAxisIndex: 0,
  })
}
```

在 `bottomMode` 处理中添加 `'cci'` 和 `'obv'`：

```ts
if (bottomMode === 'cci') {
  const cciData = indicators.cci(high, low, close)
  const min = Math.min(...cciData.filter(v => v != null)) as number
  const max = Math.max(...cciData.filter(v => v != null)) as number
  series.push({
    type: 'line' as const,
    name: 'CCI',
    data: cciData.map((v, i) => v != null ? [i, v] : null),
    smooth: true,
    lineStyle: { width: 1, color: '#a78bfa' },
    xAxisIndex: 0, yAxisIndex: 2,
  })
  // ±100 参考线
  markLine: {
    silent: true,
    data: [
      { yAxis: 100, lineStyle: { type: 'dashed', color: '#f87171', width: 1 } },
      { yAxis: -100, lineStyle: { type: 'dashed', color: '#4ade80', width: 1 } },
    ],
  }
}
if (bottomMode === 'obv') {
  const obvData = indicators.obv(close, volume)
  series.push({
    type: 'line' as const,
    name: 'OBV',
    data: obvData.map((v, i) => [i, v]),
    smooth: true,
    lineStyle: { width: 1, color: '#fb923c' },
    areaStyle: { color: 'rgba(251, 146, 60, 0.1)' },
    xAxisIndex: 0, yAxisIndex: 2,
  })
}
```

在 `buildKlineOption` 参数中加入 `topOverlay` 类型：`'none' | 'ma' | 'bb' | 'sar' | 'ema'`，`bottomMode` 类型：`'volume' | 'macd' | 'kdj' | 'rsi' | 'wr' | 'cci' | 'obv'`。

- [ ] **Step 8: 更新 CandlestickPanel overlay/mode 选择**

在 `CandlestickPanel.vue` 中找到 `topOverlay` 和 `bottomMode` ref，确认类型已扩展。更新模板中的选择器（`<select>` 或按钮）加入新选项。

- [ ] **Step 9: 编写单元测试**

读取 `frontend/src/lib/composables/__tests__/useIndicators.test.ts` 或类似文件，追加测试。

```ts
describe('sar', () => {
  it('computes SAR correctly for uptrend', () => {
    const high = [10, 11, 12, 13, 14, 15]
    const low = [9, 10, 11, 12, 13, 14]
    const close = [9.5, 10.5, 11.5, 12.5, 13.5, 14.5]
    const result = sar(high, low, close)
    expect(result.length).toBe(6)
    expect(result[0]).toBeNull()
    expect(result[result.length - 1]).toBeGreaterThan(0)
  })
})

describe('ema', () => {
  it('computes EMA correctly', () => {
    const data = [1, 2, 3, 4, 5, 6, 7, 8, 9, 10]
    const result = ema(data, 3)
    expect(result[0]).toBeNull()
    expect(result[1]).toBeNull()
    expect(result[2]).toBe(2) // SMA of first 3
  })
})
```

- [ ] **Step 10: 运行测试和类型检查**

```bash
cd frontend && npx vitest run && npx vue-tsc --noEmit
```

- [ ] **Step 11: 提交**

```bash
git add -A
git commit -m "feat: expand indicators - SAR, EMA, CCI, OBV"
```

---

### Task 3: DrawingController — 从 DrawingPanel 提取画线核心逻辑

**Files:**
- Create: `frontend/src/lib/chart/DrawingController.ts`
- Create: `frontend/src/lib/chart/types.ts`
- Read: `frontend/src/terminal/panels/DrawingPanel.vue` (参考来源)

**Interfaces:**
- Consumes: ECharts instance ref (提供 convertToPixel/convertFromPixel), HTMLCanvasElement
- Produces: `DrawingController` class with `mount()`, `destroy()`, `setMode()`, `setColor()`, `render()`, `clearAll()`

- [ ] **Step 1: 创建类型定义**

```ts
// frontend/src/lib/chart/types.ts

// 数据空间坐标（缩放平移后仍准确）
export interface DataPoint {
  dataIndex: number
  price: number
}

export type DrawingType = 'trendline' | 'horizontal' | 'fibonacci' | 'text'

export interface DrawingShape {
  id: number
  type: DrawingType
  points: DataPoint[]
  color: string
  text?: string
}

export type DrawingMode = 'cursor' | DrawingType
```

- [ ] **Step 2: 创建 DrawingController**

```ts
// frontend/src/lib/chart/DrawingController.ts
import type { EChartsType } from 'echarts'
import type { DrawingShape, DrawingMode, DataPoint } from './types'
import { useCanvasTheme } from '@/lib/canvas-theme'

export class DrawingController {
  private echarts: EChartsType | null = null
  private canvas: HTMLCanvasElement | null = null
  private ctx: CanvasRenderingContext2D | null = null
  private mode: DrawingMode = 'cursor'
  private color = '#58a6ff'
  private drawings: DrawingShape[] = []
  private activeSymbol = ''
  private isDrawing = false
  private startPoint: DataPoint | null = null
  private currentPixel: { x: number; y: number } | null = null
  private nextId = 1

  private storageKey(): string { return `drawings-v2/${this.activeSymbol}` }

  mount(echarts: EChartsType, canvas: HTMLCanvasElement, symbol: string) {
    this.echarts = echarts
    this.canvas = canvas
    this.ctx = canvas.getContext('2d')
    this.activeSymbol = symbol
    this.loadDrawings()
    this.render()
  }

  destroy() {
    this.echarts = null
    this.canvas = null
    this.ctx = null
    this.drawings = []
  }

  updateSymbol(symbol: string) {
    this.saveDrawings()
    this.activeSymbol = symbol
    this.loadDrawings()
    this.render()
  }

  setMode(mode: DrawingMode) { this.mode = mode; this.isDrawing = false; this.startPoint = null; this.currentPixel = null }
  setColor(color: string) { this.color = color }

  loadDrawings() {
    try {
      const raw = localStorage.getItem(this.storageKey()) || '[]'
      this.drawings = JSON.parse(raw)
      this.nextId = this.drawings.reduce((max, d) => Math.max(max, d.id), 0) + 1
    } catch { this.drawings = [] }
  }

  saveDrawings() {
    localStorage.setItem(this.storageKey(), JSON.stringify(this.drawings))
  }

  clearAll() {
    this.drawings = []
    this.saveDrawings()
    this.render()
  }

  render() {
    const ctx = this.ctx
    const canvas = this.canvas
    if (!ctx || !canvas || !this.echarts) return
    canvas.width = canvas.clientWidth
    canvas.height = canvas.clientHeight

    // 清除画布（透明背景，不遮挡 ECharts）
    ctx.clearRect(0, 0, canvas.width, canvas.height)

    // 绘制已保存图形
    for (const d of this.drawings) {
      this.drawShape(ctx, d)
    }

    // 绘制进行中的图形
    if (this.isDrawing && this.startPoint && this.currentPixel) {
      const preview: DrawingShape = {
        id: -1,
        type: this.mode as DrawingShape['type'],
        points: [this.startPoint, {
          dataIndex: this.startPoint.dataIndex,
          price: this.startPoint.price,
        }],
        color: this.color,
      }

      // 实时预览需要用像素坐标直接画（还没完成，没有第二个数据点）
      const echarts = this.echarts
      const startPixel = echarts.convertToPixel({ seriesIndex: 0 }, [this.startPoint.dataIndex, this.startPoint.price])
      if (!startPixel) return
      const previewPoints: { x: number; y: number }[] = [{ x: startPixel[0], y: startPixel[1] }]
      if (this.currentPixel) {
        previewPoints.push({ x: this.currentPixel.x, y: this.currentPixel.y })
      }
      this.drawShapePixels(ctx, { ...preview, points: previewPoints as any })
    }
  }

  private toDataPoint(pixelX: number, pixelY: number): DataPoint | null {
    if (!this.echarts) return null
    const coord = this.echarts.convertFromPixel({ seriesIndex: 0 }, [pixelX, pixelY])
    if (!coord || !Array.isArray(coord) || coord.length < 2) return null
    return { dataIndex: Math.round(coord[0]), price: coord[1] }
  }

  private drawShape(ctx: CanvasRenderingContext2D, d: DrawingShape) {
    if (!this.echarts) return
    const pixels = d.points.map(p => this.echarts!.convertToPixel({ seriesIndex: 0 }, [p.dataIndex, p.price]))
    if (pixels.some(p => !p)) return
    this.drawShapePixels(ctx, { ...d, points: pixels.filter(Boolean).map(p => ({ x: p![0], y: p![1] })) })
  }

  private drawShapePixels(ctx: CanvasRenderingContext2D, d: any) {
    ctx.strokeStyle = d.color
    ctx.fillStyle = d.color
    ctx.lineWidth = 2
    ctx.font = '13px monospace'
    ctx.setLineDash([])

    const [a, b] = d.points
    if (!b && d.type !== 'text') return

    switch (d.type) {
      case 'trendline':
        ctx.beginPath(); ctx.moveTo(a.x, a.y); ctx.lineTo(b.x, b.y); ctx.stroke()
        break
      case 'horizontal':
        ctx.beginPath(); ctx.moveTo(0, b.y); ctx.lineTo(ctx.canvas.width, b.y); ctx.stroke()
        ctx.fillText(b.y.toFixed(2), 6, b.y - 4)
        break
      case 'fibonacci': {
        const dx = b.x - a.x
        const dy = b.y - a.y
        const ratios = [0, 0.236, 0.382, 0.5, 0.618, 0.786, 1]
        const colors = ['#f87171', '#fb923c', '#fbbf24', '#4ade80', '#22d3ee', '#818cf8', '#e879f9']
        for (let i = 0; i < ratios.length; i++) {
          const y = a.y + ratios[i] * dy
          ctx.strokeStyle = colors[i]
          ctx.lineWidth = 1
          ctx.setLineDash([4, 4])
          ctx.beginPath(); ctx.moveTo(0, y); ctx.lineTo(ctx.canvas.width, y); ctx.stroke()
          ctx.fillText((ratios[i] * 100).toFixed(1) + '%', 6, y - 4)
        }
        ctx.setLineDash([])
        break
      }
      case 'text': {
        const p = d.points[0]
        if (p) ctx.fillText(d.text || '', p.x, p.y)
        break
      }
    }
  }

  // 鼠标事件
  onMouseDown(e: MouseEvent) {
    if (this.mode === 'cursor' || !this.canvas) return
    this.isDrawing = true
    const rect = this.canvas.getBoundingClientRect()
    const px = e.clientX - rect.left
    const py = e.clientY - rect.top
    this.currentPixel = { x: px, y: py }
    const dp = this.toDataPoint(px, py)
    if (dp) this.startPoint = dp
  }

  onMouseMove(e: MouseEvent) {
    if (!this.isDrawing || !this.canvas) return
    const rect = this.canvas.getBoundingClientRect()
    this.currentPixel = { x: e.clientX - rect.left, y: e.clientY - rect.top }
    this.render()
  }

  onMouseUp(_e: MouseEvent) {
    if (!this.isDrawing || !this.startPoint || !this.canvas) return
    this.isDrawing = false

    if (this.mode === 'text') {
      const text = prompt('输入文字:')
      if (!text) { this.render(); return }
      this.drawings.push({
        id: this.nextId++,
        type: 'text',
        points: [this.startPoint],
        color: this.color,
        text,
      })
    } else {
      const rect = this.canvas.getBoundingClientRect()
      // 获取结束点数据坐标
      const endDp = this.toDataPoint(this.currentPixel!.x, this.currentPixel!.y)
      if (!endDp) { this.render(); return }
      this.drawings.push({
        id: this.nextId++,
        type: this.mode as DrawingShape['type'],
        points: [this.startPoint, endDp],
        color: this.color,
      })
    }

    this.startPoint = null
    this.currentPixel = null
    this.saveDrawings()
    this.render()
  }
}
```

- [ ] **Step 3: 运行类型检查**

```bash
cd frontend && npx vue-tsc --noEmit
```

- [ ] **Step 4: 提交**

```bash
git add -A
git commit -m "feat: DrawingController - canvas overlay drawing engine with echarts coord mapping"
```

---

### Task 4: Canvas Overlay + 画线模式集成到 CandlestickPanel

**Files:**
- Modify: `frontend/src/terminal/panels/CandlestickPanel.vue`
- Modify: `frontend/src/terminal/components/panel/KlineChart.vue`

**Interfaces:**
- Consumes: `DrawingController` from Task 3
- Produces: 画线工具栏 UI + Canvas overlay DOM + 快捷键 Shift+D/Esc

- [ ] **Step 1: KlineChart 暴露 ECharts 实例**

在 `KlineChart.vue` 中：

```ts
const chartRef = shallowRef<InstanceType<typeof VChart>>()

// 暴露 ECharts 实例
const getEchartsInstance = () => chartRef.value?.getEChartsInstance?.() ?? null
defineExpose({ refreshSize, getEchartsInstance })
```

- [ ] **Step 2: CandlestickPanel 集成 DrawingController**

在 script 中：

```ts
import { DrawingController } from '@/lib/chart/DrawingController'
import { onMounted, nextTick } from 'vue'

const klineChartRef = ref<InstanceType<typeof KlineChart> | null>(null)

const drawingController = ref<DrawingController | null>(null)
const drawingMode = ref(false)

function toggleDrawingMode() {
  drawingMode.value = !drawingMode.value
  if (dc) dc.setMode(drawingMode.value ? 'trendline' : 'cursor')
}
```

- [ ] **Step 3: Canvas overlay 模板**

在 `CandlestickPanel.vue` 模板中，KlineChart 周围包裹：

```vue
<div class="chart-area" style="position: relative; flex: 1;">
  <KlineChart
    ref="klineChartRef"
    :option="option"
    :symbol="symbol"
    :loading="loading"
  />
  <canvas
    ref="drawingCanvasRef"
    class="canvas-overlay"
    :class="{ 'drawing-mode': drawingMode }"
    @mousedown="onDrawingMouseDown"
    @mousemove="onDrawingMouseMove"
    @mouseup="onDrawingMouseUp"
  />
</div>
```

样式：

```css
.canvas-overlay {
  position: absolute;
  top: 0; left: 0;
  width: 100%; height: 100%;
  pointer-events: none;
  z-index: 10;
}
.canvas-overlay.drawing-mode {
  pointer-events: auto;
  cursor: crosshair;
}
```

- [ ] **Step 4: 画线工具栏模板**

在 InfoBar 和 chart-area 之间或 chart-area 内浮动：

```vue
<div class="drawing-toolbar" v-if="drawingMode">
  <button @click="dc?.setMode('cursor')" :class="{active: dc?.mode === 'cursor'}" title="光标">↖</button>
  <button @click="dc?.setMode('trendline')" :class="{active: dc?.mode === 'trendline'}" title="趋势线">╱</button>
  <button @click="dc?.setMode('horizontal')" :class="{active: dc?.mode === 'horizontal'}" title="水平线">━</button>
  <button @click="dc?.setMode('fibonacci')" :class="{active: dc?.mode === 'fibonacci'}" title="斐波那契">F</button>
  <button @click="dc?.setMode('text')" :class="{active: dc?.mode === 'text'}" title="文字">T</button>
  <input type="color" v-model="drawingColor" @input="dc?.setColor(drawingColor)" />
  <button @click="dc?.clearAll()" title="全部清除">✕</button>
</div>
```

- [ ] **Step 5: 事件绑定**

在 script 中：

```ts
const drawingCanvasRef = ref<HTMLCanvasElement | null>(null)
let dc: DrawingController | null = null

onMounted(() => {
  nextTick(() => {
    const echarts = klineChartRef.value?.getEchartsInstance?.()
    if (echarts && drawingCanvasRef.value) {
      dc = new DrawingController()
      dc.mount(echarts, drawingCanvasRef.value, symbol.value)
      drawingController.value = dc
    }
  })
})

// ECharts 'finished' 事件触发画线重绘
watch(option, () => {
  nextTick(() => dc?.render())
})

function onDrawingMouseDown(e: MouseEvent) { dc?.onMouseDown(e) }
function onDrawingMouseMove(e: MouseEvent) { dc?.onMouseMove(e) }
function onDrawingMouseUp(e: MouseEvent) { dc?.onMouseUp(e) }

const drawingColor = ref('#58a6ff')
watch(drawingColor, (c) => dc?.setColor(c))
```

- [ ] **Step 6: 快捷键绑定**

在 `CandlestickPanel.vue` 中添加：

```ts
function onKeyDown(e: KeyboardEvent) {
  if (e.key === 'd' && e.shiftKey) {
    e.preventDefault()
    toggleDrawingMode()
  }
  if (e.key === 'Escape' && drawingMode.value) {
    drawingMode.value = false
    dc?.setMode('cursor')
  }
}

onMounted(() => {
  window.addEventListener('keydown', onKeyDown)
})
onUnmounted(() => {
  window.removeEventListener('keydown', onKeyDown)
})
```

- [ ] **Step 7: 运行类型检查**

```bash
cd frontend && npx vue-tsc --noEmit
```

- [ ] **Step 8: 提交**

```bash
git add -A
git commit -m "feat: integrate DrawingController into CandlestickPanel with canvas overlay"
```

---

### Task 5: 自定义十字光标 (Crosshair)

**Files:**
- Create: `frontend/src/lib/chart/Crosshair.ts`
- Modify: `frontend/src/terminal/panels/CandlestickPanel.vue`

**Interfaces:**
- Consumes: ECharts instance ref, HTMLCanvasElement (同一 Canvas overlay)
- Produces: 在 Canvas overlay 上绘制十字线 + 侧边标尺 + 浮动数据面板

- [ ] **Step 1: 创建 Crosshair 类**

```ts
// frontend/src/lib/chart/Crosshair.ts
import type { EChartsType } from 'echarts'

export interface CrosshairData {
  time: string
  open: number
  high: number
  low: number
  close: number
  volume: number
  change: number
  changePercent: number
}

export class Crosshair {
  private echarts: EChartsType | null = null
  private canvas: HTMLCanvasElement | null = null
  private ctx: CanvasRenderingContext2D | null = null
  private visible = false
  private mouseX = 0
  private mouseY = 0
  private data: CrosshairData | null = null
  private onHover: ((d: CrosshairData | null) => void) | null = null

  mount(echarts: EChartsType, canvas: HTMLCanvasElement, onHover?: (d: CrosshairData | null) => void) {
    this.echarts = echarts
    this.canvas = canvas
    this.ctx = canvas.getContext('2d')
    this.onHover = onHover ?? null
  }

  destroy() {
    this.echarts = null
    this.canvas = null
    this.ctx = null
  }

  show(x: number, y: number) {
    this.visible = true
    this.mouseX = x
    this.mouseY = y
    this.updateData()
    this.render()
  }

  hide() {
    this.visible = false
    this.data = null
    this.onHover?.(null)
    this.render()
  }

  private updateData() {
    if (!this.echarts) return
    const coord = this.echarts.convertFromPixel({ seriesIndex: 0 }, [this.mouseX, this.mouseY])
    if (!coord || !Array.isArray(coord)) { this.data = null; return }
    const dataIndex = Math.round(coord[0])
    // Get actual data from echarts - need to access the series data
    const model = this.echarts.getModel()
    const series = model.getSeriesByIndex(0)
    if (!series) { this.data = null; return }
    const rawData = series.getRawData()
    if (!rawData || dataIndex < 0 || dataIndex >= rawData.length) { this.data = null; return }
    const item = rawData.getValues(dataIndex) as any
    // Candlestick data format: [time, open, close, low, high, volume] or [time, open, high, low, close, volume]
    this.data = {
      time: String(item[0] || ''),
      open: Number(item[1]),
      high: Number(item[3] || item[2]), // depends on echarts data format
      low: Number(item[4] || item[3]),
      close: Number(item[2] || item[4]),
      volume: Number(item[5] || item[6] || 0),
      change: 0,
      changePercent: 0,
    }
    // Calculate change from prev close
    if (dataIndex > 0) {
      const prevValues = rawData.getValues(dataIndex - 1) as any
      const prevClose = Number(prevValues[2] || prevValues[4])
      this.data.change = this.data.close - prevClose
      this.data.changePercent = prevClose !== 0 ? (this.data.change / prevClose) * 100 : 0
    }
    this.onHover?.(this.data)
  }

  render() {
    const ctx = this.ctx
    const canvas = this.canvas
    if (!ctx || !canvas) return
    if (!this.visible) return

    const w = canvas.width
    const h = canvas.height

    // 垂直线
    ctx.strokeStyle = 'rgba(128, 128, 128, 0.5)'
    ctx.lineWidth = 1
    ctx.setLineDash([4, 4])
    ctx.beginPath(); ctx.moveTo(this.mouseX, 0); ctx.lineTo(this.mouseX, h); ctx.stroke()

    // 水平线
    ctx.beginPath(); ctx.moveTo(0, this.mouseY); ctx.lineTo(w, this.mouseY); ctx.stroke()
    ctx.setLineDash([])

    // 侧边价格标尺
    if (this.data) {
      const priceText = this.data.close.toFixed(2)
      ctx.fillStyle = '#333'
      ctx.fillRect(w - 80, this.mouseY - 8, 80, 16)
      ctx.fillStyle = '#fff'
      ctx.fillText(priceText, w - 76, this.mouseY + 4)

      // 底部时间标尺
      const timeText = this.data.time
      ctx.fillStyle = '#333'
      const tw = ctx.measureText(timeText).width
      ctx.fillRect(this.mouseX - tw / 2 - 4, h - 20, tw + 8, 20)
      ctx.fillStyle = '#fff'
      ctx.fillText(timeText, this.mouseX - tw / 2, h - 6)
    }

    // 数据浮层
    if (this.data) {
      const lines = [
        `时间: ${this.data.time}`,
        `开: ${this.data.open.toFixed(2)}`,
        `高: ${this.data.high.toFixed(2)}`,
        `低: ${this.data.low.toFixed(2)}`,
        `收: ${this.data.close.toFixed(2)}`,
        `涨跌: ${this.data.change >= 0 ? '+' : ''}${this.data.change.toFixed(2)}`,
        `涨幅: ${this.data.changePercent >= 0 ? '+' : ''}${this.data.changePercent.toFixed(2)}%`,
        `量: ${this.data.volume.toFixed(0)}`,
      ]
      const lineH = 18
      const boxW = 140
      const boxH = lines.length * lineH + 8
      const boxX = Math.min(this.mouseX + 16, w - boxW - 8)
      const boxY = Math.max(8, Math.min(this.mouseY - boxH / 2, h - boxH - 8))

      ctx.fillStyle = 'rgba(0, 0, 0, 0.75)'
      ctx.fillRect(boxX, boxY, boxW, boxH)
      ctx.fillStyle = '#fff'
      ctx.font = '11px monospace'
      lines.forEach((line, i) => {
        ctx.fillText(line, boxX + 6, boxY + 14 + i * lineH)
      })
    }
  }
}
```

- [ ] **Step 2: 在 CandlestickPanel 中集成 Crosshair**

在 `CandlestickPanel.vue` 中：

```ts
import { Crosshair } from '@/lib/chart/Crosshair'

let crosshair: Crosshair | null = null

// 在 onMounted 中初始化
crosshair = new Crosshair()
crosshair.mount(echartsInstance, drawingCanvasRef.value)

// ECharts 鼠标事件转发到 Crosshair
// 需要监听 ECharts 的 mousemove/mouseout
// 由于 ECharts 在 canvas 内部处理事件，需要在 echarts instance 上注册事件
echartsInstance.on('mousemove', (params: any) => {
  if (!drawingMode.value) {
    crosshair?.show(params.event?.offsetX, params.event?.offsetY)
  }
})
echartsInstance.on('mouseout', () => {
  crosshair?.hide()
})

// 画线模式下隐藏十字光标
watch(drawingMode, (mode) => {
  if (mode) crosshair?.hide()
})
```

- [ ] **Step 3: 运行类型检查**

```bash
cd frontend && npx vue-tsc --noEmit
```

- [ ] **Step 4: 提交**

```bash
git add -A
git commit -m "feat: custom crosshair with info tooltip and axis rulers"
```

---

### Task 6: 事件标记 + 大盘叠加

**Files:**
- Modify: `frontend/src/lib/chart/EventMarker.ts` (Create)
- Modify: `frontend/src/lib/buildChartOption.ts`
- Modify: `frontend/src/terminal/panels/CandlestickPanel.vue`

- [ ] **Step 1: 创建 EventMarker 工具**

```ts
// frontend/src/lib/chart/EventMarker.ts
export interface EventMarker {
  dataIndex: number
  label: string
  color: string
  type: 'line' | 'area'
}

// 从 K 线数据检测涨停/跌停
export function detectLimitUpDown(data: KlineDataItem[], limitUp?: number, limitDown?: number): EventMarker[] {
  const markers: EventMarker[] = []
  if (!data.length) return markers

  // A 股: 如果 close 接近 limit_up/down
  if (limitUp && limitDown) {
    for (let i = 0; i < data.length; i++) {
      const bar = data[i]
      if (Math.abs(bar[4] - limitUp) / limitUp < 0.001) { // close == limitUp
        markers.push({ dataIndex: i, label: '涨停', color: '#f87171', type: 'line' })
      }
      if (Math.abs(bar[4] - limitDown) / limitDown < 0.001) {
        markers.push({ dataIndex: i, label: '跌停', color: '#4ade80', type: 'line' })
      }
    }
  }
  return markers
}

// 从 GetExDividend 数据生成标记
export function exDividendMarkers(dates: number[], data: KlineDataItem[], timestamps: number[]): EventMarker[] {
  return dates
    .map(d => {
      const idx = timestamps.findIndex(t => {
        const date = new Date(t * 1000)
        const exDate = new Date(d * 1000)
        return date.getFullYear() === exDate.getFullYear() &&
               date.getMonth() === exDate.getMonth() &&
               date.getDate() === exDate.getDate()
      })
      return idx >= 0 ? { dataIndex: idx, label: '除权', color: '#818cf8', type: 'line' as const } : null
    })
    .filter(Boolean) as EventMarker[]
}
```

- [ ] **Step 2: 在 buildChartOption 中集成事件标记**

在 `buildKlineOption` 中增加 `eventMarkers?: EventMarker[]` 参数，注入 markLine：

```ts
function buildKlineOption(
  data: KlineDataItem[],
  topOverlay: OverlayType,
  bottomMode: SubChartType,
  theme: ChartThemeColors,
  indicatorCache: IndicatorCache,
  symbol: string,
  interval: string,
  eventMarkers?: EventMarker[],
  indexOverlay?: IndexOverlayData | null,
): ECBasicOption {
  const series: any[] = [/* ... 现有 series */]

  // 事件标记
  if (eventMarkers?.length) {
    series[0].markLine = {
      silent: true,
      symbol: 'none',
      data: eventMarkers.map(m => ({
        xAxis: m.dataIndex,
        label: { formatter: m.label, color: m.color, fontSize: 10 },
        lineStyle: { color: m.color, type: 'dashed' as any, width: 1 },
      })),
    }
  }

  // 大盘叠加（新增 series）
  if (indexOverlay) {
    series.push({
      type: 'line',
      name: indexOverlay.name,
      data: indexOverlay.data,
      smooth: true,
      lineStyle: { width: 1, color: '#a78bfa' },
      xAxisIndex: 0,
      yAxisIndex: 1, // 右侧 Y 轴
    })
  }
}
```

- [ ] **Step 3: CandlestickPanel 大盘叠加选择器**

在模板中添加 `<select>` 或按钮组：

```vue
<select v-model="indexOverlaySymbol" class="toolbar-select">
  <option value="">不叠加</option>
  <option value="000001">上证指数</option>
  <option value="399001">深证成指</option>
  <option value="399006">创业板指</option>
</select>
```

在 script 中：

```ts
const indexOverlaySymbol = ref('')
const indexData = ref<IndexOverlayData | null>(null)

watch(indexOverlaySymbol, async (sym) => {
  if (!sym) { indexData.value = null; return }
  try {
    const bars = await fetchOHLCV('CN', sym, '1d', 'qfq', now - 365*86400, now)
    // 归一化到主图价格区间
    indexData.value = normalizeIndex(bars, ohlcvData.value)
  } catch (e) {
    console.error('[Candlestick] loadIndex:', e)
  }
})
```

- [ ] **Step 4: 运行类型检查**

```bash
cd frontend && npx vue-tsc --noEmit
```

- [ ] **Step 5: 提交**

```bash
git add -A
git commit -m "feat: event markers on kline + index overlay"
```

---

### Task 7: 快捷键 + 右键菜单

**Files:**
- Modify: `frontend/src/terminal/panels/CandlestickPanel.vue`

- [ ] **Step 1: 扩展 onKeyDown 处理更多快捷键**

```ts
// K 线周期列表
const intervals = ['1m', '5m', '15m', '30m', '1h', '1d', '1w'] as const

function onKeyDown(e: KeyboardEvent) {
  if (e.target instanceof HTMLInputElement || e.target instanceof HTMLTextAreaElement) return

  switch (e.key) {
    case 'ArrowLeft':
      // 前一个周期
      e.preventDefault()
      const idx = intervals.indexOf(interval as any)
      if (idx > 0) interval.value = intervals[idx - 1]
      break
    case 'ArrowRight':
      e.preventDefault()
      const idx2 = intervals.indexOf(interval as any)
      if (idx2 < intervals.length - 1) interval.value = intervals[idx2 + 1]
      break
    case 'ArrowUp':
      e.preventDefault()
      // zoom in: 减少显示范围 (暂时用 dataZoom 内部处理)
      break
    case 'ArrowDown':
      e.preventDefault()
      break
    case 'g':
    case 'G':
      // 跳转到日期 - 弹出日期输入
      e.preventDefault()
      const dateStr = prompt('跳转到日期 (YYYY-MM-DD):')
      if (dateStr) jumpToDate(dateStr)
      break
    case 'Escape':
      if (drawingMode.value) {
        drawingMode.value = false
        dc?.setMode('cursor')
      }
      break
    case 'Delete':
    case 'Backspace':
      // 删除选中画线（简化：删除最后一条）
      if (drawingMode.value) {
        dc?.clearAll()
        e.preventDefault()
      }
      break
    case 'D':
      if (e.shiftKey) {
        e.preventDefault()
        toggleDrawingMode()
      }
      break
  }
}

function jumpToDate(dateStr: string) {
  // 计算 dataZoom 起始位置
  // 简单实现: 查找对应日期的 index
  const target = new Date(dateStr).getTime()
  const idx = timestamps.value.findIndex(t => {
    const d = new Date(t * 1000)
    return d.getFullYear() === new Date(target).getFullYear() &&
           d.getMonth() === new Date(target).getMonth() &&
           d.getDate() === new Date(target).getDate()
  })
  if (idx >= 0 && ohlcvData.value.length) {
    // 设置 dataZoom 到目标位置
    const start = idx / ohlcvData.value.length
    // ECharts dataZoom 通过 option 控制 dispatchAction
    const echarts = klineChartRef.value?.getEchartsInstance?.()
    echarts?.dispatchAction({
      type: 'dataZoom',
      start: Math.max(0, (1 - 0.3) * start * 100),
      end: Math.min(100, (1 + 0.3) * start * 100),
    })
  }
}
```

- [ ] **Step 2: 右键菜单**

在模板中添加条件渲染的 context menu：

```vue
<div
  v-if="contextMenu.visible"
  class="context-menu"
  :style="{ left: contextMenu.x + 'px', top: contextMenu.y + 'px' }"
>
  <div class="menu-item" @click="switchInterval('1d')">日线</div>
  <div class="menu-item" @click="switchInterval('1w')">周线</div>
  <div class="menu-separator"></div>
  <div class="menu-item" @click="switchOverlay('none')">清除叠加</div>
  <div class="menu-item" @click="switchOverlay('ma')">叠加MA</div>
  <div class="menu-item" @click="switchOverlay('bb')">叠加布林带</div>
  <div class="menu-separator"></div>
  <div class="menu-item" @click="dc?.clearAll(); contextMenu.visible = false">清除画线</div>
  <div class="menu-item" @click="copyChart()">截图</div>
</div>
```

script 中：

```ts
const contextMenu = reactive({ visible: false, x: 0, y: 0 })

function onContextMenu(e: MouseEvent) {
  e.preventDefault()
  contextMenu.x = e.clientX
  contextMenu.y = e.clientY
  contextMenu.visible = true
}

function closeContextMenu() { contextMenu.visible = false }

onMounted(() => {
  document.addEventListener('click', closeContextMenu)
})
onUnmounted(() => {
  document.removeEventListener('click', closeContextMenu)
})
```

- [ ] **Step 3: 运行类型检查**

```bash
cd frontend && npx vue-tsc --noEmit
```

- [ ] **Step 4: 提交**

```bash
git add -A
git commit -m "feat: keyboard shortcuts and context menu for candlestick chart"
```

---

### Task 8: WebSocket Hub (Go 后端)

**Files:**
- Create: `app/internal/ws/hub.go`
- Create: `app/internal/ws/client.go`
- Create: `app/internal/ws/topics/kline.go`
- Create: `app/internal/ws/topics/tick.go`
- Create: `app/internal/ws/topics/depth.go`
- Create: `app/internal/ws/handler.go`

- [ ] **Step 1: 创建类型定义文件**

```go
// app/internal/ws/topics/kline.go
package topics

type KlineUpdate struct {
    Symbol   string  `json:"symbol"`
    Interval string  `json:"interval"`
    Time     int64   `json:"time"`
    Open     float64 `json:"open"`
    High     float64 `json:"high"`
    Low      float64 `json:"low"`
    Close    float64 `json:"close"`
    Volume   float64 `json:"volume"`
    IsClosed bool    `json:"is_closed"`
}
```

```go
// app/internal/ws/topics/tick.go
package topics

type Tick struct {
    Symbol string  `json:"symbol"`
    Price  float64 `json:"price"`
    Volume float64 `json:"volume"`
    Time   int64   `json:"time"`
    Side   string  `json:"side"` // "buy" | "sell"
}
```

```go
// app/internal/ws/topics/depth.go
package topics

type DepthLevel struct {
    Price  float64 `json:"price"`
    Volume float64 `json:"volume"`
}

type DepthUpdate struct {
    Symbol string        `json:"symbol"`
    Bids   []DepthLevel  `json:"bids"`
    Asks   []DepthLevel  `json:"asks"`
}
```

- [ ] **Step 2: 创建 Hub**

```go
// app/internal/ws/hub.go
package ws

import (
    "encoding/json"
    "sync"
)

type Message struct {
    Topic string          `json:"topic"`
    Data  json.RawMessage `json:"data"`
}

type Hub struct {
    mu         sync.RWMutex
    clients    map[*Client]bool
    topics     map[string]map[*Client]bool // topic → set of clients
    register   chan *Client
    unregister chan *Client
}

func NewHub() *Hub {
    return &Hub{
        clients:    make(map[*Client]bool),
        topics:     make(map[string]map[*Client]bool),
        register:   make(chan *Client, 256),
        unregister: make(chan *Client, 256),
    }
}

func (h *Hub) Run() {
    for {
        select {
        case client := <-h.register:
            h.mu.Lock()
            h.clients[client] = true
            h.mu.Unlock()
        case client := <-h.unregister:
            h.mu.Lock()
            if _, ok := h.clients[client]; ok {
                delete(h.clients, client)
                for topic := range client.topics {
                    if subs, ok := h.topics[topic]; ok {
                        delete(subs, client)
                    }
                }
                close(client.send)
            }
            h.mu.Unlock()
        }
    }
}

func (h *Hub) Subscribe(client *Client, topic string) {
    h.mu.Lock()
    defer h.mu.Unlock()
    if h.topics[topic] == nil {
        h.topics[topic] = make(map[*Client]bool)
    }
    h.topics[topic][client] = true
    client.topics[topic] = true
}

func (h *Hub) Unsubscribe(client *Client, topic string) {
    h.mu.Lock()
    defer h.mu.Unlock()
    if subs, ok := h.topics[topic]; ok {
        delete(subs, client)
    }
    delete(client.topics, topic)
}

func (h *Hub) Broadcast(topic string, data any) {
    h.mu.RLock()
    subs := h.topics[topic]
    h.mu.RUnlock()

    raw, err := json.Marshal(data)
    if err != nil {
        return
    }

    msg := &Message{Topic: topic, Data: raw}
    rawMsg, _ := json.Marshal(msg)

    h.mu.RLock()
    defer h.mu.RUnlock()
    for client := range subs {
        select {
        case client.send <- rawMsg:
        default:
            // client send buffer full, skip
        }
    }
}
```

- [ ] **Step 3: 创建 Client**

```go
// app/internal/ws/client.go
package ws

import (
    "encoding/json"
    "log/slog"
    "time"
    "github.com/coder/websocket"
)

const (
    writeWait      = 10 * time.Second
    pongWait       = 60 * time.Second
    pingPeriod     = (pongWait * 9) / 10
    maxMessageSize = 4096
    sendBufSize    = 256
)

type Client struct {
    hub    *Hub
    conn   *websocket.Conn
    send   chan []byte
    topics map[string]bool
}

type subscribeMessage struct {
    Type   string   `json:"type"`
    Topics []string `json:"topics"`
}

func NewClient(hub *Hub, conn *websocket.Conn) *Client {
    return &Client{
        hub:    hub,
        conn:   conn,
        send:   make(chan []byte, sendBufSize),
        topics: make(map[string]bool),
    }
}

func (c *Client) ReadPump() {
    defer func() {
        c.hub.unregister <- c
        c.conn.Close(websocket.StatusNormalClosure, "connection closed")
    }()

    c.conn.SetReadLimit(maxMessageSize)
    c.conn.SetReadDeadline(time.Now().Add(pongWait))
    c.conn.SetPongHandler(func(string) error {
        c.conn.SetReadDeadline(time.Now().Add(pongWait))
        return nil
    })

    for {
        _, msg, err := c.conn.Read(nil)
        if err != nil {
            if websocket.CloseStatus(err) != websocket.StatusNormalClosure {
                slog.Error("ws read error", "err", err)
            }
            break
        }

        var sub subscribeMessage
        if err := json.Unmarshal(msg, &sub); err != nil {
            continue
        }

        switch sub.Type {
        case "subscribe":
            for _, topic := range sub.Topics {
                c.hub.Subscribe(c, topic)
            }
        case "unsubscribe":
            for _, topic := range sub.Topics {
                c.hub.Unsubscribe(c, topic)
            }
        }
    }
}

func (c *Client) WritePump() {
    ticker := time.NewTicker(pingPeriod)
    defer func() {
        ticker.Stop()
        c.conn.Close(websocket.StatusNormalClosure, "connection closed")
    }()

    for {
        select {
        case message, ok := <-c.send:
            if !ok {
                c.conn.Write(nil, websocket.MessageText, []byte{})
                return
            }
            c.conn.SetWriteDeadline(time.Now().Add(writeWait))
            if err := c.conn.Write(nil, websocket.MessageText, message); err != nil {
                return
            }
        case <-ticker.C:
            c.conn.SetWriteDeadline(time.Now().Add(writeWait))
            if err := c.conn.Write(nil, websocket.MessagePing, []byte{}); err != nil {
                return
            }
        }
    }
}
```

- [ ] **Step 4: 创建 handler**

```go
// app/internal/ws/handler.go
package ws

import (
    "log/slog"
    "net/http"
    "github.com/coder/websocket"
)

var DefaultHub = NewHub()

func init() {
    go DefaultHub.Run()
}

func ServeWS(w http.ResponseWriter, r *http.Request) {
    conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
        InsecureSkipVerify: true, // dev only; prod should verify origin
    })
    if err != nil {
        slog.Error("ws accept", "err", err)
        return
    }

    client := NewClient(DefaultHub, conn)
    DefaultHub.register <- client

    go client.WritePump()
    go client.ReadPump()
}
```

- [ ] **Step 5: 注册 Wails 路由**

在 `app/main.go` 或 `app/app.go` 中注册 WS handler：

```go
import "quantflow/internal/ws"

// 在 app.Init() 或 main() 中
// Wails v3: app.Router.Handle("/ws", http.HandlerFunc(ws.ServeWS))
```

读取 `app/main.go` 和 `app/app.go` 找到正确的路由注册方式。

- [ ] **Step 6: 编译验证**

```bash
go vet ./...
go build ./...
```

- [ ] **Step 7: 提交**

```bash
git add -A
git commit -m "feat: WebSocket hub for real-time kline/tick/depth push"
```

---

### Task 9: useWebSocket 前端 composable + 集成到 CandlestickPanel

**Files:**
- Create: `frontend/src/lib/composables/useWebSocket.ts`
- Modify: `frontend/src/terminal/panels/CandlestickPanel.vue`

- [ ] **Step 1: 创建 useWebSocket composable**

```ts
// frontend/src/lib/composables/useWebSocket.ts
import { ref, onUnmounted } from 'vue'

type MessageHandler = (data: any) => void

export function useWebSocket() {
  const ws = ref<WebSocket | null>(null)
  const connected = ref(false)
  const handlers = new Map<string, Set<MessageHandler>>()
  let reconnectTimer: ReturnType<typeof setTimeout> | null = null
  let reconnectAttempts = 0
  const maxReconnectDelay = 30000

  function connect(url: string, topics: string[]) {
    if (ws.value?.readyState === WebSocket.OPEN) return

    ws.value = new WebSocket(url)

    ws.value.onopen = () => {
      connected.value = true
      reconnectAttempts = 0
      // 订阅主题
      ws.value?.send(JSON.stringify({ type: 'subscribe', topics }))
    }

    ws.value.onmessage = (event) => {
      try {
        const msg = JSON.parse(event.data)
        const topicHandlers = handlers.get(msg.topic)
        if (topicHandlers) {
          topicHandlers.forEach(h => h(msg.data))
        }
        // 也触发通配符监听
        const wildcard = handlers.get('*')
        if (wildcard) {
          wildcard.forEach(h => h(msg))
        }
      } catch (e) {
        console.error('[WS] parse error:', e)
      }
    }

    ws.value.onclose = () => {
      connected.value = false
      ws.value = null
      scheduleReconnect(url, topics)
    }

    ws.value.onerror = () => {
      ws.value?.close()
    }
  }

  function scheduleReconnect(url: string, topics: string[]) {
    const delay = Math.min(1000 * Math.pow(2, reconnectAttempts), maxReconnectDelay)
    reconnectAttempts++
    reconnectTimer = setTimeout(() => connect(url, topics), delay)
  }

  function onMessage(topic: string, handler: MessageHandler) {
    if (!handlers.has(topic)) handlers.set(topic, new Set())
    handlers.get(topic)!.add(handler)
    return () => handlers.get(topic)?.delete(handler)
  }

  function disconnect() {
    if (reconnectTimer) clearTimeout(reconnectTimer)
    handlers.clear()
    ws.value?.close()
    ws.value = null
    connected.value = false
  }

  onUnmounted(() => disconnect())

  return { connect, onMessage, disconnect, connected }
}
```

- [ ] **Step 2: 在 CandlestickPanel 中集成 WebSocket**

```ts
import { useWebSocket } from '@/lib/composables/useWebSocket'

const { connect: wsConnect, onMessage: wsOnMessage, connected: wsConnected } = useWebSocket()

// 连接 WS
function initWebSocket() {
  const wsUrl = `ws://${window.location.host}/ws` // Wails 同源
  wsConnect(wsUrl, [`kline:${symbol.value}:${interval.value}`])

  wsOnMessage(`kline:${symbol.value}:${interval.value}`, (update: KlineUpdate) => {
    mergeOHLCVUpdate(ohlcvData.value, update)
  })
}

function mergeOHLCVUpdate(data: KlineDataItem[], update: KlineUpdate) {
  const { time, open, high, low, close, volume, isClosed } = update
  const last = data[data.length - 1]
  if (last && last[0] === time) {
    // 更新最后一根 K 线（实时更新）
    data[data.length - 1] = [time, open, high, low, close, volume]
  } else if (isClosed) {
    // 新 K 线
    data.push([time, open, high, low, close, volume])
  }
  // 触发响应式更新
  ohlcvData.value = [...data]
}

// symbol 或 interval 变更时重新订阅
watch([symbol, interval], ([sym, iv]) => {
  wsConnect(`ws://${window.location.host}/ws`, [`kline:${sym}:${iv}`])
})

onMounted(() => {
  initWebSocket()
})
```

- [ ] **Step 3: 修改 CandlestickPanel 的轮询逻辑**

当 WS 连接时，暂停轮询；断连时恢复：

```ts
watch(wsConnected, (connected) => {
  if (connected) {
    clearInterval(pollTimer)
  } else {
    pollTimer = setInterval(pollOHLCV, 30000)
  }
})
```

- [ ] **Step 4: 运行类型检查和前端测试**

```bash
cd frontend && npx vue-tsc --noEmit && npx vitest run
```

- [ ] **Step 5: 提交**

```bash
git add -A
git commit -m "feat: useWebSocket composable with auto-reconnect and topic subscription"
```

---

### Task 10: 清理 — 删除 DrawingPanel + registry 更新

**Files:**
- Delete: `frontend/src/terminal/panels/DrawingPanel.vue`
- Delete: `frontend/src/terminal/panels/__tests__/DrawingPanel.test.ts`
- Modify: `frontend/src/terminal/panels/registry.ts`

- [ ] **Step 1: 从 registry 移除 drawing**

找到 `registry.ts` 第 56 行，删除：

```ts
register('drawing', () => import('./DrawingPanel.vue'), { label: '绘图工具', category: '图表分析', description: '自由绘图标注' })
```

- [ ] **Step 2: 删除文件**

```bash
git rm frontend/src/terminal/panels/DrawingPanel.vue
git rm frontend/src/terminal/panels/__tests__/DrawingPanel.test.ts
```

- [ ] **Step 3: 确认无残留引用**

```bash
grep -r "DrawingPanel" frontend/src/ --include="*.ts" --include="*.vue"
# 预期: 只输出 DrawingController.ts 中提取的逻辑，无 DrawingPanel.vue 引用
```

- [ ] **Step 4: 运行完整检查**

```bash
cd frontend && npx vue-tsc --noEmit && npx vitest run
cd app && go vet ./...
```

- [ ] **Step 5: 最终提交**

```bash
git add -A
git commit -m "cleanup: remove standalone DrawingPanel, merged into CandlestickPanel"
```

---

## 执行完成检查清单

- [ ] 全量构建通过: `make build-full`
- [ ] Go vet 无警告
- [ ] TypeScript 类型检查通过
- [ ] 前端测试通过
- [ ] CHANGELOG.md 已更新
- [ ] 版本日期已检查（package.json, README, CHANGELOG）
