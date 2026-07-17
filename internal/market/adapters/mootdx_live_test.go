//go:build integration
// +build integration

package adapters

import (
	"context"
	"testing"
	"time"

	"quantflow/internal/python"
)

// TestMootdxAdapter_WithBridge tests mootdx with a real Python sidecar connection.
// Requires: python sidecar running on localhost:50051 (cd python && python -m src.server).
func TestMootdxAdapter_WithBridge(t *testing.T) {
	// Connect to Python sidecar
	opts := python.DefaultOptions()
	opts.DialTimeout = 5 * time.Second
	bridge, err := python.NewPythonBridge(opts)
	if err != nil {
		t.Skipf("Python sidecar not available: %v", err)
	}
	defer bridge.Close()

	// Create DataClient (what mootdx uses internally)
	dataClient := python.NewDataClient(bridge)

	// Create mootdx adapter with real DataClient
	adapter := NewMootdxAdapter(dataClient)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Check availability
	available := adapter.IsAvailable(ctx)
	t.Logf("mootdx IsAvailable=%v", available)

	// Fetch real-time quote for 600519 (贵州茅台).
	// Minute data is only available during trading hours; outside trading
	// hours we fall back to computing quote summary from recent OHLCV.
	snap, err := adapter.FetchQuote(ctx, "600519")
	if err != nil {
		t.Logf("mootdx FetchQuote(600519) note (non-trading hours?): %v", err)
	} else if snap != nil {
		t.Logf("mootdx FetchQuote(600519) OK: last=%.2f open=%.2f high=%.2f low=%.2f change=%.2f%% vol=%.0f",
			snap.Last, snap.Open, snap.High, snap.Low, snap.ChangePct, snap.Volume)
	}

	// Fetch OHLCV for the last 5 trading days
	now := time.Now()
	end := now.Unix()
	start := now.AddDate(0, 0, -10).Unix()
	bars, err := adapter.FetchOHLCV(ctx, "600519", "1D", "", start, end)
	if err != nil {
		t.Fatalf("mootdx FetchOHLCV(600519) error: %v", err)
	}
	t.Logf("mootdx FetchOHLCV(600519) OK: %d bars", len(bars))
	for i, b := range bars {
		if i >= 5 {
			t.Logf("  ... and %d more", len(bars)-5)
			break
		}
		t.Logf("  bar[%d]: date=%s o=%.2f h=%.2f l=%.2f c=%.2f v=%.0f",
			i, b.Date, b.Open, b.High, b.Low, b.Close, b.Volume)
	}
}
