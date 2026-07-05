# 实施计划：Research 测试全覆盖

参考：`docs/specs/2026-07-05-test-research.md`

## 数据结构

所有 15 个 service 共享相同的缓存模式。将通用 mock 和测试辅助提取到 `internal/research/research_test.go`：

```go
package research

import (
    "context"
    "time"
)

func newCtx() context.Context { return context.Background() }

// testCacheEntry 注入缓存条目的辅助函数
func setCacheEntry(sv mutexMap, key string, data any, expiresIn time.Duration) {
    // sv 需要是 `mu` + `cache` 可访问的 — 由于 cache 和 mu 是私有字段，
    // 测试写在同一个包内可以直接访问
}
```

由于测试在同一 `package research` 内，可以直接访问 `mu` 和 `cache` 字段。

## 文件清单

### 1. `analyst_estimates_service_test.go`
```go
func TestAnalystEstimates_GetEstimates_Cache(t *testing.T) {
    svc := NewAnalystEstimatesService(nil)
    data, err := svc.GetEstimates(newCtx(), "AAPL")
    if err != nil || len(data) == 0 {
        t.Fatal("expected mock data")
    }
    // cache hit
    cached, err := svc.GetEstimates(newCtx(), "AAPL")
    if err != nil { t.Fatal(err) }
    if len(cached) != len(data) {
        t.Error("cache returned different data")
    }
}

func TestAnalystEstimates_GetConsensus(t *testing.T) {
    svc := NewAnalystEstimatesService(nil)
    _, err := svc.GetConsensus(newCtx(), "AAPL", "2026Q2")
    if err != nil {
        t.Error("expected mock consensus data, got error:", err)
    }
}
```

### 2. `announcement_service_test.go`
```go
func TestAnnouncementService_GetAnnouncements_Cache(t *testing.T) {
    svc := NewAnnouncementService(nil)
    data, err := svc.GetAnnouncements(newCtx(), "000001", 10)
    if err != nil || len(data) == 0 { t.Fatal("expected mock data") }
    cached, err := svc.GetAnnouncements(newCtx(), "000001", 10)
    if err != nil { t.Fatal(err) }
    if len(cached) != len(data) { t.Error("cache mismatch") }
}
```

### 3-15. 其余每个 service 一个测试文件

每个文件按模式生成：
- TestXxx_GetData_CacheHit
- TestXxx_GetData_ExpiredCache（注入过期条目，验证重新获取）
- TestXxx_GetData_TypeAssertionError（注入错误类型，验证不 panic）

## 关键路径：sentiment_engine.go

`sentiment_engine.go` 是唯一依赖 Python gRPC 的 service。测试策略：

```go
func TestSentimentEngine_AnalyzeSentiment_NilBridge(t *testing.T) {
    e := NewSentimentEngine(nil) // nil bridge
    _, err := e.AnalyzeSentiment(newCtx(), "BTC", "", "news", "en")
    if err == nil {
        t.Error("expected error with nil bridge")
    }
}
```

## 验证

```bash
go test ./internal/research/... -v -count=1
go test ./internal/research/... -race -count=1
```
