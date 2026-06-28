# Plan: Financial Analyzer Integration

## Phase 1: Python — fincept/analyzer.py

**Task 1.1**: 创建 `python/src/data/fincept/analyzer.py`
- `analyze_report(financials_json)` → 三年对比表 + 异常标记 + 健康评分
- `compute_valuation(financials_json, quote_json)` → DCF三情景 + 买卖建议
- `detect_audit_risks(financials_json)` → 审计风险发现列表
- `forecast_financials(financials_json)` → 三情景预测表

**Task 1.2**: `fetcher.py` 新增 `analyzer` source 路由 4 条

## Phase 2: Go — app_research.go

**Task 2.1**: 新增 `GetFinancialAnalysis` / `GetValuation` / `GetAuditFindings` / `GetForecast`

**Task 2.2**: `wails-runtime.d.ts` 新增 TS 类型声明

## Phase 3: Frontend

**Task 3.1**: FinancialsPanel 增强 — JSON → 三年对比表 + 异常高亮 + 健康评分
**Task 3.2**: ValuationPanel 新增 — DCF估值条 + 买卖建议
**Task 3.3**: AuditPanel 新增 — 风险分组审计发现列表
**Task 3.4**: ForecastPanel 新增 — 三情景预测表
**Task 3.5**: registry + i18n + icons 注册

## Files

| File | Action |
|------|--------|
| `python/src/data/fincept/analyzer.py` | New |
| `python/src/data/fetcher.py` | Add routes |
| `app_research.go` | Add 4 methods |
| `frontend/src/types/wails-runtime.d.ts` | TS types |
| `frontend/src/terminal/panels/FinancialsPanel.vue` | Rewrite |
| `frontend/src/terminal/panels/ValuationPanel.vue` | New |
| `frontend/src/terminal/panels/AuditPanel.vue` | New |
| `frontend/src/terminal/panels/ForecastPanel.vue` | New |
| `registry.ts` `i18n` `icons` `CHANGELOG.md` | Update |
