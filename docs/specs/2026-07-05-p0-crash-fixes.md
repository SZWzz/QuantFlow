# P0 Crash Fixes — Double RUnlock, Context Leaks, Goroutine Recovery

## Motivation

审计发现三类可导致生产环境崩溃或资源泄漏的 bug：

1. **Double RUnlock panic**（3 处）— 缓存命中后类型断言失败不 return，执行到外部 `RUnlock` 导致 panic
2. **context.Background() 无超时**（20+ 方法）— adapter HTTP 调用挂起 → goroutine 永久泄漏
3. **goroutine 无 panic recovery**（8/9 处）— 任一个 `go func()` 内 panic 直接带崩整个进程

## Design

### 1. Double RUnlock 修复

**文件**: `internal/research/satellite_service.go`, `govdata_service.go`, `prediction_market_service.go`

共同模式：
```go
s.mu.RLock()
if entry, ok := s.cache[cacheKey]; ok && time.Now().Before(entry.expiresAt) {
    defer s.mu.RUnlock()
    // ... 类型断言 ...
    // BUG: 失败后不 return，继续执行到外部 RUnlock
}
s.mu.RUnlock()  // double unlock if type assertion failed
```

修复：在类型断言失败后加 `return`，或去掉 `defer` 改用手动 unlock。

最佳修复：去掉 `defer`，直接 `s.mu.RUnlock()` 再 `return nil, err`。

### 2. Context Timeout

为所有用 `context.Background()` 调用 adapter 的 Wails 方法包装超时 context。

**新增** `internal/market/request.go`（或现有工具文件）：
```go
// RequestCtx returns a context with a default timeout for market data requests.
func RequestCtx() (context.Context, context.CancelFunc) {
    return context.WithTimeout(context.Background(), 10*time.Second)
}
```

**修改**：`app.go`, `app_market.go`, `app_research.go` 中使用 `context.Background()` 的地方改为 `RequestCtx()`，并 `defer cancel()`。

### 3. Goroutine Panic Recovery

在 `app.go`, `app_market.go`, `app_research.go`, `internal/python/bridge.go`, `internal/python/ml_client.go` 中所有 `go func()` 内加：

```go
go func() {
    defer func() {
        if r := recover(); r != nil {
            slog.Error("goroutine panicked", "recover", r, "stack", string(debug.Stack()))
        }
    }()
    // ... existing code ...
}()
```

新增 `"runtime/debug"` import。

## Acceptance Criteria

- [ ] `satellite_service.go` 缓存命中+类型错误不会 panic
- [ ] `govdata_service.go` 同上
- [ ] `prediction_market_service.go` 同上
- [ ] 所有 Wails adapter 调用有 10s 超时，超时后释放 goroutine
- [ ] 所有 `go func()` 有 panic recovery
- [ ] `go test ./internal/research/ -count=1` 通过
- [ ] `go vet ./...` 通过

## Risks / Trade-offs

- 10s 超时对于慢 API（如 AKShare 60-90s）可能过短。`fetchDataCache` 已使用 `RequestCtx`（30s/10min），新增的超时应区分内部 adapter（10s）和 Python sidecar 调用（由 `bridge.go` 自己的重试逻辑处理）。
- Panic recovery 改变行为：不在源头 panic，而是记录错误。调用者可能依赖 panic 行为（不太可能）。
