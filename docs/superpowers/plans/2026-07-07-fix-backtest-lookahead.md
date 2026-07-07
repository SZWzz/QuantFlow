# Plan: Fix Backtest Look-Ahead Bias

## Spec
`docs/specs/2026-07-07-fix-backtest-lookahead.md`

## Tasks

### 1. Change SignalFunc type signature

**File**: `internal/backtest/types.go`

Change:
```go
type SignalFunc func(bar OHLCV, portfolio Portfolio) Signal
```
To:
```go
type SignalFunc func(openPrice float64, prevBar *OHLCV, portfolio Portfolio) Signal
```

### 2. Update engine_cn.go

**File**: `internal/backtest/engine_cn.go`

At the signal call site, pass `bar.Open` and a pointer to the previous bar (tracked in the loop). Replace:
```go
signal := strategy.SignalFunc(bar, portfolio)
```
With:
```go
signal := strategy.SignalFunc(bar.Open, prevBar, portfolio)
```
Add `prevBar` tracking in the loop:
```go
var prevBar *OHLCV
for _, bar := range bars {
    signal := strategy.SignalFunc(bar.Open, prevBar, portfolio)
    // ... execution logic ...
    prevBar = &bar
}
```

### 3. Update engine_us.go

Same pattern as engine_cn.go but for US market rules.

### 4. Update strategy implementations

Search for all `SignalFunc` implementations in `internal/strategy/` and `internal/workflow/nodes/`. Update each to match new signature.

### 5. Update tests

Update all test callbacks that implement `SignalFunc`.

### 6. Verify

```bash
cd /Volumes/etx/coding/quantflow/app && go build ./...
cd /Volumes/etx/coding/quantflow/app && go test ./... -count=1
```
