# Spec: 财务深度分析面板 — Financial Analyzer 集成

## Motivation

QuantFlow 已有财报数据（新浪三表）、行情、研报，但缺少分析层——FinancialsPanel 至今仍是裸 JSON。需要：三年对比趋势、财务健康评分、DCF 估值、审计风险检测、财务预测。

## Design

### 数据流

```
Python: fincept/analyzer.py (新)
  report_analysis() / valuation() / audit() / forecast()
       ↓ gRPC FetchData
Go: 4 个新 App 方法（Go预取财报JSON→传Python计算）
       ↓ Wails IPC
Frontend: 1增强(Financials) + 3新增(Valuation/Audit/Forecast)
```

### Python：fincept/analyzer.py

4 个纯计算函数，输入为 Go 预取的财报/行情 JSON：

| 函数 | 功能 | 输出 |
|------|------|------|
| `analyze_report(json)` | 三年对比 + 异常标记 + 健康评分 | `{periods, anomalies, score}` |
| `compute_valuation(json)` | DCF三情景 + 买卖建议 | `{scenarios, fair_value, buy_sell}` |
| `detect_audit_risks(json)` | 收入质量/商誉/现金流异常 | `{findings[{level, metric, detail}]}` |
| `forecast_financials(json)` | 线性回归 + 三情景预测 | `{segments, forecast_table}` |

### Go：4 个新 App 方法

每个方法先通过已有 adapter 获取财报/行情 JSON，再调 `FetchData("analyzer", type, ...)` → Python：

| Go 方法 | Python type | 面板 |
|---------|------------|------|
| `GetFinancialAnalysis(symbol)` | `report_analysis` | FinancialsPanel 增强 |
| `GetValuation(symbol)` | `valuation` | ValuationPanel 新 |
| `GetAuditFindings(symbol)` | `audit` | AuditPanel 新 |
| `GetForecast(symbol)` | `forecast` | ForecastPanel 新 |

### 前端

- **FinancialsPanel**：裸JSON→三年对比表+异常红色高亮+健康评分卡片
- **ValuationPanel**：DCF估值区间条+可比公司+买卖建议
- **AuditPanel**：按风险等级分组的审计发现列表
- **ForecastPanel**：业务拆分+三情景预测表

### 文件

| 文件 | 操作 |
|------|------|
| `python/src/data/fincept/analyzer.py` | 新 |
| `python/src/data/fetcher.py` | 改 — analyzer路由 |
| `app_research.go` | 改 — 4个方法 |
| `frontend/src/terminal/panels/FinancialsPanel.vue` | 改 — JSON→结构化 |
| `frontend/src/terminal/panels/ValuationPanel.vue` | 新 |
| `frontend/src/terminal/panels/AuditPanel.vue` | 新 |
| `frontend/src/terminal/panels/ForecastPanel.vue` | 新 |
| registry.ts / i18n / icons / wails-runtime.d.ts | 改 |

## Acceptance Criteria

- [ ] FinancialsPanel 显示三年对比表+异常高亮+健康评分
- [ ] ValuationPanel 显示 DCF 估值区间+买卖建议
- [ ] AuditPanel 显示按风险等级分组的审计发现
- [ ] ForecastPanel 显示三情景预测表
- [ ] `go build` + `vue-tsc` 通过
