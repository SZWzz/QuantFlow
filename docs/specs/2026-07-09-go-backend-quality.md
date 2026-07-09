# Go Backend Quality: Concurrency, Financial Correctness, Refactoring

## Motivation

Phase 12 review identified 9 concrete issues in the Go backend:

### Concurrency (3 issues)
- **Channel leak**: `topicBroker.subscribe` leaks a `chan MarketMessage(64)` per unsubscribe cycle (`hub.go:40-47`)
- **Busy-wait**: `ExecutionQueue.processLoop` spins at 100% CPU when queue is empty (`queue.go:113-157`)
- **God method**: `ServiceStartup` is ~450 lines — untestable, hard to reason about (`app_startup.go`)

### Financial Correctness (5 issues)
- **Hardcoded risk-free rate 2%** in Sharpe/Sortino (`metrics.go:67`). Misleading in low-rate (Japan, EU) or high-rate (Turkey, Brazil) markets.
- **Stamp duty not rounded to cents** (`engine_cn.go:80`). `tradeValue * rate` leaves full float precision; cumulative rounding error over thousands of trades.
- **Wash sale basis: wrong reference price** (`wash_sale.go`). Compares sale price to repurchase price, not to original cost basis.
- **US engine default 100 shares** (`engine_us.go:193`, `runner.go:147`). US stocks allow fractional shares — default should be 1.
- **`golang.org/x/net v0.53.0` stale** (`go.mod`). Latest is v0.35+ with security fixes.

### Other (1 issue)
- **Package-level `execQueue` var** (`app_startup.go`). Breaks encapsulation; should be a field on `App`.

## Design

### 1a. Fix topicBroker Channel Leak

**Current:**
```go
unsubscribe := func() {
    b.mu.Lock()
    defer b.mu.Unlock()
    delete(b.subscribers, subID)
    // Channel not closed — race with publish() sending outside lock
    // Rely on GC — never happens
}
```

**Fix:**
Replace the `chan MarketMessage` with a wrapper that has a `closed` flag:

```go
type subscriber struct {
    ch     chan MarketMessage
    closed bool
}

func (b *topicBroker) subscribe(subID string) (<-chan MarketMessage, func()) {
    b.mu.Lock()
    defer b.mu.Unlock()
    
    sub := &subscriber{ch: make(chan MarketMessage, 64)}
    b.subscribers[subID] = sub
    
    unsubscribe := func() {
        b.mu.Lock()
        defer b.mu.Unlock()
        if s, ok := b.subscribers[subID]; ok {
            s.closed = true
            delete(b.subscribers, subID)
            close(s.ch) // safe: publish() checks closed before send
        }
    }
    return sub.ch, unsubscribe
}

func (b *topicBroker) publish(msg MarketMessage) {
    b.mu.Lock()
    for id, sub := range b.subscribers {
        if sub.closed {
            delete(b.subscribers, id) // cleanup already-unsubscribed
            continue
        }
        select {
        case sub.ch <- msg:
        default: // slow consumer
        }
    }
    b.mu.Unlock()
}
```

**Modified files:**
- `internal/market/hub.go` — Replace `map[string]chan MarketMessage` with `map[string]*subscriber`; update `subscribe`/`publish`/`subscriberCount`
- `internal/market/hub_test.go` — Add test for rapid subscribe/unsubscribe cycles (1000 iterations, no goroutine leak)

### 1b. Fix processLoop Busy-Wait

**Current:** `for { lock → check queue → unlock → execute → lock → update → unlock }` spins when queue empty (then returns after processing all items).

**Fix:** Use `sync.Cond`:

```go
type ExecutionQueue struct {
    mu       sync.Mutex
    cond     *sync.Cond
    queue    []*QueuedWorkflow
    running  bool
    ctx      context.Context
    engine   Engine
}

func NewExecutionQueue(engine Engine, ctx context.Context) *ExecutionQueue {
    q := &ExecutionQueue{ctx: ctx, engine: engine}
    q.cond = sync.NewCond(&q.mu)
    return q
}

func (q *ExecutionQueue) Enqueue(wf *Workflow) (string, error) {
    q.mu.Lock()
    id := q.generateID()
    q.queue = append(q.queue, &QueuedWorkflow{ID: id, Workflow: wf})
    q.cond.Signal() // wake up processLoop
    q.mu.Unlock()
    return id, nil
}

func (q *ExecutionQueue) processLoop() {
    q.mu.Lock()
    defer q.mu.Unlock()
    for {
        for len(q.queue) == 0 {
            q.cond.Wait() // sleep until signaled
        }
        // process... (release lock during execution)
    }
}
```

**Modified files:**
- `internal/workflow/queue.go` — Replace spin-loop with `sync.Cond`; add constructor if missing

### 1c. Decompose ServiceStartup

Break into focused methods:

```go
func (a *App) ServiceStartup(ctx context.Context, app *application.App) error {
    steps := []struct{
        name string
        fn   func(context.Context) error
    }{
        {"initConfig", a.initConfig},
        {"initDatabase", a.initDatabase},
        {"initCredentialManager", a.initCredentialManager},
        {"initMarketData", a.initMarketData},
        {"initResearch", a.initResearch},
        {"initWorkflow", a.initWorkflow},
        {"initTrading", a.initTrading},
        {"initPythonBridge", a.initPythonBridge},
        {"initQuotePoller", a.initQuotePoller},
    }
    for _, step := range steps {
        if err := step.fn(ctx); err != nil {
            return fmt.Errorf("startup %s: %w", step.name, err)
        }
    }
    return nil
}
```

**Modified files:**
- `app_startup.go` — Extract methods; keep `ServiceStartup` as orchestrator
- `app.go` — Add new methods as needed

### 2a. Configurable Risk-Free Rate

```go
type MetricsConfig struct {
    RiskFreeRate float64 // annualized, e.g. 0.02 for 2%
    TradingDays  int     // e.g. 252
}

var DefaultMetricsConfig = MetricsConfig{
    RiskFreeRate: 0.02,
    TradingDays:  252,
}
```

The `ComputeMetrics` function now accepts `MetricsConfig`. The `Config` struct in each engine carries this. Default = 2% + 252 days (backward compatible).

**Modified files:**
- `internal/backtest/metrics.go` — Add `MetricsConfig`; update `ComputeMetrics` signature; update all callers
- `internal/portfolio/risk.go` — Same change
- `internal/backtest/runner.go` — Wire config through

### 2b. Stamp Duty Rounding

```go
func (e *CNEngine) stampDuty(tradeValue float64) float64 {
    raw := tradeValue * e.stampDutyRate
    // Round to nearest cent (0.01)
    return math.Round(raw*100) / 100
}
```

**Modified files:**
- `internal/backtest/engine_cn.go` — Update `stampDuty` method
- `internal/backtest/engine_hk.go` — Same for HK stamp duty + trade fee

### 2c. Wash Sale Basis Fix

```go
// Current (wrong):
lossAmt = t.Quantity * (t.Price - st[j].Price)  // sale vs repurchase

// Fix:
lossAmt = t.Quantity * (t.Price - originalCostBasis)  // sale vs original cost
```

The `WashSaleTracker` needs to store the original cost basis per lot, not just the repurchase price.

**Modified files:**
- `internal/trading/wash_sale.go` — Store original cost basis in `WashSaleLot`; update loss calculation
- `internal/trading/wash_sale_test.go` — Add test case for the corrected calculation

### 2d. US Engine Default 1 Share

```go
func (e *USEngine) processUSBuySignal(...) {
    qty := signal.Quantity
    if qty <= 0 {
        qty = 1  // was 100
    }
```

**Modified files:**
- `internal/backtest/engine_us.go:193` — Change `qty = 100` to `qty = 1`
- `internal/backtest/runner.go:147` — Change `qty = 100` to `qty = 1` (if this is a generic default; verify it only affects US path)

### 2e. Update golang.org/x/net

```bash
go get golang.org/x/net@v0.35.0
go mod tidy
```

**Modified files:**
- `go.mod` — Version bump
- `go.sum` — Auto-updated

### 3. Fix execQueue Package Variable

Move `execQueue` from package-level var to `App` field.

**Modified files:**
- `app_startup.go` — Move `var execQueue` → `a.execQueue`
- `app.go` — Add `execQueue *workflow.ExecutionQueue` field
- Any file referencing `execQueue` — Update to `a.execQueue`

## Acceptance Criteria

- [ ] 1000 rapid subscribe/unsubscribe cycles: 0 goroutine leak (verify with `runtime.NumGoroutine`)
- [ ] `processLoop` uses 0% CPU when queue is empty
- [ ] `ServiceStartup` is an orchestrator (<30 lines); each init method is <60 lines
- [ ] Sharpe/Sortino use configurable rf rate (default 2%, backward compatible)
- [ ] Stamp duty rounds to nearest cent
- [ ] Wash sale uses original cost basis (test assertion added)
- [ ] US engine buys minimum 1 share (not 100)
- [ ] `golang.org/x/net` updated to latest
- [ ] `execQueue` is a field on `App`, not a package var
- [ ] All existing tests pass

## Risks / Trade-offs

- **Shrinkflation**: The wash sale "fix" changes PnL for users who trade around wash sales. The effect is small for most retail traders but theoretically correct.
- **Stamp duty rounding**: In practice, brokerages round to the nearest cent. The float-based approach may differ from a specific broker's rounding by 0.01 CNY per trade — acceptable for backtesting.
- **ServiceStartup decomposition**: Existing tests that mock `ServiceStartup` will need updating. The new methods are public (`initConfig`, etc.) so they can be tested individually.
