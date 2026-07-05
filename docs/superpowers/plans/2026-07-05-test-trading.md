# 实施计划：Trading Core 测试全覆盖

参考：`docs/specs/2026-07-05-test-trading.md`

## Task 1: risk_pipeline 完整测试

**`internal/trading/risk_pipeline_test.go`**（新建/补充）：

```go
func TestCheckDrawdown_Normal(t *testing.T) {
    r := NewRiskPipeline(RiskConfig{
        PeakEquity:      100000,
        MaxDrawdownPct:  0.20,
    })
    err := r.CheckDrawdown(90000) // 10% drawdown, < 20%
    if err != nil {
        t.Error("expected no error for 10% drawdown, got:", err)
    }
}

func TestCheckDrawdown_Exceeds(t *testing.T) {
    r := NewRiskPipeline(RiskConfig{
        PeakEquity:      100000,
        MaxDrawdownPct:  0.20,
    })
    err := r.CheckDrawdown(75000) // 25% drawdown, > 20%
    if err == nil {
        t.Error("expected error for 25% drawdown")
    }
}

func TestCheckDrawdown_NewPeak(t *testing.T) {
    r := NewRiskPipeline(RiskConfig{
        PeakEquity:      100000,
        MaxDrawdownPct:  0.20,
    })
    _ = r.CheckDrawdown(110000) // new peak
    err := r.CheckDrawdown(90000) // 18.18% drawdown from new peak
    if err != nil {
        t.Error("expected no error, peak should have been updated")
    }
}

func TestCheckDrawdown_ConcurrentSafe(t *testing.T) {
    r := NewRiskPipeline(RiskConfig{PeakEquity: 100000, MaxDrawdownPct: 0.20})
    var wg sync.WaitGroup
    for i := 0; i < 10; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            r.CheckDrawdown(95000)
        }()
    }
    wg.Wait()
    // 不 panic = 通过
}

func TestCheckOrder_MaxPosition(t *testing.T) {
    r := NewRiskPipeline(RiskConfig{MaxPositionPct: 0.25})
    err := r.CheckOrder(&Order{Quantity: 100, Price: 500}, nil, 100000)
    // 100*500 = 50000 = 50% of 100000 > 25%
    if err == nil {
        t.Error("expected error for exceeding max position")
    }
}
```

## Task 2: 引擎端到端测试

**`internal/trading/engine_cn_test.go`**、`engine_us_test.go`、`engine_hk_test.go`：

```go
func TestEngineCN_BuyThenSell(t *testing.T) {
    broker := newMockBroker()
    engine := NewEngineCN(broker, DefaultRiskConfig())
    
    // 买入
    order := &Order{Symbol: "000001", Side: Buy, Quantity: 100, Price: 10.0}
    fill, err := engine.ExecuteOrder(context.Background(), order)
    if err != nil { t.Fatal(err) }
    if fill.Quantity != 100 { t.Error("expected full fill") }
    
    // 检查持仓
    pos := engine.GetPosition("000001")
    if pos.Shares != 100 { t.Error("expected 100 shares") }
    
    // 卖出
    sellOrder := &Order{Symbol: "000001", Side: Sell, Quantity: 100, Price: 11.0}
    sellFill, err := engine.ExecuteOrder(context.Background(), sellOrder)
    if err != nil { t.Fatal(err) }
    if sellFill.Quantity != 100 { t.Error("expected full fill") }
    
    // 检查 PnL
    if sellFill.PnL <= 0 { t.Error("expected positive PnL") }
}
```

## Task 3: 回测 runner 止损/止盈测试

**`internal/backtest/runner_test.go`**（补充）：

```go
func TestRunner_StopLossHit(t *testing.T) {
    data := []Kline{
        {Close: 100, Time: day(1)},
        {Close: 101, Time: day(2)},
        {Close: 95,  Time: day(3)},  // -5% → 触发 5% 止损
        {Close: 96,  Time: day(4)},
    }
    strategy := &Strategy{
        Entry:   func(ctx) bool { return ctx.BarIndex == 1 },
        Exit:    func(ctx) bool { return false },
    }
    config := RiskConfig{StopLossPct: 0.05, InitialCapital: 100000}
    runner := NewRunner(data, strategy, config)
    result, err := runner.Run(context.Background())
    if err != nil { t.Fatal(err) }
    if len(result.Trades) != 2 { // 入场 + 止损出场
        t.Error("expected 2 trades (entry + stop loss)")
    }
}
```

## Task 4: Portfolio 测试

**`internal/portfolio/portfolio_test.go`**（补充）：

```go
func TestPortfolio_AddRemovePosition(t *testing.T) {
    p := NewPortfolio(100000)
    p.AddPosition("AAPL", 10, 150.0)
    if p.Positions["AAPL"].Shares != 10 {
        t.Error("expected 10 shares")
    }
    p.RemovePosition("AAPL")
    if _, ok := p.Positions["AAPL"]; ok {
        t.Error("position should be removed")
    }
}

func TestPortfolio_TotalEquity(t *testing.T) {
    p := NewPortfolio(100000)
    p.AddPosition("AAPL", 10, 150.0)  // 1500
    p.AddPosition("MSFT", 5, 300.0)   // 1500
    // 现金: 100000 - 1500 - 1500 = 97000
    // 持仓市值: 1500 + 1500 = 3000
    if p.Cash != 97000 { t.Error("cash mismatch") }
    if p.GetEquity() != 100000 { t.Error("total equity mismatch") }
}
```

## 验证

```bash
go test ./internal/trading/... -v -count=1
go test ./internal/backtest/... -v -count=1
go test ./internal/portfolio/... -v -count=1
go test ./internal/trading/... ./internal/backtest/... ./internal/portfolio/... -race -count=1
```
