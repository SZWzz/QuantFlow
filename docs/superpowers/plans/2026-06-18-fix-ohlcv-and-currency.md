# Implementation Plan: Fix OHLCV Index Order and Currency Symbol Wiring

## Task 1: Fix CandlestickPanel OHLCV index order

**File:** `frontend/src/terminal/panels/CandlestickPanel.vue:92`

Change:
```
data: ohlcvData.value.map((d) => [d[1], d[4], d[3], d[2]]),
```
To:
```
data: ohlcvData.value.map((d) => [d[1], d[2], d[3], d[4]]),
```

Rationale: `generateMockData` pushes `[date, open, close, low, high, volume]`, indices 0–5. ECharts needs `[open, close, low, high]` = `[d[1], d[2], d[3], d[4]]`.

**Test:** Run `npx vitest run` in frontend/ (no dedicated test for this, but existing tests should pass).

## Task 2: Fix PortfolioSummary currency symbol in positions table

**File:** `frontend/src/terminal/panels/PortfolioSummary.vue`

Change line 281 from:
```
<td class="num">{{ fmtMoney(pos.marketValue) }}</td>
```
To:
```
<td class="num">{{ fmtMoney(pos.marketValue, pos.symbol) }}</td>
```

Change line 283 from:
```
{{ pnlSign(pos.pnl) }}{{ fmtMoney(pos.pnl) }}
```
To:
```
{{ pnlSign(pos.pnl) }}{{ fmtMoney(pos.pnl, pos.symbol) }}
```

**Test:** Run `npx vitest run` in frontend/.

## Post-execution

- Update CHANGELOG.md with the fixes under the existing `[2026.6.18]` section
- Verify version date is current (2026.6.18)
