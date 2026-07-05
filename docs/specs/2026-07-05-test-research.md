# Test Coverage: Internal Research (16 services, currently 1 test)

## Motivation

`internal/research` 包含 16 个源文件但只有 1 个测试文件（`satellite_service_test.go`），覆盖率约 6%。P0 double RUnlock bug 就发生在这个包 —— testing 本可以提前捕捉。每个 service 遵循相同的 TTL 缓存 + 类型断言 + mock fallback 模式，同一类 bug 可能潜伏在其他 15 个文件中。

## Design

### 测试策略

#### 原则
1. **不依赖外部 API** — 所有 service 接受 **可选的 adapter 参数**（nil = mock 模式），测试用 nil adapter 验证 mock fallback 路径
2. **缓存穿透测试** — 对每个 service 测试：未命中 → 获取 → 缓存 → 命中返回
3. **类型断言错误** — mock adapter 返回错误类型数据，验证 service 不 panic

#### 需要测试的 service

| # | Service 文件 | 核心方法 | 预期测试数 |
|---|-------------|----------|-----------|
| 1 | `analyst_estimates_service.go` | GetEstimates, GetConsensus | 3 |
| 2 | `announcement_service.go` | GetAnnouncements | 3 |
| 3 | `capital_service.go` | GetMarginTrading, GetBlockTrades, GetHolderChanges, GetDividendHistory | 5 |
| 4 | `congress_trading_service.go` | GetCongressTrades | 3 |
| 5 | `financials_service.go` | GetFinancials, GetFinancialMetrics | 3 |
| 6 | `fundflow_service.go` | GetMinuteFlow, GetDailyFlow | 3 |
| 7 | `geopolitics_service.go` | GetTopicRisks, GetTopicDetail | 3 |
| 8 | `govdata_service.go` | GetIndicator, GetAllSignals | 4 |
| 9 | `insider_trading_service.go` | GetInsiderTrades | 4 |
| 10 | `northbound_service.go` | GetMinuteFlow, GetHistory | 3 |
| 11 | `peer_comparison_service.go` | GetPeers | 3 |
| 12 | `prediction_market_service.go` | GetEvents, GetEventDetail, GetPriceHistory | 5 |
| 13 | `repo.go` | Save, List, GetByID, Delete | 4 |
| 14 | `satellite_service.go` | GetRegionSnapshots, GetRegionDetail, ExtractSignals | 5 |
| 15 | `sentiment_engine.go` | AnalyzeSentiment, GetSentimentHistory | 4 |

### 测试模式模板

```go
func TestXxxService_GetData_CacheHit(t *testing.T) {
    svc := NewXxxService(nil) // nil adapter = mock-only
    // First call populates cache with mock data
    data, err := svc.GetData(ctx, "test-input")
    if err != nil { t.Fatal(err) }
    if len(data) == 0 { t.Fatal("expected mock data") }
    
    // Second call should hit cache
    cached, err := svc.GetData(ctx, "test-input")
    if err != nil { t.Fatal(err) }
    if !reflect.DeepEqual(data, cached) {
        t.Error("cached data differs from original")
    }
}
```

```go
func TestXxxService_GetData_TypeAssertionError(t *testing.T) {
    svc := NewXxxService(nil)
    // Inject bad type into cache
    svc.mu.Lock()
    svc.cache["bad-key"] = &cacheEntry{
        data:      "this is a string, not the expected type",
        expiresAt: time.Now().Add(time.Hour),
    }
    svc.mu.Unlock()
    // Should return error, not panic
    _, err := svc.GetData(ctx, "bad-key")
    if err == nil {
        t.Error("expected error for type mismatch, got nil")
    }
}
```

```go
func TestXxxService_GetData_ExpiredCache(t *testing.T) {
    svc := NewXxxService(nil)
    svc.mu.Lock()
    svc.cache["expired"] = &cacheEntry{
        data:      mockData(),
        expiresAt: time.Now().Add(-time.Hour), // expired
    }
    svc.mu.Unlock()
    // Should fall through to mock adapter (nil = fresh mock)
    data, err := svc.GetData(ctx, "expired")
    if err != nil { t.Fatal(err) }
    if data == nil { t.Fatal("expected data despite expired cache") }
}
```

## Acceptance Criteria

- [ ] 15 个 service 文件各有对应的 `*_test.go`，每个至少 3 个测试
- [ ] 所有测试不依赖外部网络（nil adapter + mock data path）
- [ ] 覆盖缓存命中、缓存过期、类型断言错误三种路径
- [ ] `go test ./internal/research/... -count=1` 全部通过
- [ ] 测试总行覆盖 > 60%（目前 ~6%）

## Risks / Trade-offs

- 部分 service（如 `fundflow_service.go`）的 adapter 接口有 3+ 个方法，mock 实现需要为其每个方法返回数据。建议用 `testify/mock` 或手写 mock struct。
- `sentiment_engine.go` 依赖 Python gRPC，mock 需跳过 bridge 调用。检查是否有 `bridge == nil` 路径可走。
