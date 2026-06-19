package ai

import (
	"encoding/json"
	"sync"
	"time"
)

// AgentEvent is emitted by the agent loop at each step.
// The frontend subscribes to these events via Wails IPC.
type AgentEvent struct {
	RunID     string      `json:"run_id"`
	Timestamp int64       `json:"ts"`
	Type      string      `json:"type"` // "step_start", "think", "tool_call", "tool_result", "finished", "error"
	Data      interface{} `json:"data"`
}

// EventEmitter manages agent event subscribers by run ID.
type EventEmitter struct {
	mu          sync.RWMutex
	subscribers map[string][]chan AgentEvent
}

// NewEventEmitter creates a new EventEmitter.
func NewEventEmitter() *EventEmitter {
	return &EventEmitter{
		subscribers: make(map[string][]chan AgentEvent),
	}
}

// Subscribe returns a channel that receives events for the given run ID.
// Channel is buffered to avoid blocking the emitter.
func (e *EventEmitter) Subscribe(runID string) <-chan AgentEvent {
	e.mu.Lock()
	defer e.mu.Unlock()
	ch := make(chan AgentEvent, 64)
	e.subscribers[runID] = append(e.subscribers[runID], ch)
	return ch
}

// Emit sends an event to all subscribers for the given run ID.
// Non-blocking: if a subscriber's buffer is full, the event is dropped.
func (e *EventEmitter) Emit(event AgentEvent) {
	if event.Timestamp == 0 {
		event.Timestamp = time.Now().UnixMilli()
	}
	e.mu.RLock()
	subs := e.subscribers[event.RunID]
	e.mu.RUnlock()

	for _, ch := range subs {
		select {
		case ch <- event:
		default:
			// Drop event for slow subscriber
		}
	}
}

// CloseRun removes all subscribers for a run ID and closes their channels.
func (e *EventEmitter) CloseRun(runID string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	for _, ch := range e.subscribers[runID] {
		close(ch)
	}
	delete(e.subscribers, runID)
}

// MarshalEvent serializes an agent event to JSON bytes for frontend delivery.
func MarshalEvent(event AgentEvent) ([]byte, error) {
	return json.Marshal(event)
}
