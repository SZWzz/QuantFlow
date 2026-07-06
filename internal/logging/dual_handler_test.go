package logging

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"
)

func TestDualHandlerWritesToInner(t *testing.T) {
	var buf bytes.Buffer
	inner := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	rb := NewRingBuffer(100)
	dh := newDualHandler(inner, rb)

	ctx := context.Background()
	rec := slog.NewRecord(time.Now(), slog.LevelInfo, "hello dual", 0)
	if err := dh.Handle(ctx, rec); err != nil {
		t.Fatalf("Handle: %v", err)
	}

	if !strings.Contains(buf.String(), "hello dual") {
		t.Errorf("expected inner handler to receive message, got %q", buf.String())
	}
}

func TestDualHandlerWritesToRingBuffer(t *testing.T) {
	var buf bytes.Buffer
	inner := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	rb := NewRingBuffer(100)
	dh := newDualHandler(inner, rb)

	ctx := context.Background()
	rec := slog.NewRecord(time.Now(), slog.LevelInfo, "ring test", 0)
	if err := dh.Handle(ctx, rec); err != nil {
		t.Fatalf("Handle: %v", err)
	}

	lines := rb.Lines(0, 10)
	if len(lines) != 1 {
		t.Fatalf("expected 1 line in ring buffer, got %d", len(lines))
	}
	if lines[0].Message != "ring test" {
		t.Errorf("expected 'ring test', got %q", lines[0].Message)
	}
}

func TestDualHandlerLevelFilter(t *testing.T) {
	var buf bytes.Buffer
	inner := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelError})
	rb := NewRingBuffer(100)
	dh := newDualHandler(inner, rb)

	ctx := context.Background()
	infoRec := slog.NewRecord(time.Now(), slog.LevelInfo, "should skip", 0)
	_ = dh.Handle(ctx, infoRec)

	errRec := slog.NewRecord(time.Now(), slog.LevelError, "should capture", 0)
	_ = dh.Handle(ctx, errRec)

	lines := rb.Lines(0, 10)
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}
	if lines[0].Message != "should capture" {
		t.Errorf("expected 'should capture', got %q", lines[0].Message)
	}
}

func TestDualHandlerWithAttrs(t *testing.T) {
	var buf bytes.Buffer
	inner := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	rb := NewRingBuffer(100)
	dh := newDualHandler(inner, rb)

	dh2 := dh.WithAttrs([]slog.Attr{slog.String("key1", "val1")})
	ctx := context.Background()
	rec := slog.NewRecord(time.Now(), slog.LevelInfo, "with attrs", 0)
	if err := dh2.Handle(ctx, rec); err != nil {
		t.Fatalf("Handle: %v", err)
	}

	lines := rb.Lines(0, 10)
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}
	if lines[0].Attrs["key1"] != "val1" {
		t.Errorf("expected attrs key1=val1, got %v", lines[0].Attrs)
	}
	if !strings.Contains(buf.String(), "key1=val1") {
		t.Errorf("expected attrs in inner handler output, got %q", buf.String())
	}
}
