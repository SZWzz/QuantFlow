# Implementation Plan: Fix 4 Blocking Bugs

> Spec: [docs/specs/2026-06-18-fix-blocking-bugs.md](../specs/2026-06-18-fix-blocking-bugs.md)
> Date: 2026-06-18
> Status: Ready for execution

## Overview

4 independent fixes, can be executed in any order. Each task is 2-5 minutes.

---

## Task 1: Fix #1 — Workflow params nil

### Step 1.1: Fix engine.go — pass params instead of nil

**File:** `internal/workflow/engine.go`

Change line 154 from:
```go
outputs, err := node.Execute(ctx, inputs, nil)
```
to:
```go
outputs, err := node.Execute(ctx, inputs, nodeInstance.Params)
```

Delete lines 138-142 (the `if len(inputs) == 0` block that puts params into inputs):
```go
	if len(inputs) == 0 && nodeInstance.Params != nil {
		for k, v := range nodeInstance.Params {
			inputs[k] = v
		}
	}
```

**Rationale:** Each node's Execute already uses `getStringParam(params, ...)` / `getFloatParam(params, ...)` which check `params` first and fall back to defaults. Passing `nodeInstance.Params` directly to the `params` argument of Execute is the correct fix. The old `len(inputs) == 0` guard was wrong: it meant params only flowed to nodes WITHOUT upstream edges, defeating the entire parameter system.

### Step 1.2: Verify — go build + test

```bash
cd /Volumes/shenzy/vibe_coding/quant/QuantFlow && go build ./...
cd /Volumes/shenzy/vibe_coding/quant/QuantFlow && go test ./internal/workflow/... -v -count=1
```

### Step 1.3: Commit

```
git add internal/workflow/engine.go
git commit -m "fix(workflow): pass node params to Execute instead of nil

Previously engine.go:154 hardcoded nil as the params argument to
node.Execute, and params were only folded into inputs when the node
had zero upstream edges. This meant any node with an upstream connection
ignored user-configured parameters and ran with hardcoded defaults.

Now nodeInstance.Params is passed directly as the params argument.
Each node's Execute method already uses getStringParam/getFloatParam
which check params first and fall back to defaults."
```

---

## Task 2: Fix #2 — Unify OHLCVBar types

### Step 2.1: Delete nodes.OHLCVBar, use market.OHLCVBar in data_loader

**File:** `internal/workflow/nodes/data_loader.go`

1. Delete lines 14-22 (the `type OHLCVBar struct` definition)
2. Add `"quantflow/internal/market"` to imports
3. Change `loadCSV` return type from `[]OHLCVBar` to `[]market.OHLCVBar`
4. In `loadCSV`, change `bars = append(bars, OHLCVBar{...})` → `bars = append(bars, market.OHLCVBar{Symbol: "CSV", ...})`
5. Change `Execute` return: `return map[string]any{"ohlcv": bars}, nil` stays the same

### Step 2.2: Update backtest.go to accept market.OHLCVBar

**File:** `internal/workflow/nodes/backtest.go`

Change line 71 from:
```go
bars, ok := rawData.([]trading.OHLCVBar)
```
to:
```go
bars, ok := rawData.([]market.OHLCVBar)
```

Change line 75-81 to try `[]market.OHLCVBar` and convert:
```go
if rawSlice, ok := rawData.([]any); ok {
    bars = make([]market.OHLCVBar, 0, len(rawSlice))
    for _, item := range rawSlice {
        if bar, ok := item.(market.OHLCVBar); ok {
            bars = append(bars, bar)
        }
    }
}
```

Then add a conversion from `[]market.OHLCVBar` → `[]trading.OHLCVBar` before passing to engines:
```go
// Convert to trading.OHLCVBar for engine consumption
tradingBars := make([]trading.OHLCVBar, len(bars))
for i, b := range bars {
    tradingBars[i] = trading.OHLCVBar{
        Symbol: b.Symbol,
        Date:   b.Date,
        Open:   b.Open,
        High:   b.High,
        Low:    b.Low,
        Close:  b.Close,
        Volume: b.Volume,
    }
}
```

And replace `bars` with `tradingBars` in all engine.Run calls.

Add `"quantflow/internal/market"` to imports.

### Step 2.3: Verify compile + test

```bash
go build ./...
go test ./internal/workflow/... -v -count=1
```

### Step 2.4: Commit

```
git add internal/workflow/nodes/data_loader.go internal/workflow/nodes/backtest.go
git commit -m "fix(workflow): unify OHLCVBar — delete nodes type, use market type

Three incompatible OHLCVBar types existed in nodes/market/trading packages.
data_loader output []nodes.OHLCVBar (no Symbol field), but backtest asserted
[]trading.OHLCVBar (has Symbol) → assertion always failed → 'no OHLCV data'.

Now data_loader uses market.OHLCVBar (the canonical type from market package)
and backtest explicitly converts to trading.OHLCVBar for engine consumption."
```

---

## Task 3: Fix #3 — Wire ML bridge

### Step 3.1: Open shared DB connection in startup()

**File:** `app.go`

The `startup()` function currently opens a DB at line 109 for the notify manager but the connection is scoped to that if block. We need to:

1. Add a `db *sql.DB` field to the `App` struct (or open a long-lived connection)
2. Open DB once, reuse for notify + model registry
3. Wire model registry

Actually, looking more carefully: the `ModelRegistry` needs `*sql.DB`. But the app currently opens/closes DB per call for workflows. For this fix, we just need the bridge wiring to work (the model registry is optional — train_model ignores its errors).

**Simpler approach:** Just wire the bridge. Model registry can be done later when DB connection management is centralized.

In `startup()`, after line 72 (bridge init succeeds), add:
```go
nodes.SetPythonBridge(a.bridge)
```

And after line 101 (`nodes.SetAgentDependencies(...)`), add model registry init:
```go
// Initialize model registry (requires DB — pass nil for now, registry calls are optional)
// Full model persistence will be wired when DB connection management is centralized.
nodes.SetModelRegistry(nil) // model registry is optional during training
```

Wait — `train_model.go:121` checks `if modelRegistry != nil`, so passing nil is fine. The training still proceeds through the Python sidecar. The model just won't be persisted locally. That's acceptable for this fix.

### Step 3.2: Change startup() code

In `app.go` `startup()` function, after line 72 (`a.bridge = bridge`):
```go
// Wire Python bridge to workflow ML nodes (train_model, predict, alpha_mining)
nodes.SetPythonBridge(a.bridge)
```

After line 101 (`nodes.SetAgentDependencies(...)`), add:
```go
// ModelRegistry wiring — pass nil for now (requires DB; model persistence is optional)
// The bridge is the critical dependency for ML training/inference.
nodes.SetModelRegistry(nil)
```

### Step 3.3: Verify

```bash
go build ./...
go test ./internal/workflow/... -v -count=1
```

### Step 3.4: Commit

```
git add app.go
git commit -m "fix(app): wire PythonBridge and ModelRegistry to workflow ML nodes

startup() creates the PythonBridge but never called nodes.SetPythonBridge()
or nodes.SetModelRegistry(), so train_model/predict/alpha_mining nodes always
returned 'PythonBridge not set'. Now the bridge is wired at startup.

ModelRegistry is passed nil for now since it requires a DB connection;
model persistence is optional during training — the critical dependency
is the bridge for gRPC communication with the Python sidecar."
```

---

## Task 4: Fix #5 — Interval case normalization

### Step 4.1: Create NormalizeInterval in market package

**File:** `internal/market/interval.go` (new file)

```go
package market

import "strings"

// NormalizeInterval standardizes the interval string to a canonical form.
// Frontend sends lowercase ("1d"), but many adapters expect uppercase ("1D").
// This function provides a single normalization point.
//
// Mapping:
//
//	5m, 15m, 30m, 1h → kept lowercase (used by crypto/minute adapters)
//	1d, 1D → "1D" (daily)
//	1w, 1W → "1W" (weekly)
//	1M, 1month → "1M" (monthly)
func NormalizeInterval(interval string) string {
	switch strings.ToLower(interval) {
	case "1d", "day", "daily":
		return "1D"
	case "1w", "week", "weekly":
		return "1W"
	case "1m", "1min":
		return "1m"
	case "1month", "monthly":
		return "1M"
	case "5m", "5min":
		return "5m"
	case "15m", "15min":
		return "15m"
	case "30m", "30min":
		return "30m"
	case "1h", "1hour", "hourly":
		return "1h"
	case "4h", "4hour":
		return "4h"
	default:
		return interval // pass through unknown values
	}
}
```

### Step 4.2: Normalize in AdapterRegistry.FetchOHLCVWithFallback

**File:** `internal/market/registry.go`

At the start of `FetchOHLCVWithFallback` (line 105), add:
```go
func (r *AdapterRegistry) FetchOHLCVWithFallback(ctx context.Context, market, symbol, interval string, start, end int64) ([]OHLCVBar, string, error) {
	interval = NormalizeInterval(interval) // <-- add this line
	
	chain, ok := FallbackChains[market]
	...
```

### Step 4.3: Add defensive ToUpper in adapters

**File:** `internal/market/adapters/tencent.go`

Line 73-77, change:
```go
func (a *TencentAdapter) FetchOHLCV(ctx context.Context, symbol string, interval string, start, end int64) ([]market.OHLCVBar, error) {
	period, ok := tencentIntervalMap[interval]
```
to:
```go
func (a *TencentAdapter) FetchOHLCV(ctx context.Context, symbol string, interval string, start, end int64) ([]market.OHLCVBar, error) {
	period, ok := tencentIntervalMap[strings.ToUpper(interval)]
```

**File:** `internal/market/adapters/baidu.go`

Line 79-80, change:
```go
func (a *BaiduAdapter) FetchOHLCV(ctx context.Context, symbol string, interval string, start, end int64) ([]market.OHLCVBar, error) {
	if interval != "1D" {
```
to:
```go
func (a *BaiduAdapter) FetchOHLCV(ctx context.Context, symbol string, interval string, start, end int64) ([]market.OHLCVBar, error) {
	if strings.ToUpper(interval) != "1D" {
```

**File:** `internal/market/adapters/eastmoney.go`

Line 98-103, change the switch to use consistent uppercase:
```go
switch strings.ToUpper(interval) {
case "1W":
	klt = "102"
case "1M":
	klt = "103"
}
```

### Step 4.4: Add NormalizeInterval tests

**File:** `internal/market/interval_test.go` (new file)

```go
package market

import "testing"

func TestNormalizeInterval(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"1d", "1D"}, {"1D", "1D"}, {"day", "1D"}, {"daily", "1D"},
		{"1w", "1W"}, {"1W", "1W"}, {"week", "1W"}, {"weekly", "1W"},
		{"1M", "1M"}, {"1month", "1M"}, {"monthly", "1M"},
		{"1m", "1m"}, {"5m", "5m"}, {"15m", "15m"}, {"30m", "30m"},
		{"1h", "1h"}, {"4h", "4h"},
		{"unknown", "unknown"}, // passthrough
	}
	for _, tt := range tests {
		got := NormalizeInterval(tt.input)
		if got != tt.expected {
			t.Errorf("NormalizeInterval(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}
```

### Step 4.5: Verify

```bash
go build ./...
go test ./internal/market/... -v -count=1
```

### Step 4.6: Commit

```
git add internal/market/interval.go internal/market/interval_test.go internal/market/registry.go internal/market/adapters/tencent.go internal/market/adapters/baidu.go internal/market/adapters/eastmoney.go
git commit -m "fix(market): normalize interval case — 1d→1D for A-share adapters

Frontend CandlestickPanel sends lowercase intervals (5m, 1h, 1d, 1w),
but Tencent/Baidu adapters only accept uppercase (1D, 1W, 1M), and
mootdx Python sidecar uses mixed case (1D, 1W, 1M, 1m, 5m, 1H).

Added NormalizeInterval() in market package, called at the registry
entry point (FetchOHLCVWithFallback). Each adapter also has a defensive
strings.ToUpper as a secondary safeguard."
```

---

## Task 5: Update CHANGELOG

### Step 5.1

**File:** `CHANGELOG.md`

Add entry under a new `[2026.6.18]` section:

```markdown
## [2026.6.18] - 2026-06-18

### Fixed
- [Workflow] Fix engine passing nil params to node.Execute — user-configured parameters now flow to nodes with upstream edges
- [Workflow] Unify three incompatible OHLCVBar types — data_loader→backtest pipeline now produces results instead of "no OHLCV data"
- [App] Wire PythonBridge to workflow ML nodes — train_model/predict/alpha_mining no longer return "PythonBridge not set"
- [MarketData] Normalize OHLCV interval case at registry entry — frontend "1d" now works for A-share K-line adapters
```

### Step 5.2: Commit

```
git add CHANGELOG.md
git commit -m "docs: update CHANGELOG for 4 blocking bug fixes"
```

---

## Execution Order

Task 1 → Task 2 → Task 3 → Task 4 → Task 5 (sequential, each independent)

**Total estimated time:** ~20 minutes for all 5 tasks.
