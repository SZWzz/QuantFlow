# Plan: Market Overview 指数 K 线图

**Spec**: `docs/specs/2026-07-02-market-overview-index-kline.md`

## Tasks

### Task 1: Go 后端 — GetMarketOverview 增加 OHLCV 并行获取

**Files to modify:**
- `app_market.go`

**Changes:**
1. In `app_market.go:396` (the goroutine for each index), after `FetchQuote` succeeds, add `FetchOHLCV` call
2. In result map (line 406-415), add `"ohlcv"` field
3. Add 30s TTL cache for index OHLCV data to avoid repeated calls

```go
// In the goroutine (around line 382-401):
go func(idx idxDef) {
    defer wg.Done()
    var snap *market.QuoteSnapshot
    var err error
    // ... existing FetchQuote logic ...

    // Also fetch daily OHLCV for mini K-line display
    var ohlcv []market.OHLCVBar
    ohlcvKey := "index_ohlcv:" + idx.code
    if cached, ok := overviewOHLCVCache.get(ohlcvKey); ok {
        ohlcv = cached
    } else {
        now := time.Now()
        start := now.AddDate(0, 0, -90).Unix() // fetch 90 days
        end := now.Unix()
        // Use empty fqfactor (raw prices, indices don't need adjustment)
        candidateMkt := marketName
        if candidateMkt == "CN" {
            // For HK/US indices, TryOHLCVWithFallback uses different adapters
        }
        bars, _, err2 := a.marketReg.FetchOHLCVWithFallback(ctx, marketName, idx.code, "1D", "", start, end)
        if err2 == nil && len(bars) > 0 {
            ohlcv = bars
            overviewOHLCVCache.set(ohlcvKey, bars) // cache for 60s
        }
    }

    ch <- idxResult{idx.code, snap, ohlcv}
}(idx)
```

Also add a new cache for OHLCV alongside the existing `overviewCache`:

```go
var overviewOHLCVCache = newOverviewCache(60 * time.Second)
```

Update `idxResult` struct to include OHLCV:

```go
type idxResult struct {
    code  string
    snap  *market.QuoteSnapshot
    ohlcv []market.OHLCVBar
}
```

Update result map building to include ohlcv:

```go
for r := range ch {
    // existing fields...
    ohlcvArr := make([]map[string]interface{}, 0, len(r.ohlcv))
    for _, b := range r.ohlcv {
        ohlcvArr = append(ohlcvArr, map[string]interface{}{
            "open":   b.Open,
            "high":   b.High,
            "low":    b.Low,
            "close":  b.Close,
        })
    }
    result = append(result, map[string]interface{}{
        "code":       r.code,
        "name":       getIndexName(r.code, indices),
        "price":      r.snap.Last,
        "change":     r.snap.Change,
        "change_pct": r.snap.ChangePct,
        "ohlcv":      ohlcvArr, // new field
    })
}
```

**Cache struct** (add to existing section):
```go
var overviewOHLCVCache = &overviewCacheType{
    data: make(map[string]cacheEntry),
    ttl:  60 * time.Second,
}
```

**Commit**: `[Engine] GetMarketOverview: add parallel OHLCV fetch for index K-line mini-charts`

---

### Task 2: Frontend 类型 — IndexSnapshot + fetchMarketOverview 透传 ohlcv

**Files to modify:**
- `frontend/src/stores/data.ts`

**Changes:**

1. `IndexSnapshot` interface 新增 `ohlcv` 字段:
```typescript
export interface IndexSnapshot {
  symbol: string
  name: string
  last: number
  changePct: number
  sparkline: number[]
  ohlcv?: OHLCVBar[]
}
```

2. `fetchMarketOverview` mapping (line 146-152) 透传:
```typescript
if (overviewResult) {
  if (overviewResult.indices) {
    indices = (overviewResult.indices as any[]).map((idx: any) => ({
      symbol: idx.code,
      name: idx.name,
      last: idx.price,
      changePct: idx.change_pct,
      sparkline: [],
      ohlcv: idx.ohlcv as OHLCVBar[] | undefined,
    }))
  }
}
```

**Commit**: `[Frontend] IndexSnapshot: add ohlcv field for mini K-line display`

---

### Task 3: PanelCard — SVG mini candlestick 渲染

**Files to modify:**
- `frontend/src/terminal/components/panel/PanelCard.vue`

**Changes:**

1. Add `ohlcv` prop:
```typescript
const props = withDefaults(defineProps<{
  title: string
  value?: number
  change?: number
  format?: 'price' | 'percent' | 'volume' | 'number'
  sparkline?: number[]
  ohlcv?: { open: number; high: number; low: number; close: number }[]
  clickable?: boolean
}>(), {
  format: 'price',
})
```

2. Add computed `candlePoints` to compute SVG positions:
```typescript
const candleCount = computed(() => Math.min(props.ohlcv?.length ?? 0, 30))

const candles = computed(() => {
  const d = props.ohlcv
  if (!d?.length) return []
  const data = d.slice(-30)
  let minLow = Infinity, maxHigh = -Infinity
  for (const c of data) {
    if (c.low < minLow) minLow = c.low
    if (c.high > maxHigh) maxHigh = c.high
  }
  const range = maxHigh - minLow || 1
  const candleW = 100 / data.length // width per candle
  const pad = candleW * 0.15          // padding per side
  const barW = candleW - pad * 2      // bar width (>0)
  return data.map((c, i) => {
    const x = i * candleW + pad
    const yHigh = 30 - ((c.high - minLow) / range) * 28 - 1
    const yLow = 30 - ((c.low - minLow) / range) * 28 - 1
    const yOpen = 30 - ((c.open - minLow) / range) * 28 - 1
    const yClose = 30 - ((c.close - minLow) / range) * 28 - 1
    const top = Math.min(yOpen, yClose)
    const bot = Math.max(yOpen, yClose)
    const isUp = c.close >= c.open
    return { x, yHigh, yLow, yOpen, yClose, top, bot, barW, isUp, wickW: Math.max(barW * 0.3, 0.5) }
  })
})
```

3. Replace SVG sparkline block in template:
```vue
<svg
  v-if="ohlcv?.length"
  class="sparkline"
  viewBox="0 0 100 30"
  preserveAspectRatio="none"
>
  <template v-for="(c, i) in candles" :key="i">
    <!-- wick shadow line -->
    <line
      :x1="c.x + c.barW / 2" :y1="c.yHigh"
      :x2="c.x + c.barW / 2" :y2="c.yLow"
      :stroke="c.isUp ? 'var(--color-up)' : 'var(--color-down)'"
      stroke-width="0.8"
    />
    <!-- body rect -->
    <rect
      :x="c.x" :y="c.top"
      :width="c.barW" :height="Math.max(c.bot - c.top, 1)"
      :fill="c.isUp ? 'var(--color-up)' : 'var(--color-down)'"
      :rx="0.5"
    />
  </template>
</svg>
<div v-else-if="sparkline?.length" class="sparkline"> 
  <!-- keep original polyline sparkline as fallback -->
</div>
```

**Commit**: `[Frontend] PanelCard: add mini candlestick SVG rendering for index K-line`

---

### Task 4: MarketOverviewPanel — 透传 ohlcv prop

**Files to modify:**
- `frontend/src/terminal/panels/MarketOverviewPanel.vue`

**Changes:**
In template line 164-174, pass `:ohlcv="idx.ohlcv"` to PanelCard:
```vue
<PanelCard
  v-for="idx in indices"
  :key="idx.symbol"
  :title="idx.name"
  :value="idx.last"
  :change="idx.changePct / 100"
  format="price"
  :sparkline="idx.sparkline"
  :ohlcv="idx.ohlcv"
  clickable
  @click="onIndexClick(idx)"
/>
```

**Commit**: `[Frontend] MarketOverviewPanel: pass ohlcv to PanelCard for mini K-line`

---

### Task 5: 构建验证

```bash
wails3 build
go vet ./...
```

Verify:
- CN tab shows mini candlesticks for all 5 indices
- Switch to HK/US tab, verify mini candlesticks appear
- No console errors about missing data

**Commit**: `[Frontend] verify index K-line build and display`

---

### Task 6: CHANGELOG + Version date

**Files to modify:**
- `CHANGELOG.md`
- `frontend/package.json`
- `README.md`

**Commit**: `chore: update CHANGELOG for index K-line feature`
