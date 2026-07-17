package logging

import (
	"fmt"
	"testing"
	"time"

	"quantflow/internal/ws"
)

func TestRingBufferPushAndLines(t *testing.T) {
	rb := NewRingBuffer(3)
	e1 := LogEntry{ID: 1, Time: time.Now(), Level: "info", Message: "msg1"}
	e2 := LogEntry{ID: 2, Time: time.Now(), Level: "warn", Message: "msg2"}
	e3 := LogEntry{ID: 3, Time: time.Now(), Level: "error", Message: "msg3"}

	rb.Push(e1)
	rb.Push(e2)
	rb.Push(e3)

	lines := rb.Lines(0, 10)
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}
}

func TestRingBufferOverflow(t *testing.T) {
	rb := NewRingBuffer(2)
	rb.Push(LogEntry{ID: 1, Message: "a"})
	rb.Push(LogEntry{ID: 2, Message: "b"})
	rb.Push(LogEntry{ID: 3, Message: "c"})

	lines := rb.Lines(0, 10)
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines after overflow, got %d", len(lines))
	}
	if lines[0].Message != "b" || lines[1].Message != "c" {
		t.Fatalf("expected oldest dropped, got %+v", lines)
	}
}

func TestRingBufferLinesAfterID(t *testing.T) {
	rb := NewRingBuffer(10)
	rb.Push(LogEntry{ID: 1, Message: "a"})
	rb.Push(LogEntry{ID: 2, Message: "b"})
	rb.Push(LogEntry{ID: 3, Message: "c"})

	lines := rb.Lines(1, 10)
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines after ID 1, got %d", len(lines))
	}
	if lines[0].Message != "b" || lines[1].Message != "c" {
		t.Fatalf("got %+v", lines)
	}
}

func TestRingBufferZeroCapacityDefaultsTo5000(t *testing.T) {
	rb := NewRingBuffer(0)
	if rb.max != 5000 {
		t.Fatalf("expected max=5000, got %d", rb.max)
	}
	rb.Push(LogEntry{ID: 1, Message: "a"})
	lines := rb.Lines(0, 10)
	if lines == nil {
		t.Fatal("expected non-nil slice")
	}
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}
}

func TestRingBufferEmptyReturnsEmptySlice(t *testing.T) {
	rb := NewRingBuffer(100)
	lines := rb.Lines(0, 10)
	if lines == nil {
		t.Fatal("expected non-nil slice, got nil")
	}
	if len(lines) != 0 {
		t.Fatal("expected empty slice")
	}
}

func TestRingBufferLastN(t *testing.T) {
	rb := NewRingBuffer(10)
	for i := 0; i < 10; i++ {
		rb.Push(LogEntry{Message: fmt.Sprintf("msg %d", i)})
	}

	last5 := rb.LastN(5)
	if len(last5) != 5 {
		t.Errorf("expected 5, got %d", len(last5))
	}
	if last5[0].Message != "msg 5" {
		t.Errorf("expected msg 5, got %s", last5[0].Message)
	}

	// Request more than available
	last20 := rb.LastN(20)
	if len(last20) != 10 {
		t.Errorf("expected 10, got %d", len(last20))
	}

	// Request 0
	last0 := rb.LastN(0)
	if len(last0) != 0 {
		t.Errorf("expected 0, got %d", len(last0))
	}
}

func TestRingBufferLinesLimit(t *testing.T) {
	rb := NewRingBuffer(100)
	for i := 1; i <= 20; i++ {
		rb.Push(LogEntry{ID: int64(i), Message: "msg"})
	}
	lines := rb.Lines(0, 5)
	if len(lines) != 5 {
		t.Fatalf("expected 5 lines, got %d", len(lines))
	}
}

func TestRingBufferSetHub(t *testing.T) {
	rb := NewRingBuffer(10)
	// Without hub, Push should not panic
	rb.Push(LogEntry{Message: "no hub"})

	// With hub, Push should not panic even if hub has no subscribers
	hub := ws.NewHub()
	go hub.Run()
	defer hub.Shutdown()
	rb.SetHub(hub)
	rb.Push(LogEntry{Message: "with hub"})

	lines := rb.Lines(0, 10)
	if len(lines) < 1 {
		t.Fatal("expected at least 1 line after pushes")
	}
}
