# 财务审计面板重设计 — 实施计划

Phase 对应 spec `docs/specs/2026-06-30-audit-panel-redesign.md`。

## Task 1: 重写 AuditPanel.vue

**文件**: `frontend/src/terminal/panels/AuditPanel.vue`

完全替换文件内容，新增：
- 并行调用 `GetAuditFindings` + `GetFinancialAnalysis`
- Risk gauge 仪表盘（SVG 进度条）
- KPI 卡片行（ROE/负债率/净利率/毛利率/营收增长）
- 评分明细（可折叠展开）
- 异常发现列表（分类卡片）
- 财务历史表格（12期）

完整代码见下方。
