# Review Fixes Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix 6 bugs + 1 doc issue discovered in the 2026-07-06 code review

**Architecture:** 7 independent fixes across frontend (Vue/TS), Go backend, and internal logging. No cross-task dependencies.

**Tech Stack:** TypeScript (Crosshair, buildChartOption, useLogger), Go (ring_buffer, mootdx, eastmoney)

## Global Constraints

- Crosshair black box must show only T, Chg, Chg%, Vol — not O/H/L/C
- Volume must not be hardcoded to 0
- ECharts tooltip OHLC order must be O→H→L→C
- `RingBuffer(0)` and `RingBuffer(-1)` both produce 5000-capacity buffer
- `useLogger` must import `GetLogs` from `@/lib/wails`
- mootdx `Volume` must be multiplied by 100 (手→shares)
- EastMoney comment must explicitly note `close→high→low` non-standard order

---

### Task 1: Crosshair — Fix volume + Remove redundant OHLC from black box

**Files:**
- Modify: `frontend/src/lib/chart/Crosshair.ts`

**Interfaces:**
- Consumes: `CrosshairData` (unchanged, internal only)
- Produces: Updated canvas rendering with volume fixed and OHLC removed from floating box

- [ ] **Step 1: Read the file to confirm current state**

```bash
cat frontend/src/lib/chart/Crosshair.ts
```

- [ ] **Step 2: Fix volume extraction in `updateData()`**

Change line 71 from:
```typescript
volume: 0,
```
to:
```typescript
volume: Number(item[off + 4] ?? 0),
```

Candlestick raw data layout: `[dataIndex?, open, close, low, high, volume]` → volume is at offset `off + 4`.

- [ ] **Step 3: Remove O/H/L/C from floating box in `render()`**

Change lines 121-144 from:
```typescript
if (this.data) {
  const lines = [
    `T: ${this.data.time}`,
    `O: ${this.data.open.toFixed(2)}`,
    `H: ${this.data.high.toFixed(2)}`,
    `L: ${this.data.low.toFixed(2)}`,
    `C: ${this.data.close.toFixed(2)}`,
    `Chg: ${this.data.change >= 0 ? '+' : ''}${this.data.change.toFixed(2)}`,
    `${this.data.changePercent >= 0 ? '+' : ''}${this.data.changePercent.toFixed(2)}%`,
    `Vol: ${(this.data.volume / 10000).toFixed(0)}万`,
  ]
  const lineH = 16
  const boxW = 130
  const boxH = lines.length * lineH + 8
  ...
```

to:
```typescript
if (this.data) {
  const lines = [
    `T: ${this.data.time}`,
    `Chg: ${this.data.change >= 0 ? '+' : ''}${this.data.change.toFixed(2)}`,
    `${this.data.changePercent >= 0 ? '+' : ''}${this.data.changePercent.toFixed(2)}%`,
    `Vol: ${(this.data.volume / 10000).toFixed(0)}万`,
  ]
  const lineH = 16
  const boxW = 130
  const boxH = lines.length * lineH + 8
  ...
```

- [ ] **Step 4: Verify the file compiles**

```bash
cd frontend && npx vue-tsc --noEmit --noErrorTruncation 2>&1 | head -20
```

Expected: no errors (or only pre-existing errors unrelated to Crosshair.ts).

- [ ] **Step 5: Commit**

```bash
git add frontend/src/lib/chart/Crosshair.ts
git commit -m "fix: crosshair volume hardcoded to 0, remove redundant OHLC from black box"
```

---

### Task 2: Tooltip OHLC Order — O→H→L→C

**Files:**
- Modify: `frontend/src/lib/buildChartOption.ts`

**Interfaces:**
- Pure visual change inside `tooltip.formatter`

- [ ] **Step 1: Read and confirm current order at lines 258-261 and 269-272**

```bash
cat -n frontend/src/lib/buildChartOption.ts | sed -n '254,276p'
```

- [ ] **Step 2: Reorder OHLC from O→C→L→H to O→H→L→C**

Change lines 258-261 from:
```typescript
lines.push(`<div style="margin-top:4px">${t('kline.open')}: ${item.open.toFixed(2)}</div>`)
lines.push(`<div>${t('kline.close')}: ${item.close.toFixed(2)}</div>`)
lines.push(`<div>${t('kline.low')}: ${item.low.toFixed(2)}</div>`)
lines.push(`<div>${t('kline.high')}: ${item.high.toFixed(2)}</div>`)
```

to:
```typescript
lines.push(`<div style="margin-top:4px">${t('kline.open')}: ${item.open.toFixed(2)}</div>`)
lines.push(`<div>${t('kline.high')}: ${item.high.toFixed(2)}</div>`)
lines.push(`<div>${t('kline.low')}: ${item.low.toFixed(2)}</div>`)
lines.push(`<div>${t('kline.close')}: ${item.close.toFixed(2)}</div>`)
```

Change lines 269-272 from:
```typescript
lines.push(`<div style="margin-top:4px">${t('kline.open')}: ${d[off].toFixed(2)}</div>`)
lines.push(`<div>${t('kline.close')}: ${d[off + 1].toFixed(2)}</div>`)
lines.push(`<div>${t('kline.low')}: ${d[off + 2].toFixed(2)}</div>`)
lines.push(`<div>${t('kline.high')}: ${d[off + 3].toFixed(2)}</div>`)
```

to:
```typescript
lines.push(`<div style="margin-top:4px">${t('kline.open')}: ${d[off].toFixed(2)}</div>`)
lines.push(`<div>${t('kline.high')}: ${d[off + 3].toFixed(2)}</div>`)
lines.push(`<div>${t('kline.low')}: ${d[off + 2].toFixed(2)}</div>`)
lines.push(`<div>${t('kline.close')}: ${d[off + 1].toFixed(2)}</div>`)
```

- [ ] **Step 3: Verify the file compiles**

```bash
cd frontend && npx vue-tsc --noEmit --noErrorTruncation 2>&1 | head -20
```

Expected: no new errors.

- [ ] **Step 4: Commit**

```bash
git add frontend/src/lib/buildChartOption.ts
git commit -m "fix: tooltip OHLC order O->C->L->H -> O->H->L->C"
```

---

### Task 3: RingBuffer(0) — Remove no-op special case

**Files:**
- Modify: `internal/logging/ring_buffer.go`

- [ ] **Step 1: Read current NewRingBuffer**

```bash
cat -n internal/logging/ring_buffer.go | sed -n '25,41p'
```

- [ ] **Step 2: Change `capacity == 0` handling**

Change from:
```go
func NewRingBuffer(capacity int) *RingBuffer {
	if capacity < 0 {
		capacity = 5000
	}
	if capacity == 0 {
		return &RingBuffer{
			buffer: nil,
			max:    0,
			nextID: 1,
		}
	}
```

to:
```go
func NewRingBuffer(capacity int) *RingBuffer {
	if capacity <= 0 {
		capacity = 5000
	}
```

Also remove the `if rb.max == 0 { return }` guard in Push (line 46-48) since max will never be 0.

Change `Push` from:
```go
func (rb *RingBuffer) Push(entry LogEntry) {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	if rb.max == 0 {
		return
	}
```

to:
```go
func (rb *RingBuffer) Push(entry LogEntry) {
	rb.mu.Lock()
	defer rb.mu.Unlock()
```

- [ ] **Step 3: Run Go tests for the logging package**

```bash
cd app && go test ./internal/logging/... -v -count=1
```

Expected: PASS (adds or passes).

- [ ] **Step 4: Run Go vet**

```bash
cd app && go vet ./internal/logging/...
```

Expected: no errors.

- [ ] **Step 5: Commit**

```bash
git add internal/logging/ring_buffer.go
git commit -m "fix: RingBuffer(0) no-op diverged from spec, now defaults to 5000"
```

---

### Task 4: useLogger — Use typed GetLogs wrapper

**Files:**
- Modify: `frontend/src/lib/composables/useLogger.ts`

- [ ] **Step 1: Read current file**

```bash
cat -n frontend/src/lib/composables/useLogger.ts
```

- [ ] **Step 2: Import `GetLogs` from `@/lib/wails`**

Change line 1:
```typescript
import { ref, onMounted, onUnmounted } from 'vue'
import type { LogEntry } from '@/lib/wails'
```

to:
```typescript
import { ref, onMounted, onUnmounted } from 'vue'
import { GetLogs, type LogEntry } from '@/lib/wails'
```

- [ ] **Step 3: Replace raw window.go usage with typed call**

Change lines 17-36 from:
```typescript
async function poll() {
  try {
    const go = (window as any).go
    if (!go || !go.main || !go.main.App || !go.main.App.GetLogs) {
      error.value = 'Wails bridge not ready'
      return
    }
    const newEntries: LogEntry[] = await go.main.App.GetLogs(lastID.value)
    ...
```

to:
```typescript
async function poll() {
  try {
    const newEntries: LogEntry[] = await GetLogs(lastID.value)
    ...
```

- [ ] **Step 4: Verify compilation**

```bash
cd frontend && npx vue-tsc --noEmit --noErrorTruncation 2>&1 | head -20
```

Expected: no new errors.

- [ ] **Step 5: Commit**

```bash
git add frontend/src/lib/composables/useLogger.ts
git commit -m "refactor: useLogger uses typed GetLogs from @/lib/wails"
```

---

### Task 5: mootdx Volume Normalization (手→shares)

**Files:**
- Modify: `internal/market/adapters/mootdx.go`

- [ ] **Step 1: Read the mootdx adapter ohlcv bars section**

```bash
cat -n internal/market/adapters/mootdx.go | sed -n '140,155p'
```

- [ ] **Step 2: Add volume normalization**

Change line 150 from:
```go
			Volume: b.Volume,
```

to:
```go
			Volume: b.Volume * 100, // 手→shares, consistent with EastMoney adapter
```

- [ ] **Step 3: Run Go vet and tests**

```bash
cd app && go vet ./internal/market/adapters/... && go test ./internal/market/adapters/... -v -count=1 2>&1 | head -30
```

Expected: no errors.

- [ ] **Step 4: Commit**

```bash
git add internal/market/adapters/mootdx.go
git commit -m "fix: mootdx volume unit not normalized (手→shares), was 100x smaller than EastMoney"
```

---

### Task 6: EastMoney Comment Clarification

**Files:**
- Modify: `internal/market/adapters/eastmoney.go`

- [ ] **Step 1: Read current comment block**

```bash
cat -n internal/market/adapters/eastmoney.go | sed -n '179,186p'
```

- [ ] **Step 2: Update comment to note non-standard ordering**

Change lines 179-182 from:
```go
		// Each kline is comma-separated. With fields2=f51..f57, the expected order is:
		// [0]=date, [1]=open, [2]=close, [3]=high, [4]=low, [5]=volume, [6]=amount
		// EastMoney may return additional fields (amplitude, pct, change, turnover)
		// if more fields are requested; we only consume the first 7.
```

to:
```go
		// Each kline is comma-separated. With fields2=f51..f57, the expected order is:
		// [0]=date, [1]=open, [2]=close, [3]=high, [4]=low, [5]=volume, [6]=amount
		// NOTE: EastMoney's CSV order is NOT standard OHLCV (open, high, low, close).
		// It returns close BEFORE high/low. Do NOT reorder to match OHLCV convention!
		// EastMoney may return additional fields (amplitude, pct, change, turnover)
		// if more fields are requested; we only consume the first 7.
```

- [ ] **Step 3: Run Go vet**

```bash
cd app && go vet ./internal/market/adapters/...
```

Expected: no errors.

- [ ] **Step 4: Commit**

```bash
git add internal/market/adapters/eastmoney.go
git commit -m "docs: eastmoney comment clarifies non-standard field order"
```

---

### Task 7: Update CHANGELOG and Version Date

**Files:**
- Modify: `CHANGELOG.md`
- Modify: `frontend/package.json`
- Modify: `README.md`

- [ ] **Step 1: Read current CHANGELOG, package.json, README**

```bash
cat CHANGELOG.md | head -10
cat frontend/package.json | grep '"version"'
grep -n 'version\|Version' README.md | head -5
```

- [ ] **Step 2: Update CHANGELOG**

Add entry at the top:
```markdown
## [2026.7.7] - 2026-07-07

### Fixed
- [Frontend] Crosshair volume hardcoded to 0; restored correct extraction from candlestick raw data
- [Frontend] Crosshair black box no longer redundantly shows OHLC (duplicated ECharts tooltip); now shows only Chg/Chg%/Vol
- [Frontend] ECharts tooltip OHLC display order corrected to O→H→L→C (industry standard)
- [Frontend] useLogger now uses typed GetLogs wrapper from @/lib/wails instead of raw window.go
- [Storage] RingBuffer(0) no longer creates silent no-op buffer; aligns with spec (capacity≤0 → 5000)
- [MarketData] mootdx OHLCV volume normalized from 手 to shares (×100), consistent with EastMoney adapter
- [Docs] EastMoney adapter comment now explicitly notes non-standard field order to prevent future breakage
```

- [ ] **Step 3: Update version in package.json**

Set `"version"` to `"2026.7.7"`.

- [ ] **Step 4: Update README version badge if present**

Look for version badge and update to match.

- [ ] **Step 5: Commit**

```bash
git add CHANGELOG.md frontend/package.json README.md
git commit -m "chore: update CHANGELOG and version to 2026.7.7"
```
