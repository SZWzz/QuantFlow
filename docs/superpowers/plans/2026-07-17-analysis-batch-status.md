# 2026-07-17 分析维度补齐 Plan 执行状态

> Spec: `docs/specs/2026-07-17-analysis-dimensions.md`
> 9 个 plan（P0: 3 + P1: 3 + P2: 3），按「链路打通 → 深度分析 → 增强功能」优先级执行。

**最后更新**: 2026-07-17

---

## 📊 总览

| 优先级 | Plan | Tasks | 新面板 | 新 Adapter | 数据就绪 |
|--------|------|-------|--------|-----------|---------|
| P0 | [SectorDashboard](./2026-07-17-p0-sector-dashboard.md) | 4 | 1 | 0 | ✅ 全部现有 |
| P0 | [ValuationBand](./2026-07-17-p0-valuation-band.md) | 3 | 0(增强) | 0 | ✅ 全部现有 |
| P0 | [DupontAnalysis](./2026-07-17-p0-dupont-analysis.md) | 3 | 1 | 0 | ✅ 全部现有 |
| P1 | [MacroDashboard](./2026-07-17-p1-macro-dashboard.md) | 3 | 1 | 0 | ✅ 全部现有 |
| P1 | [MarketStyle](./2026-07-17-p1-market-style.md) | 3 | 1 | 0 | ✅ 全部现有 |
| P1 | [EventStudy](./2026-07-17-p1-event-study.md) | 3 | 1 | 0 | ✅ 全部现有 |
| P2 | [ShareholderAnalysis](./2026-07-17-p2-shareholder-analysis.md) | 3 | 1 | 1 | ❌ 需新 adapter |
| P2 | [UnlockCalendar](./2026-07-17-p2-unlock-calendar.md) | 3 | 1 | 1 | ❌ 需新 adapter |
| P2 | [FactorAttribution](./2026-07-17-p2-factor-attribution.md) | 3 | 1 | 0 | ⚠️ Python engine 已有 |

---

## 执行计划

### 批次 A: P0 — 打通分析链路（本次）
```
SectorDashboard → ValuationBand → DupontAnalysis
```
- 3 个 plan，10 个 task
- 0 个新 adapter
- 产出：行业→个股→估值→财务 完整分析链

### 批次 B: P1 — 宏观 + 市场 + 事件（下次）
```
MacroDashboard → MarketStyle → EventStudy
```
- 3 个 plan，9 个 task
- 0 个新 adapter
- 产出：宏观视角 + 市场风格 + 事件驱动

### 批次 C: P2 — 增强功能（待定）
```
ShareholderAnalysis → UnlockCalendar → FactorAttribution
```
- 3 个 plan，9 个 task
- 2 个新 adapter（股东 + 解禁）
- 产出：股东结构 + 解禁预警 + 因子归因

---

## 依赖关系

- **ValuationBand** 可独立（仅依赖 OHLCV + EPS）
- **DupontAnalysis** 可独立（仅依赖 FinancialData）
- **SectorDashboard** 依赖 ValuationBand 的 PE 分位计算（共用 `computePEPercentile`）
- P1 三个 plan 相互独立
- P2 ShareholderAnalysis + UnlockCalendar 共享 EastMoney datacenter API 模式

---

## 执行记录

### 批次 A: P0

| # | Plan | 状态 | 完成日期 |
|---|------|------|---------|
| 1 | SectorDashboard | ⏳ 待执行 | - |
| 2 | ValuationBand | ⏳ 待执行 | - |
| 3 | DupontAnalysis | ⏳ 待执行 | - |
