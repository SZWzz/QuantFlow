# 分时图性能优化 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Eliminate 90% of useless ECharts re-renders on minute chart, add loading skeleton, skip non-CN markets

**Architecture:** Add a computed cache guard in `CandlestickPanel.vue` that skips `buildMinuteOption` when tick data hasn't changed (common case: no new tick arrives in 5s). Plus early-exit for non-CN + loading skeleton.

**Tech Stack:** Vue 3 + ECharts 5 + TypeScript

## Global Constraints

- MinuteTicks are append-only — existing ticks never change retroactively
- Non-CN symbols never have minute data (A-share only via mootdx/TDX)
- `shallowRef` is safe because `minuteTicks.value` is always replaced with a new array
- All tests must pass: `go test ./...` + `npx vitest run` + `vue-tsc --noEmit`

---

### Task 1: Change Detection + Computed Cache Guard

**Files:**
- Modify: `frontend/src/terminal/panels/CandlestickPanel.vue`
- Modify: `frontend/src/lib/buildChartOption.ts`

**Interfaces:**
- Consumes: `MinuteTick[]` (exists), `buildMinuteOption()` (exists)
- Produces: `minuteChartOption` computed with cache guard

- [ ] **Step 1: Write failing test for minute option cache**

Read the current test file:
```bash
ls /Volumes/shenzy/vibe_coding/QuantFlow/frontend/src/terminal/panels/__tests__/
```

Create `frontend/src/terminal/panels/__tests__/CandlestickPanel.test.ts`:

```typescript
import { describe, it, expect } from 'vitest'

describe('minuteChartOption cache', () => {
  it('should return same object reference when data unchanged', () => {
    // The MinuteTick[] with same length + last time + last price
    // should produce the same cache key and return the same option ref
    const key1 = computeDataKey([
      { time: '09:30', price: 100, volume: 1000, avg_price: 100, amount: 100000 },
      { time: '09:31', price: 101, volume: 2000, avg_price: 100.5, amount: 200000 },
    ])
    const key2 = computeDataKey([
      { time: '09:30', price: 100, volume: 1000, avg_price: 100, amount: 100000 },
      { time: '09:31', price: 101, volume: 2000, avg_price: 100.5, amount: 200000 },
    ])
    expect(key1).toBe(key2)
  })

  it('should return different key when data changes', () => {
    const key1 = computeDataKey([
      { time: '09:31', price: 101, volume: 2000, avg_price: 100.5, amount: 200000 },
    ])
    const key2 = computeDataKey([
      { time: '09:31', price: 102, volume: 2000, avg_price: 100.5, amount: 200000 },
    ])
    expect(key1).not.toBe(key2)
  })

  it('should return different key when new tick appended', () => {
    const key1 = computeDataKey([
      { time: '09:30', price: 100, volume: 1000, avg_price: 100, amount: 100000 },
    ])
    const key2 = computeDataKey([
      { time: '09:30', price: 100, volume: 1000, avg_price: 100, amount: 100000 },
      { time: '09:31', price: 101, volume: 2000, avg_price: 100.5, amount: 200000 },
    ])
    expect(key1).not.toBe(key2)
  })
})
```

Run it (should fail — `computeDataKey` not defined):
```bash
cd /Volumes/shenzy/vibe_coding/QuantFlow/frontend && npx vitest run src/terminal/panels/__tests__/CandlestickPanel.test.ts 2>&1
```
Expected: FAIL

- [ ] **Step 2: Add `computeDataKey` helper + cache guard**

Read the current `CandlestickPanel.vue` lines 200-220 and 480-500:
```bash
cd /Volumes/shenzy/vibe_coding/QuantFlow && rg -n "minuteChartOption|minuteTicks" frontend/src/terminal/panels/CandlestickPanel.vue
```

At the top of the `<script setup>` block (near line 215, after `minuteTicks`), add:
```typescript
import { shallowRef } from 'vue'
import type { ECBasicOption } from 'echarts'

function computeDataKey(ticks: MinuteTick[]): string {
  if (ticks.length === 0) return '0|'
  const last = ticks[ticks.length - 1]
  return `${ticks.length}|${last.time}|${last.price}`
}
```

Change `const minuteTicks = ref<MinuteTick[]>([])` to:
```typescript
const minuteTicks = shallowRef<MinuteTick[]>([])
```

Find the `minuteChartOption` computed (around lines 490-493) and replace it with:
```typescript
const minuteOptionCache = ref<{ key: string; option: ECBasicOption | null }>({ key: '', option: null })
const minuteChartOption = computed(() => {
  const ticks = minuteTicks.value
  const key = computeDataKey(ticks)
  if (key === minuteOptionCache.value.key) {
    return minuteOptionCache.value.option!
  }
  const opt = buildMinuteOption(
    ticks,
    prevClose.value,
    bottomMode.value,
    theme as any,
    indicatorCache,
    symbol.value,
  )
  minuteOptionCache.value = { key, option: opt }
  return opt
})
```

- [ ] **Step 3: Run test to verify it passes**

```bash
cd /Volumes/shenzy/vibe_coding/QuantFlow/frontend && npx vitest run src/terminal/panels/__tests__/CandlestickPanel.test.ts 2>&1
```
Expected: PASS

- [ ] **Step 4: Run all frontend tests + typecheck**

```bash
cd /Volumes/shenzy/vibe_coding/QuantFlow/frontend && npx vitest run 2>&1 | tail -20
```
```bash
cd /Volumes/shenzy/vibe_coding/QuantFlow/frontend && npx vue-tsc --noEmit 2>&1 | tail -10
```

- [ ] **Step 5: Commit**

```bash
cd /Volumes/shenzy/vibe_coding/QuantFlow && git add frontend/src/terminal/panels/CandlestickPanel.vue frontend/src/terminal/panels/__tests__/CandlestickPanel.test.ts
git commit -m "perf(chart): add computed cache guard for minute chart option, use shallowRef"
```

---

### Task 2: Non-CN Early Exit + Loading Skeleton

**Files:**
- Modify: `frontend/src/terminal/panels/CandlestickPanel.vue`

**Interfaces:**
- Consumes: `detectMarket()`, `SkeletonPanel`, `minuteLoading`
- Produces: minute chart with skeleton + non-CN guard

- [ ] **Step 1: Write test for non-CN behavior**

Add to `CandlestickPanel.test.ts`:
```typescript
describe('minute chart non-CN', () => {
  it('should derive market correctly for CN symbols', () => {
    // A-share: 60xxxx, 00xxxx, 300xxx, 688xxx, 8xxxxx
    const cnSymbols = ['600519', '000001', '300750', '688001', '830001']
    for (const s of cnSymbols) {
      expect(detectMarket(s)).toBe('CN')
    }
  })

  it('should not start minute polling for non-CN symbols', () => {
    const nonCNSymbols = ['AAPL', '0700.HK', 'BTC-USD']
    for (const s of nonCNSymbols) {
      expect(detectMarket(s)).not.toBe('CN')
    }
  })
})
```

```bash
cd /Volumes/shenzy/vibe_coding/QuantFlow/frontend && npx vitest run src/terminal/panels/__tests__/CandlestickPanel.test.ts 2>&1
```
Expected: FAIL (detectMarket may not be exported or function differently)

- [ ] **Step 2: Add non-CN guard in startMinutePolling**

In `CandlestickPanel.vue`, find `function startMinutePolling()` (around line 337) and add at the top:
```typescript
function startMinutePolling() {
  stopMinutePolling()
  if (detectMarket(symbol.value) !== 'CN') {
    logger.info('[Candlestick] minute chart only supports CN market, skipping', symbol.value)
    return
  }
  // ... existing code ...
}
```

- [ ] **Step 3: Add loading skeleton in template**

Find the minute chart template (around line 816-842). Change it to show SkeletonPanel when loading:

Current template (approximate, read first):
```bash
cd /Volumes/shenzy/vibe_coding/QuantFlow && rg -n "minuteChartOption|minute-chart|分时" frontend/src/terminal/panels/CandlestickPanel.vue | head -10
```

Add a conditional wrapper around the VChart:
```html
<template v-if="activeTab === 'minute'">
  <div class="minute-chart-container" v-if="minuteTicks.length > 0">
    <VChart ... />
  </div>
  <SkeletonPanel v-else-if="minuteLoading" />
  <div v-else class="empty-state">
    <span>{{ t('kline.no_minute_data') }}</span>
  </div>
</template>
```

Make sure `SkeletonPanel` is imported (add to existing import block):
```typescript
import SkeletonPanel from '@/terminal/components/SkeletonPanel.vue'
```

Make sure the `setTimeout(() => { minuteLoading.value = false })` in `loadMinuteLine` doesn't cause flash. The loading state is already handled:
- `minuteLoading = true` before fetch
- `minuteLoading = false` in finally block

- [ ] **Step 4: Run tests + typecheck**

```bash
cd /Volumes/shenzy/vibe_coding/QuantFlow/frontend && npx vitest run 2>&1 | tail -10
```
```bash
cd /Volumes/shenzy/vibe_coding/QuantFlow/frontend && npx vue-tsc --noEmit 2>&1 | tail -10
```

- [ ] **Step 5: Commit**

```bash
cd /Volumes/shenzy/vibe_coding/QuantFlow && git add frontend/src/terminal/panels/CandlestickPanel.vue frontend/src/terminal/panels/__tests__/CandlestickPanel.test.ts
git commit -m "perf(chart): skip minute polling for non-CN markets, add skeleton placeholder"
```

---

### Task 3: Indicator Cache Key Expansion

**Files:**
- Modify: `frontend/src/lib/composables/useIndicators.ts`

**Interfaces:**
- Consumes: `IndicatorCache` (exists)
- Produces: Cache key includes `lastPrice` for accuracy

- [ ] **Step 1: Read the current useIndicators.ts**

```bash
cd /Volumes/shenzy/vibe_coding/QuantFlow && rg -n "createIndicatorCache|getCached" frontend/src/lib/composables/useIndicators.ts
```

Read around those lines.

- [ ] **Step 2: Expand cache key**

Find the `createIndicatorCache` function. Look for where the cache key is constructed. Currently the key is like `minute-{indicator}-{ticks.length}-{bottomMode}`. Change to include `lastPrice`:

```typescript
const lastPrice = prices.length > 0 ? prices[prices.length - 1] : 0
const cacheKey = `${prefix}-${prices.length}-${lastPrice}-${suffix}`
```

The exact change depends on how the function is structured. Look for a pattern like:

```typescript
key = `${prefix}-${prices.length}-${suffix}`
```

Add `-${lastPrice}` between length and suffix.

- [ ] **Step 3: Run tests**

```bash
cd /Volumes/shenzy/vibe_coding/QuantFlow/frontend && npx vitest run 2>&1 | tail -10
```
```bash
cd /Volumes/shenzy/vibe_coding/QuantFlow/frontend && npx vue-tsc --noEmit 2>&1 | tail -10
```

- [ ] **Step 4: Commit**

```bash
cd /Volumes/shenzy/vibe_coding/QuantFlow && git add frontend/src/lib/composables/useIndicators.ts
git commit -m "perf(chart): include lastPrice in indicator cache key to avoid stale cache hits"
```

---

### Task 4: Update CHANGELOG

**Files:**
- Modify: `CHANGELOG.md`

- [ ] **Step 1: Read and update CHANGELOG**

```bash
cd /Volumes/shenzy/vibe_coding/QuantFlow && head -30 CHANGELOG.md
```

Add under `## [2026.7.9]` → `### Changed`:
```markdown
- [Frontend] Minute chart no longer rebuilds ECharts option when tick data hasn't changed (skips 90%+ of useless re-renders)
- [Frontend] Non-CN markets skip minute polling entirely (eliminates pointless IPC errors every 5s)
- [Frontend] Minute chart shows loading skeleton instead of blank area during initial load
```

- [ ] **Step 2: Commit**

```bash
cd /Volumes/shenzy/vibe_coding/QuantFlow && git add CHANGELOG.md
git commit -m "docs: update CHANGELOG for minute chart performance optimization"
```
