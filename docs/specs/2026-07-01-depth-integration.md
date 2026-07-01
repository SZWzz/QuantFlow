# Depth Order Book Integration into CandlestickPanel

## Motivation
Standalone `MarketDepthPanel` is redundant — depth data (5-level order book) is most useful alongside the minute chart (分时图). Users want a single cohesive view: price chart + depth sidebar, toggleable to maximize chart area when depth is not needed.

## Design

### Data Flow
```
Go: GetDepth(mkt, symbol) → DepthSnapshot{Bids, Asks} 
  └─ Tencent adapter (CN/HK), error (US/CRYPTO)

Frontend: CandlestickPanel
  ├─ minute tab: KlineChart (ECharts)
  └─ [toggle] depth sidebar
       ├─ Last price row (name, price, change%)
       ├─ 5-level bid/ask order book with visual bars
       └─ Simulated fallback when real depth unavailable
```

### Changes

1. **Delete** `MarketDepthPanel.vue` — no longer needed
2. **Delete** `__tests__/MarketDepthPanel.test.ts` — removed with panel
3. **Edit** `registry.ts` — remove `market-depth` registration
4. **Edit** `icons.ts` — remove `market-depth: 'depth'` mapping
5. **Edit** `CandlestickPanel.vue` — add depth sidebar to minute tab:
   - New script: `showDepth` ref, `depthData` state, `loadDepth()` calling `GetDepth()`, `maxSize` computed, `formatSize`/`barWidth` helpers
   - Only on `activeTab === 'minute'` — toggle button in indicator bar or chart-header
   - Right sidebar (280px): last-price row + 5-level order book with visual bars
   - `chart-body` flex layout: left chart flex:1, right sidebar when shown

### Removed
- `market-depth` from panel registry and icon map

## Acceptance Criteria
- [ ] `MarketDepthPanel.vue` and its test file deleted
- [ ] No references to `market-depth` in registry or icons
- [ ] Minute tab has a toggle button to show/hide depth sidebar
- [ ] Depth sidebar shows last price, 5 bids and 5 asks with size bars
- [ ] Real depth when `GetDepth` returns data; simulated fallback from bid/ask quote
- [ ] Chart resizes correctly when sidebar opens/closes (flex layout)
- [ ] Depth data refreshes on symbol change

## Risks / Trade-offs
- US/CRYPTO markets have no real depth — will show simulated from bid/ask spread (acceptable)
- Removing a panel might break existing user layouts that reference `market-depth` id (only in-memory panels, no persistence yet — safe)
