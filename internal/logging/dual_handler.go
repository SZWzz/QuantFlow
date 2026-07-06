package logging

import (
	"context"
	"log/slog"
	"sync"
)

type dualHandler struct {
	inner slog.Handler
	rb    *RingBuffer
	mu    sync.Mutex
	attrs []slog.Attr
}

func newDualHandler(inner slog.Handler, rb *RingBuffer) *dualHandler {
	return &dualHandler{inner: inner, rb: rb}
}

func (h *dualHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

func (h *dualHandler) Handle(ctx context.Context, rec slog.Record) error {
	if !h.Enabled(ctx, rec.Level) {
		return nil
	}
	if err := h.inner.Handle(ctx, rec); err != nil {
		return err
	}

	attrs := make(map[string]any)
	h.mu.Lock()
	for _, a := range h.attrs {
		attrs[a.Key] = a.Value.Any()
	}
	h.mu.Unlock()
	rec.Attrs(func(a slog.Attr) bool {
		attrs[a.Key] = a.Value.Any()
		return true
	})

	entry := LogEntry{
		Time:    rec.Time,
		Level:   rec.Level.String(),
		Message: rec.Message,
		Attrs:   attrs,
	}
	h.rb.Push(entry)

	return nil
}

func (h *dualHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	h.mu.Lock()
	combined := make([]slog.Attr, len(h.attrs)+len(attrs))
	copy(combined, h.attrs)
	copy(combined[len(h.attrs):], attrs)
	h.mu.Unlock()

	return &dualHandler{
		inner: h.inner.WithAttrs(attrs),
		rb:    h.rb,
		attrs: combined,
	}
}

func (h *dualHandler) WithGroup(name string) slog.Handler {
	return &dualHandler{
		inner: h.inner.WithGroup(name),
		rb:    h.rb,
		attrs: h.attrs,
	}
}
