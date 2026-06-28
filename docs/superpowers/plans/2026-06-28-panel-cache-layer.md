# Panel Cache Layer — Implementation Plan

**Spec**: `docs/specs/2026-06-28-panel-cache-layer.md`

---

## Task 1: Add cache layer to Pinia dataStore

**Files**: `frontend/src/stores/data.ts`

**Changes**:
1. Add `CacheEntry<T>` interface after existing interfaces
2. Add `ref<Map<string, CacheEntry>>('cache')` 
3. Add `setCached<T>(key, data, ttlMs)`, `getCached<T>(key)`, `clearCached(key?)` methods
4. Export them in the return block

**Code for the additions** (insert after line 55, before `export const useDataStore`):

```ts
interface CacheEntry<T = any> {
  data: T
  timestamp: number
  ttl: number
}

const cache = ref<Map<string, CacheEntry>>(new Map())

function setCached<T>(key: string, data: T, ttlMs: number): void {
  cache.value.set(key, { data, timestamp: Date.now(), ttl: ttlMs })
}

function getCached<T>(key: string): T | null {
  const entry = cache.value.get(key)
  if (!entry) return null
  if (Date.now() - entry.timestamp > entry.ttl) {
    cache.value.delete(key)
    return null
  }
  return entry.data as T
}

function clearCached(key?: string): void {
  if (key) cache.value.delete(key)
  else cache.value = new Map()
}
```

Add to return block: `cache, setCached, getCached, clearCached`

**Verify**: `npx vue-tsc --noEmit` passes.

---

## Task 2: Update HeatmapPanel to use TTL cache

**Files**: `frontend/src/terminal/panels/HeatmapPanel.vue`

**Changes**:
1. Replace `if (!dataStore.marketOverview) refresh()` with cache check
2. Replace `refresh()` to use `getCached`/`setCached`
3. On refresh button, call `dataStore.clearCached(cacheKey)` before fetching

**Code**: See diff in spec. Wrapped in `onMounted`:

```ts
const cacheKey = computed(() => `market:overview:${activeMarket.value}`)

onMounted(async () => {
  const cached = dataStore.getCached<MarketOverview>(cacheKey.value)
  if (cached) {
    // dataStore.marketOverview stays as-is from cache; no fetch needed
    // But we still need the ref for template — actually the template reads from dataStore.marketOverview
    // So we need to set it from cache:
    // No — the cache stores the raw data. MarketOverview type is used directly.
    // Actually the simplest approach: just use the cache to guard the fetch, same as before
    return
  }
  await refresh()
})

async function refresh() {
  dataStore.clearCached(cacheKey.value)
  // ...rest stays the same
}
```

Wait, I need to think more carefully. The template reads `dataStore.marketOverview?.sectors`. So if we cache the fetch result and skip the fetch, `dataStore.marketOverview` might be null or stale from another panel.

Better approach:
- `refresh()` writes to cache AND to `dataStore.marketOverview`
- `onMounted` checks cache first; if cache hit, writes cache to `dataStore.marketOverview` and doesn't fetch
- Refresh button clears cache and re-fetches

Let me revise:

```ts
const cacheKey = computed(() => `market:overview:${activeMarket.value}`)

onMounted(async () => {
  const cached = dataStore.getCached<MarketOverview>(cacheKey.value)
  if (cached) {
    dataStore.marketOverview = cached
    return
  }
  await refresh()
})

async function refresh() {
  loading.value = true
  try {
    await dataStore.fetchMarketOverview(activeMarket.value)
    dataStore.setCached(cacheKey.value, dataStore.marketOverview, 30_000)
  } finally {
    loading.value = false
  }
}
```

And the refresh button calls `dataStore.clearCached(cacheKey.value)` then `refresh()` — but actually `refresh()` sets the cache at the end, so clearing first isn't needed if we always overwrite.

Actually the simplest: `refresh()` always fetches and then sets cache. No need to clear first since `setCached` overwrites.

---

## Task 3: Update GovDataPanel to use TTL cache

**Files**: `frontend/src/terminal/panels/GovDataPanel.vue`

**Changes**:
1. Add cache key computation based on `activeSource`
2. In `onMounted`, check cache before calling `loadSignals()`
3. In `loadSignals()`, set cache after fetching
4. When switching source, clear that source's old cache entry before re-fetching
5. Cache indicator detail data as well

**Code**:

Add after imports:
```ts
import { useDataStore } from '@/stores/data'
const dataStore = useDataStore()
```

Cache key:
```ts
const signalsCacheKey = computed(() => `gov:signals:${activeSource.value}`)
```

Modify `onMounted`:
```ts
onMounted(async () => {
  const cached = dataStore.getCached<MacroSignal[]>(signalsCacheKey.value)
  if (cached) {
    signals.value = cached
    return
  }
  await loadSignals()
})
```

In `loadSignals()`, after successfully setting `signals.value`:
```ts
dataStore.setCached(signalsCacheKey.value, signals.value, 300_000)
```

For source switch (template `@click`), clear and refetch:
```
dataStore.clearCached(`gov:signals:${activeSource.value}`) before loadSignals
```

Actually, the template already does `activeSource = s.key; activeCategory = 'all'; loadSignals()`. Since `loadSignals` sets the cache at the end, and the cache key depends on `activeSource`, switching source will use a different cache key automatically. But we want to force re-fetch when user clicks a source tab — they explicitly chose a new source. So we should clear the old cache key and the new one.

Actually no — when user clicks "FRED" while on "CN", `loadSignals()` runs, which sets `gov:signals:fred`. Next time they click "FRED" again, the cache hit returns the previously fetched data. That's the desired behavior.

For the detail cache, in `loadIndicatorDetail`:
```ts
const detailCacheKey = `gov:detail:${signal.indicator_id}`
const cached = dataStore.getCached<IndicatorPoint[]>(detailCacheKey)
if (cached) {
  indicatorData.value = cached
  chartLoading.value = false
  return
}
// ... fetch logic ...
dataStore.setCached(detailCacheKey, indicatorData.value, 600_000)
```

---

## Task 4: Verify

Run:
```bash
cd frontend && npx vue-tsc --noEmit && npx vitest run
```
