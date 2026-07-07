# Review Fixes — Crosshair, Tooltip, RingBuffer, Adapters

## Motivation

Code review from 2026-07-06 found 6 issues across the frontend and backend:

1. **Crosshair volume hardcoded to 0** — volume extraction was dropped during ECharts 5.5+ offset refactor
2. **Canvas black box shows redundant OHLC** — duplicates ECharts tooltip content
3. **Tooltip OHLC order non-standard** — O→C→L→H instead of O→H→L→C
4. **RingBuffer(0) diverges from spec** — spec says `<=0 → default 5000`, code treats `==0` as no-op
5. **useLogger bypasses typed Wails wrapper** — uses raw `window.go` instead of imported `GetLogs`
6. **mootdx volume unit not normalized** — miss 100× conversion (手→shares) unlike EastMoney adapter
7. **EastMoney OHLC comment lacks API caveat** — order differs from standard OHLCV, future maintainers may "fix" it

## Design

### 1. Crosshair Canvas Overlay — Fix volume + Remove redundant OHLC

**`frontend/src/lib/chart/Crosshair.ts`**

- The floating black box (lines 121-145) currently renders 8 lines: T, O, H, L, C, Chg, Chg%, Vol
- ECharts tooltip already shows O/H/L/C → the canvas box duplicates this
- `CrosshairData` has zero external consumers (`onHover` callback never wired at call site)

Changes:
- **Remove** O, H, L, C from the floating box rendering
- **Keep** T, Chg, Chg%, Vol (4 lines — ECharts tooltip doesn't show Chg/Chg%/Vol in cursor-following form)
- **Fix** `volume: 0` → extract from raw data at correct offset
- The Y-axis price label (close) and X-axis time label remain unchanged
- `CrosshairData` interface kept unchanged (internal, harmless)

Data flow:
```
ECharts candlestick raw → updateData() → CrosshairData → render() → canvas
                              ↓
                         onHover?.(data)  [no-op, no consumer]
```

### 2. ECharts Tooltip OHLC Order

**`frontend/src/lib/buildChartOption.ts`** lines 258-261, 269-272

Change display order from O→C→L→H to O→H→L→C (industry standard, matches Bloomberg convention).

### 3. RingBuffer(0) Align with Spec

**`internal/logging/ring_buffer.go`** lines 25-41

Remove the `capacity == 0` → no-op special case. Align with spec: `capacity <= 0` defaults to 5000.

### 4. useLogger — Use Typed Wails Wrapper

**`frontend/src/lib/composables/useLogger.ts`** lines 19-24

Replace raw `(window as any).go.main.App.GetLogs` with imported `GetLogs` from `@/lib/wails`. The `setupWailsBridge()` already provides a Proxy that routes through `Call.ByName`, so behavior is identical — but using the typed import avoids breakage if `window.go` shim changes.

### 5. mootdx Volume Unit Normalization

**`internal/market/adapters/mootdx.go`** line 150

mootdx returns volume in 手 (lots), same as EastMoney's raw data. EastMoney adapter normalizes with `* 100` (line 207). mootdx adapter misses this — volume will be 100× smaller for the same security.

Change `Volume: b.Volume` → `Volume: b.Volume * 100`.

### 6. EastMoney Comment Clarification

**`internal/market/adapters/eastmoney.go`** lines 179-182

Add note that EastMoney's CSV format returns `date, open, close, high, low, volume, amount` (close before high), which deviates from standard OHLCV ordering. This is correct for this API and must not be "fixed" to standard order.

## Acceptance Criteria

- [ ] Crosshair black box shows only T, Chg, Chg%, Vol (4 lines), not O/H/L/C
- [ ] Crosshair volume correctly displays (not hardcoded 0)
- [ ] ECharts tooltip OHLC order is O→H→L→C
- [ ] `RingBuffer(-1)` and `RingBuffer(0)` both produce 5000-capacity buffer
- [ ] `RingBuffer` no longer has the `capacity == 0` → no-op special case
- [ ] `useLogger` uses imported `GetLogs` from `@/lib/wails`
- [ ] mootdx OHLCV volume is 100× larger (手→shares)
- [ ] EastMoney comment explicitly notes non-standard field order

## Risks / Trade-offs

- **CrosshairData field removal considered** — but since the interface has zero external consumers and changing it would not affect anything, keeping it unchanged minimizes diff. The `onHover` callback is dead code (never wired), but removing it is out of scope for this fix.
- **mootdx volume *100** could cause double-counting if mootdx later changes unit. The debug logging in `fetcher.py` (lines 177-179) will surface any unit changes.
