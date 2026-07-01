# 财务报表面板三表展示 — 实施计划

## Task 1: Go 后端新增 GetFinancialStatements

**文件**: `app_research.go`

新增方法：
```go
func (a *App) GetFinancialStatements(symbol string) (map[string]interface{}, error) {
    ctx := context.Background()
    income, _ := a.sinaFinAdpt.FetchIncomeStatement(ctx, symbol, 12)
    balance, _ := a.sinaFinAdpt.FetchBalanceSheet(ctx, symbol, 12)
    cashflow, _ := a.sinaFinAdpt.FetchCashFlow(ctx, symbol, 12)
    if income == nil && balance == nil {
        return nil, fmt.Errorf("no financial data for %s", symbol)
    }
    return map[string]interface{}{
        "income":   formatFinPeriods(income),
        "balance":  formatFinPeriods(balance),
        "cashflow": formatFinPeriods(cashflow),
    }, nil
}
```

## Task 2: 重写 FinancialsPanel.vue

**文件**: `frontend/src/terminal/panels/FinancialsPanel.vue`

三 Tab：利润表 / 资产负债表 / 现金流量表，每张表行=科目名，列=报告期。
