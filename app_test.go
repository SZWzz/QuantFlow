package main

import (
	"context"
	"log/slog"
	"quantflow/internal/config"
	"quantflow/internal/logging"
	"quantflow/internal/market"
	"testing"
)

// TestApp_RegisterMarketAdapters_AllWired verifies startup's adapter registration
// populates every data source so the fallback chains (mootdx first for CN) are real.
// Runs without a Python bridge: mootdx then has a nil DataClient and must report
// IsAvailable()==false (graceful degradation), while the others are still registered.
func TestApp_RegisterMarketAdapters_AllWired(t *testing.T) {
	a := &App{
		cfg:       config.DefaultConfig(),
		marketReg: market.NewAdapterRegistry(),
		bridge:    nil, // no Python sidecar → mootdx degrades
	}
	a.registerMarketAdapters()

	// Explicit expected roster: count is derived from the list so adding an
	// adapter forces an intentional update here (and the membership loop
	// below names exactly which one is missing).
	expected := []string{
		// CN chain
		"mootdx", "sina", "tushare", "eastmoney", "tencent", "baidu", "akshare",
		// HK / minute
		"akshare_hk_minute", "qos",
		// US
		"yahoo", "finnhub", "polygon",
		// CRYPTO
		"gateio", "okx", "binance", "binance_futures", "coingecko",
	}
	if got := a.marketReg.Count(); got != len(expected) {
		t.Errorf("registered adapter count = %d, want %d", got, len(expected))
	}
	for _, name := range expected {
		if a.marketReg.Get(name) == nil {
			t.Errorf("adapter %q not registered", name)
		}
	}

	// mootdx with a nil DataClient must be unavailable (no TDX probe, no panic).
	mootdx := a.marketReg.Get("mootdx")
	if mootdx == nil {
		t.Fatal("mootdx adapter not registered")
	}
	if mootdx.IsAvailable(context.Background()) {
		t.Error("mootdx should be unavailable when the Python bridge is absent (nil DataClient)")
	}
}

// TestApp_GetQuote_NoRegistryErrors guards the IPC guard: GetQuote must fail
// cleanly when the registry was never initialized.
func TestApp_GetQuote_NoRegistryErrors(t *testing.T) {
	a := &App{} // marketReg is nil
	_, _, err := a.GetQuote(context.Background(), "CN", "600519")
	if err == nil {
		t.Fatal("GetQuote should error when market registry is nil")
	}
}

// TestApp_FetchOHLCV_NoRegistryErrors guards the IPC guard for OHLCV.
func TestApp_FetchOHLCV_NoRegistryErrors(t *testing.T) {
	a := &App{} // marketReg is nil
	_, _, err := a.FetchOHLCV(context.Background(), "CN", "600519", "1D", "", 0, 0)
	if err == nil {
		t.Fatal("FetchOHLCV should error when market registry is nil")
	}
}

func TestGetLogs(t *testing.T) {
	app := &App{}
	logging.Setup("debug")
	slog.Info("test log for GetLogs")
	logs := app.GetLogs(0)
	if len(logs) == 0 {
		t.Fatal("expected at least 1 log entry")
	}
	lastID := logs[len(logs)-1].ID
	newLogs := app.GetLogs(int(lastID))
	if len(newLogs) != 0 {
		t.Fatalf("expected 0 new entries after last ID, got %d", len(newLogs))
	}
	slog.Info("second test log")
	newLogs = app.GetLogs(int(lastID))
	if len(newLogs) != 1 {
		t.Fatalf("expected 1 new entry, got %d", len(newLogs))
	}
}
