# 财务报表面板：三表展示

## Motivation

当前 FinancialsPanel 显示的是 Python `analyze_report` 的聚合数据（健康评分 + 指标表格），与 AuditPanel 功能重叠。审计面板已有评分/异常/明细，财务报表应专注展示原始三张表：利润表、资产负债表、现金流量表。

## Design

### 数据流

```
Go 新增 GetFinancialStatements(symbol)
  ├── sinaFinAdpt.FetchIncomeStatement(symbol, 12)
  ├── sinaFinAdpt.FetchBalanceSheet(symbol, 12)
  └── sinaFinAdpt.FetchCashFlow(symbol, 12)
  └── 返回 { income: [...], balance: [...], cashflow: [...] }

前端 FinancialsPanel
  └── 3 tabs: 利润表 | 资产负债表 | 现金流量表
      └── 每张表：行=科目，列=报告期
```

### 界面布局

```
┌─────────────────────────────────────────────────┐
│ 财务报表  [600519] 贵州茅台              [⟳]   │
├─────────────────────────────────────────────────┤
│ [利润表] [资产负债表] [现金流量表]               │
├─────────────────────────────────────────────────┤
│ 科目          │ 2025Q4  │ 2025Q3 │ 2025Q2 │     │
│ ─────────────────────────────────────────────── │
│ 营业总收入     │ 123.4亿 │ 112.3亿│ 98.7亿 │     │
│ 营业成本       │  10.2亿 │   9.8亿│  8.5亿 │     │
│ 毛利润         │ 113.2亿 │ 102.5亿│ 90.2亿 │     │
│ 销售费用       │   3.2亿 │   3.0亿│  2.8亿 │     │
│ 管理费用       │   5.1亿 │   4.8亿│  4.5亿 │     │
│ 净利润         │  45.6亿 │  41.2亿│ 38.5亿 │     │
└─────────────────────────────────────────────────┘
```

### 修改文件

| 文件 | 改动 |
|------|------|
| `app_research.go` | 新增 `GetFinancialStatements()` Wails 方法 |
| `frontend/src/terminal/panels/FinancialsPanel.vue` | 完全重写：删除评分列，三表 Tab 切换 |

### API 变更

新增 `GetFinancialStatements(symbol string) → {income, balance, cashflow}`

## Acceptance Criteria

- [ ] 三表 Tab 切换，每张表以"行=科目、列=报告期"展示
- [ ] 科目名保持中文原始名称（来自 Sina）
- [ ] 数值自动格式化（亿/万）
- [ ] 删除右侧评分/异常列
- [ ] `go vet ./...` 通过

## Risks / Trade-offs

- Sina 返回的中文科目标题可能包含空格/不一致，保持原样展示不作映射
- 12 期数据较多，横向滚动需要处理好
