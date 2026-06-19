# Fix OHLCV Index Order and Currency Symbol Wiring

## Motivation

Two bugs found during code review of recent frontend changes:

1. **CandlestickPanel.vue** — ECharts candlestick series expects `[open, close, lowest, highest]` per item, but the data mapping passes `[d[1], d[4], d[3], d[2]]` which corresponds to `[open, high, low, close]`. Every candle body and wick renders incorrectly.

2. **PortfolioSummary.vue** — `fmtMoney` accepts an optional `symbol` parameter to determine the correct currency prefix (¥/HK$/USDT/$), but the positions table calls it without `pos.symbol`, so all rows display `$` regardless of market. A股 (000001.SZ) should show ¥, BTCUSDT should show USDT.

## Design

### Fix 1: OHLCV index order

`generateMockData` pushes `[date, open, close, low, high, volume]` (indices 0–5). ECharts candlestick `data` expects `[open, close, low, high]`.

Change line 92 from:
```
data: ohlcvData.value.map((d) => [d[1], d[4], d[3], d[2]]),
```
to:
```
data: ohlcvData.value.map((d) => [d[1], d[2], d[3], d[4]]),
```

### Fix 2: Currency symbol in positions table

`fmtMoney(n, symbol?)` already has the logic. The template calls on lines 281 and 283 omit the second argument:

- Line 281: `{{ fmtMoney(pos.marketValue) }}` → `{{ fmtMoney(pos.marketValue, pos.symbol) }}`
- Line 283: `{{ pnlSign(pos.pnl) }}{{ fmtMoney(pos.pnl) }}` → `{{ pnlSign(pos.pnl) }}{{ fmtMoney(pos.pnl, pos.symbol) }}`

## Acceptance Criteria

- [ ] CandlestickPanel K线图正确渲染（open→open, close→close, low→low, high→high）
- [ ] PortfolioSummary 持仓表中 A 股显示 ¥、BTCUSDT 显示 USDT
- [ ] 所有现有测试通过

## Risks / Trade-offs

Minimal. These are two-line fixes with no behavioral side effects.
