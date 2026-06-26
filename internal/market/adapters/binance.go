package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"quantflow/internal/market"
)

const binanceBaseURL = "https://api.binance.com/api/v3"

// BinanceAdapter fetches crypto market data from Binance (free, no auth for public endpoints).
type BinanceAdapter struct {
	client *http.Client
}

// NewBinanceAdapter creates a new Binance adapter.
func NewBinanceAdapter() *BinanceAdapter {
	return &BinanceAdapter{
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

func (a *BinanceAdapter) Name() string      { return "binance" }
func (a *BinanceAdapter) Markets() []string  { return []string{"CRYPTO"} }
func (a *BinanceAdapter) RequiresAuth() bool { return false }

func (a *BinanceAdapter) IsAvailable(ctx context.Context) bool {
	req, _ := http.NewRequestWithContext(ctx, "GET", binanceBaseURL+"/ticker/price?symbol=BTCUSDT", nil)
	resp, err := a.client.Do(req)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func (a *BinanceAdapter) FetchQuote(ctx context.Context, symbol string) (*market.QuoteSnapshot, error) {
	// Price ticker
	priceURL := fmt.Sprintf("%s/ticker/price?symbol=%s", binanceBaseURL, symbol)
	req, _ := http.NewRequestWithContext(ctx, "GET", priceURL, nil)

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, market.NewTransientErrorf("binance: http error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("binance: HTTP %d", resp.StatusCode)
	}

	var priceResp struct {
		Symbol string `json:"symbol"`
		Price  string `json:"price"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&priceResp); err != nil {
		return nil, fmt.Errorf("binance: parse error: %w", err)
	}

	last := parseFloat(priceResp.Price)

	// 24h ticker for change info
	tickerURL := fmt.Sprintf("%s/ticker/24hr?symbol=%s", binanceBaseURL, symbol)
	req2, _ := http.NewRequestWithContext(ctx, "GET", tickerURL, nil)
	resp2, err := a.client.Do(req2)
	if err != nil {
		// 24h ticker is optional — return basic quote
		return &market.QuoteSnapshot{
			Symbol:    symbol,
			Last:      last,
			Timestamp: time.Now().UnixMilli(),
		}, nil
	}
	defer resp2.Body.Close()

	var ticker24 struct {
		OpenPrice  string `json:"openPrice"`
		HighPrice  string `json:"highPrice"`
		LowPrice   string `json:"lowPrice"`
		Volume     string `json:"volume"`
		PriceChange string `json:"priceChange"`
		PriceChangePercent string `json:"priceChangePercent"`
	}
	if err := json.NewDecoder(resp2.Body).Decode(&ticker24); err != nil {
		return &market.QuoteSnapshot{Symbol: symbol, Last: last, Timestamp: time.Now().UnixMilli()}, nil
	}

	return &market.QuoteSnapshot{
		Symbol:    symbol,
		Last:      last,
		Open:      parseFloat(ticker24.OpenPrice),
		High:      parseFloat(ticker24.HighPrice),
		Low:       parseFloat(ticker24.LowPrice),
		Volume:    parseFloat(ticker24.Volume),
		Change:    parseFloat(ticker24.PriceChange),
		ChangePct: parseFloat(ticker24.PriceChangePercent),
		Timestamp: time.Now().UnixMilli(),
	}, nil
}

func (a *BinanceAdapter) FetchOHLCV(ctx context.Context, symbol string, interval string, _ string, start, end int64) ([]market.OHLCVBar, error) {
	// Map interval to Binance format
	binanceInterval := interval
	switch interval {
	case "1m", "5m", "15m", "30m", "1h", "4h", "1d", "1w", "1M":
		binanceInterval = interval
	default:
		binanceInterval = "1d"
	}

	limit := 500
	url := fmt.Sprintf("%s/klines?symbol=%s&interval=%s&limit=%d&startTime=%d000&endTime=%d000",
		binanceBaseURL, symbol, binanceInterval, limit, start, end)

	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	resp, err := a.client.Do(req)
	if err != nil {
		return nil, market.NewTransientErrorf("binance: http error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("binance: HTTP %d", resp.StatusCode)
	}

	var rawKlines [][]any
	if err := json.NewDecoder(resp.Body).Decode(&rawKlines); err != nil {
		return nil, fmt.Errorf("binance: parse error: %w", err)
	}

	bars := make([]market.OHLCVBar, 0, len(rawKlines))
	for _, k := range rawKlines {
		if len(k) < 6 {
			continue
		}
		ts := int64(toFloat(k[0]))
		bars = append(bars, market.OHLCVBar{
			Symbol: symbol,
			Date:   time.Unix(ts/1000, 0).Format("2006-01-02 15:04"),
			Open:   toFloat(k[1]),
			High:   toFloat(k[2]),
			Low:    toFloat(k[3]),
			Close:  toFloat(k[4]),
			Volume: toFloat(k[5]),
		})
	}

	return bars, nil
}

func (a *BinanceAdapter) HealthCheck(ctx context.Context) error {
	_, err := a.FetchQuote(ctx, "BTCUSDT")
	return err
}

func toFloat(v any) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case string:
		return parseFloat(val)
	}
	return 0
}

func parseFloat(s string) float64 {
	var f float64
	fmt.Sscanf(s, "%f", &f)
	return f
}
