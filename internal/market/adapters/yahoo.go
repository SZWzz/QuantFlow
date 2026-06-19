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

const yahooBaseURL = "https://query1.finance.yahoo.com/v8/finance/chart"

// YahooAdapter fetches market data from Yahoo Finance (free, no auth required).
type YahooAdapter struct {
	client *http.Client
}

// NewYahooAdapter creates a new Yahoo Finance adapter.
func NewYahooAdapter() *YahooAdapter {
	return &YahooAdapter{
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

func (a *YahooAdapter) Name() string      { return "yfinance" }
func (a *YahooAdapter) Markets() []string  { return []string{"US", "HK"} }
func (a *YahooAdapter) RequiresAuth() bool { return false }

func (a *YahooAdapter) IsAvailable(ctx context.Context) bool {
	req, err := http.NewRequestWithContext(ctx, "GET", yahooBaseURL+"/AAPL?interval=1d&range=1d", nil)
	if err != nil {
		return false
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := a.client.Do(req)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func (a *YahooAdapter) FetchQuote(ctx context.Context, symbol string) (*market.QuoteSnapshot, error) {
	bars, err := a.FetchOHLCV(ctx, symbol, "1d", time.Now().AddDate(0, 0, -2).Unix(), time.Now().Unix())
	if err != nil {
		return nil, fmt.Errorf("yahoo: %w", err)
	}
	if len(bars) == 0 {
		return nil, fmt.Errorf("yahoo: no data for %s", symbol)
	}

	last := bars[len(bars)-1]
	prev := bars[0]
	change := last.Close - prev.Close
	changePct := 0.0
	if prev.Close > 0 {
		changePct = (change / prev.Close) * 100
	}

	return &market.QuoteSnapshot{
		Symbol:    symbol,
		Last:      last.Close,
		Open:      last.Open,
		High:      last.High,
		Low:       last.Low,
		Volume:    last.Volume,
		Change:    change,
		ChangePct: changePct,
		Timestamp: time.Now().UnixMilli(),
	}, nil
}

func (a *YahooAdapter) FetchOHLCV(ctx context.Context, symbol string, interval string, start, end int64) ([]market.OHLCVBar, error) {
	url := fmt.Sprintf("%s/%s?interval=%s&period1=%d&period2=%d", yahooBaseURL, symbol, interval, start, end)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("yahoo: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("Accept", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, market.NewTransientErrorf("yahoo: http error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("yahoo: HTTP %d: %s", resp.StatusCode, string(body))
	}

	var result yahooChartResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("yahoo: parse error: %w", err)
	}

	if result.Chart.Error != nil {
		return nil, fmt.Errorf("yahoo: API error: %s", result.Chart.Error.Description)
	}

	if len(result.Chart.Result) == 0 {
		return nil, fmt.Errorf("yahoo: no results for %s", symbol)
	}

	r := result.Chart.Result[0]
	timestamps := r.Timestamp
	quote := r.Indicators.Quote[0]
	opens := quote.Open
	highs := quote.High
	lows := quote.Low
	closes := quote.Close
	volumes := quote.Volume

	bars := make([]market.OHLCVBar, 0, len(timestamps))
	for i, ts := range timestamps {
		if i >= len(opens) || i >= len(closes) {
			break
		}
		if opens[i] == 0 && closes[i] == 0 {
			continue // skip null bars
		}
		bars = append(bars, market.OHLCVBar{
			Symbol: symbol,
			Date:   time.Unix(ts, 0).Format("2006-01-02"),
			Open:   safeFloat(opens, i),
			High:   safeFloat(highs, i),
			Low:    safeFloat(lows, i),
			Close:  safeFloat(closes, i),
			Volume: safeFloat(volumes, i),
		})
	}

	return bars, nil
}

func (a *YahooAdapter) HealthCheck(ctx context.Context) error {
	_, err := a.FetchQuote(ctx, "AAPL")
	return err
}

func safeFloat(arr []float64, i int) float64 {
	if i < len(arr) {
		return arr[i]
	}
	return 0
}

// Yahoo Finance v8 API response structures
type yahooChartResponse struct {
	Chart yahooChart `json:"chart"`
}

type yahooChart struct {
	Result []yahooResult `json:"result"`
	Error  *yahooError   `json:"error"`
}

type yahooResult struct {
	Timestamp  []int64          `json:"timestamp"`
	Indicators yahooIndicators  `json:"indicators"`
}

type yahooIndicators struct {
	Quote []yahooQuote `json:"quote"`
}

type yahooQuote struct {
	Open   []float64 `json:"open"`
	High   []float64 `json:"high"`
	Low    []float64 `json:"low"`
	Close  []float64 `json:"close"`
	Volume []float64 `json:"volume"`
}

type yahooError struct {
	Code        string `json:"code"`
	Description string `json:"description"`
}
