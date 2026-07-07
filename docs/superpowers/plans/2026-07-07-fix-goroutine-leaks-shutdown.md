# Plan: Fix Goroutine Leaks and Graceful Shutdown

## Spec
`docs/specs/2026-07-07-fix-goroutine-leaks-shutdown.md`

## Tasks

### 1. WebSocket Hub graceful shutdown

**File**: `internal/ws/hub.go`
- Add `done chan struct{}` to `Hub` struct
- Add `select` with `<-h.done` case in `Run()` loop
- Add `Shutdown()` method

**File**: `internal/ws/client.go`
- Propagate hub shutdown to client read/write pumploops

**File**: `app.go`
- Call `wsHub.Shutdown()` in `ServiceShutdown()`

### 2. ExecutionQueue cancellable context

**File**: `internal/workflow/queue.go`
- Add `ctx context.Context` and `cancel context.CancelFunc` fields
- Initialize in `NewExecutionQueue` with `context.WithCancel()`
- Use `q.ctx` instead of `context.Background()` in `processLoop`
- Add `Shutdown()` method

### 3. SentimentEngine bounded concurrency

**File**: `internal/research/sentiment_engine.go`
- Replace unbounded goroutine spawn with `errgroup` + `SetLimit(10)`

### 4. Debounced cache saves

**File**: `internal/market/registry.go`
- Add `debouncedSaveQuotes()` using `time.AfterFunc`
- Replace `go r.saveLastQuotes()` with `r.debouncedSaveQuotes()`

### 5. Verify

```bash
cd /Volumes/etx/coding/quantflow/app && go test -race ./internal/ws/... ./internal/workflow/... ./internal/research/... ./internal/market/...
```
