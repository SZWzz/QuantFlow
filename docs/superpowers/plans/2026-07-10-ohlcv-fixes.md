# Plan: OHLCV Fixes

## Task 1: Extend index OHLCV date range

**File**: `app_market.go:585`

Change `now.AddDate(0, 0, -90).Unix()` to `now.AddDate(-10, 0, 0).Unix()`

**Verification**: `grep` shows the old value is gone.

## Task 2: Wire OHLCVCache into App struct and startup

**File**: `app.go:50` — add `ohlcvCache *market.OHLCVCache` field after `minuteCache`

**File**: `app_startup.go:65-70` — init `OHLCVCache` alongside `MinuteCache`:
```go
oc, err := market.NewOHLCVCache(a.db)
if err != nil {
    slog.Error("failed to init ohlcv cache", "err", err)
} else {
    a.ohlcvCache = oc
}
```

**Verification**: Build passes.

## Task 3: Use cache in FetchOHLCVWithFallback

**File**: `internal/market/registry.go:280-317`

Add cache check at the start of `FetchOHLCVWithFallback`. Pass the cache as an optional parameter or use a package-level reference. Simpler: expose `SetOHLCVCache` on registry and check before adapter loop.

```go
func (r *AdapterRegistry) SetOHLCVCache(c *OHLCVCache) {
    r.ohlcvCache = c
}

// At start of FetchOHLCVWithFallback:
if r.ohlcvCache != nil {
    if cached, err := r.ohlcvCache.Get(symbol, interval, start, end); err == nil && len(cached) > 0 {
        slog.Debug("ohlcv cache hit", "symbol", symbol, "interval", interval, "bars", len(cached))
        return cached, "cache", nil
    }
}
```

And after successful fetch:
```go
if r.ohlcvCache != nil {
    if err := r.ohlcvCache.Set(symbol, interval, bars); err != nil {
        slog.Warn("ohlcv cache set failed", "symbol", symbol, "error", err)
    }
}
```

## Task 4: Bypass mootdx for index OHLCV in GetMarketOverview

**File**: `app_market.go:610-616`

Instead of `FetchOHLCVWithFallback`, try tencent directly for index symbols (same pattern as Sina for quotes):

```go
var ohlcv []market.OHLCVBar
if cached, ok := indexOhlcvCache.get(idx.code); ok {
    ohlcv = cached
} else {
    // Try tencent directly for index OHLCV (bypass mootdx which doesn't support indices)
    tencent := a.marketReg.Get("tencent")
    if tencent != nil {
        bars, err := tencent.FetchOHLCV(ctx, idx.code, "1D", "", start, end)
        if err == nil && len(bars) > 0 {
            ohlcv = bars
        }
    }
    if len(ohlcv) == 0 {
        bars, _, err2 := a.marketReg.FetchOHLCVWithFallback(ctx, marketName, idx.code, "1D", "", start, end)
        if err2 == nil && len(bars) > 0 {
            ohlcv = bars
        }
    }
    if len(ohlcv) > 0 {
        indexOhlcvCache.set(idx.code, ohlcv)
    }
}
```

**Verification**: Build passes.
