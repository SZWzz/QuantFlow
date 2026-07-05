# 实施计划：P0 Crash Fixes

参考：`docs/specs/2026-07-05-p0-crash-fixes.md`

## Task 1: 修复 Double RUnlock（3 文件）

### satellite_service.go

**位置**: `internal/research/satellite_service.go:60-67`

当前代码：
```go
s.mu.RLock()
if entry, ok := s.cache[cacheKey]; ok && time.Now().Before(entry.expiresAt) {
    defer s.mu.RUnlock()
    if snapshots, ok := entry.data.([]adapters.RegionSnapshot); ok {
        return snapshots, nil
    }
}
s.mu.RUnlock()
```

改为：
```go
s.mu.RLock()
if entry, ok := s.cache[cacheKey]; ok && time.Now().Before(entry.expiresAt) {
    if snapshots, ok := entry.data.([]adapters.RegionSnapshot); ok {
        s.mu.RUnlock()
        return snapshots, nil
    }
    s.mu.RUnlock()
    return nil, fmt.Errorf("cache entry %q has unexpected type %T", cacheKey, entry.data)
}
s.mu.RUnlock()
```

### govdata_service.go

**位置**: `internal/research/govdata_service.go:124-130`

同样模式，同样修复：
```go
s.mu.RLock()
if entry, ok := s.cache[cacheKey]; ok && time.Now().Before(entry.expiresAt) {
    if entries, ok := entry.data.([]adapters.SignalEntry); ok {
        s.mu.RUnlock()
        return entries, nil
    }
    s.mu.RUnlock()
    return nil, fmt.Errorf("cache entry %q has unexpected type %T", cacheKey, entry.data)
}
s.mu.RUnlock()
```

### prediction_market_service.go

**位置**: `internal/research/prediction_market_service.go:88-97`

同样模式，同样修复。

---

## Task 2: 添加 RequestCtx 工具

**新建文件**: 现有合适位置？检查 `internal/market/request.go` 或 `internal/market/retry.go`。

在 `internal/market/retry.go` 末尾加：

```go
// RequestCtx returns a context with a 10-second timeout for market data requests.
// Callers must call cancel() to release resources.
func RequestCtx() (context.Context, context.CancelFunc) {
    return context.WithTimeout(context.Background(), 10*time.Second)
}
```

---

## Task 3: 替换 Wails 方法中的 context.Background()

需要替换的文件和方法（全部改为 `ctx, cancel := market.RequestCtx(); defer cancel()`）：

**app.go**:
- `GetCapitalData` (line ~1056)
- `GetNews` (line ~1277)
- `SearchResearch` (line ~1040)
- `GetAnnouncements` (line ~1081)
- `GetDragonTiger` (line ~1099)
- `GetDailyDragonTiger` (line ~1118)

**app_market.go**:
- `GetFundFlow` (line ~363)
- `GetNorthboundFlow` (line ~391)
- `GetMarketOverview`（检查）
- `GetEarningsCalendar` (line ~866)
- `GetShortInterest` (line ~851)

**app_research.go**:
- `GetSentiment` (line ~21)
- `GetPredictionEventDetail` (line ~198)
- `GetSatelliteDetail` (line ~294)
- 其他多处

**注意**: Python sidecar 调用（如 `FetchData`）不应使用 10s 超时 — 它们已有自己的重试逻辑。区分方法：检查 context 是否传入 bridge/fetchData。

---

## Task 4: 添加 goroutine panic recovery

**app.go**:
```
app.go:105     — engine.Run goroutine
app.go:1105    — dragonTigerCache.Save()
app.go:1154    — industryRanksCache.Save()
app_market.go:377  — fundFlowCache.Save() (×2)
app_market.go:760  — depthCache.Save() (×2)
app_research.go:389, 401  — abnormalStocksCache.Save() (×2)
```

全部加：
```go
go func() {
    defer func() {
        if r := recover(); r != nil {
            slog.Error("goroutine panicked", "recover", r, "stack", string(debug.Stack()))
        }
    }()
    // ...
}()
```

需在 `app.go` import 中加 `"runtime/debug"`。

**internal/python/bridge.go:157**:
```go
go func() {
    defer func() {
        if r := recover(); r != nil {
            slog.Error("bridge goroutine panicked", "recover", r)
        }
    }()
    done <- cmd.Wait()
}()
```

**internal/python/ml_client.go:91**:
```go
go func() {
    defer func() {
        if r := recover(); r != nil {
            slog.Error("ml client goroutine panicked", "recover", r)
        }
    }()
    // ...
}()
```

---

## Task 5: 验证

```bash
go vet ./...
go test ./internal/research/ -count=1
go test ./internal/market/ -count=1
go build ./...
```
