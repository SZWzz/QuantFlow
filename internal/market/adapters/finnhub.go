package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"quantflow/internal/market"
)

const finnhubBaseURL = "https://finnhub.io/api/v1"

// FinnhubAdapter fetches US stock data from Finnhub (free tier: 60 req/min).
// Requires FINNHUB_API_KEY env var (free registration at finnhub.io).
// Provides real-time quotes and OHLCV for US equities.
type FinnhubAdapter struct {
	client *http.Client
	apiKey string
}

// NewFinnhubAdapter creates a new Finnhub adapter.
// Reads FINNHUB_API_KEY from environment. Returns adapter even without key
// (IsAvailable will return false gracefully).
func NewFinnhubAdapter() *FinnhubAdapter {
	return &FinnhubAdapter{
		client: &http.Client{Timeout: 10 * time.Second},
		apiKey: os.Getenv("FINNHUB_API_KEY"),
	}
}

// SetAPIKey updates the Finnhub API key from app config.
func (a *FinnhubAdapter) SetAPIKey(key string) {
	if key != "" {
		a.apiKey = key
	}
}

func (a *FinnhubAdapter) Name() string      { return "finnhub" }
func (a *FinnhubAdapter) Markets() []string  { return []string{"US"} }
func (a *FinnhubAdapter) RequiresAuth() bool { return true }

func (a *FinnhubAdapter) IsAvailable(ctx context.Context) bool {
	if a.apiKey == "" {
		return false
	}
	req, _ := http.NewRequestWithContext(ctx, "GET",
		finnhubBaseURL+"/quote?symbol=AAPL&token="+a.apiKey, nil)
	resp, err := a.client.Do(req)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// FetchQuote returns real-time US stock quote from Finnhub.
func (a *FinnhubAdapter) FetchQuote(ctx context.Context, symbol string) (*market.QuoteSnapshot, error) {
	if a.apiKey == "" {
		return nil, fmt.Errorf("finnhub: API key not configured (set FINNHUB_API_KEY)")
	}

	url := fmt.Sprintf("%s/quote?symbol=%s&token=%s", finnhubBaseURL, symbol, a.apiKey)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("finnhub: %w", err)
	}

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, market.NewTransientErrorf("finnhub: http error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, market.NewTransientErrorf("finnhub: rate limited (60/min free tier)")
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("finnhub: HTTP %d: %s", resp.StatusCode, string(body))
	}

	var q struct {
		Current     float64 `json:"c"`
		High        float64 `json:"h"`
		Low         float64 `json:"l"`
		Open        float64 `json:"o"`
		PrevClose   float64 `json:"pc"`
		Timestamp   int64   `json:"t"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&q); err != nil {
		return nil, fmt.Errorf("finnhub: parse error: %w", err)
	}

	change := q.Current - q.PrevClose
	changePct := 0.0
	if q.PrevClose > 0 {
		changePct = (change / q.PrevClose) * 100
	}

	return &market.QuoteSnapshot{
		Symbol:    symbol,
		Last:      q.Current,
		Open:      q.Open,
		High:      q.High,
		Low:       q.Low,
		Volume:    0, // Finnhub quote endpoint doesn't include volume
		Change:    change,
		ChangePct: changePct,
		Timestamp: q.Timestamp * 1000, // Finnhub uses Unix seconds
	}, nil
}

// FetchOHLCV fetches historical OHLCV from Finnhub's stock/candle endpoint.
func (a *FinnhubAdapter) FetchOHLCV(ctx context.Context, symbol string, interval string, start, end int64) ([]market.OHLCVBar, error) {
	if a.apiKey == "" {
		return nil, fmt.Errorf("finnhub: API key not configured (set FINNHUB_API_KEY)")
	}

	// Map interval to Finnhub resolution
	resolution := "D"
	switch interval {
	case "1m", "5m", "15m", "30m", "60m", "1h":
		resolution = interval
	case "1wk", "1w", "week":
		resolution = "W"
	case "1mo", "1M", "month":
		resolution = "M"
	}

	url := fmt.Sprintf("%s/stock/candle?symbol=%s&resolution=%s&from=%d&to=%d&token=%s",
		finnhubBaseURL, symbol, resolution, start, end, a.apiKey)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("finnhub OHLCV: %w", err)
	}

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, market.NewTransientErrorf("finnhub OHLCV: http error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("finnhub OHLCV: HTTP %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Close     []float64 `json:"c"`
		High      []float64 `json:"h"`
		Low       []float64 `json:"l"`
		Open      []float64 `json:"o"`
		Volume    []float64 `json:"v"`
		Timestamp []int64   `json:"t"`
		Status    string    `json:"s"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("finnhub OHLCV parse: %w", err)
	}
	if result.Status == "no_data" {
		return nil, fmt.Errorf("finnhub OHLCV: no data for %s", symbol)
	}

	bars := make([]market.OHLCVBar, 0, len(result.Timestamp))
	for i, ts := range result.Timestamp {
		if i >= len(result.Open) || i >= len(result.Close) {
			break
		}
		bars = append(bars, market.OHLCVBar{
			Symbol: symbol,
			Date:   time.Unix(ts, 0).Format("2006-01-02"),
			Open:   result.Open[i],
			High:   result.High[i],
			Low:    result.Low[i],
			Close:  result.Close[i],
			Volume: result.Volume[i],
		})
	}
	return bars, nil
}

func (a *FinnhubAdapter) HealthCheck(ctx context.Context) error {
	_, err := a.FetchQuote(ctx, "AAPL")
	return err
}
