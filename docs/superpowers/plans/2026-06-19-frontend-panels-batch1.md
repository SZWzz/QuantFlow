# 17 Frontend Panels Batch 1 — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan task-by-task.

**Goal:** Add 17 new frontend panels (5 market + 7 charts/portfolio + 5 trading). Zero Go backend changes.

**Architecture:** Each panel is a self-contained Vue SFC (`<script setup lang="ts">`). Panels use existing Pinia stores or inline fetch with mock fallback. New `stats.ts` for pure-frontend statistical functions.

**Spec ref:** `docs/superpowers/specs/2026-06-19-frontend-panels-batch1-design.md`

## Global Constraints
- Props: `defineProps<{ panelId: string; params?: Record<string, any> }>()`
- Registration: `defineAsyncComponent` in `registry.ts`
- Mock fallback when `(window as any).go?.main?.App` is unavailable
- CSS: use `--color-bg`, `--color-text`, `--color-border` variables
- Each panel has a vitest test: mount + exists + title check + key elements check
- No Go/Python changes

## Execution Order
Phase 0 (shared infra) → Phase 1 (market, 5) → Phase 2 (charts, 7) → Phase 3 (trading, 5) → Phase 4 (finalize)

---

## Phase 0: Shared Infrastructure

### Task 0.1: Create stats.ts library
- **Create:** `frontend/src/lib/stats.ts`
- **Content:** `pearsonMatrix()`, `histogramBins()`, `simulateGBM()`, `computeDrawdowns()`, `sharpeRatio()` — pure TS, no deps

### Task 0.2: Extend data store
- **Modify:** `frontend/src/stores/data.ts` — add `MarketOverview` interface, `marketOverview` ref, `fetchMarketOverview()` method
- **Modify:** `frontend/src/stores/data.test.ts` — add marketOverview tests

### Task 0.3: Extend portfolio store
- **Modify:** `frontend/src/stores/portfolio.ts` — add `Order`, `Trade`, `EquityCurvePoint` interfaces; `orders`/`trades`/`equityCurve` refs; `fetchOrders()`/`fetchTrades()`/`fetchEquityCurve()` methods
- **Modify:** `frontend/src/stores/portfolio.test.ts` — add orders/trades tests

---

## Phase 1: Market Panels (5)

### Task 1.1: MarketOverviewPanel
- **Create:** `frontend/src/terminal/panels/MarketOverviewPanel.vue`
- **Create:** `frontend/src/terminal/panels/__tests__/MarketOverviewPanel.test.ts`
- **Modify:** `frontend/src/terminal/panels/registry.ts` — register `market-overview`
- **Spec:** 7 index cards + breadth bar + sector rankings

### Task 1.2: MarketDepthPanel
- **Create:** `frontend/src/terminal/panels/MarketDepthPanel.vue`
- **Create:** `frontend/src/terminal/panels/__tests__/MarketDepthPanel.test.ts`
- **Modify:** `frontend/src/terminal/panels/registry.ts` — register `market-depth`
- **Spec:** 5-level bid/ask ladder + tick timeline

### Task 1.3: HeatmapPanel
- **Create:** `frontend/src/terminal/panels/HeatmapPanel.vue`
- **Create:** `frontend/src/terminal/panels/__tests__/HeatmapPanel.test.ts`
- **Modify:** `frontend/src/terminal/panels/registry.ts` — register `heatmap`
- **Spec:** ECharts treemap, area=market cap, color=% change

### Task 1.4: TickerTapePanel
- **Create:** `frontend/src/terminal/panels/TickerTapePanel.vue`
- **Create:** `frontend/src/terminal/panels/__tests__/TickerTapePanel.test.ts`
- **Modify:** `frontend/src/terminal/panels/registry.ts` — register `ticker-tape`
- **Spec:** Horizontal scrolling quote bar, CSS animation

### Task 1.5: CryptoOverviewPanel
- **Create:** `frontend/src/terminal/panels/CryptoOverviewPanel.vue`
- **Create:** `frontend/src/terminal/panels/__tests__/CryptoOverviewPanel.test.ts`
- **Modify:** `frontend/src/terminal/panels/registry.ts` — register `crypto-overview`
- **Spec:** BTC/ETH dominance bars + Top 20 table

---

## Phase 2: Charts & Portfolio Panels (7)

### Task 2.1: EquityCurvePanel
- **Create:** `frontend/src/terminal/panels/EquityCurvePanel.vue`
- **Create:** `frontend/src/terminal/panels/__tests__/EquityCurvePanel.test.ts`
- **Modify:** `frontend/src/terminal/panels/registry.ts` — register `equity-curve`
- **Spec:** ECharts dual-axis NAV + drawdown + 5 stats cards

### Task 2.2: SurfaceChartPanel
- **Create:** `frontend/src/terminal/panels/SurfaceChartPanel.vue`
- **Create:** `frontend/src/terminal/panels/__tests__/SurfaceChartPanel.test.ts`
- **Modify:** `frontend/src/terminal/panels/registry.ts` — register `surface-chart`
- **Spec:** ECharts GL 3D volatility surface, mock SVI model

### Task 2.3: CorrelationPanel
- **Create:** `frontend/src/terminal/panels/CorrelationPanel.vue`
- **Create:** `frontend/src/terminal/panels/__tests__/CorrelationPanel.test.ts`
- **Modify:** `frontend/src/terminal/panels/registry.ts` — register `correlation`
- **Spec:** ECharts heatmap N×N Pearson matrix

### Task 2.4: DistributionPanel
- **Create:** `frontend/src/terminal/panels/DistributionPanel.vue`
- **Create:** `frontend/src/terminal/panels/__tests__/DistributionPanel.test.ts`
- **Modify:** `frontend/src/terminal/panels/registry.ts` — register `distribution`
- **Spec:** Histogram + normal fit + skewness/kurtosis stats

### Task 2.5: DrawingPanel
- **Create:** `frontend/src/terminal/panels/DrawingPanel.vue`
- **Create:** `frontend/src/terminal/panels/__tests__/DrawingPanel.test.ts`
- **Modify:** `frontend/src/terminal/panels/registry.ts` — register `drawing`
- **Spec:** Canvas drawing tools (trendline/Fibonacci/text), localStorage persistence

### Task 2.6: MonteCarloPanel
- **Create:** `frontend/src/terminal/panels/MonteCarloPanel.vue`
- **Create:** `frontend/src/terminal/panels/__tests__/MonteCarloPanel.test.ts`
- **Modify:** `frontend/src/terminal/panels/registry.ts` — register `monte-carlo`
- **Spec:** GBM simulation paths + terminal distribution + VaR/CVaR

### Task 2.7: RebalancePanel
- **Create:** `frontend/src/terminal/panels/RebalancePanel.vue`
- **Create:** `frontend/src/terminal/panels/__tests__/RebalancePanel.test.ts`
- **Modify:** `frontend/src/terminal/panels/registry.ts` — register `rebalance`
- **Spec:** Current vs target allocation chart + trade list

---

## Phase 3: Trading Panels (5)

### Task 3.1: OrderBlotterPanel
- **Create:** `frontend/src/terminal/panels/OrderBlotterPanel.vue`
- **Create:** `frontend/src/terminal/panels/__tests__/OrderBlotterPanel.test.ts`
- **Modify:** `frontend/src/terminal/panels/registry.ts` — register `order-blotter`
- **Spec:** Filterable order table + cancel button + stats footer

### Task 3.2: ExecutionPanel
- **Create:** `frontend/src/terminal/panels/ExecutionPanel.vue`
- **Create:** `frontend/src/terminal/panels/__tests__/ExecutionPanel.test.ts`
- **Modify:** `frontend/src/terminal/panels/registry.ts` — register `execution`
- **Spec:** Trade history table with pagination

### Task 3.3: BasketOrderPanel
- **Create:** `frontend/src/terminal/panels/BasketOrderPanel.vue`
- **Create:** `frontend/src/terminal/panels/__tests__/BasketOrderPanel.test.ts`
- **Modify:** `frontend/src/terminal/panels/registry.ts` — register `basket-order`
- **Spec:** Multi-symbol basket builder + execution progress

### Task 3.4: BrokerStatusPanel
- **Create:** `frontend/src/terminal/panels/BrokerStatusPanel.vue`
- **Create:** `frontend/src/terminal/panels/__tests__/BrokerStatusPanel.test.ts`
- **Modify:** `frontend/src/terminal/panels/registry.ts` — register `broker-status`
- **Spec:** Broker card grid with connection status/latency/balance

### Task 3.5: ActionCenterPanel
- **Create:** `frontend/src/terminal/panels/ActionCenterPanel.vue`
- **Create:** `frontend/src/terminal/panels/__tests__/ActionCenterPanel.test.ts`
- **Modify:** `frontend/src/terminal/panels/registry.ts` — register `action-center`
- **Spec:** Event feed with stop-loss/profit/dividend/approval items

---

## Phase 4: Finalize

### Task 4.1: Run tests + fix + update CHANGELOG
- Run `npx vitest run` — all 93+ tests must pass
- Update `CHANGELOG.md` with all 17 panels
- Verify registry has all 46 panel IDs (29 existing + 17 new)
