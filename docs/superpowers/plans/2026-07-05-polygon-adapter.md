# 实施计划：Polygon.io Adapter

参考：`docs/specs/2026-07-05-polygon-adapter.md`

## Task 1: 创建限速器

在 `internal/market/adapters/polygon.go` 中添加 rate limiter：

```go
package adapters

import (
    "context"
    "time"
)

type rateLimiter struct {
    tokens chan struct{}
    closeCh chan struct{}
}

func newRateLimiter(ratePerSec int, burst int) *rateLimiter {
    rl := &rateLimiter{
        tokens:  make(chan struct{}, burst),
        closeCh: make(chan struct{}),
    }
    // fill initial burst
    for i := 0; i < burst; i++ {
        rl.tokens <- struct{}{}
    }
    go func() {
        ticker := time.NewTicker(time.Second / time.Duration(ratePerSec))
        defer ticker.Stop()
        for {
            select {
            case <-ticker.C:
                select {
                case rl.tokens <- struct{}{}:
                default:
                }
            case <-rl.closeCh:
                return
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

func (rl *rateLimiter) Close() {
    close(rl.closeCh)
}
```

---

## Task 2: 实现 Polygon 数据结构 + HTTP 客户端

```go
type PolygonAdapter struct {
    apiKey  string
    client  *http.Client
    limiter *rateLimiter
    baseURL string
}

type PolygonConfig struct {
    APIKey string  `json:"apiKey"`
    Timeout int    `json:"timeout"`
}

type polygonQuote struct {
    Status  string `json:"status"`
    Results []struct {
        Price float64 `json:"p"`
        Size  int     `json:"s"`
        Timestamp int64 `json:"t"`
    } `json:"results"`
}

type polygonAgg struct {
    Results []struct {
        Open  float64 `json:"o"`
        High  float64 `json:"h"`
        Low   float64 `json:"l"`
        Close float64 `json:"c"`
        Volume float64 `json:"v"`
        Timestamp int64 `json:"t"`
    } `json:"results"`
}

func NewPolygonAdapter(cfg PolygonConfig) *PolygonAdapter {
    timeout := cfg.Timeout
    if timeout <= 0 {
        timeout = 15
    }
    return &PolygonAdapter{
        apiKey:  cfg.APIKey,
        client:  &http.Client{Timeout: time.Duration(timeout) * time.Second},
        limiter: newRateLimiter(1, 5), // 5 req/min for free tier
        baseURL: "https://api.polygon.io",
    }
}

func (p *PolygonAdapter) request(ctx context.Context, path string) ([]byte, error) {
    if err := p.limiter.Wait(ctx); err != nil {
        return nil, err
    }
    req, err := http.NewRequestWithContext(ctx, "GET",
        p.baseURL+path+"&apiKey="+p.apiKey, nil)
    if err != nil {
        return nil, err
    }
    resp, err := p.client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    if resp.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(resp.Body)
        return nil, fmt.Errorf("Polygon HTTP %d: %s", resp.StatusCode, string(body))
    }
    return io.ReadAll(resp.Body)
}
```

---

## Task 3: 实现接口方法

```go
func (p *PolygonAdapter) Name() string { return "polygon" }

func (p *PolygonAdapter) GetQuote(ctx context.Context, symbol string) (float64, error) {
    data, err := p.request(ctx, "/v2/last/trade/"+symbol+"?")
    if err != nil {
        return 0, err
    }
    var resp polygonQuote
    if err := json.Unmarshal(data, &resp); err != nil {
        return 0, err
    }
    if len(resp.Results) == 0 {
        return 0, fmt.Errorf("no quote for %s", symbol)
    }
    return resp.Results[0].Price, nil
}

func (p *PolygonAdapter) GetKlines(ctx context.Context, symbol string,
    interval string, from, to time.Time) ([]Kline, error) {
    // Map interval to Polygon multiplier/timespan
    mult, span := parseInterval(interval)
    fromStr := from.Format("2006-01-02")
    toStr := to.Format("2006-01-02")
    path := fmt.Sprintf("/v2/aggs/ticker/%s/range/%d/%s/%s/%s?",
        symbol, mult, span, fromStr, toStr)
    data, err := p.request(ctx, path)
    if err != nil {
        return nil, err
    }
    var resp polygonAgg
    if err := json.Unmarshal(data, &resp); err != nil {
        return nil, err
    }
    klines := make([]Kline, len(resp.Results))
    for i, r := range resp.Results {
        klines[i] = Kline{
            Open:   r.Open, High: r.High,
            Low:    r.Low, Close: r.Close,
            Volume: r.Volume,
            Time:   time.UnixMilli(r.Timestamp),
        }
    }
    return klines, nil
}

func parseInterval(interval string) (mult int, span string) {
    switch interval {
    case "1m":  return 1, "minute"
    case "5m":  return 5, "minute"
    case "15m": return 15, "minute"
    case "1h":  return 1, "hour"
    case "1d":  return 1, "day"
    case "1w":  return 1, "week"
    default:    return 1, "day"
    }
}

func (p *PolygonAdapter) GetDaily(ctx context.Context, symbol string) (Kline, error) {
    data, err := p.request(ctx, "/v2/aggs/ticker/"+symbol+"/prev?")
    if err != nil {
        return Kline{}, err
    }
    var resp polygonAgg
    if err := json.Unmarshal(data, &resp); err != nil {
        return Kline{}, err
    }
    if len(resp.Results) == 0 {
        return Kline{}, fmt.Errorf("no daily data for %s", symbol)
    }
    r := resp.Results[0]
    return Kline{
        Open: r.Open, High: r.High, Low: r.Low, Close: r.Close,
        Volume: r.Volume, Time: time.UnixMilli(r.Timestamp),
    }, nil
}

func (p *PolygonAdapter) GetCompanyProfile(ctx context.Context, symbol string) (CompanyProfile, error) {
    data, err := p.request(ctx, "/v3/reference/tickers/"+symbol+"?")
    if err != nil {
        return CompanyProfile{}, err
    }
    var resp struct {
        Results struct {
            Name        string  `json:"name"`
            MarketCap   float64 `json:"market_cap"`
            Sector      string  `json:"sic_sector"`
            Industry    string  `json:"sic_industry"`
        } `json:"results"`
    }
    if err := json.Unmarshal(data, &resp); err != nil {
        return CompanyProfile{}, err
    }
    return CompanyProfile{
        Name: resp.Results.Name, MarketCap: resp.Results.MarketCap,
        Sector: resp.Results.Sector, Industry: resp.Results.Industry,
    }, nil
}

func (p *PolygonAdapter) GetNews(ctx context.Context, symbol string, limit int) ([]News, error) {
    if limit <= 0 { limit = 10 }
    path := fmt.Sprintf("/v2/reference/news?ticker=%s&limit=%d&", symbol, limit)
    data, err := p.request(ctx, path)
    if err != nil {
        return nil, err
    }
    var resp struct {
        Results []struct {
            Title      string `json:"title"`
            Summary    string `json:"summary"`
            Published  string `json:"published_utc"`
            ArticleURL string `json:"article_url"`
        } `json:"results"`
    }
    if err := json.Unmarshal(data, &resp); err != nil {
        return nil, err
    }
    news := make([]News, len(resp.Results))
    for i, r := range resp.Results {
        t, _ := time.Parse(time.RFC3339, r.Published)
        news[i] = News{
            Title: r.Title, Summary: r.Summary,
            PublishedAt: t, URL: r.ArticleURL,
        }
    }
    return news, nil
}
```

---

## Task 4: 注册适配器

在 `app.go` 或其他注册点添加 Polygon 适配器：

```go
if cfg.Polygon.APIKey != "" {
    market.RegisterAdapter(adapters.NewPolygonAdapter(cfg.Polygon))
}
```

---

## Task 5: 验证

```go test ./internal/market/adapters/... -count=1
go vet ./...
go build ./...
```
