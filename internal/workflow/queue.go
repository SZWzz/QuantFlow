package workflow

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// QueuedWorkflow represents a workflow waiting for or in execution.
type QueuedWorkflow struct {
	RunID       string                  `json:"run_id"`
	WorkflowID  string                  `json:"workflow_id"`
	Workflow    *Workflow               `json:"-"`
	Status      string                  `json:"status"` // "queued" | "running" | "completed" | "failed"
	Result      *ExecutionResult        `json:"result,omitempty"`
	QueuePos    int                     `json:"queue_position"`
	EnqueuedAt  time.Time               `json:"enqueued_at"`
	StartedAt   time.Time               `json:"started_at,omitempty"`
	FinishedAt  time.Time               `json:"finished_at,omitempty"`
	Error       string                  `json:"error,omitempty"`
	NodeResults map[string]*NodeResult  `json:"node_results,omitempty"`
}

// ExecutionQueue manages asynchronous workflow execution.
// Workflows are queued and executed sequentially by a background goroutine.
// Callers poll GetStatus to track progress.
type ExecutionQueue struct {
	mu      sync.Mutex
	queue   []*QueuedWorkflow
	running bool
	engine  *Engine
}

// NewExecutionQueue creates an ExecutionQueue.
func NewExecutionQueue(engine *Engine) *ExecutionQueue {
	return &ExecutionQueue{engine: engine}
}

// Enqueue adds a workflow to the execution queue and returns a run ID.
// If the worker is not running, it starts automatically.
func (q *ExecutionQueue) Enqueue(wf *Workflow) (string, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	runID := fmt.Sprintf("run_%d", time.Now().UnixNano())
	qw := &QueuedWorkflow{
		RunID:       runID,
		WorkflowID:  wf.ID,
		Workflow:    wf,
		Status:      "queued",
		EnqueuedAt:  time.Now(),
		NodeResults: make(map[string]*NodeResult),
	}
	q.queue = append(q.queue, qw)

	if !q.running {
		q.running = true
		go q.processLoop()
	}
	return runID, nil
}

// GetStatus returns the current state of a queued/running/completed workflow.
// Returns nil if the run ID is not found (already cleaned up).
func (q *ExecutionQueue) GetStatus(runID string) *QueuedWorkflow {
	q.mu.Lock()
	defer q.mu.Unlock()
	for _, qw := range q.queue {
		if qw.RunID == runID {
			// Update queue position
			for i, w := range q.queue {
				if w.RunID == runID {
					qw.QueuePos = i
					break
				}
			}
			return qw
		}
	}
	return nil
}

// Cancel marks a queued workflow as cancelled.
func (q *ExecutionQueue) Cancel(runID string) {
	q.mu.Lock()
	defer q.mu.Unlock()
	for _, qw := range q.queue {
		if qw.RunID == runID && qw.Status == "queued" {
			qw.Status = "failed"
			qw.Error = "cancelled"
			qw.FinishedAt = time.Now()
		}
	}
	// Remove cancelled items from the queue
	var kept []*QueuedWorkflow
	for _, qw := range q.queue {
		if qw.Status != "failed" || qw.RunID != runID {
			kept = append(kept, qw)
		}
	}
	q.queue = kept
}

// processLoop runs in a goroutine, executing queued workflows sequentially.
func (q *ExecutionQueue) processLoop() {
	for {
		q.mu.Lock()
		if len(q.queue) == 0 {
			q.running = false
			q.mu.Unlock()
			return
		}
		// Find the first queued item
		var current *QueuedWorkflow
		for _, qw := range q.queue {
			if qw.Status == "queued" {
				current = qw
				break
			}
		}
		if current == nil {
			q.running = false
			q.mu.Unlock()
			return
		}
		current.Status = "running"
		current.StartedAt = time.Now()
		q.mu.Unlock()

		// Execute
		result, err := q.engine.Execute(context.Background(), current.Workflow)

		q.mu.Lock()
		current.FinishedAt = time.Now()
		if err != nil {
			current.Status = "failed"
			current.Error = err.Error()
		} else {
			current.Status = "completed"
			current.Result = result
			if result != nil {
				for i := range result.NodeResults {
					nr := &result.NodeResults[i]
					current.NodeResults[nr.NodeID] = nr
				}
			}
		}
		q.mu.Unlock()
	}
}
