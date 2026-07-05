# Test Coverage: Trading Core (engine, backtest, portfolio)

## Motivation

交易核心（`internal/trading`、`internal/backtest`、`internal/portfolio`）共 23 个源文件但有 9 个测试文件，覆盖不均：

| 包 | 源文件 | 测试文件 | 关键漏测 |
|---|:---:|:---:|---|
| `internal/trading` | 11 | 4 | `risk_pipeline.go`（风控）、`engine_*.go`（A股/港股/美股引擎）、`rebalance.go` |
| `internal/backtest` | 7 | 3 | 回测 runner 的止损/止盈逻辑、多标的回测 |
| `internal/portfolio` | 5 | 2 | 组合再平衡、权重计算、PnL 归因 |

交易逻辑的正确性直接影响资金，测试优先级最高。

## Design

### 1. internal/trading — 引擎 + 风控

**风险管线 `risk_pipeline.go`**：
- `CheckDrawdown(equity)` — max drawdown 计算 + 并发安全
- `CheckOrder(order, position, portfolioValue)` — 仓位限制、止损止盈

```go
func TestCheckDrawdown_ExceedsThreshold(t *testing.T) {
    r := NewRiskPipeline(RiskConfig{
        InitialCapital:  100000,
        PeakEquity:      100000,
        MaxDrawdownPct:  0.20,
    })
    // 从峰值下跌 25%
    err := r.CheckDrawdown(75000)
    if err == nil {
        t.Error("expected drawdown error")
    }
}
```

**引擎 `engine_cn.go`, `engine_us.go`, `engine_hk.go`**：
- 下单 → 成交 → 持仓 → PnL 完整生命周期
- T+1 规则（A股）、T+2 规则（港股）
- 止损/止盈自动平仓

### 2. internal/backtest — 回测执行

**`runner.go`**：
- 给定行情数据 + 策略 → 执行回测 → 输出 metrics
- 止损/止盈路径（之前 P0 bug）
- 多标的组合回测

```go
func TestRunner_StopLoss(t *testing.T) {
    data := generateTestKlines([]float64{100, 101, 102, 95, 96})
    strategy := &Strategy{
        Entry:   func(ctx) bool { return true }, // 立即入场
        StopLoss: 0.05,
    }
    runner := NewRunner(data, strategy, DefaultRiskConfig())
    result, err := runner.Run(ctx)
    if err != nil { t.Fatal(err) }
    if result.TotalReturn >= 0 {
        t.Error("expected negative return with stop loss hit")
    }
}
```

### 3. internal/portfolio — 组合管理

**`portfolio.go`**：
- 添加/移除持仓
- 权重计算
- PnL 归因

```go
func TestPortfolio_AddPosition(t *testing.T) {
    p := NewPortfolio(100000)
    p.AddPosition("AAPL", 10, 150.0)
    if p.Positions["AAPL"].Shares != 10 {
        t.Error("expected 10 shares")
    }
    if p.GetEquity() != 100000 + 10*150 {
        t.Error("equity calc wrong")
    }
}
```

## Acceptance Criteria

- [ ] `risk_pipeline.go` 全覆盖测试（max drawdown、仓位限制、止损止盈）
- [ ] A股/港股/美股引擎各有一个端到端下单→成交→PnL 测试
- [ ] 回测 runner 测试止损、止盈、多标的组合
- [ ] portfolio 测试添加/移除持仓、PnL 计算、再平衡
- [ ] `go test ./internal/trading/... -count=1` 全部通过
- [ ] `go test ./internal/backtest/... -count=1` 全部通过
- [ ] `go test ./internal/portfolio/... -count=1` 全部通过
- [ ] 竞态检测器 `go test -race` 无报错

## Risks / Trade-offs

- 引擎测试需要 mock broker 和 mock market data。现有 `broker_test.go` 已有 mock broker 模式可复用。
- 港股 T+2 和 A 股 T+1 规则在回测中模拟起来复杂，可用简化模型：T+N 日资金可用。
- 回测 runner 的部分逻辑依赖时间轴推进，`time.Now()` 调用需要可注入（或使用 `clock` interface），否则测试不稳定。
