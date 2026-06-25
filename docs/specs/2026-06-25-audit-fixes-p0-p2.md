# Audit Fixes: P0–P2

## P0: Critical

| ID | Issue | Fix | Files |
|----|-------|-----|-------|
| C1 | P&L 每次 mark-to-market 覆盖已实现盈亏 | `Position` 新增 `RealizedPnl`，卖单累加，`TotalPnl = RealizedPnl + (marketPrice - avgPrice) * qty` | `trading/types.go`, `trading/oms.go` |
| C2 | PlaceOrderLive 零风险检查 | 调用 `RiskPipeline.CheckOrder()` | `trading/oms.go` |
| C3 | MaxDrawdownPct 从未检查 | `RiskPipeline` 加 `CheckDrawdown(equity, peak)` 断路器 | `trading/risk_pipeline.go` |
| C4 | API 密钥暴露给前端 | `GetConfig()` 过滤 `api_keys` | `app.go` |

## P1: High

| ID | Issue | Fix | Files |
|----|-------|-----|-------|
| H1 | RiskDashboard 假数据 | 面板加"占位" banner | `RiskDashboard.vue` |
| H2 | GetEquityCurve 不存在 | 移除调用，注释说明 | `stores/portfolio.ts` |
| H4 | 止损成交静默失败 | `paper_engine.go` 错误 → notify store | `trading/paper_engine.go` |
| C5 | gRPC 绑定 [::] | 改为 `localhost:{port}` | `python/src/server.py` |

## P2: Next

| ID | Issue | Fix | Files |
|----|-------|-----|-------|
| H7 | 回测未暴露 | 加 `RunBacktest(jsonDef)` Wails 绑定 | `app.go` |
| M7 | MarketDataHub 未启动 | `ServiceStartup` 初始化 hub | `app.go` |
| M8 | Scheduler cron 未启动 | `ServiceStartup` 创建 + 启动 Scheduler | `app.go` |

## AC

- [ ] P&L 含已实现+未实现
- [ ] 实时下单前风险检查
- [ ] 回撤超限阻止新开仓
- [ ] 前端拿不到 api_keys
- [ ] 止损失败通知
- [ ] gRPC localhost only
- [ ] `go vet` + `npm run build` 通过
