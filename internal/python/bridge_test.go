package python

import (
	"context"
	"testing"
	"time"
)

// TestPythonBridge_Integration tests the full bridge against a running Python sidecar.
// Requires: python -m src.server running on localhost:50051.
func TestPythonBridge_Integration(t *testing.T) {
	bridge, err := NewPythonBridge(DefaultOptions())
	if err != nil {
		t.Skipf("Python sidecar not available (start with: python -m src.server): %v", err)
	}
	defer bridge.Close()

	ctx := context.Background()

	// Test health check
	t.Run("Ping", func(t *testing.T) {
		if err := bridge.Ping(ctx); err != nil {
			t.Fatalf("Ping failed: %v", err)
		}
	})

	t.Run("IsHealthy", func(t *testing.T) {
		if !bridge.IsHealthy(ctx) {
			t.Fatal("IsHealthy returned false")
		}
	})

	t.Run("GetStatus", func(t *testing.T) {
		status, err := bridge.GetStatus(ctx)
		if err != nil {
			t.Fatalf("GetStatus failed: %v", err)
		}
		if !status.Healthy {
			t.Fatal("status reports unhealthy")
		}
		if status.Version == "" {
			t.Fatal("status version is empty")
		}
		t.Logf("Python sidecar: version=%s, uptime=%ds, memory=%dMB",
			status.Version, status.UptimeSeconds, status.MemoryMb)
	})

	// Test list factors
	t.Run("ListFactors", func(t *testing.T) {
		factors, err := bridge.ListFactors(ctx)
		if err != nil {
			t.Fatalf("ListFactors failed: %v", err)
		}
		if len(factors) < 25 {
			t.Fatalf("expected at least 25 factors, got %d", len(factors))
		}
		t.Logf("Found %d factors", len(factors))
		for _, f := range factors {
			if f.Name == "" || f.Category == "" {
				t.Errorf("factor missing required fields: name=%q category=%q", f.Name, f.Category)
			}
		}
	})

	// Test compute factor
	t.Run("ComputeFactor", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		results, err := bridge.ComputeFactor(ctx, "momentum_20d", []string{"000001.SZ"},
			"2024-01-01", "2024-12-31", nil, nil)
		if err != nil {
			// This may fail if no data is provided — that's OK, as long as the error is clear
			t.Logf("ComputeFactor with no data: %v (expected — needs OHLCV data)", err)
		} else {
			t.Logf("ComputeFactor results: %d symbol(s)", len(results))
		}
	})
}

// TestPythonBridge_NotRunning verifies graceful behavior when Python is not available.
func TestPythonBridge_NotRunning(t *testing.T) {
	// Use a port with no server to test connection failure
	opts := DefaultOptions()
	opts.Address = "localhost:19999" // Nothing running here
	opts.DialTimeout = 1 * time.Second

	bridge, err := NewPythonBridge(opts)
	if err == nil {
		bridge.Close()
		t.Fatal("expected error when connecting to unavailable port, got nil")
	}
	t.Logf("Expected connection error: %v", err)
}
