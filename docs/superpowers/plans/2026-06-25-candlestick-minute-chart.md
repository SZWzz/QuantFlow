# CandlestickPanel 分时图 Tab 实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** CandlestickPanel 内嵌 K线/分时双 tab 切换，分时图每 10s 从 mootdx 刷新分钟走势线 + 成交量。

**Architecture:** Go 后端新增 `GetMinuteLine` API → 调用 mootdx Python sidecar `FetchData(data_type="quote")` → 前端 CandlestickPanel 新增 `activeTab` 状态，分钟 tab 首次全量拉取 + 每 10s 增量轮询。K线 tab 分钟级 interval 增加 30s 自动刷新。

**Tech Stack:** Go 1.22/Wails v3 + Vue 3/TS + ECharts + Python mootdx gRPC sidecar

## Global Constraints

- Go 1.22+/Wails v3 + Vue 3/TS + Python 3.12 gRPC + SQLite WAL
- A 股涨红色/跌绿色（`#ef4444` / `#22c55e`）
- Wails v3 IPC：context.Context 自动注入，前端不传
- `(window as any).go.main.App.Method(...)` 调用模式
- 所有 mock 数据已清理，走真实 API

---

## File Structure

```
新增:
  internal/market/minuteline.go        # MinuteTick 类型 + 适配器接口
  internal/market/adapters/mootdx_minuteline.go  # mootdx 适配器

修改:
  internal/market/types.go             # 确保 MinuteTick 在 market 包
  internal/market/adapters/mootdx.go   # 挂上 FetchMinuteLine
  app.go                               # 新增 GetMinuteLine 导出方法
  frontend/src/terminal/panels/CandlestickPanel.vue  # tab 切换 + 分时图
```

---

### Task 1: Go 后端 MinuteTick 类型 + 适配器接口

**Files:**
- Create: `internal/market/minuteline.go`

**Interfaces:**
- Produces: `MinuteTick` struct, `MinuteLineProvider` interface

- [ ] **Step 1: 创建类型和接口**

```go
// minuteline.go — minute-line (intraday) data types and provider interface.
package market

// MinuteTick represents one minute's trade data during the day.
type MinuteTick struct {
	Time    string  `json:"time"`    // "09:35"
	Price   float64 `json:"price"`   // 该分钟均价
	Volume  float64 `json:"volume"`  // 该分钟成交量
	AvgPrice float64 `json:"avg_price"` // 日内累计均价
}

// MinuteLineProvider is implemented by adapters that can fetch intraday
// minute-line data (primarily mootdx via TDX TCP protocol).
type MinuteLineProvider interface {
	FetchMinuteLine(symbol string) ([]MinuteTick, error)
}
```

- [ ] **Step 2: 验证编译**

```bash
cd /Volumes/shenzy/vibe_coding/QuantFlow && go build -o /dev/null ./internal/market/...
```

- [ ] **Step 3: Commit**

```bash
git add internal/market/minuteline.go
git commit -m "[Market] add MinuteTick type and MinuteLineProvider interface"
```

---

### Task 2: mootdx 适配器实现 FetchMinuteLine

**Files:**
- Create: `internal/market/adapters/mootdx_minuteline.go`

**Interfaces:**
- Consumes: `MinuteTick`, `MinuteLineProvider` (Task 1)
- Produces: `MootdxAdapter.FetchMinuteLine(symbol string) ([]MinuteTick, error)`

- [ ] **Step 1: 实现 FetchMinuteLine**

```go
// mootdx_minuteline.go — mootdx adapter for intraday minute-line data.
package adapters

import (
	"encoding/json"
	"fmt"

	pb "quantflow/internal/python/proto"
	"quantflow/internal/market"
)

// FetchMinuteLine returns today's minute-by-minute price/volume ticks
// via the mootdx Python sidecar.
func (a *MootdxAdapter) FetchMinuteLine(symbol string) ([]market.MinuteTick, error) {
	if a.dataClient == nil {
		return nil, fmt.Errorf("mootdx: Python sidecar not connected")
	}

	resp, err := a.dataClient.FetchData(nil, &pb.FetchDataRequest{
		Source:   "mootdx",
		DataType: "quote",
		Symbols:  []string{symbol},
		Params:   map[string]string{"field": "minute"},
	})
	if err != nil {
		return nil, fmt.Errorf("mootdx minuteline: %w", err)
	}

	var raw []rawMinuteTick
	if err := json.Unmarshal(resp.Data, &raw); err != nil {
		return nil, fmt.Errorf("mootdx minuteline parse: %w", err)
	}

	ticks := make([]market.MinuteTick, 0, len(raw))
	for _, r := range raw {
		ticks = append(ticks, market.MinuteTick{
			Time:     r.Time,
			Price:    r.Price,
			Volume:   r.Volume,
			AvgPrice: r.AvgPrice,
		})
	}
	return ticks, nil
}

// rawMinuteTick mirrors the JSON from Python sidecar's _fetch_mootdx_quote output.
type rawMinuteTick struct {
	Time     string  `json:"time"`
	Price    float64 `json:"price"`
	Volume   float64 `json:"volume"`
	AvgPrice float64 `json:"avg_price"`
}
```

- [ ] **Step 2: 验证编译**

```bash
cd /Volumes/shenzy/vibe_coding/QuantFlow && go build -o /dev/null . 2>&1 | head -3
```

- [ ] **Step 3: Commit**

```bash
git add internal/market/adapters/mootdx_minuteline.go
git commit -m "[Adapter] add mootdx FetchMinuteLine for intraday data"
```

---

### Task 3: app.go 新增 GetMinuteLine 导出方法

**Files:**
- Modify: `app.go`

**Interfaces:**
- Consumes: `MinuteTick` (Task 1), `FetchMinuteLine` (Task 2)
- Produces: `App.GetMinuteLine("CN", symbol) → ([]MinuteTick, string, error)`

- [ ] **Step 1: 在 app.go 中新增方法（插入在 GetQuote 之后）**

```go
// GetMinuteLine returns today's intraday minute-by-minute ticks for a CN symbol.
// Data is fetched via mootdx (TDX TCP protocol) when the Python sidecar is available.
// Returns an empty slice on weekends, before market open, or when mootdx is unavailable.
func (a *App) GetMinuteLine(ctx context.Context, symbol string) ([]market.MinuteTick, string, error) {
	adpt := a.getMootdxAdapter()
	if adpt == nil {
		return nil, "unavailable", fmt.Errorf("mootdx adapter not available")
	}
	ticks, err := adpt.FetchMinuteLine(symbol)
	if err != nil {
		return nil, "unavailable", err
	}
	return ticks, "mootdx", nil
}
```

- [ ] **Step 2: 验证编译**

```bash
go build -o /dev/null .
```

- [ ] **Step 3: Commit**

```bash
git add app.go
git commit -m "[App] add GetMinuteLine API for intraday chart"
```

---

### Task 4: CandlestickPanel 前端 — tab 切换 + 分时图逻辑

**Files:**
- Modify: `frontend/src/terminal/panels/CandlestickPanel.vue`

**Interfaces:**
- Consumes: `App.GetMinuteLine("CN", symbol)`, `App.FetchOHLCV("CN", symbol, interval, start, end)`
- Produces: kline/minute tab switch, minute chart with 10s auto-refresh

- [ ] **Step 1: 添加 activeTab 状态和分时数据接口**

在 `<script setup>` 的 state 区域新增：

```ts
// Tab state
const activeTab = ref<'kline' | 'minute'>('kline')

// Minute chart data
interface MinuteTick {
  time: string     // "09:35"
  price: number    // 均价
  volume: number   // 成交量
  avg_price: number
}
const minuteTicks = ref<MinuteTick[]>([])
const prevClose = ref(0)
const minuteLoading = ref(false)
let minuteTimer: ReturnType<typeof setInterval> | null = null
```

- [ ] **Step 2: 实现分时图加载函数**

```ts
async function loadMinuteLine() {
  const app = (window as any).go?.main?.App
  if (!app) return
  minuteLoading.value = true
  try {
    const result = await app.GetMinuteLine('CN', symbol.value)
    const ticks = Array.isArray(result) ? result[0] : result
    if (!Array.isArray(ticks) || ticks.length === 0) {
      minuteTicks.value = []
      return
    }
    // Merge: keep existing ticks, update latest or append new ones
    const existing = new Map(minuteTicks.value.map(t => [t.time, t]))
    for (const t of ticks) {
      existing.set(t.time, t)
    }
    minuteTicks.value = Array.from(existing.values()).sort((a, b) => a.time.localeCompare(b.time))
    // Derive prevClose from first tick's open or GetQuote
    if (prevClose.value === 0 && minuteTicks.value.length > 0) {
      prevClose.value = minuteTicks.value[0].price
    }
  } catch {
    // silent
  } finally {
    minuteLoading.value = false
  }
}

function startMinutePolling() {
  stopMinutePolling()
  loadMinuteLine()
  minuteTimer = setInterval(loadMinuteLine, 10000)
}

function stopMinutePolling() {
  if (minuteTimer) { clearInterval(minuteTimer); minuteTimer = null }
}
```

- [ ] **Step 3: 分时图 ECharts 配置**

```ts
const minuteChartOption = computed(() => {
  if (!minuteTicks.value.length) return {}
  const times = minuteTicks.value.map(t => t.time)
  const prices = minuteTicks.value.map(t => t.price)
  const volumes = minuteTicks.value.map(t => t.volume)
  const isUp = prices.length > 0 && prices[prices.length - 1] >= prevClose.value
  const lineColor = isUp ? '#ef4444' : '#22c55e'

  return {
    backgroundColor: 'transparent',
    grid: { top: 20, right: 60, bottom: 40, left: 60 },
    xAxis: {
      type: 'category', data: times,
      axisLabel: { color: '#6b7280', fontSize: 10, interval: 30 },
      axisLine: { lineStyle: { color: '#374151' } },
    },
    yAxis: [
      {
        type: 'value', name: '价格',
        position: 'left',
        axisLabel: { color: '#6b7280', fontSize: 10 },
        splitLine: { lineStyle: { color: '#1f2937' } },
        min: (val: { min: number; max: number }) => Math.floor(val.min * 0.995 * 100) / 100,
        max: (val: { min: number; max: number }) => Math.ceil(val.max * 1.005 * 100) / 100,
      },
      {
        type: 'value', name: '量',
        position: 'right',
        axisLabel: { color: '#6b7280', fontSize: 10, formatter: (v: number) => v >= 1e4 ? (v/1e4).toFixed(1)+'万' : v },
        splitLine: { show: false },
      }
    ],
    series: [
      {
        type: 'line', data: prices, yAxisIndex: 0,
        smooth: false, symbol: 'none',
        lineStyle: { color: lineColor, width: 1.5 },
        areaStyle: {
          color: {
            type: 'linear', x: 0, y: 0, x2: 0, y2: 1,
            colorStops: [
              { offset: 0, color: lineColor === '#ef4444' ? 'rgba(239,68,68,0.3)' : 'rgba(34,197,94,0.3)' },
              { offset: 1, color: 'rgba(0,0,0,0)' }
            ]
          }
        },
      },
      {
        type: 'line', data: minuteTicks.value.map(t => t.avg_price), yAxisIndex: 0,
        smooth: true, symbol: 'none',
        lineStyle: { color: '#f59e0b', width: 1, type: 'dashed' },
        name: '均价',
      },
      {
        type: 'bar', data: volumes, yAxisIndex: 1,
        itemStyle: { color: '#374151' },
        barWidth: 1,
      },
    ],
    tooltip: { trigger: 'axis' },
    // 昨收参考线 (markLine)
    markLine: prevClose.value > 0 ? {
      silent: true, symbol: 'none',
      lineStyle: { color: '#6b7280', type: 'dashed', width: 1 },
      data: [{ yAxis: prevClose.value, label: { formatter: `昨收 ${prevClose.value.toFixed(2)}`, color: '#6b7280', fontSize: 10 } }],
    } : undefined,
  }
})
```

- [ ] **Step 4: Tab 切换联动轮询**

在 `watch` 或 `onMounted` 中添加：

```ts
// Watch tab switch for minute polling
watch(activeTab, (tab) => {
  if (tab === 'minute') {
    startMinutePolling()
  } else {
    stopMinutePolling()
  }
})

// Watch symbol change — reload
watch(() => symbol.value, () => {
  if (activeTab.value === 'minute') {
    minuteTicks.value = []
    prevClose.value = 0
    loadMinuteLine()
  }
})

// K-line auto-refresh for minute intervals
watch(interval, (iv) => {
  if (['1m','5m','15m','30m','1h'].includes(iv as string)) {
    if (activeTab.value === 'kline') {
      startKlineRefresh()
    }
  }
})

let klineTimer: ReturnType<typeof setInterval> | null = null

function startKlineRefresh() {
  if (klineTimer) clearInterval(klineTimer)
  klineTimer = setInterval(loadOHLCV, 30000)
}

function stopKlineRefresh() {
  if (klineTimer) { clearInterval(klineTimer); klineTimer = null }
}

onUnmounted(() => {
  stopMinutePolling()
  stopKlineRefresh()
})
```

- [ ] **Step 5: 模板添加 tab 切换**

在 panel-header 区域增加 tab 按钮：

```html
<div class="tab-btns">
  <button :class="{ active: activeTab === 'kline' }" class="tab-btn" @click="activeTab = 'kline'">K线</button>
  <button :class="{ active: activeTab === 'minute' }" class="tab-btn" @click="activeTab = 'minute'">分时</button>
</div>
```

分时 tab 时隐藏 interval 按钮，渲染 minuteChartOption：

```html
<VChart v-if="hasECharts && activeTab === 'kline'" :option="chartOption" autoresize class="candlestick-chart" />
<VChart v-else-if="hasECharts && activeTab === 'minute'" :option="minuteChartOption" autoresize class="minute-chart" />
<div v-if="activeTab === 'minute' && !minuteTicks.length" class="no-data">暂无分时数据</div>
```

- [ ] **Step 6: CSS 新增 tab 按钮样式**

```css
.tab-btns { display: flex; gap: 4px; margin-right: 12px; }
.tab-btn {
  padding: 3px 12px; border: 1px solid #374151; border-radius: 4px;
  background: #1f2937; color: #9ca3af; font-size: 12px; cursor: pointer;
}
.tab-btn.active { background: #374151; color: #e5e7eb; border-color: #534ab7; }
.no-data { color: #6b7280; padding: 40px; text-align: center; }
```

- [ ] **Step 7: 类型检查 + 构建验证**

```bash
cd frontend && npx vue-tsc --noEmit 2>&1 | grep "error TS" | grep -vc "PropertyPanel\|DockView.test\|AIChatPanel\|workflow.ts\|CorrelationPanel.test\|DrawingPanel"
# 期望: 0 new errors
npm run build 2>&1 | grep "built in"
```

- [ ] **Step 8: Commit**

```bash
git add frontend/src/terminal/panels/CandlestickPanel.vue
git commit -m "[Frontend] add minute-line tab to CandlestickPanel with 10s polling"
```

---

### Task 5: 全量构建 + 推送

- [ ] **Step 1: 构建**

```bash
cd /Volumes/shenzy/vibe_coding/QuantFlow/frontend && npm run build
cd /Volumes/shenzy/vibe_coding/QuantFlow && CGO_ENABLED=1 go build -ldflags="-s -w" -o quantflow .
ls -lh quantflow
```

- [ ] **Step 2: Commit + Push**

```bash
git add -A
git commit -m "[Release] build candlestick minute-line feature"
git push origin main
```
