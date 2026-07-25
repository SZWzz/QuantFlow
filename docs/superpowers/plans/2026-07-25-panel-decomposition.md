# Large Panel Component Decomposition

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Decompose the 15 largest panel components (423–992 lines) into smaller, focused sub-components. Each resulting file should be under 400 lines with a single clear responsibility.

**Architecture:** Extract cohesive sections of each large panel into sibling child components under `frontend/src/terminal/panels/` (co-located with the parent). Each extraction follows the pattern: identify a visual/logical section, extract its template + script + scoped styles, define Props/Emits interface, test.

**Panels to decompose (sorted by line count):**

| Panel | Lines | Likely splits |
|-------|-------|---------------|
| CandlestickPanel.vue | 992 | ChartCanvas, ChartToolbar, IndicatorOverlay, TradeMarkers |
| MarketScannerPanel.vue | 947 | ScannerFilters, ScannerTable, ScannerRow |
| PortfolioSummary.vue | 883 | PortfolioHeader, HoldingsTable, RiskMetricsCard, PnLChart |
| SettingsPanel.vue | 838 | SettingsSection (reusable), SettingsFormField |
| GovDataPanel.vue | 824 | GovDataChart, GovDataTable, GovDataFilterBar |
| FinancialsPanel.vue | 772 | FinancialsTable, FinancialsChart, FinancialsHeader |
| WelcomePanel.vue | 686 | WelcomeQuickLinks, WelcomeSystemStatus, WelcomeRecentProjects |
| RebalancePanel.vue | 670 | RebalanceTargetTable, RebalanceSimulation, RebalanceResult |
| MarketOverviewPanel.vue | 651 | OverviewIndices, OverviewSector, OverviewSentiment |
| WatchlistPanel.vue | 552 | WatchlistTable, WatchlistRow, WatchlistToolbar |
| TradeHistory.vue | 519 | TradeHistoryTable, TradeHistoryFilter, TradeDetailModal |
| BasketOrderPanel.vue | 466 | BasketOrderTable, BasketOrderForm, BasketOrderSummary |
| IndicatorPanel.vue | 453 | IndicatorList, IndicatorChart, IndicatorConfig |
| TradingJournalPanel.vue | 430 | JournalEntryList, JournalEntryForm, JournalStats |
| AuditPanel.vue | 423 | AuditTable, AuditFilter, AuditDetailDrawer |

**Tech Stack:** Vue 3 Composition API, `<script setup lang="ts">`, CSS variables

---

### Task 1: Decompose CandlestickPanel.vue (~992 lines → 4 files)

**Files:**
- Create: `frontend/src/terminal/panels/CandlestickToolbar.vue`
- Create: `frontend/src/terminal/panels/CandlestickChart.vue`
- Create: `frontend/src/terminal/panels/CandlestickIndicatorOverlay.vue`
- Create: `frontend/src/terminal/panels/CandlestickTradeMarkers.vue`
- Modify: `frontend/src/terminal/panels/CandlestickPanel.vue`

**Approach:** Read the existing `CandlestickPanel.vue` first. Then extract:

1. **CandlestickToolbar** — symbol selector, interval picker, indicator toggle buttons, drawing tools
   - Props: `symbol: string`, `interval: string`, `indicators: string[]`
   - Emits: `update:symbol`, `update:interval`, `update:indicators`

2. **CandlestickChart** — the main ECharts K线 canvas
   - Props: `symbol: string`, `interval: string`, `data: OHLCVBar[]`
   - Emits: `range-change`, `crosshair-move`

3. **CandlestickIndicatorOverlay** — technical indicator overlay (MA, MACD, RSI, etc.)
   - Props: `data: OHLCVBar[]`, `activeIndicators: string[]`, `indicatorParams: Record<string, any>`
   - Emits: none (purely presentational, reading from props)

4. **CandlestickTradeMarkers** — buy/sell markers overlaid on chart
   - Props: `trades: TradeMarker[]`
   - Emits: `select-trade`

- [ ] **Step 1: Read CandlestickPanel.vue and identify boundaries**

```bash
cd frontend && npx vitest run src/terminal/panels/__tests__/CandlestickPanel.spec.ts 2>/dev/null
```
Expected: PASS (capture baseline)

- [ ] **Step 2: Extract CandlestickToolbar.vue**

Create file with the toolbar section (symbol picker, timeframe tabs, indicator toggle). Template + moved script + styles. Import in parent.

- [ ] **Step 3: Extract CandlestickChart.vue**

Create file with the ECharts initialization, resize handling, and data binding logic.

- [ ] **Step 4: Extract CandlestickIndicatorOverlay.vue**

Create file with the series overlay computation for each active indicator.

- [ ] **Step 5: Extract CandlestickTradeMarkers.vue**

Create file with annotated trade entry/exit markers on the chart.

- [ ] **Step 6: Wire up parent CandlestickPanel.vue**

Replace inline sections with `import` and template references. Parent becomes ~300 lines.

- [ ] **Step 7: Write tests for extracted components**

Create `frontend/src/terminal/panels/__tests__/CandlestickToolbar.spec.ts` etc.

- [ ] **Step 8: Run all tests**

```bash
cd frontend && npx vitest run
```
Expected: PASS (all existing + new tests)

- [ ] **Step 9: Run type check**

```bash
cd frontend && npx vue-tsc --noEmit
```
Expected: No errors

- [ ] **Step 10: Commit**

```bash
git add frontend/src/terminal/panels/
git commit -m "refactor(panel): decompose CandlestickPanel into 4 sub-components"
```

---

### Task 2: Decompose MarketScannerPanel.vue (~947 lines → 3 files)

**Files:**
- Create: `frontend/src/terminal/panels/ScannerFilters.vue`
- Create: `frontend/src/terminal/panels/ScannerTable.vue`
- Modify: `frontend/src/terminal/panels/MarketScannerPanel.vue`

**Same pattern as Task 1.**

**ScannerFilters** — market/type/sector/cap filter row, search input
- Props: `filters: ScannerFilters`, `markets: string[]`
- Emits: `update:filters`

**ScannerTable** — the data grid with sortable columns, pagination
- Props: `results: ScannerResult[]`, `columns: ColumnDef[]`, `sort: SortState`
- Emits: `update:sort`, `select-row`

- [ ] **Step 1: Read MarketScannerPanel.vue, capture baseline tests → Extract ScannerFilters → Extract ScannerTable → Wire parent → Write tests → Verify → Commit**

```bash
git commit -m "refactor(panel): decompose MarketScannerPanel into ScannerFilters + ScannerTable"
```

---

### Task 3: Decompose PortfolioSummary.vue (~883 lines → 4 files)

**Files:**
- Create: `frontend/src/terminal/panels/PortfolioHeader.vue`
- Create: `frontend/src/terminal/panels/HoldingsTable.vue`
- Create: `frontend/src/terminal/panels/RiskMetricsCard.vue`
- Create: `frontend/src/terminal/panels/PnLChart.vue` (if chart block exists standalone)
- Modify: `frontend/src/terminal/panels/PortfolioSummary.vue`

- [ ] **Step 1: Read → Extract each subsection → Wire parent → Test → Commit**

```bash
git commit -m "refactor(panel): decompose PortfolioSummary into PortfolioHeader, HoldingsTable, RiskMetricsCard, PnLChart"
```

---

### Task 4: Decompose SettingsPanel.vue (~838 lines → 2 files)

**Files:**
- Create: `frontend/src/terminal/panels/SettingsSection.vue`
- Modify: `frontend/src/terminal/panels/SettingsPanel.vue`

**SettingsSection** — a generic labelled section block used repeatedly:
```vue
<SettingsSection title="通知设置" icon="bell">
  <label><input type="checkbox" v-model="pushEnabled"> 推送通知</label>
</SettingsSection>
```

- [ ] **Step 1: Read → Extract SettingsSection → Replace all inline section blocks → Test → Commit**

```bash
git commit -m "refactor(panel): decompose SettingsPanel, extract reusable SettingsSection"
```

---

### Task 5: Decompose GovDataPanel.vue (~824 lines → 3 files)

**Files:**
- Create: `frontend/src/terminal/panels/GovDataFilterBar.vue`
- Create: `frontend/src/terminal/panels/GovDataChart.vue`
- Create: `frontend/src/terminal/panels/GovDataTable.vue`
- Modify: `frontend/src/terminal/panels/GovDataPanel.vue`

- [ ] **Step 1: Read → Extract each → Wire parent → Test → Commit**

```bash
git commit -m "refactor(panel): decompose GovDataPanel into GovDataFilterBar + GovDataChart + GovDataTable"
```

---

### Task 6: Decompose FinancialsPanel.vue (~772 lines → 3 files)

**Files:**
- Create: `frontend/src/terminal/panels/FinancialsHeader.vue`
- Create: `frontend/src/terminal/panels/FinancialsTable.vue`
- Create: `frontend/src/terminal/panels/FinancialsChart.vue`
- Modify: `frontend/src/terminal/panels/FinancialsPanel.vue`

- [ ] **Step 1: Read → Extract each → Wire parent → Test → Commit**

```bash
git commit -m "refactor(panel): decompose FinancialsPanel into FinancialsHeader + FinancialsTable + FinancialsChart"
```

---

### Task 7: Decompose remaining 9 panels (~423–686 lines each)

**Same extraction pattern.** Each panel gets its own commit. For brevity, here's the checklist:

- [ ] **WelcomePanel** → `WelcomeQuickLinks.vue` + `WelcomeSystemStatus.vue` + `WelcomeRecentProjects.vue`
  ```bash
  git commit -m "refactor(panel): decompose WelcomePanel into 3 sub-components"
  ```

- [ ] **RebalancePanel** → `RebalanceTargetTable.vue` + `RebalanceSimulation.vue` + `RebalanceResult.vue`
  ```bash
  git commit -m "refactor(panel): decompose RebalancePanel into 3 sub-components"
  ```

- [ ] **MarketOverviewPanel** → `OverviewIndices.vue` + `OverviewSector.vue` + `OverviewSentiment.vue`
  ```bash
  git commit -m "refactor(panel): decompose MarketOverviewPanel into 3 sub-components"
  ```

- [ ] **WatchlistPanel** → `WatchlistToolbar.vue` + `WatchlistTable.vue`
  ```bash
  git commit -m "refactor(panel): decompose WatchlistPanel into WatchlistToolbar + WatchlistTable"
  ```

- [ ] **TradeHistory** → `TradeHistoryFilter.vue` + `TradeHistoryTable.vue` + `TradeDetailModal.vue`
  ```bash
  git commit -m "refactor(panel): decompose TradeHistory into 3 sub-components"
  ```

- [ ] **BasketOrderPanel** → `BasketOrderForm.vue` + `BasketOrderTable.vue` + `BasketOrderSummary.vue`
  ```bash
  git commit -m "refactor(panel): decompose BasketOrderPanel into 3 sub-components"
  ```

- [ ] **IndicatorPanel** → `IndicatorList.vue` + `IndicatorChart.vue` + `IndicatorConfig.vue`
  ```bash
  git commit -m "refactor(panel): decompose IndicatorPanel into 3 sub-components"
  ```

- [ ] **TradingJournalPanel** → `JournalEntryList.vue` + `JournalEntryForm.vue` + `JournalStats.vue`
  ```bash
  git commit -m "refactor(panel): decompose TradingJournalPanel into 3 sub-components"
  ```

- [ ] **AuditPanel** → `AuditFilter.vue` + `AuditTable.vue` + `AuditDetailDrawer.vue`
  ```bash
  git commit -m "refactor(panel): decompose AuditPanel into 3 sub-components"
  ```

---

### Task 8: Verify full test suite

- [ ] **Step 1: Run all tests**

```bash
cd frontend && npx vitest run
```
Expected: PASS

- [ ] **Step 2: Type check**

```bash
cd frontend && npx vue-tsc --noEmit
```
Expected: No errors

- [ ] **Step 3: Commit**

```bash
git add frontend/src/terminal/panels/
git commit -m "chore: post-decomposition test & type verification"
```

---

### Task 9: Update CHANGELOG

**Files:**
- Modify: `CHANGELOG.md`

- [ ] **Step 1: Add entries**

```markdown
### Changed
- [Panel] Decomposed CandlestickPanel into CandlestickToolbar, CandlestickChart, CandlestickIndicatorOverlay, CandlestickTradeMarkers
- [Panel] Decomposed MarketScannerPanel into ScannerFilters, ScannerTable
- [Panel] Decomposed PortfolioSummary into PortfolioHeader, HoldingsTable, RiskMetricsCard, PnLChart
- [Panel] Decomposed SettingsPanel, extracted reusable SettingsSection
- [Panel] Decomposed GovDataPanel into GovDataFilterBar, GovDataChart, GovDataTable
- [Panel] Decomposed FinancialsPanel into FinancialsHeader, FinancialsTable, FinancialsChart
- [Panel] Decomposed WelcomePanel, RebalancePanel, MarketOverviewPanel, WatchlistPanel, TradeHistory, BasketOrderPanel, IndicatorPanel, TradingJournalPanel, AuditPanel
```

- [ ] **Step 2: Commit**

```bash
git add CHANGELOG.md && git commit -m "chore: update CHANGELOG for panel decomposition"
```
