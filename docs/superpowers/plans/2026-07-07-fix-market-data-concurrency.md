# Plan: Fix Market Data Concurrency

## Spec
`docs/specs/2026-07-07-fix-market-data-concurrency.md`

## Tasks

### 1. Fix TOCTOU race in MarketDataHub.Publish

**File**: `internal/market/hub.go`

Change the double-checked locking: replace `RLock`/`RUnlock` with `Lock`/`Unlock` and remove the `!ok` second check.

### 2. Fix OffHoursCache reference leak

**File**: `internal/market/offhours.go`

Change `Get` signature from returning `any` to accepting `dest any` and deep-copy via JSON round-trip. Update all callers in the codebase.

### 3. Wire EastMoney rate limiter

**File**: `internal/market/adapters/eastmoney_rate_limit.go`
**File**: `internal/market/adapters/eastmoney.go`

Create a token-bucket limiter and call `Wait()` before each HTTP request.

### 4. Add concurrent publish test

**File**: `internal/market/hub_test.go`

### 5. Verify

```bash
cd /Volumes/etx/coding/quantflow/app && go test -race ./internal/market/...
```
