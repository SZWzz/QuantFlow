package logging

import (
	"testing"
	"time"
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
