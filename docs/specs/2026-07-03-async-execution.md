# Asynchronous Workflow Execution with Polling-Based Status

## Motivation
Currently `RunWorkflow` is a synchronous request-response call: frontend POSTs JSON → Go executes entire DAG → returns result. For workflows with long-running ML/training nodes, this blocks the UI. ComfyUI uses an async queue system with WebSocket streaming; this spec adopts a simpler polling-based approach suitable for v1.

## Design

### Backend
1. **`QueueWorkflow(jsonDef string) (runID string, error)`** — enqueues workflow, returns immediately with runID
2. **`GetExecutionStatus(runID string) (*QueuedWorkflow, error)`** — returns full status snapshot with per-node results
3. **`CancelExecution(runID string) error`** — cancels a queued/running execution
4. **`ExecutionQueue`** — goroutine-based sequential queue in `internal/workflow/queue.go`
5. **`ExecutionQueue` worker loop** — dequeues one workflow at a time, runs it via `engine.Execute()`, collects per-node results

### Frontend
1. Call `QueueWorkflow` instead of `RunWorkflow` → get `runId`
2. Poll `GetExecutionStatus(runId)` every 200ms via `setTimeout`
3. Update `nodeStatuses` map and node visuals from poll results
4. Fallback to sync `RunWorkflow` if `QueueWorkflow` not available

### Data flow
```
Frontend                      Go Backend
   │                              │
   ├── QueueWorkflow(json) ──────►│
   │◄───── runID ────────────────┤
   │                              │
   │  [every 200ms]              │
   ├── GetExecutionStatus(runID)─►│
   │◄── {status, nodeResults[]} ─┤
   │                              │
   │  Update node visual          │
   │  from nodeResults            │
   │                              │
   │  When status=completed|failed│
   │  stop polling                │
```

### Files modified/created
- `internal/workflow/queue.go` — **new file**: `ExecutionQueue` struct with enqueue/process/cancel
- `internal/workflow/engine.go` — add per-node callback support in `Execute`
- `app.go` — `QueueWorkflow`, `GetExecutionStatus`, `CancelExecution`
- `frontend/src/stores/workflow.ts` — async execution flow (queue + poll)
- `frontend/src/workflow/canvas/WorkflowCanvas.vue` — real-time status updates from poll

### API changes
**New Go exported functions:**
```go
func (a *App) QueueWorkflow(ctx context.Context, jsonDef string) (string, error)
func (a *App) GetExecutionStatus(runID string) (*workflow.QueuedWorkflow, error)
func (a *App) CancelExecution(runID string) error
```

**New Go types (in `internal/workflow/`):**
```go
type QueuedWorkflow struct {
    RunID       string
    Status      string // "queued", "running", "completed", "failed"
    NodeResults map[string]*NodeResult
    ...
}
type ExecutionQueue struct { ... }
```

**Pinia store changes:**
- `asyncRunId`, `queuePosition`, `pollTimer` refs
- `startAsyncRun()`, `startPolling()`, `stopPolling()` actions

## Acceptance Criteria
- [ ] `QueueWorkflow` returns immediately with runID
- [ ] Frontend polls `GetExecutionStatus` every 200ms and updates node statuses
- [ ] Multiple workflows can be queued (sequential execution for now)
- [ ] Interrupt/kill queued execution via `CancelExecution`
- [ ] Fallback to sync execution if async not available
- [ ] Changelog updated

## Risks / Trade-offs
- **Polling vs WebSocket**: Polling adds 200ms latency to status updates and extra HTTP overhead. WebSocket would be more efficient but adds Wails v3 WS dependency. Polling is simpler and works with Wails IPC directly.
- **Sequential queue**: Only one workflow executes at a time. Parallel execution requires a more complex scheduler with resource limits (deferred to v2).
- **No persistence**: Queue is in-memory only. Restarting the app loses queued workflows. Persistence deferred to v2.
