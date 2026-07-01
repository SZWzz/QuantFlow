# Depth Integration — Implementation Plan

## Task 1: Delete MarketDepthPanel and its references

**Files to delete:**
- `frontend/src/terminal/panels/MarketDepthPanel.vue`

**Files to edit:**
- `frontend/src/terminal/panels/registry.ts` — remove line 23: `register('market-depth', ...)`
- `frontend/src/lib/icons.ts` — remove line 193: `'market-depth': 'depth',`

**Files to delete:**
- `frontend/src/terminal/panels/__tests__/MarketDepthPanel.test.ts`

**Commands:**
```bash
rm frontend/src/terminal/panels/MarketDepthPanel.vue
rm frontend/src/terminal/panels/__tests__/MarketDepthPanel.test.ts
```

Edit `registry.ts`: remove line 23
Edit `icons.ts`: remove line 193

## Task 2: Add depth sidebar to CandlestickPanel.vue

### Script additions (after line 89 activeTab)

```typescript
// ── Depth sidebar (minute tab) ──
const showDepth = ref(false)
const depthData = ref<{ bids: {price:number;size:number}[]; asks: {price:number;size:number}[] } | null>(null)
const depthLoading = ref(false)
const depthSimulated = ref(false)
const depthPrice = ref(0)
const depthChange = ref(0)
const depthChangePct = ref(0)

const depthMaxSize = computed(() => {
  if (!depthData.value) return 1
  const all = [...depthData.value.bids, ...depthData.value.asks]
  return Math.max(...all.map(l => l.size), 1)
})

function formatSize(size: number): string {
  if (size >= 10000) return (size / 10000).toFixed(1) + '万'
  return size.toFixed(0)
}

function barWidth(size: number): string {
  return ((size / depthMaxSize.value) * 100).toFixed(0) + '%'
}

async function loadDepth() {
  const app = (window as any).go?.main?.App
  if (!app) return
  depthLoading.value = true
  try {
    const mkt = detectMarket(symbol.value)
    const quoteResult = await app.GetQuote(mkt, symbol.value)
    const snapshot = Array.isArray(quoteResult) ? quoteResult[0] : quoteResult
    if (snapshot) {
      depthPrice.value = snapshot.last || 0
      depthChange.value = snapshot.change || 0
      depthChangePct.value = snapshot.change_pct || snapshot.changePct || 0
    }
    const depthResult = await app.GetDepth(mkt, symbol.value).catch(() => null)
    if (depthResult && depthResult.bids?.length > 0) {
      depthData.value = {
        bids: depthResult.bids.map((l: any) => ({ price: l.price, size: l.size })),
        asks: depthResult.asks.map((l: any) => ({ price: l.price, size: l.size })),
      }
      depthSimulated.value = false
    } else if (snapshot?.bid > 0 && snapshot?.ask > 0) {
      const bids: {price:number;size:number}[] = []
      const asks: {price:number;size:number}[] = []
      const step = (snapshot.ask - snapshot.bid) / 5 || 0.02
      for (let i = 0; i < 5; i++) {
        bids.push({ price: +(snapshot.bid - i * step).toFixed(2), size: Math.round(1000 / (i + 1)) })
        asks.push({ price: +(snapshot.ask + i * step).toFixed(2), size: Math.round(800 / (i + 1)) })
      }
      depthData.value = { bids, asks }
      depthSimulated.value = true
    } else {
      depthData.value = null
    }
  } catch(e) {
    console.error('[Candlestick] depth:', e)
    depthData.value = null
  } finally {
    depthLoading.value = false
  }
}
```

### Toggle mechanism
- In the indicator bar (or chart header), add a toggle button next to sub-chart group:
  ```html
  <button class="depth-toggle" @click="toggleDepth" :class="{ active: showDepth }">📊 {{ $t('misc.depth') }}</button>
  ```
- `toggleDepth` toggles `showDepth` and calls `loadDepth()` if depthData is null

### Template changes for minute tab
Replace current minute chart block:
```html
<template v-else-if="activeTab === 'minute'">
  <div class="minute-layout">
    <div class="minute-chart-area" :class="{ 'with-depth': showDepth }">
      <KlineChart v-if="minuteTicks.length" ... />
      <div v-else class="chart-fallback no-data">{{ $t('kline.no_minute_data') }}</div>
    </div>
    <div v-if="showDepth" class="depth-sidebar">
      <!-- last-price row -->
      <div class="dp-last-price" :style="{ color: marketChangeColor }">
        <span class="dp-name">{{ name || symbol }}</span>
        <span class="dp-price">{{ depthPrice.toFixed(2) }}</span>
        <span class="dp-change">{{ depthChange >= 0 ? '+' : '' }}{{ depthChange.toFixed(2) }}</span>
      </div>
      <!-- order book header -->
      <div class="dp-ob-header">
        <span>买({{ $t('quote.bid') }})</span><span>{{ $t('common.size') }}</span><span></span>
        <span>卖({{ $t('quote.ask') }})</span><span>{{ $t('common.size') }}</span><span></span>
      </div>
      <!-- 5 levels -->
      <div v-for="i in 5" :key="i" class="dp-ob-row">
        <span class="dp-bid-price">{{ depthData?.bids[5-i]?.price.toFixed(2) ?? '' }}</span>
        <span class="dp-bid-size">{{ depthData?.bids[5-i] ? formatSize(depthData.bids[5-i].size) : '' }}</span>
        <span class="dp-bar-wrap"><span class="dp-bar bid" :style="{width: depthData?.bids[5-i] ? barWidth(depthData.bids[5-i].size) : '0%' }"></span></span>
        <span class="dp-ask-price">{{ depthData?.asks[i-1]?.price.toFixed(2) ?? '' }}</span>
        <span class="dp-ask-size">{{ depthData?.asks[i-1] ? formatSize(depthData.asks[i-1].size) : '' }}</span>
        <span class="dp-bar-wrap"><span class="dp-bar ask" :style="{width: depthData?.asks[i-1] ? barWidth(depthData.asks[i-1].size) : '0%' }"></span></span>
      </div>
      <div v-if="depthSimulated" class="dp-sim-badge">模拟</div>
    </div>
  </div>
</template>
```

### Style additions
- `.minute-layout`: `display: flex; flex: 1; min-height: 0`
- `.minute-chart-area`: `flex: 1; min-width: 0`
- `.depth-sidebar`: `width: 280px; margin-left: 8px; display: flex; flex-direction: column; gap: 4px`
- Copy order book styles from MarketDepthPanel
- `.depth-toggle` button style similar to `.indicator-btn`

## Task 3: Update CHANGELOG

Add entry:
```markdown
### Added
- [Frontend] Collapsible depth order book sidebar in CandlestickPanel minute chart view

### Removed
- [Frontend] Standalone MarketDepthPanel (functionality integrated into CandlestickPanel)
```

## Task 4: Commit
Commit with message:
```
feat: integrate depth order book into CandlestickPanel minute chart

- Remove standalone MarketDepthPanel, registry entry, icon, and test
- Add toggleable depth sidebar to minute tab showing 5-level bid/ask
- Real depth via GetDepth (CN/HK), simulated fallback from bid/ask spread
- Chart resizes via flex layout when sidebar opens/closes

Spec: docs/specs/2026-07-01-depth-integration.md
```
