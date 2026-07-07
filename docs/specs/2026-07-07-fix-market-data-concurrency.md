# Fix Market Data Concurrency Issues

## Motivation

Three concurrency bugs in the market data layer:

1. **TOCTOU race in `MarketDataHub.Publish`** (`internal/market/hub.go:110-130`) — Two goroutines publishing to the same new topic both see `ok == false`, both create a `topicBroker`, one orphaned.

2. **OffHoursCache returns reference types** (`internal/market/offhours.go`) — Callers can mutate cached slices/maps, corrupting future reads.

3. **EastMoney adapter lacks rate limiting** (`internal/market/adapters/eastmoney.go`) — Rapid panel refresh triggers anti-scraping blocks.

## Design

### 1. Fix TOCTOU race

**File**: `internal/market/hub.go`

Change the double-checked locking pattern to hold write lock for the entire check-and-create:

```go
func (h *MarketDataHub) Publish(topic string, data any) {
    h.mu.Lock()  // was RLock
    broker, ok := h.topics[topic]
    if !ok {
        broker = newTopicBroker()
        h.topics[topic] = broker
    }
    h.mu.Unlock()
    broker.publish(msg)
}
```

This eliminates the race entirely at the cost of serializing all publishes to the same hub. For a single-user desktop app this is acceptable.

### 2. Fix OffHoursCache reference leak

**File**: `internal/market/offhours.go`

Add a `deepCopy` function that JSON round-trips cached values on read:

```go
func (c *OffHoursCache) Get(key string, dest any) error {
    c.mu.RLock()
    raw, ok := c.data[key]
    c.mu.RUnlock()
    if !ok { return ErrCacheMiss }
    // Deep-copy via JSON to prevent caller mutation
    data, _ := json.Marshal(raw)
    return json.Unmarshal(data, dest)
}
```

Change the `Get` API from returning `any` to accepting a destination pointer, matching `json.Unmarshal` convention.

### 3. Add EastMoney rate limiter

**File**: `internal/market/adapters/eastmoney_rate_limit.go` (already exists but not wired)

Create a token-bucket rate limiter (1 request per 200ms = 5/s) wired into `EastMoneyAdapter.FetchOHLCV` and `FetchQuote`:

```go
var emLimiter = rate.NewLimiter(rate.Every(200*time.Millisecond), 1)

func (a *EastMoneyAdapter) FetchOHLCV(ctx context.Context, ...) (...) {
    if err := emLimiter.Wait(ctx); err != nil { return nil, err }
    // ... existing fetch logic
}
```

Apply same pattern to Tencent and Sina adapters if they exist.

### Modified files

| File | Change |
|------|--------|
| `internal/market/hub.go` | TOCTOU: RLock → Lock in Publish |
| `internal/market/offhours.go` | Add deep-copy Get API |
| `internal/market/adapters/eastmoney_rate_limit.go` | Wire token-bucket into adapter |
| `internal/market/adapters/eastmoney.go` | Call rate limiter before HTTP requests |
| `internal/market/adapters/eastmoney_signals.go` | Same rate limiter |
| `internal/market/offhours_test.go` | Update tests for new Get signature |
| `internal/market/hub_test.go` | Add concurrent publish test |

### API changes

- `OffHoursCache.Get(key string) any` → `OffHoursCache.Get(key string, dest any) error`
- No other API changes

## Acceptance Criteria

- [ ] `TestConcurrentPublish` passes with race detector (`-race`)
- [ ] OffHoursCache callers cannot mutate cached data (test with slice append)
- [ ] EastMoney adapter rate-limits to ≤5 req/s
- [ ] All existing tests pass
- [ ] `go test -race ./internal/market/...` passes

## Risks / Trade-offs

- **Publish serialization**: Single lock serializes all publishes. For a single-user desktop app with <100 topics, this is fine. If we ever support multi-user, we'd need sharded locks.
- **JSON deep-copy**: Adds allocation overhead. Cache reads on hot paths (real-time quote ticks) should be measured. If too slow, use `gob` or manual copy for hot types.
