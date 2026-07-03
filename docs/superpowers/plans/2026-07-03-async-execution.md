# Async Execution — Implementation Plan

## Task 1: Go execution queue (`internal/workflow/queue.go`)

**File**: `internal/workflow/queue.go` — new file

```go
package workflow

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type QueuedWorkflow struct {
	RunID       string
	Workflow    *Workflow
	Status      string // "queued", "running", "completed", "failed"
	Result      *ExecutionResult
	EnqueuedAt  time.Time
	StartedAt   time.Time
	FinishedAt  time.Time
	Error       string
	NodeResults map[string]*NodeResult
}

type ExecutionQueue struct {
	mu       sync.Mutex
	queue    []*QueuedWorkflow
	running  bool
	engine   *Engine
	notifyFn func(runID string, nodeID string, status string)
}

func NewExecutionQueue(engine *Engine, notifyFn func(string, string, string)) *ExecutionQueue {
	return &ExecutionQueue{engine: engine, notifyFn: notifyFn}
}

func (q *ExecutionQueue) Enqueue(wf *Workflow) (string, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	runID := fmt.Sprintf("run_%d", time.Now().UnixNano())
	q.queue = append(q.queue, &QueuedWorkflow{
		RunID: runID, Workflow: wf, Status: "queued", EnqueuedAt: time.Now(),
		NodeResults: make(map[string]*NodeResult),
	})
	if !q.running {
		q.running = true
		go q.processLoop()
	}
	return runID, nil
}

func (q *ExecutionQueue) processLoop() {
	for {
		q.mu.Lock()
		if len(q.queue) == 0 {
			q.running = false
			q.mu.Unlock()
			return
		}
		wf := q.queue[0]
		wf.Status = "running"
		wf.StartedAt = time.Now()
		q.mu.Unlock()

		result, err := q.engine.Execute(context.Background(), wf.Workflow)

		q.mu.Lock()
		wf.FinishedAt = time.Now()
		if err != nil {
			wf.Status = "failed"
			wf.Error = err.Error()
		} else {
			wf.Status = "completed"
			wf.Result = result
			for i := range result.NodeResults {
				nr := &result.NodeResults[i]
				wf.NodeResults[nr.NodeID] = nr
			}
		}
		q.queue = q.queue[1:]
		q.mu.Unlock()
	}
}

func (q *ExecutionQueue) GetStatus(runID string) *QueuedWorkflow {
	q.mu.Lock()
	defer q.mu.Unlock()
	for _, wf := range q.queue {
		if wf.RunID == runID {
			return wf
		}
	}
	return nil
}

func (q *ExecutionQueue) Cancel(runID string) {
	q.mu.Lock()
	defer q.mu.Unlock()
	for i, wf := range q.queue {
		if wf.RunID == runID {
			wf.Status = "failed"
			wf.Error = "cancelled"
			q.queue = append(q.queue[:i], q.queue[i+1:]...)
			return
		}
	}
}
```

## Task 2: app.go — QueueWorkflow + GetExecutionStatus

**File**: `app.go` — add to existing file

```go
var execQueue *workflow.ExecutionQueue

// in ServiceStartup, after engine init:
execQueue = workflow.NewExecutionQueue(a.engine, func(runID, nodeID, status string) {
	slog.Debug("node status update", "run_id", runID, "node_id", nodeID, "status", status)
})

func (a *App) QueueWorkflow(ctx context.Context, jsonDef string) (string, error) {
	var wf workflow.Workflow
	if err := json.Unmarshal([]byte(jsonDef), &wf); err != nil {
		return "", fmt.Errorf("parse json: %w", err)
	}
	return execQueue.Enqueue(&wf)
}

func (a *App) GetExecutionStatus(runID string) (*workflow.QueuedWorkflow, error) {
	qwf := execQueue.GetStatus(runID)
	if qwf == nil {
		return nil, fmt.Errorf("run %q not found", runID)
	}
	return qwf, nil
}

func (a *App) CancelExecution(runID string) error {
	execQueue.Cancel(runID)
	return nil
}
```

## Task 3: Frontend async execution flow

**File**: `frontend/src/stores/workflow.ts` — modify existing store

```ts
const asyncRunId = ref<string | null>(null)
const queuePosition = ref<number>(0)
const pollTimer = ref<number | null>(null)

async function startAsyncRun() {
	const wfJSON = toWorkflowJSON()
	resetExecution()
	executionStatus.value = 'running'
	const app = (window as any).go?.main?.App
	if (!app?.QueueWorkflow) {
		// fallback to sync
		const result = await RunWorkflow(JSON.stringify(wfJSON))
		applyResult(result)
		return
	}
	const runId = await app.QueueWorkflow(JSON.stringify(wfJSON))
	asyncRunId.value = runId
	startPolling(runId)
}

function startPolling(runId: string) {
	const app = (window as any).go?.main?.App
	if (!app?.GetExecutionStatus) return
	const poll = async () => {
		try {
			const status = await app.GetExecutionStatus(runId)
			if (!status) { stopPolling(); return }
			queuePosition.value = status.queue_position || 0
			// Update node statuses from status.node_results
			if (status.node_results) {
				for (const nr of status.node_results) {
					const existing = nodeStatuses.value.get(nr.node_id)
					if (existing?.status !== nr.status) {
						nodeStatuses.value.set(nr.node_id, nr)
						updateNodeVisual(nr.node_id, nr.status)
					}
				}
			}
			if (status.status === 'completed' || status.status === 'failed') {
				executionStatus.value = status.status
				asyncRunId.value = null
				stopPolling()
				return
			}
			pollTimer.value = window.setTimeout(poll, 200)
		} catch { stopPolling() }
	}
	poll()
}

function stopPolling() {
	if (pollTimer.value !== null) {
		clearTimeout(pollTimer.value)
		pollTimer.value = null
	}
}
```

## Commit

```bash
git add -A && git commit -m "[Workflow] Async execution queue with polling status (#spec 2026-07-03-async-execution)"
```
