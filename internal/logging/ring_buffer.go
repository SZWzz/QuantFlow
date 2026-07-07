package logging

import (
	"sync"
	"time"
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

func (rb *RingBuffer) Push(entry LogEntry) {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	entry.ID = rb.nextID
	rb.nextID++
	if rb.count < rb.max {
		rb.buffer[rb.count] = entry
		rb.count++
	} else {
		rb.buffer[rb.head] = entry
		rb.head = (rb.head + 1) % rb.max
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
