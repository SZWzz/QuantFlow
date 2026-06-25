# Audit Fixes Plan — P0 → P1 → P2 (11 tasks)

## P0: Critical (4 tasks)

**P0-1: Fix P&L (C1)**
- `trading/types.go`: Position + `RealizedPnl float64`
- `trading/oms.go`: FillOrder 累加 RealizedPnl, UpdateMarketPrice PnL = Realized + Unrealized
- Commit: `[Fix] P&L: add RealizedPnl accumulator`

**P0-2: Real-time Risk Checks (C2+C3)**
- `trading/risk_pipeline.go`: + `CheckDrawdown()` circuit breaker
- `trading/oms.go`: PlaceOrderLive 调用前 CheckDrawdown + CheckOrder
- Commit: `[Fix] risk: order checks + drawdown circuit breaker`

**P0-3: Filter API Keys (C4)**
- `app.go`: GetConfig() delete api_keys
- Commit: `[Fix] security: strip api_keys from GetConfig`

**P0-4: Build verify P0**
- `go vet` + `go build`
- Commit: `[Chore] verify P0 fixes`

---

## P1: High (4 tasks)

**P1-1: RiskDashboard Placeholder (H1)**
- `RiskDashboard.vue`: 顶部黄色 banner
- Commit: `[Frontend] RiskDashboard: placeholder warning`

**P1-2: Fix GetEquityCurve (H2)**
- `stores/portfolio.ts`: 移除不存在的方法调用
- Commit: `[Frontend] portfolio: remove GetEquityCurve`

**P1-3: Stop-Loss Alert (H4)**
- `trading/paper_engine.go`: 成交失败 → notify
- Commit: `[Fix] paper engine: notify on stop-loss failure`

**P1-4: gRPC localhost (C5)**
- `python/server.py`: `[::]` → `localhost`
- Commit: `[Fix] security: gRPC localhost only`

---

## P2: Next (2 tasks)

**P2-1: Backtest API (H7)**
- `app.go`: 加 `RunBacktest` Wails 绑定
- Commit: `[Go] expose RunBacktest API`

**P2-2: Hub + Scheduler Startup (M7+M8)**
- `app.go`: ServiceStartup 初始化 MarketDataHub + Scheduler
- Commit: `[Go] init MarketDataHub + Scheduler`

---

## Build + CHANGELOG
Commit: `[Chore] CHANGELOG: P0-P2 audit fixes`
