package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"quantflow/internal/market"
)

// rateLimiter implements a token bucket rate limiter with a background
// goroutine that refills tokens at a fixed rate.
type rateLimiter struct {
	tokens  chan struct{}
	closeCh chan struct{}
}

func newRateLimiter(ratePerSec int, burst int) *rateLimiter {
	rl := &rateLimiter{
		tokens:  make(chan struct{}, burst),
		closeCh: make(chan struct{}),
	}
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

// PolygonConfig configures the Polygon.io adapter.
type PolygonConfig struct {
	APIKey  string `json:"apiKey"`
	Timeout int    `json:"timeout"`
}

// PolygonAdapter fetches US equity data from Polygon.io.
// Free tier: 5 requests/minute, data delayed 15 minutes.
type PolygonAdapter struct {
	apiKey  string
	client  *http.Client
	limiter *rateLimiter
	baseURL string
}

// NewPolygonAdapter creates a new Polygon adapter with the given config.
func NewPolygonAdapter(cfg PolygonConfig) *PolygonAdapter {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 15
	}
	return &PolygonAdapter{
		apiKey:  cfg.APIKey,
		client:  &http.Client{Timeout: time.Duration(timeout) * time.Second},
		limiter: newRateLimiter(1, 5),
		baseURL: "https://api.polygon.io",
	}
}

// ── market.Adapter interface ──────────────────────────────────────

func (p *PolygonAdapter) Name() string      { return "polygon" }
func (p *PolygonAdapter) Markets() []string  { return []string{"US"} }
func (p *PolygonAdapter) RequiresAuth() bool { return true }

func (p *PolygonAdapter) IsAvailable(ctx context.Context) bool {
	if p.apiKey == "" {
		return false
	}
	_, err := p.GetQuote(ctx, "AAPL")
	return err == nil
}

func (p *PolygonAdapter) FetchQuote(ctx context.Context, symbol string) (*market.QuoteSnapshot, error) {
	price, err := p.GetQuote(ctx, symbol)
	if err != nil {
		return nil, err
	}
	daily, dailyErr := p.GetDaily(ctx, symbol)
	change := 0.0
	changePct := 0.0
	if dailyErr == nil && daily.Close != 0 {
		change = price - daily.Close
		changePct = (change / daily.Close) * 100
	}
	return &market.QuoteSnapshot{
		Symbol:    symbol,
		Last:      price,
		Change:    change,
		ChangePct: changePct,
		Timestamp: time.Now().UnixMilli(),
	}, nil
}

func (p *PolygonAdapter) FetchOHLCV(ctx context.Context, symbol string, interval string, _ string, start, end int64) ([]market.OHLCVBar, error) {
	from := time.Unix(start, 0)
	to := time.Unix(end, 0)
	klines, err := p.GetKlines(ctx, symbol, interval, from, to)
	if err != nil {
		return nil, err
	}
	bars := make([]market.OHLCVBar, len(klines))
	for i, k := range klines {
		bars[i] = market.OHLCVBar{
			Symbol: symbol,
			Date:   k.Time.Format("2006-01-02"),
			Open:   k.Open,
			High:   k.High,
			Low:    k.Low,
			Close:  k.Close,
			Volume: k.Volume,
		}
	}
	return bars, nil
}

func (p *PolygonAdapter) HealthCheck(ctx context.Context) error {
	_, err := p.GetQuote(ctx, "AAPL")
	return err
}

// ── Internal helper ───────────────────────────────────────────────

func (p *PolygonAdapter) request(ctx context.Context, path string) ([]byte, error) {
	if err := p.limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("polygon: rate limit: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, "GET", p.baseURL+path+"&apiKey="+p.apiKey, nil)
	if err != nil {
		return nil, fmt.Errorf("polygon: create request: %w", err)
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("polygon: http: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("polygon: HTTP %d: %s", resp.StatusCode, string(body))
	}
	return io.ReadAll(resp.Body)
}

// ── Response types ────────────────────────────────────────────────

type polygonQuote struct {
	Status  string `json:"status"`
	Results []struct {
		Price     float64 `json:"p"`
		Size      int     `json:"s"`
		Timestamp int64   `json:"t"`
	} `json:"results"`
}

type polygonAgg struct {
	Results []struct {
		Open      float64 `json:"o"`
		High      float64 `json:"h"`
		Low       float64 `json:"l"`
		Close     float64 `json:"c"`
		Volume    float64 `json:"v"`
		Timestamp int64   `json:"t"`
	} `json:"results"`
}

// Kline represents a single OHLCV candlestick from Polygon.io.
type Kline struct {
	Open   float64
	High   float64
	Low    float64
	Close  float64
	Volume float64
	Time   time.Time
}

// CompanyProfile represents basic company information from Polygon.io.
type CompanyProfile struct {
	Name      string
	MarketCap float64
	Sector    string
	Industry  string
}

// News represents a news article from Polygon.io.
type News struct {
	Title       string
	Summary     string
	PublishedAt time.Time
	URL         string
}

// ── Public API methods ────────────────────────────────────────────

// GetQuote returns the last trade price for a symbol.
func (p *PolygonAdapter) GetQuote(ctx context.Context, symbol string) (float64, error) {
	data, err := p.request(ctx, "/v2/last/trade/"+symbol+"?")
	if err != nil {
		return 0, err
	}
	var resp polygonQuote
	if err := json.Unmarshal(data, &resp); err != nil {
		return 0, fmt.Errorf("polygon: parse quote: %w", err)
	}
	if len(resp.Results) == 0 {
		return 0, fmt.Errorf("polygon: no quote for %s", symbol)
	}
	return resp.Results[0].Price, nil
}

// GetKlines returns historical OHLCV bars for a symbol.
func (p *PolygonAdapter) GetKlines(ctx context.Context, symbol string, interval string, from, to time.Time) ([]Kline, error) {
	mult, span := parseInterval(interval)
	fromStr := from.Format("2006-01-02")
	toStr := to.Format("2006-01-02")
	path := fmt.Sprintf("/v2/aggs/ticker/%s/range/%d/%s/%s/%s?", symbol, mult, span, fromStr, toStr)
	data, err := p.request(ctx, path)
	if err != nil {
		return nil, err
	}
	var resp polygonAgg
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("polygon: parse klines: %w", err)
	}
	klines := make([]Kline, len(resp.Results))
	for i, r := range resp.Results {
		klines[i] = Kline{
			Open: r.Open, High: r.High, Low: r.Low, Close: r.Close,
			Volume: r.Volume, Time: time.UnixMilli(r.Timestamp),
		}
	}
	return klines, nil
}

// GetDaily returns the previous trading day's OHLCV bar.
func (p *PolygonAdapter) GetDaily(ctx context.Context, symbol string) (Kline, error) {
	data, err := p.request(ctx, "/v2/aggs/ticker/"+symbol+"/prev?")
	if err != nil {
		return Kline{}, err
	}
	var resp polygonAgg
	if err := json.Unmarshal(data, &resp); err != nil {
		return Kline{}, fmt.Errorf("polygon: parse daily: %w", err)
	}
	if len(resp.Results) == 0 {
		return Kline{}, fmt.Errorf("polygon: no daily data for %s", symbol)
	}
	r := resp.Results[0]
	return Kline{
		Open: r.Open, High: r.High, Low: r.Low, Close: r.Close,
		Volume: r.Volume, Time: time.UnixMilli(r.Timestamp),
	}, nil
}

// GetCompanyProfile returns company name, market cap, sector, and industry.
func (p *PolygonAdapter) GetCompanyProfile(ctx context.Context, symbol string) (CompanyProfile, error) {
	data, err := p.request(ctx, "/v3/reference/tickers/"+symbol+"?")
	if err != nil {
		return CompanyProfile{}, err
	}
	var resp struct {
		Results struct {
			Name      string  `json:"name"`
			MarketCap float64 `json:"market_cap"`
			Sector    string  `json:"sic_sector"`
			Industry  string  `json:"sic_industry"`
		} `json:"results"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return CompanyProfile{}, fmt.Errorf("polygon: parse profile: %w", err)
	}
	return CompanyProfile{
		Name: resp.Results.Name, MarketCap: resp.Results.MarketCap,
		Sector: resp.Results.Sector, Industry: resp.Results.Industry,
	}, nil
}

// GetNews returns recent news articles for a symbol.
func (p *PolygonAdapter) GetNews(ctx context.Context, symbol string, limit int) ([]News, error) {
	if limit <= 0 {
		limit = 10
	}
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
		return nil, fmt.Errorf("polygon: parse news: %w", err)
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

// parseInterval maps common interval strings to Polygon multiplier and timespan.
func parseInterval(interval string) (mult int, span string) {
	switch interval {
	case "1m":
		return 1, "minute"
	case "5m":
		return 5, "minute"
	case "15m":
		return 15, "minute"
	case "1h":
		return 1, "hour"
	case "1d", "day":
		return 1, "day"
	case "1w", "week":
		return 1, "week"
	default:
		return 1, "day"
	}
}
