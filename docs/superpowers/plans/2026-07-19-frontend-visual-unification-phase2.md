# 前端视觉统一 Phase 2 Implementation Plan — 全量面板迁移

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 把剩余 79 个面板全部迁移到统一骨架与共享组件层，达到 Phase 1 打样验收的"标准长相"，并通过 stylelint error 级守卫防退化。

**Architecture:** 复用 Phase 1 已建好的迁移模式库（`.superpowers/sdd/migration-playbook.md`，与计划同源的 P1–P7 模式）。Task 1 先打地基补丁（终审遗留组件级修复）；Tasks 2–13 按 12 个 dispatch 分批迁移（每批一次实现 + 一次审查 + 一个 commit）；Task 14 收尾：stylelint 转 error + 全量验收。

**Tech Stack:** Vue 3 (`<script setup>`), TypeScript, vitest + @vue/test-utils, ECharts, stylelint。

**前置：** Phase 0+1 已合入 main（merge `e056bd1`）并继续在分支 `feat/frontend-visual-unification` 上工作。共享组件（`@/terminal/components/panel`）：PanelHeader（含 slots）、PanelTabs、PanelTable（mono/sortable/rowClass/hideHeader）、StatItem、EmptyState、ErrorState、LoadingState；composable：`useChartTheme`、`useFlashOnUpdate`；全局类：`.panel-header`/`.empty-state`/`.panel-error`/`.section-title`/`.flash-up`/`.flash-down`。

## Global Constraints

- 继承 Phase 1 计划全部 Global Constraints（见 `docs/superpowers/plans/2026-07-18-frontend-visual-unification.md`）
- **不得删除既有功能**（WatchlistPanel 教训）：共享组件表达不了的交互保留自绘但 token 化，并在报告中列出；功能裁减只有用户明确批准才可做
- 每面板迁移走 `.superpowers/sdd/migration-playbook.md` 的 P1–P7 + 自检 greps
- 命令式渲染图表面板迁移时，option 改为 computed（BacktestPanel 范式），使其随主题切换重绘
- 所有命令在 `frontend/` 下执行；conventional commits；每个 dispatch 一个 commit
- 每个 Task 结束必须 `npm run typecheck` 无错误、`npm run test` 全绿

## Phase 2 退出标准（终审遗留，全部满足才算完成）

- [ ] stylelint 三条守卫规则（hex / px font-size / border:none）转为 `"error"`，**保留** `defaultSeverity: "warning"`；全量 `lint:styles` 守卫类 warning = 0
- [ ] PanelTable 支持 sticky 表头（分组/长表场景）
- [ ] `grep -rn "color: var(--color-down)" frontend/src/terminal/panels | grep "\.up\b"` 无结果（无涨跌色反转残留）
- [ ] `.btn-sm` 提升为全局类，panels 内重复定义清零
- [ ] WatchlistPanel 分组/排序/闪烁有单元测试
- [ ] typecheck 0 错误、vitest 全绿、build 成功、e2e 无新增失败（基线：market-watch 6 + order-entry 1 既有失败）

---

### Task 1: 地基补丁包（终审遗留组件级修复）

**Files:**
- Modify: `frontend/src/terminal/components/panel/PanelTable.vue`、`types.ts`
- Modify: `frontend/src/assets/themes.css`（全局 `.btn-sm`）
- Modify: `frontend/src/terminal/panels/SatellitePanel.vue`（涨跌色反转）
- Modify: `frontend/src/terminal/panels/CandlestickPanel.vue`、`frontend/src/terminal/panels/candlestick/ChartToolbar.vue`（死 emit）
- Modify: `frontend/src/terminal/panels/MarketOverviewPanel.vue`（loadError 接通或删除）
- Modify: `frontend/src/terminal/panels/BacktestPanel.vue`（tooltip font-size）
- Test: `frontend/src/terminal/__tests__/PanelTable.test.ts`（追加）、`frontend/src/terminal/panels/__tests__/WatchlistPanel.test.ts`（追加分组/排序用例）

**Interfaces:**
- Produces: PanelTable 新 prop `stickyHeader?: boolean`（默认 false）；`Column` 不变。全局 `.btn-sm`。后续 Tasks 2–13 依赖这些修正。

- [ ] **Step 1: PanelTable sticky 表头（TDD）**

测试（追加到 PanelTable.test.ts）：

```typescript
it('applies sticky class to header when stickyHeader', () => {
  const w = mount(PanelTable, { props: { columns: cols, data, stickyHeader: true } })
  expect(w.find('.table-header-row').classes()).toContain('sticky')
})

it('colorize does not paint non-numeric placeholder', () => {
  const w = mount(PanelTable, { props: { columns: cols, data: [{ name: 'X', price: 1, chg: undefined }] } })
  const chgCell = w.findAll('.td.colorize')[0]
  expect(chgCell.attributes('style') ?? '').not.toContain('var(--color-down)')
})

it('mono:false overrides numeric format auto-mono', () => {
  const c2: Column[] = [{ key: 'v', label: 'V', format: 'price', mono: false }]
  const w = mount(PanelTable, { props: { columns: c2, data: [{ v: 1.5 }] } })
  expect(w.find('.td').classes()).not.toContain('mono')
})
```

实现：
1. props 加 `stickyHeader?: boolean`（默认 false）；`.table-header-row` 绑定 `{ sticky: stickyHeader }`；样式：

```css
.table-header-row.sticky {
  position: sticky;
  top: 0;
  z-index: 1;
  background: var(--color-bg-panel);
}
```

2. `colorize` 守卫：td 的 style 绑定处改为仅在 `typeof row[col.key] === 'number' && Number.isFinite(row[col.key])` 时应用 colorize 颜色。
3. 跑测试确认 3 个新用例通过。

- [ ] **Step 2: WatchlistPanel 分组/排序测试（追加到既有测试文件）**

```typescript
it('groups symbols by market with group headers', () => {
  // 用既有 mock 数据挂载，断言 .group-header 数量 ≥1 且默认展开时行可见
})

it('clicking sortable th emits sort and reorders rows', () => {
  // 点击涨跌幅表头两次，断言行顺序变化（asc→desc）
})
```

按面板现有 mock 方式实现，断言真实 DOM 顺序。

- [ ] **Step 3: 小修复批量**

1. `themes.css` 追加全局类（panel 迁移批会删除 panels 内的重复定义）：

```css
/* ── 小按钮修饰符（与 .btn 组合使用） ─────────────────────────── */
.btn-sm {
  padding: var(--space-xs) var(--space-sm);
  font-size: var(--font-xs);
  border-radius: var(--radius-sm);
}
```

2. `SatellitePanel.vue`：`.summary-badge.up { color: var(--color-down) }` → `var(--color-up)`（同类 `.down` 反转一并检查）。
3. `ChartToolbar.vue` 删除 `'addToWorkflow'` emit 声明；`CandlestickPanel.vue` 删除对应 `@addToWorkflow` 监听与 handler（若 handler 另有调用路径则保留 handler 只删监听）。
4. `MarketOverviewPanel.vue`：`loadError` 接通 —— 在数据拉取 catch 中赋值 `loadError.value = e?.message || '加载失败'`（ErrorState 已存在）；若该面板拉取函数无 catch，补最小 try/catch。
5. `BacktestPanel.vue`：tooltip formatter HTML 中的 `font-size:12px` → `font-size:var(--font-xs)`（ECharts tooltip 渲染在 DOM 中，inline style 支持 var()）。

- [ ] **Step 4: 验证 + Commit**

```bash
cd frontend && npm run typecheck && npm run test
git add -A frontend/src frontend/AGENTS.md 2>/dev/null; git add frontend/src/assets/themes.css frontend/src/terminal
git commit -m "feat(frontend): phase2 groundwork — PanelTable sticky header, colorize guard, global btn-sm, small fixes"
```

---

### Task 2–13: 面板分批迁移（12 个 dispatch）

**每批统一要求（不逐批重复）：**

1. 读 `.superpowers/sdd/migration-playbook.md`（P1–P7 + token 映射表 + 自检 greps），对批次内每个面板完整应用
2. **功能盘点先行**：列出该面板全部交互（排序/筛选/展开/右键/拖拽/弹窗），迁移后逐一核对还在；组件表达不了的保留自绘但 token 化，写进报告
3. 命令式 ECharts → computed option（useChartTheme）
4. panels 内重复定义的 `.btn-sm` 随迁移删除（全局类已在）
5. 自检 greps（playbook 末尾）对每个面板无输出；`npm run typecheck && npm run test` 全绿
6. 一批一个 commit：`refactor(frontend): migrate <batch-name> panels to shared skeleton`

**批次清单**（panel 后为行数与提示：hdr=手写 panel-header；bar=toolbar/tab-bar 变体；table=原生 table；ech=含 echarts）：

- [ ] **Task 2 · 债券基金期货组**（~870 行）：Bonds(147,hdr,table)、BrokerStatus(67,hdr)、Funds(114,hdr)、Futures(128,hdr,table)、SEC13F(129,hdr)、ShortInterest(155,hdr)、SectorDashboard(130,bar,table)
- [ ] **Task 3 · 套利日历组**（~1560 行）：CBArbitrage(257,hdr)、CryptoOverview(174,hdr,table)、DefiTVL(172,hdr)、DepthComparison(201,hdr)、EarningsCalendar(161,hdr)、EconomicCalendar(216,hdr)、ExDividend(198,hdr)、WhaleTracking(181,hdr)
- [ ] **Task 4 · 费率港股组**（~1430 行）：FundingRate(190,hdr)、GasFee(166,hdr)、Heatmap(174,hdr,ech)、HKConnect(286,hdr)、HKDerivatives(203,hdr)、Liquidation(217,hdr)、Margin(193,hdr)
- [ ] **Task 5 · 工具条变体组**（~760 行）：Dupont(145,bar,ech)、EventStudy(87,bar,ech)、FactorAttribution(84,bar,ech)、MacroDashboard(75,bar)、MarketStyle(84)、Notify(59,bar)、Schedule(99,bar)、Shareholder(68,bar,table)、UnlockCalendar(60,bar)
- [ ] **Task 6 · 系统辅助组**（~1150 行）：AIChat(286,bar)、ApiKeyManager(159)、BrokerConfig(75)、HelpCenter(89)、RLMonitor(57)、SystemMonitor(148)、TickerTape(156)、ScenarioAnalysis(186,hdr)
- [ ] **Task 7 · 大图表组 A**（~2700 行）：Audit(656,table,ech)、Backtest(324,ech)、Correlation(628,hdr,table,ech)、Distribution(475,hdr,table,ech)、Financials(623,hdr,bar,ech)
- [ ] **Task 8 · 大图表组 B**（~2780 行）：Geopolitics(537,hdr,ech)、GovData(826,ech)、HKSettlement(461,hdr)、IPOCalendar(459,hdr)、MonteCarlo(501,hdr,table)
- [ ] **Task 9 · 预测估值组**（~2450 行）：PredictionMarket(388,hdr,table,ech)、Satellite(582,hdr,ech)、Valuation(395,hdr,ech)、SectorRotation(235,hdr)、SurfaceChart(79,hdr,ech)、CongressTrading(172,hdr,bar,table)、DarkPool(322,hdr,bar,table)、FundFlow(283,hdr,table)
- [ ] **Task 10 · 研究工具组**（~2400 行）：AlphaMining(225)、FactorAnalysis(309)、IndicatorPanel(453)、LayoutTemplate(193)、LogPanel(309)、PredictionDashboard(126)、StockScanner(389)、StockResearch(195,hdr,bar,table)、DailyReport(252,hdr)
- [ ] **Task 11 · 交易执行组**（~2520 行）：BasketOrder(466)、Options(324,hdr,table)、OrderEntry(185)、TradeHistory(519,bar,table)、TradingJournal(430,bar)、WashSale(207,hdr,table)、Chanlun(388,hdr,table)
- [ ] **Task 12 · 组合再平衡组**（~1550 行）：PortfolioSummary(883,bar,table,ech)、Rebalance(670,table,ech)
- [ ] **Task 13 · 扫描设置欢迎组**（~2430 行）：MarketScanner(948,bar)、Settings(838)、WelcomePanel(644)、StoragePanel(212,hdr,table)

---

### Task 14: stylelint 转 error + Phase 2 验收

**Files:**
- Modify: `frontend/.stylelintrc.json`

- [ ] **Step 1: 守卫规则转 error（保留 defaultSeverity）**

`.stylelintrc.json` 中三条规则的 severity 从 `"warning"` 改为 `"error"`；`defaultSeverity: "warning"` 保持不变。同时把 `frontend/src/terminal/components/panel/` 下组件的既有合理违例（如有）通过行内豁免注释处理，而不是放宽规则。

- [ ] **Step 2: 清零守卫类 warning**

```bash
cd frontend && npm run lint:styles
# 期望：0 errors；warning 中 color-no-hex / font-size px / border:none 三类计数为 0
```

若迁移批引入了守卫类违例，逐一修复至清零。

- [ ] **Step 3: 全量验收**

```bash
cd frontend
npm run typecheck && npm run test && npm run build
npx playwright test --config e2e/playwright.config.ts 2>&1 | tail -5
# e2e 基线：market-watch 6 + order-entry 1 既有失败，不得新增失败
```

- [ ] **Step 4: 截图走查 + 用户确认**

用 `frontend/e2e/specs/screenshot-walkthrough.spec.ts` 截图（亮/暗/紧凑/宽松），抽查 5 个不同批次迁移的面板；`wails dev` 由用户走查确认。

- [ ] **Step 5: Commit**

```bash
git add frontend/.stylelintrc.json frontend/src
git commit -m "chore(frontend): promote stylelint guards to error level, phase 2 complete"
```

---

## Self-Review 记录

- **Spec coverage**：规格 §5 Phase 2 四批 → 本计划按模式重组为 12 批（79 面板全覆盖，逐一点名）；§4.4 stylelint error 转换 → Task 14；终审退出标准 → Task 1 + Task 14。
- **Placeholder scan**：迁移批的统一要求集中在"每批统一要求"块（完整规则），批次清单只列面板与提示，无 TBD。
- **Type consistency**：PanelTable `stickyHeader` 在 Task 1 定义；`.btn-sm` 全局类在 Task 1 定义、Tasks 2–13 消费。
