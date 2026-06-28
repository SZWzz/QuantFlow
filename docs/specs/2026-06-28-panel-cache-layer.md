# Panel Data Cache Layer

## Motivation

GovDataPanel (宏观经济) and HeatmapPanel (板块热力图) fetch data on every mount with zero caching. Switching panels — or even switching sources within GovDataPanel — triggers a full reload, wasting bandwidth and creating unnecessary latency. Users see loading spinners every time they navigate, even for data that hasn't changed (e.g., BIS dataflows, historical indicator series).

Other panels (MarketOverview, Candlestick) have ad-hoc caching but no shared mechanism. A generic cache layer in the Pinia `dataStore` solves this once for all panels.

## Design

Add a generic TTL-based cache to `dataStore`. Each entry stores `{ data, timestamp, ttl }`. Exported methods handle get/set with expiry checks. Panels use these methods instead of raw refs.

### Cache Entry

```ts
interface CacheEntry<T = any> {
  data: T
  timestamp: number   // Date.now() when set
  ttl: number         // milliseconds
}
```

### API (added to `useDataStore`)

```ts
function setCached<T>(key: string, data: T, ttlMs: number): void
function getCached<T>(key: string): T | null       // returns null if expired or missing
function clearCached(key?: string): void              // omit key = clear all
```

### Cache Key Conventions

| Key pattern | TTL | Used by |
|---|---|---|
| `market:overview:{CN,HK,US}` | 30s | HeatmapPanel, MarketOverviewPanel |
| `gov:signals:{fred,cn,bis}` | 5min | GovDataPanel signal list |
| `gov:detail:{indicator_id}` | 10min | GovDataPanel indicator time series |

### Data Flow

```
Panel mounted
  → cacheKey = build from source/params
  → cached = dataStore.getCached(key)
  → if cached: render immediately
  → if !cached OR stale: fetch → dataStore.setCached(key, data, ttl) → render
```

For GovDataPanel source-switch: always re-fetch (user explicitly chose a different source), but cache the result so switching away and back is instant.

### Files Modified

| File | Change |
|---|---|
| `frontend/src/stores/data.ts` | Add `CacheEntry` type, cache Map, `setCached`/`getCached`/`clearCached` |
| `frontend/src/terminal/panels/HeatmapPanel.vue` | Use `getCached`/`setCached` with 30s TTL instead of raw `marketOverview` guard |
| `frontend/src/terminal/panels/GovDataPanel.vue` | Use `getCached`/`setCached` for signals (5min TTL) and detail (10min TTL) |

No new files. No Go/Python changes. No schema changes.

## Acceptance Criteria

- [ ] HeatmapPanel: first mount fetches; switching away and back within 30s uses cache (no fetch)
- [ ] HeatmapPanel: waiting 30s+ and switching back re-fetches (TTL expired)
- [ ] HeatmapPanel: clicking refresh always re-fetches and updates cache
- [ ] GovDataPanel: first mount fetches; switching away and back within 5min uses cache
- [ ] GovDataPanel: switching source re-fetches (different cache key)
- [ ] GovDataPanel: clicking refresh re-fetches that source
- [ ] GovDataPanel: indicator detail cached for 10min
- [ ] `clearCached()` without arg clears everything; with key clears single entry
- [ ] All existing functionality preserved (no regressions in MarketOverviewPanel etc.)

## Risks / Trade-offs

- **Memory**: Cache lives in Pinia ref (RAM). Macro signals are small (~50 items). Market overview is ~30 sectors + ~5 indices. Risk is negligible (< 100KB).
- **Stale data during trading hours**: 30s TTL for market data is acceptable — users can always click refresh. If we need shorter, we can make TTL configurable per call.
- **No persistence**: Cache is lost on page refresh. Acceptable — data is cheap to re-fetch on cold start. If we want persistence later, we can add a `sessionStorage` backend to the same API.
