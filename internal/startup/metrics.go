package startup

import (
	"quantflow/internal/logging"
	"time"
)

// Metrics holds startup timing data for diagnostics.
type Metrics struct {
	StartTime time.Time     `json:"start_time"`
	ReadyTime time.Time     `json:"ready_time"`
	TotalMs   int64         `json:"total_ms"`
	Phases    []PhaseMetric `json:"phases"`
}

// PhaseMetric measures a single startup phase.
type PhaseMetric struct {
	Name      string `json:"name"`
	ElapsedMs int64  `json:"elapsed_ms"`
}

var metrics = &Metrics{}

// Start marks the beginning of startup measurement.
func Start() {
	metrics.StartTime = time.Now()
	metrics.Phases = nil
}

// Track records timing for a named phase.
func Track(name string, fn func()) {
	start := time.Now()
	fn()
	elapsed := time.Since(start).Milliseconds()
	metrics.Phases = append(metrics.Phases, PhaseMetric{Name: name, ElapsedMs: elapsed})
	logging.Ring.Push(logging.LogEntry{
		Time:    time.Now(),
		Level:   "info",
		Message: "startup phase: " + name,
		Attrs:   map[string]any{"elapsed_ms": elapsed},
	})
}

// Done marks startup as complete and returns the metrics.
func Done() *Metrics {
	metrics.ReadyTime = time.Now()
	metrics.TotalMs = metrics.ReadyTime.Sub(metrics.StartTime).Milliseconds()
	return metrics
}

// GetMetrics returns the current startup metrics (nil if Start() not called).
func GetMetrics() *Metrics {
	return metrics
}
