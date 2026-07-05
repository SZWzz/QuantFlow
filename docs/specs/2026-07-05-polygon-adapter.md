# Polygon.io Adapter Implementation

## Motivation

`internal/market/adapters/polygon.go` 当前只返回 `"not implemented"`。Polygon.io 是美股市场主要实时/历史数据源之一，实现它使美股行情和基本面数据可用。

## Design

### API 覆盖

实现以下 Polygon REST API 端点：

| 功能 | Polygon API | 适配器方法 |
|------|-------------|-----------|
| 实时报价 | `GET /v2/last/trade/{stocksTicker}` | `GetQuote(symbol)` |
| 历史K线 | `GET /v2/aggs/ticker/{stocksTicker}/range/{multiplier}/{timespan}/{from}/{to}` | `GetKlines(symbol, interval, from, to)` |
| 日线 | `GET /v2/aggs/ticker/{stocksTicker}/prev` | `GetDaily(symbol)` |
| 公司基本面 | `GET /v3/reference/tickers/{ticker}` | `GetCompanyProfile(symbol)` |
| 新闻 | `GET /v2/reference/news?ticker={ticker}` | `GetNews(symbol, limit)` |

### 配置

```go
type PolygonConfig struct {
    APIKey string `json:"apiKey"`
    Timeout int   `json:"timeout"` // 秒
}
```

API Key 从配置文件或环境变量读取。

### 速率限制

Polygon 免费版限制 5 次/分钟。实现 token bucket 限速：

```go
type rateLimiter struct {
    tokens chan struct{}
}

func newRateLimiter(rate, burst int) *rateLimiter {
    rl := &rateLimiter{tokens: make(chan struct{}, burst)}
    go func() {
        ticker := time.NewTicker(time.Second / time.Duration(rate))
        defer ticker.Stop()
        for range ticker.C {
            select {
            case rl.tokens <- struct{}{}:
            default:
            }
        }
    }()
    return rl
}

func (rl *rateLimiter) Wait(ctx context.Context) error {
    select {
    case <-rl.tokens:
        return nil
    case <-ctx.Done():
        return ctx.Err()
    }
}
```

## Acceptance Criteria

- [ ] `GetQuote` 返回实时价格和成交量
- [ ] `GetKlines` 返回正确的 OHLCV 数据
- [ ] `GetCompanyProfile` 返回公司名称、市值、行业
- [ ] `GetNews` 返回新闻列表
- [ ] 速率限制正常工作（免费版 5 req/min 不触发 429）
- [ ] API Key 从配置/环境变量读取
- [ ] `go test ./internal/market/adapters/... -count=1` 通过

## Risks / Trade-offs

- Polygon 免费版数据延迟 15 分钟，实时数据需要付费订阅。文档需注明。
- 不需要 WebSocket 实现（免费版不支持实时流）。实时数据优先级：Alpaca WebSocket > Polygon REST。
- 限速 token bucket goroutine 会造成一个永久 goroutine。可行的替代：使用 time.Timer 的 blocking channel（更复杂但不需要后台 goroutine）。
