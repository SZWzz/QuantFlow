# Panel Health Review Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix 15 TypeScript compilation errors across 9 panels, 2 Go build/test failures, and the i18n test infrastructure issue so `npx vue-tsc --noEmit`, `go test ./...`, and `go build ./...` all pass cleanly.

**Architecture:** Two parallel tracks — Go backend fixes (compile + test), Vue frontend fixes (type errors + test infra). Each fix is a single-file change following existing code patterns. No new files needed.

**Tech Stack:** Go 1.22+, Vue 3 Composition API + `<script setup lang="ts">`, vue-i18n, marked v18+

## Global Constraints

- Do NOT add comments (per QuantFlow code style rule in CLAUDE.md)
- All panel code follows `<script setup lang="ts">` pattern
- Use `(window as any).go?.main?.App` optional chaining for Go bridge access
- Go `slog` for logging, no panic in library code
- Every commit must include CHANGELOG update + version date check per CLAUDE.md rules 2 + 3

---

### Task 1: Fix eastmoney_signals.go — %w → %v

**Files:**
- Modify: `internal/market/adapters/eastmoney_signals.go:282,302`

**Interfaces:**
- Consumes: `market.NewTransientErrorf` (returns `error`, does NOT support `%w` directive)
- Produces: Two lines changed from `%w` to `%v`

- [ ] **Step 1: Change %w to %v at line 282**

```go
// line 282 — before:
return nil, market.NewTransientErrorf("eastmoney_signals industry: %w", err)
// line 282 — after:
return nil, market.NewTransientErrorf("eastmoney_signals industry: %v", err)
```

- [ ] **Step 2: Change %w to %v at line 302**

```go
// line 302 — before:
return nil, market.NewTransientErrorf("eastmoney_signals industry: %w", err)
// line 302 — after:
return nil, market.NewTransientErrorf("eastmoney_signals industry: %v", err)
```

- [ ] **Step 3: Verify Go build**

Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow && go build ./internal/market/adapters`
Expected: No output (success)

- [ ] **Step 4: Commit**

```bash
cd /Volumes/shenzy/vibe_coding/QuantFlow
git add internal/market/adapters/eastmoney_signals.go
git commit -m "fix: eastmoney_signals %w directive on non-wrapping NewTransientErrorf"
```

---

### Task 2: Fix app_test.go — nil cfg panic

**Files:**
- Modify: `app_test.go:17-21`

**Interfaces:**
- Consumes: `config.DefaultConfig()` — returns `*config.Config` with empty `APIKeys` map
- Produces: `App{marketReg, bridge: nil, cfg: config.DefaultConfig()}` — no nil deref in `registerMarketAdapters()`

- [ ] **Step 1: Add `cfg` field to App literal in TestApp_RegisterMarketAdapters_AllWired**

```go
// before (lines 17-21):
	a := &App{
		marketReg: market.NewAdapterRegistry(),
		bridge:    nil, // no Python sidecar → mootdx degrades
	}
// after:
	a := &App{
		cfg:       config.DefaultConfig(),
		marketReg: market.NewAdapterRegistry(),
		bridge:    nil, // no Python sidecar → mootdx degrades
	}
```

Need to add `"quantflow/internal/config"` import if not already present (check imports in app_test.go — since `config` is not used anywhere else in the test file, it needs adding).

- [ ] **Step 2: Add config import if missing**

```go
// app_test.go — add to import block:
	"quantflow/internal/config"
```

- [ ] **Step 3: Verify Go build and test**

Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow && go build ./... && go test -run TestApp_RegisterMarketAdapters_AllWired ./...`
Expected: Build OK, test passes (may show API key warnings but no panic)

- [ ] **Step 4: Commit**

```bash
cd /Volumes/shenzy/vibe_coding/QuantFlow
git add app_test.go
git commit -m "fix: app_test inject cfg to prevent nil deref in registerMarketAdapters"
```

---

### Task 3: Fix AIChatPanel.vue — marked renderer.code signature

**Files:**
- Modify: `frontend/src/terminal/panels/AIChatPanel.vue:38-44`

**Interfaces:**
- Consumes: `marked.Renderer` class (v18+), `hljs` from `highlight.js`
- Produces: Renderer.code with single-object parameter matching marked v18 API

- [ ] **Step 1: Update renderer.code function signature and body**

```ts
// before (lines 38-44):
renderer.code = function (code: string, language: string | undefined) {
  const lang = language || ''
  if (lang && hljs.getLanguage(lang)) {
    return '<pre><code class="hljs ' + lang + '">' + hljs.highlight(code, { language: lang }).value + '</code></pre>'
  }
  return '<pre><code class="hljs">' + hljs.highlightAuto(code).value + '</code></pre>'
}

// after:
renderer.code = function ({ text, lang }: { text: string; lang?: string }) {
  const language = lang || ''
  if (language && hljs.getLanguage(language)) {
    return '<pre><code class="hljs ' + language + '">' + hljs.highlight(text, { language: language }).value + '</code></pre>'
  }
  return '<pre><code class="hljs">' + hljs.highlightAuto(text).value + '</code></pre>'
}
```

- [ ] **Step 2: Verify TypeScript**

Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow/frontend && npx vue-tsc --noEmit 2>&1 | grep AIChatPanel`
Expected: No output (error eliminated)

- [ ] **Step 3: Commit**

```bash
cd /Volumes/shenzy/vibe_coding/QuantFlow
git add frontend/src/terminal/panels/AIChatPanel.vue
git commit -m "fix: AIChatPanel marked Renderer.code signature for v18+ API"
```

---

### Task 4: Fix CandlestickPanel.vue — d[0] → d.date in jumpToDate

**Files:**
- Modify: `frontend/src/terminal/panels/CandlestickPanel.vue:561`

**Interfaces:**
- Consumes: `ohlcvData` — `ref<KlineDataItem[]>` where `KlineDataItem = { date: string; open: number; high: number; low: number; close: number; volume: number }`
- Produces: `timestamps` as `string[]` of date strings

- [ ] **Step 1: Fix array index access**

```ts
// before (line 561):
  const timestamps = ohlcvData.value.map(d => d[0])
// after:
  const timestamps = ohlcvData.value.map(d => d.date)
```

This changes `timestamps` from `any[]` to `string[]`, which works with all subsequent usage (`timestamps[i]` comparison with `target / 1000` in `Math.abs` — `string - number` is `NaN` but works with `> <` comparison).

Note: The `timestamps[i]` values are date strings like `"2026-01-15"` and `target / 1000` is a Unix timestamp in seconds. The `Math.abs` comparison still works because `string - number` coerces to `NaN` but the `>` comparison uses lexicographic ordering... Wait, actually `Math.abs(timestamps[i] - target / 1000)` where timestamps[i] is a string like "2026-01-15" — this would give `NaN`. This was already broken before (with `d[0]` returning `any`, the same coercion happened). The original intent was probably to store timestamps as numbers.

Let me fix this properly — extract numeric timestamps by parsing the date string:

```ts
  const timestamps = ohlcvData.value.map(d => new Date(d.date).getTime() / 1000)
```

This produces `number[]` and makes the `jumpToDate()` logic work correctly.

- [ ] **Step 2: Verify TypeScript**

Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow/frontend && npx vue-tsc --noEmit 2>&1 | grep CandlestickPanel`
Expected: No output (error eliminated)

- [ ] **Step 3: Commit**

```bash
cd /Volumes/shenzy/vibe_coding/QuantFlow
git add frontend/src/terminal/panels/CandlestickPanel.vue
git commit -m "fix: CandlestickPanel jumpToDate parse date strings to timestamps"
```

---

### Task 5: Fix CorrelationPanel.vue — corrMatrix type from fetchWithCache

**Files:**
- Modify: `frontend/src/terminal/panels/CorrelationPanel.vue:70-74`

**Interfaces:**
- Consumes: `fetchWithCache<T>(key, fn)` — returns `{ data: T }`
- Produces: `corrMatrix` typed as `Record<string, Record<string, number>>`

- [ ] **Step 1: Add type parameter to fetchWithCache**

```ts
// before (lines 69-74):
    const key = 'correlation:' + syms.join(',') + ':' + lookback.value
    const { data: corrMatrix } = await fetchWithCache(key, () => app.GetCorrelationMatrix(syms, lookback.value))
    // Convert map[string]map[string]float64 to 2D array ordered by syms
    const m: number[][] = syms.map(si =>
      syms.map(sj => corrMatrix?.[si]?.[sj] ?? 0)
    )

// after:
    const key = 'correlation:' + syms.join(',') + ':' + lookback.value
    type CorrMap = Record<string, Record<string, number>>
    const { data: corrMatrix } = await fetchWithCache<CorrMap>(key, () => app.GetCorrelationMatrix(syms, lookback.value))
    // Convert map[string]map[string]float64 to 2D array ordered by syms
    const m: number[][] = syms.map(si =>
      syms.map(sj => corrMatrix?.[si]?.[sj] ?? 0)
    )
```

- [ ] **Step 2: Verify TypeScript**

Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow/frontend && npx vue-tsc --noEmit 2>&1 | grep CorrelationPanel`
Expected: No output (error eliminated)

- [ ] **Step 3: Commit**

```bash
cd /Volumes/shenzy/vibe_coding/QuantFlow
git add frontend/src/terminal/panels/CorrelationPanel.vue
git commit -m "fix: CorrelationPanel corrMatrix type via fetchWithCache generic param"
```

---

### Task 6: Fix DefiTVLPanel.vue — sort bVal type narrowing

**Files:**
- Modify: `frontend/src/terminal/panels/DefiTVLPanel.vue:32-34`

**Interfaces:**
- Consumes: `Protocol` interface with `string | number` fields
- Produces: sort comparator that handles both string and number fields correctly

- [ ] **Step 1: Narrow bVal in sort comparator**

```ts
// before (lines 31-35):
  arr.sort((a, b) => {
    const aVal = a[sortKey.value as keyof Protocol] ?? 0
    const bVal = b[sortKey.value as keyof Protocol] ?? 0
    return (typeof aVal === 'number' ? aVal - bVal : String(aVal).localeCompare(String(bVal))) * sortDir.value
  })

// after:
  arr.sort((a, b) => {
    const aVal = a[sortKey.value as keyof Protocol] ?? 0
    const bVal = b[sortKey.value as keyof Protocol] ?? 0
    if (typeof aVal === 'number' && typeof bVal === 'number') {
      return (aVal - bVal) * sortDir.value
    }
    return String(aVal).localeCompare(String(bVal)) * sortDir.value
  })
```

- [ ] **Step 2: Verify TypeScript**

Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow/frontend && npx vue-tsc --noEmit 2>&1 | grep DefiTVLPanel`
Expected: No output (error eliminated)

- [ ] **Step 3: Commit**

```bash
cd /Volumes/shenzy/vibe_coding/QuantFlow
git add frontend/src/terminal/panels/DefiTVLPanel.vue
git commit -m "fix: DefiTVLPanel sort comparator bVal type narrowing"
```

---

### Task 7: Fix WhaleTrackingPanel.vue — sort bVal type narrowing

**Files:**
- Modify: `frontend/src/terminal/panels/WhaleTrackingPanel.vue:32-34`

**Interfaces:**
- Consumes: `WhaleTx` interface with `string | number` fields
- Produces: sort comparator handling both string and number fields

Same fix pattern as Task 6.

- [ ] **Step 1: Narrow bVal in sort comparator**

```ts
// before (lines 31-35):
  arr.sort((a, b) => {
    const aVal = a[sortKey.value as keyof WhaleTx] ?? 0
    const bVal = b[sortKey.value as keyof WhaleTx] ?? 0
    return (typeof aVal === 'number' ? aVal - bVal : String(aVal).localeCompare(String(bVal))) * sortDir.value
  })

// after:
  arr.sort((a, b) => {
    const aVal = a[sortKey.value as keyof WhaleTx] ?? 0
    const bVal = b[sortKey.value as keyof WhaleTx] ?? 0
    if (typeof aVal === 'number' && typeof bVal === 'number') {
      return (aVal - bVal) * sortDir.value
    }
    return String(aVal).localeCompare(String(bVal)) * sortDir.value
  })
```

- [ ] **Step 2: Verify TypeScript**

Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow/frontend && npx vue-tsc --noEmit 2>&1 | grep WhaleTrackingPanel`
Expected: No output (error eliminated)

- [ ] **Step 3: Commit**

```bash
cd /Volumes/shenzy/vibe_coding/QuantFlow
git add frontend/src/terminal/panels/WhaleTrackingPanel.vue
git commit -m "fix: WhaleTrackingPanel sort comparator bVal type narrowing"
```

---

### Task 8: Fix DistributionPanel.vue — missing useI18n() destructure

**Files:**
- Modify: `frontend/src/terminal/panels/DistributionPanel.vue`

**Interfaces:**
- Consumes: `vue-i18n`'s `useI18n()` composable
- Produces: `t` function available in `<script setup>` scope for i18n translations

- [ ] **Step 1: Add `const { t } = useI18n()` after existing imports**

The file already imports `useI18n` at line 3. Add the destructure call after the `useSymbolContext()` line (around line 39, before `symbol` ref):

```ts
// after line 39: const ctx = useSymbolContext()
const { t } = useI18n()
```

- [ ] **Step 2: Verify TypeScript**

Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow/frontend && npx vue-tsc --noEmit 2>&1 | grep DistributionPanel`
Expected: No output (all 8 `t` errors eliminated)

- [ ] **Step 3: Commit**

```bash
cd /Volumes/shenzy/vibe_coding/QuantFlow
git add frontend/src/terminal/panels/DistributionPanel.vue
git commit -m "fix: DistributionPanel add missing useI18n() destructure"
```

---

### Task 9: Fix IndicatorPanel.vue — align type mismatch

**Files:**
- Modify: `frontend/src/terminal/panels/IndicatorPanel.vue:139`

**Interfaces:**
- Consumes: Column type from `PanelTable` (requires `align: 'left'`)
- Produces: Column definitions with compatible align value

- [ ] **Step 1: Change align value**

```ts
// before (line 139):
      align: 'right' as const,
// after:
      align: 'left' as const,
```

- [ ] **Step 2: Verify TypeScript**

Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow/frontend && npx vue-tsc --noEmit 2>&1 | grep IndicatorPanel`
Expected: No output (error eliminated)

- [ ] **Step 3: Commit**

```bash
cd /Volumes/shenzy/vibe_coding/QuantFlow
git add frontend/src/terminal/panels/IndicatorPanel.vue
git commit -m "fix: IndicatorPanel column align right→left (PanelTable type constraint)"
```

---

### Task 10: Fix LimitUpDownPanel.vue — switchMarket signature

**Files:**
- Modify: `frontend/src/terminal/panels/LimitUpDownPanel.vue:75`

**Interfaces:**
- Consumes: `PanelHeader` `@tab-change` emits `(key: string)`
- Produces: `switchMarket` accepts `string` with runtime validation

- [ ] **Step 1: Change switchMarket parameter type**

```ts
// before (line 75):
function switchMarket(mkt: 'SH' | 'SZ') {
// after:
function switchMarket(mkt: string) {
  if (mkt !== 'SH' && mkt !== 'SZ') return
```

- [ ] **Step 2: Verify TypeScript**

Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow/frontend && npx vue-tsc --noEmit 2>&1 | grep LimitUpDownPanel`
Expected: No output (error eliminated)

- [ ] **Step 3: Commit**

```bash
cd /Volumes/shenzy/vibe_coding/QuantFlow
git add frontend/src/terminal/panels/LimitUpDownPanel.vue
git commit -m "fix: LimitUpDownPanel switchMarket accept string with runtime guard"
```

---

### Task 11: Fix MarketOverviewPanel.vue — switchMarket signature

**Files:**
- Modify: `frontend/src/terminal/panels/MarketOverviewPanel.vue:81`

**Interfaces:**
- Consumes: `PanelHeader` `@tab-change` emits `(key: string)`
- Produces: `switchMarket` accepts `string` with runtime validation

- [ ] **Step 1: Change switchMarket parameter type**

```ts
// before (line 81):
function switchMarket(mkt: typeof activeMarket.value) {
// after:
function switchMarket(mkt: string) {
  if (mkt !== 'CN' && mkt !== 'HK' && mkt !== 'US') return
```

- [ ] **Step 2: Verify TypeScript**

Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow/frontend && npx vue-tsc --noEmit 2>&1 | grep MarketOverviewPanel`
Expected: No output (error eliminated)

- [ ] **Step 3: Commit**

```bash
cd /Volumes/shenzy/vibe_coding/QuantFlow
git add frontend/src/terminal/panels/MarketOverviewPanel.vue
git commit -m "fix: MarketOverviewPanel switchMarket accept string with runtime guard"
```

---

### Task 12: Fix test infrastructure — register vue-i18n in test wrappers

**Files:**
- Modify: `frontend/src/__tests__/setup.ts` (create if not exists, or modify existing)

Check if `setup.ts` exists first:
```bash
ls frontend/src/__tests__/setup.ts 2>/dev/null && echo EXISTS || echo NOT_FOUND
```

**Interfaces:**
- Consumes: `vue-i18n` instance from `@/lib/i18n`
- Produces: i18n registered globally via `config.globalProperties` or `app.use()`

- [ ] **Step 1: Check if setup.ts exists and configure i18n**

If `setup.ts` exists but doesn't configure i18n, add:
```ts
// frontend/src/__tests__/setup.ts
import { createI18n } from 'vue-i18n'
import { config } from '@vue/test-utils'

const i18n = createI18n({
  locale: 'zh',
  fallbackLocale: 'zh',
  messages: { zh: {}, en: {} },
  legacy: false,
})

config.global.plugins = [i18n]
```

If `setup.ts` doesn't exist, create it:
```ts
import { createI18n } from 'vue-i18n'
import { config } from '@vue/test-utils'

const i18n = createI18n({
  locale: 'zh',
  fallbackLocale: 'zh',
  messages: { zh: {}, en: {} },
  legacy: false,
})

config.global.plugins = [i18n]
```

- [ ] **Step 2: Check if vitest.config.ts references setup.ts**

Verify:
```bash
grep -n "setup" frontend/vitest.config.ts 2>/dev/null || grep -n "setup" frontend/vite.config.ts 2>/dev/null
```

If no setup file reference exists, add to `vitest.config.ts`:
```ts
// inside defineConfig({ test: { ... } })
setupFiles: ['./src/__tests__/setup.ts'],
```

- [ ] **Step 3: Run tests to verify i18n fix**

Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow/frontend && npx vitest run 2>&1 | tail -5`
Expected: 47 failed tests reduce significantly (many i18n-related failures resolved)

- [ ] **Step 4: Commit**

```bash
cd /Volumes/shenzy/vibe_coding/QuantFlow
git add frontend/src/__tests__/setup.ts frontend/vitest.config.ts
git commit -m "fix: register vue-i18n in test setup for component mount wrappers"
```

---

### Task 13: Update CHANGELOG and version date (if stale)

- [ ] **Step 1: Check today's date**

Run: `date +%Y.%-m.%-d`
Expected: `2026.7.7`

- [ ] **Step 2: Verify version in `frontend/package.json`**

```bash
grep '"version"' frontend/package.json
```

If not `2026.7.7`, update.

- [ ] **Step 3: Update CHANGELOG.md**

```markdown
## [2026.7.7] - 2026-07-07

### Fixed
- [Terminal] AIChatPanel marked renderer.code signature for marked v18+ API
- [Terminal] CandlestickPanel jumpToDate parse date strings to Unix timestamps
- [Terminal] CorrelationPanel corrMatrix type via fetchWithCache generic param
- [Terminal] DefiTVLPanel/WhaleTrackingPanel sort comparator bVal type narrowing
- [Terminal] DistributionPanel add missing useI18n() destructure
- [Terminal] IndicatorPanel column align right→left (PanelTable type constraint)
- [Terminal] LimitUpDownPanel/MarketOverviewPanel switchMarket accept string with runtime guard
- [Engine] eastmoney_signals %w directive on non-wrapping NewTransientErrorf
- [Engine] app_test inject cfg to prevent nil deref in registerMarketAdapters
- [Frontend] Register vue-i18n in test setup for component mount wrappers
```

- [ ] **Step 4: Verify README badge version**

```bash
grep "version" README.md | head -3
```

Update if stale.

- [ ] **Step 5: Commit**

```bash
cd /Volumes/shenzy/vibe_coding/QuantFlow
git add CHANGELOG.md frontend/package.json README.md
git commit -m "chore: update CHANGELOG and version to 2026.7.7"
```
