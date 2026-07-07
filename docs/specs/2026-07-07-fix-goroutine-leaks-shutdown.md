# Fix Goroutine Leaks and Graceful Shutdown

## Motivation

Four goroutine leak / shutdown issues:

1. **WebSocket Hub** (`internal/ws/hub.go:30-51`) — `Run()` starts an infinite `for` loop with no shutdown signal. Leaks all client handlers on app exit.

2. **ExecutionQueue** (`internal/workflow/queue.go:132`) — Uses `context.Background()`, so queued workflows cannot be cancelled on shutdown.

3. **SentimentEngine.BatchAnalyze** (`internal/research/sentiment_engine.go:104-127`) — Spawns N goroutines without limit (N = symbol count).

4. **Off-hours cache saves** (`internal/market/registry.go:198`, `app.go`) — Spawns `go func() { cache.Save() }()` per quote fetch, creating goroutine buildup.

## Design

### 1. WebSocket Hub graceful shutdown

**File**: `internal/ws/hub.go`

Add a `done` channel to `Hub` struct. `Run()` selects on `done` in the loop body:

```go
type Hub struct {
    clients    map[*Client]bool
    register   chan *Client
    unregister chan *Client
    done       chan struct{}
}

func (h *Hub) Run() {
    for {
        select {
        case client := <-h.register: ...
        case client := <-h.unregister: ...
        case <-h.done: return
        }
    }
}

func (h *Hub) Shutdown() { close(h.done) }
```

**File**: `app.go:279` — Call `wsHub.Shutdown()` in `ServiceShutdown()`.

### 2. ExecutionQueue cancellable context

**File**: `internal/workflow/queue.go`

Store a cancellable context in `ExecutionQueue`:

```go
type ExecutionQueue struct {
    ctx    context.Context
    cancel context.CancelFunc
    // ...
}

func NewExecutionQueue(engine *Engine) *ExecutionQueue {
    ctx, cancel := context.WithCancel(context.Background())
    return &ExecutionQueue{ctx: ctx, cancel: cancel, ...}
}

func (q *ExecutionQueue) Shutdown() { q.cancel() }
```

Change `processLoop` line 132 to use `q.ctx` instead of `context.Background()`.

### 3. SentimentEngine bounded concurrency

**File**: `internal/research/sentiment_engine.go`

Replace unbounded goroutine spawn with bounded worker pool using `errgroup`:

```go
func (e *SentimentEngine) BatchAnalyze(ctx context.Context, symbols []string, ...) ([]SentimentResult, error) {
    g, ctx := errgroup.WithContext(ctx)
    g.SetLimit(10)  // max 10 concurrent
    results := make([]SentimentResult, len(symbols))
    for i, sym := range symbols {
        i, sym := i, sym
        g.Go(func() error {
            result, err := e.AnalyzeSentiment(ctx, sym, ...)
            if err != nil { return err }
            results[i] = result
            return nil
        })
    }
    return results, g.Wait()
}
```

### 4. Debounced cache saves

**File**: `internal/market/registry.go`

Replace per-call goroutine with a debounce timer:

```go
var saveMu sync.Mutex
var saveTimer *time.Timer

func (r *Registry) debouncedSaveQuotes() {
    saveMu.Lock()
    defer saveMu.Unlock()
    if saveTimer != nil { saveTimer.Stop() }
    saveTimer = time.AfterFunc(5*time.Second, func() {
        r.saveLastQuotes()
    })
}
```

Call `debouncedSaveQuotes()` instead of `go r.saveLastQuotes()`.

### Modified files

| File | Change |
|------|--------|
| `internal/ws/hub.go` | Add `done` channel, `Shutdown()` |
| `internal/ws/hub_test.go` | Add shutdown test |
| `internal/ws/client.go` | Propagate hub done to client read/write loops |
| `app.go` | Call `wsHub.Shutdown()` + `queue.Shutdown()` in `ServiceShutdown()` |
| `internal/workflow/queue.go` | Add cancellable context, `Shutdown()` |
| `internal/workflow/queue_test.go` | Add shutdown cancellation test |
| `internal/research/sentiment_engine.go` | Replace unbounded goroutines with errgroup worker pool |
| `internal/research/sentiment_engine_test.go` | Add concurrency limit test |
| `internal/market/registry.go` | Replace `go r.saveLastQuotes()` with debounced version |

### API changes

- `Hub.Shutdown()` — new method
- `ExecutionQueue.Shutdown()` — new method
- No changes to exported function signatures

## Acceptance Criteria

- [ ] `go test -race ./internal/ws/...` passes
- [ ] `go test -race ./internal/workflow/...` passes
- [ ] SentimentEngine with 1000 symbols uses ≤10 goroutines (verify with `GOMAXPROCS` debug)
- [ ] Cache save goroutines don't accumulate under rapid quote polling
- [ ] App shutdown completes within 1s (currently may hang forever)

## Risks / Trade-offs

- **Debounce delay**: Cache saves wait up to 5s after last quote. Acceptable — data is already in memory.
- **SentimentEngine concurrency limit**: 10 is a safe default. Could be configurable if needed.
- **Backwards compatible**: All changes are additive (new `Shutdown()` methods), existing code unaffected.
