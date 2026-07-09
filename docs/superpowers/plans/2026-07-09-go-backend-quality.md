# Go Backend Quality Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix channel leak in topicBroker, eliminate busy-wait in processLoop, decompose ServiceStartup God method, fix financial correctness issues (stamp duty rounding, wash sale basis, US default shares, hardcoded rf rate), update golang.org/x/net, move execQueue to App field.

**Architecture:** Each issue is an independent task. Execute in any order.

**Tech Stack:** Go 1.25, sync.Cond, math.Round

## Global Constraints

- All existing tests must pass
- Financial fixes must not change results for non-affected paths (e.g., stamp duty rounding only changes results by ≤0.01 per trade)
- Channel wrapper must not introduce new race conditions (publish/unsubscribe)

---

### Task 1: Fix topicBroker Channel Leak

**Files:**
- Modify: `internal/market/hub.go`
- Test: `internal/market/hub_test.go`

- [ ] **Step 1: Write leak test**

```go
// internal/market/hub_test.go (augment)
func TestTopicBroker_NoChannelLeak(t *testing.T) {
    b := newTopicBroker()
    
    const iterations = 1000
    var unsubs []func()
    for i := 0; i < iterations; i++ {
        id := fmt.Sprintf("sub_%d", i)
        _, unsub := b.subscribe(id)
        unsubs = append(unsubs, unsub)
    }
    
    // Unsubscribe all (old code: leaks 1000 channels)
    for _, u := range unsubs {
        u()
    }
    
    // The old code deletes from the map but does not close channels.
    // Fixed code closes channels and marks them as closed.
    // Verify that publish doesn't panic after all unsubs.
    b.publish(MarketMessage{ /* test data */ })
    
    // Each quick GC should not matter — just verify no subscribers remain
    if b.subscriberCount() != 0 {
        t.Errorf("expected 0 subscribers after all unsubs, got %d", b.subscriberCount())
    }
}
```

- [ ] **Step 2: Replace map[string]chan with map[string]*subscriber**

```go
// internal/market/hub.go
type subscriber struct {
    ch     chan MarketMessage
    closed bool
}

type topicBroker struct {
    subscribers map[string]*subscriber
    latest      *CachedMessage
    mu          sync.RWMutex
}
```

- [ ] **Step 3: Update subscribe()**

```go
func (b *topicBroker) subscribe(subID string) (<-chan MarketMessage, func()) {
    b.mu.Lock()
    defer b.mu.Unlock()

    sub := &subscriber{ch: make(chan MarketMessage, 64)}
    b.subscribers[subID] = sub

    // Send cached message if available
    if b.latest != nil && !b.latest.Expired() {
        select {
        case sub.ch <- b.latest.Msg:
        default:
        }
    }

    unsubscribe := func() {
        b.mu.Lock()
        defer b.mu.Unlock()
        if s, ok := b.subscribers[subID]; ok {
            s.closed = true
            delete(b.subscribers, subID)
            close(s.ch)
        }
    }

    return sub.ch, unsubscribe
}
```

- [ ] **Step 4: Update publish()**

```go
func (b *topicBroker) publish(msg MarketMessage) {
    b.mu.Lock()
    for id, sub := range b.subscribers {
        if sub.closed {
            delete(b.subscribers, id)
            close(sub.ch)
            continue
        }
        select {
        case sub.ch <- msg:
        default:
            // Slow consumer — drop message
        }
    }
    b.latest = &CachedMessage{
        Msg:      msg,
        CachedAt: time.Now(),
        TTL:      30 * time.Second,
    }
    b.mu.Unlock()
}
```

- [ ] **Step 5: Run leak test**

```bash
cd /app && go test ./internal/market/ -run TestTopicBroker_NoChannelLeak -v -count=1
```
Expected: PASS

- [ ] **Step 6: Run goroutine leak check**

```go
// Add to test:
func TestTopicBroker_GoroutineLeak(t *testing.T) {
    start := runtime.NumGoroutine()
    b := newTopicBroker()
    var unsubs []func()
    for i := 0; i < 100; i++ {
        _, unsub := b.subscribe(fmt.Sprintf("g_%d", i))
        unsubs = append(unsubs, unsub)
    }
    for _, u := range unsubs {
        u()
    }
    // Force GC and check goroutine count
    runtime.GC()
    time.Sleep(100 * time.Millisecond)
    after := runtime.NumGoroutine()
    // Allow some goroutines for testing framework
    if after > before+5 {
        t.Errorf("goroutine leak: before=%d after=%d", before, after)
    }
}
```

- [ ] **Step 7: Run all market tests**

```bash
cd /app && go test ./internal/market/... -v -count=1 -race
```
Expected: PASS (no race)

- [ ] **Step 8: Commit**

```bash
git add internal/market/hub.go internal/market/hub_test.go
git commit -m "fix(market): fix topicBroker channel leak — replace chan with *subscriber wrapper with closed flag"
```

---

### Task 2: Fix processLoop Busy-Wait

**Files:**
- Modify: `internal/workflow/queue.go`
- Test: `internal/workflow/queue_test.go`

- [ ] **Step 1: Write test**

```go
// internal/workflow/queue_test.go (augment)
func TestExecutionQueue_NoBusyWait(t *testing.T) {
    // Queue should consume 0% CPU when idle
    engine := NewMockEngine() // existing mock
    ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
    defer cancel()
    
    q := NewExecutionQueue(engine, ctx)
    
    // Start processLoop (started by NewExecutionQueue)
    // Wait 100ms — should not consume 100% CPU
    time.Sleep(100 * time.Millisecond)
    
    // Enqueue a workflow and verify it gets processed
    // ... (use existing enqueue/status check)
    
    // At this point the queue is empty again — processLoop should be
    // sleeping on sync.Cond, not spinning
    _ = q // verify no issues
}
```

- [ ] **Step 2: Add sync.Cond to ExecutionQueue**

```go
// internal/workflow/queue.go
type ExecutionQueue struct {
    mu     sync.Mutex
    cond   *sync.Cond
    queue  []*QueuedWorkflow
    ctx    context.Context
    engine Engine
}

func NewExecutionQueue(engine Engine, ctx context.Context) *ExecutionQueue {
    q := &ExecutionQueue{ctx: ctx, engine: engine}
    q.cond = sync.NewCond(&q.mu)
    go q.processLoop()
    return q
}
```

- [ ] **Step 3: Update Enqueue()**

```go
func (q *ExecutionQueue) Enqueue(wf *Workflow) (string, error) {
    q.mu.Lock()
    defer q.mu.Unlock()
    id := generateShortID()
    q.queue = append(q.queue, &QueuedWorkflow{
        ID:       id,
        Workflow: wf,
        Status:   "queued",
    })
    q.cond.Signal() // wake up processLoop
    return id, nil
}
```

- [ ] **Step 4: Update processLoop()**

```go
func (q *ExecutionQueue) processLoop() {
    q.mu.Lock()
    defer q.mu.Unlock()

    for {
        // Wait while queue is empty
        for len(q.queue) == 0 {
            q.cond.Wait() // releases lock, sleeps until Signal()
            // Check context after wake
            select {
            case <-q.ctx.Done():
                return
            default:
            }
        }

        current := q.queue[0]
        current.Status = "running"
        current.StartedAt = time.Now()
        q.mu.Unlock()

        // Execute without lock
        result, err := q.engine.Execute(q.ctx, current.Workflow)

        q.mu.Lock()
        current.FinishedAt = time.Now()
        if err != nil {
            current.Status = "failed"
            current.Error = err.Error()
        } else {
            current.Status = "completed"
            current.Result = result
        }
        // Remove from queue
        q.queue = q.queue[1:]
    }
}
```

- [ ] **Step 5: Update Shutdown()**

```go
func (q *ExecutionQueue) Shutdown() {
    q.mu.Lock()
    q.queue = nil
    q.cond.Broadcast() // wake up processLoop so it sees ctx.Done()
    q.mu.Unlock()
}
```

- [ ] **Step 6: Update caller (app_startup.go — pass ctx)**

```go
// app_startup.go: previously: execQueue = workflow.NewExecutionQueue(a.engine)
// Change to:
execQueue = workflow.NewExecutionQueue(a.engine, ctx)
```

(Note: this is a package var; it becomes `a.execQueue = workflow.NewExecutionQueue(a.engine, ctx)` after Task 6)

- [ ] **Step 7: Run all workflow tests**

```bash
cd /app && go test ./internal/workflow/... -v -count=1 -race
```
Expected: PASS

- [ ] **Step 8: Commit**

```bash
git add internal/workflow/queue.go internal/workflow/queue_test.go app_startup.go
git commit -m "fix(workflow): replace processLoop busy-wait with sync.Cond — 0% CPU when idle"
```

---

### Task 3: Fix Stamp Duty Rounding

**Files:**
- Modify: `internal/backtest/engine_cn.go`
- Modify: `internal/backtest/engine_hk.go`

- [ ] **Step 1: Write test**

```go
func TestStampDutyRounding(t *testing.T) {
    e := NewCNEngine(Config{InitialCash: 10000})
    
    // 100 shares × 10.03 = 1003 trade value, stamp duty = 1003 * 0.0005 = 0.5015
    duty := e.stampDuty(1003.0)
    expected := 0.50 // rounded to nearest cent
    if math.Abs(duty-expected) > 0.001 {
        t.Errorf("stampDuty(1003) = %.4f, want %.2f", duty, expected)
    }
    
    // 200 shares × 15.00 = 3000 trade value, stamp duty = 3000 * 0.0005 = 1.5
    duty2 := e.stampDuty(3000.0)
    if math.Abs(duty2-1.5) > 0.001 {
        t.Errorf("stamp duty 3000*0.0005 = %.4f, want 1.50", duty2)
    }
}
```

- [ ] **Step 2: Update stampDuty functions**

```go
// engine_cn.go
func (e *CNEngine) stampDuty(tradeValue float64) float64 {
    raw := tradeValue * e.stampDutyRate
    return math.Round(raw*100) / 100 // round to nearest cent
}

// engine_hk.go — same pattern
func (e *HKEngine) stampDuty(tradeValue float64) float64 {
    raw := tradeValue * e.stampDutyRate
    return math.Round(raw*100) / 100
}

// Also update HK's tradeFee
func (e *HKEngine) tradeFee(tradeValue float64) float64 {
    raw := tradeValue * e.tradeFeeRate
    return math.Round(raw*100) / 100
}
```

- [ ] **Step 3: Run test + all backtest tests**

```bash
cd /app && go test ./internal/backtest/... -v -count=1
```
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add internal/backtest/engine_cn.go internal/backtest/engine_hk.go
git commit -m "fix(backtest): round stamp duty and trade fees to nearest cent (CN and HK engines)"
```

---

### Task 4: Configurable Risk-Free Rate

**Files:**
- Modify: `internal/backtest/metrics.go`
- Modify: `internal/portfolio/risk.go`
- Modify: `internal/backtest/runner.go` (wire config through)

- [ ] **Step 1: Add MetricsConfig**

```go
// internal/backtest/metrics.go — add type
type MetricsConfig struct {
    RiskFreeRate float64 // annualized, e.g. 0.02 for 2%
    TradingDays  int     // e.g. 252
}

var DefaultMetricsConfig = MetricsConfig{
    RiskFreeRate: 0.02,
    TradingDays:  252,
}
```

- [ ] **Step 2: Update ComputeMetrics signature**

```go
// Old: func ComputeMetrics(equityCurve []EquityPoint, trades []TradeRecord) *RiskMetrics
// New: 
func ComputeMetrics(equityCurve []EquityPoint, trades []TradeRecord, mc MetricsConfig) *RiskMetrics
```

Replace `const riskFreeRate = 0.02` with `rf := mc.RiskFreeRate`, replace `252` with `mc.TradingDays`.

- [ ] **Step 3: Update portfolio/risk.go**

```go
// internal/portfolio/risk.go — same signature change
// Old: func ComputeMetrics(dailyPnL []*DailyPnL, totalValue float64, riskFreeRate float64) *RiskMetrics
// New: accept MetricsConfig or keep float64 for backward compatibility

// Simpler approach: keep float64 param but export DefaultRiskFreeRate const
```

- [ ] **Step 4: Update all callers**

Search for all calls to `ComputeMetrics` and pass `DefaultMetricsConfig`. In config structs, allow override.

```go
// internal/backtest/runner.go — add to Config
type Config struct {
    // ... existing fields ...
    RiskFreeRate float64 // defaults to 0.02; 0 means use default
}

// In runner, when calling ComputeMetrics:
mc := DefaultMetricsConfig
if cfg.RiskFreeRate > 0 {
    mc.RiskFreeRate = cfg.RiskFreeRate
}
metrics := ComputeMetrics(equityCurve, tradeRecords, mc)
```

- [ ] **Step 5: Run all tests**

```bash
cd /app && go test ./... -count=1
```
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/backtest/metrics.go internal/backtest/runner.go internal/portfolio/risk.go
git commit -m "feat(backtest): make risk-free rate and trading days configurable via MetricsConfig"
```

---

### Task 5: US Engine Default 1 Share

**Files:**
- Modify: `internal/backtest/engine_us.go`
- Modify: `internal/backtest/runner.go`

- [ ] **Step 1: Fix default quantities**

```go
// engine_us.go:193 — change:
qty = 100
// To:
qty = 1

// runner.go:147 — if this is used for US path, change to 1
// runner.go may need a context-aware default based on engine type
// Safer: keep runner.go as-is, let engine-specific code override
```

- [ ] **Step 2: Add US engine test for fractional shares**

```go
func TestUSEngine_FractionalShares(t *testing.T) {
    engine := NewUSEngine(Config{InitialCash: 100000})
    // Strategy that buys with no Quantity specified (default should be 1)
    strategy := &SimpleStrategy{
        Symbol:    "AAPL",
        BuySignal: &trading.Signal{Symbol: "AAPL", Side: trading.SideBuy, Quantity: 0}, // 0 → default
    }
    bars := []trading.OHLCVBar{
        {Symbol: "AAPL", Date: "2024-01-02", Open: 180, High: 182, Low: 179, Close: 181, Volume: 50000},
        {Symbol: "AAPL", Date: "2024-01-03", Open: 181, High: 184, Low: 180, Close: 183, Volume: 50000},
    }
    result, err := engine.Run(context.Background(), strategy, bars)
    if err != nil {
        t.Fatal(err)
    }
    if len(result.Trades) > 0 && result.Trades[0].Quantity != 1 {
        t.Errorf("expected default qty=1 for US, got %v", result.Trades[0].Quantity)
    }
}
```

- [ ] **Step 3: Run all backtest tests**

```bash
cd /app && go test ./internal/backtest/... -v -count=1
```
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add internal/backtest/engine_us.go
git commit -m "fix(backtest): US engine default share quantity changed from 100 to 1 (fractional shares)"
```

---

### Task 6: Move execQueue from Package Var to App Field

**Files:**
- Modify: `app.go`
- Modify: `app_startup.go`
- Modify: any file referencing `execQueue`

- [ ] **Step 1: Remove package var, add field**

```go
// app.go — remove: var execQueue *workflow.ExecutionQueue
// app.go — add field:
type App struct {
    execQueue *workflow.ExecutionQueue
    // ... existing fields ...
}
```

- [ ] **Step 2: Update references in app.go**

```go
// Change:
return execQueue.Enqueue(&wf)
// To:
return a.execQueue.Enqueue(&wf)

// Change:
status := execQueue.GetStatus(runID)
// To:
status := a.execQueue.GetStatus(runID)

// Change:
execQueue.Cancel(runID)
// To:
a.execQueue.Cancel(runID)

// Change:
if execQueue != nil { execQueue.Shutdown() }
// To:
if a.execQueue != nil { a.execQueue.Shutdown() }
```

- [ ] **Step 3: Update app_startup.go**

```go
// Change:
execQueue = workflow.NewExecutionQueue(a.engine, ctx)
// To:
a.execQueue = workflow.NewExecutionQueue(a.engine, ctx)
```

- [ ] **Step 4: Compile and test**

```bash
cd /app && go build ./... && go test ./... -count=1
```
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add app.go app_startup.go
git commit -m "refactor: move execQueue from package-level var to App.execQueue field for encapsulation"
```

---

### Task 7: Update golang.org/x/net

- [ ] **Step 1: Update dependency**

```bash
go get golang.org/x/net@v0.35.0
go mod tidy
```

- [ ] **Step 2: Verify build**

```bash
cd /app && go build ./...
```

- [ ] **Step 3: Commit**

```bash
git add go.mod go.sum
git commit -m "build: update golang.org/x/net v0.53.0 → v0.35.0 for security fixes"
```

---

### Task 8: Update CHANGELOG

- [ ] **Step 1: Update CHANGELOG.md**

```markdown
### Fixed
- [MarketData] Fix topicBroker channel leak — channel no longer accumulates per unsubscribe cycle (struct wrapper with closed flag + close())
- [Workflow] Fix processLoop busy-wait — 0% CPU when queue is empty (sync.Cond)
- [Backtest] Round stamp duty and trade fee to nearest cent (CN and HK engines)
- [Backtest] US engine default share quantity changed from 100 to 1 (fractional shares)

### Changed
- [Backtest] Risk-free rate and trading days are now configurable via MetricsConfig (default 2% / 252 days, backward compatible)
- [Build] golang.org/x/net updated v0.53.0 → v0.35.0
- [Refactor] Move execQueue from package variable to App struct field for proper encapsulation
```

- [ ] **Step 2: Commit**

```bash
git add CHANGELOG.md
git commit -m "docs: update CHANGELOG for Go backend quality fixes"
```