package logging

import (
	"encoding/json"
	"sync"
	"time"

	"quantflow/internal/ws"
)

type LogEntry struct {
	ID      int64          `json:"id"`
	Time    time.Time      `json:"time"`
	Level   string         `json:"level"`
	Message string         `json:"message"`
	Attrs   map[string]any `json:"attrs,omitempty"`
}

type RingBuffer struct {
	mu     sync.Mutex
	buffer []LogEntry
	nextID int64
	head   int
	count  int
	max    int
	hub    *ws.Hub // WebSocket hub for broadcasting new entries
}

func NewRingBuffer(capacity int) *RingBuffer {
	if capacity <= 0 {
		capacity = 5000
	}
	return &RingBuffer{
		buffer: make([]LogEntry, capacity),
		max:    capacity,
		nextID: 1,
	}
}

// SetHub wires a WebSocket hub for real-time log broadcast.
func (rb *RingBuffer) SetHub(hub *ws.Hub) {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	rb.hub = hub
}

func (rb *RingBuffer) Push(entry LogEntry) {
	rb.mu.Lock()
	entry.ID = rb.nextID
	rb.nextID++
	if rb.count < rb.max {
		rb.buffer[rb.count] = entry
		rb.count++
	} else {
		rb.buffer[rb.head] = entry
		rb.head = (rb.head + 1) % rb.max
	}
	hub := rb.hub
	rb.mu.Unlock()

	// Broadcast to WebSocket subscribers outside the lock
	if hub != nil {
		// Marshal the entry as JSON for the broadcast (ignore marshal errors)
		if data, err := json.Marshal(entry); err == nil {
			hub.Broadcast("system:notification", json.RawMessage(data))
		}
	}
}

func (rb *RingBuffer) Lines(afterID int64, limit int) []LogEntry {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	if rb.count == 0 || limit <= 0 {
		return []LogEntry{}
	}

	var result []LogEntry
	for i := 0; i < rb.count; i++ {
		idx := (rb.head + i) % rb.max
		entry := rb.buffer[idx]
		if entry.ID > afterID {
			result = append(result, entry)
			if len(result) >= limit {
				break
			}
		}
	}
	return result
}

// LastN returns the last n entries from the buffer, most recent first.
func (rb *RingBuffer) LastN(n int) []LogEntry {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	if rb.count == 0 || n <= 0 {
		return []LogEntry{}
	}

	if n > rb.count {
		n = rb.count
	}

	result := make([]LogEntry, n)
	for i := 0; i < n; i++ {
		idx := (rb.head + rb.count - n + i) % rb.max
		result[i] = rb.buffer[idx]
	}
	return result
}
