package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"quantflow/internal/market"
)

const binanceFuturesAPI = "https://fapi.binance.com/fapi/v1"

// BinanceFuturesAdapter queries Binance USDⓈ-M perpetual futures.
type BinanceFuturesAdapter struct {
	client *http.Client
}

func NewBinanceFuturesAdapter() *BinanceFuturesAdapter {
	return &BinanceFuturesAdapter{client: &http.Client{Timeout: 10 * time.Second}}
}

func (b *BinanceFuturesAdapter) Name() string     { return "binance_futures" }
func (b *BinanceFuturesAdapter) Markets() []string { return []string{"CRYPTO"} }
func (b *BinanceFuturesAdapter) RequiresAuth() bool { return false }

func (b *BinanceFuturesAdapter) IsAvailable(ctx context.Context) bool {
	req, _ := http.NewRequestWithContext(ctx, "GET", binanceFuturesAPI+"/ticker/price?symbol=BTCUSDT", nil)
	resp, err := b.client.Do(req)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func (b *BinanceFuturesAdapter) FetchOHLCV(ctx context.Context, symbol string, interval string, _ string, start, end int64) ([]market.OHLCVBar, error) {
	url := fmt.Sprintf("%s/klines?symbol=%s&interval=%s&limit=500", binanceFuturesAPI, symbol, interval)
	if start > 0 {
		url += fmt.Sprintf("&startTime=%d000", start)
	}
	if end > 0 {
		url += fmt.Sprintf("&endTime=%d000", end)
	}

	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	resp, err := b.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("binance_futures FetchOHLCV: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("binance_futures: HTTP %d", resp.StatusCode)
	}

	var raw [][]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("binance_futures decode: %w", err)
	}

	bars := make([]market.OHLCVBar, 0, len(raw))
	for _, r := range raw {
		if len(r) < 6 {
			continue
		}
		ts := int64(r[0].(float64)) / 1000
		bars = append(bars, market.OHLCVBar{
			Symbol: symbol,
			Date:   time.Unix(ts, 0).Format("2006-01-02"),
			Open:   parseFloatSafe(fmt.Sprint(r[1])),
			High:   parseFloatSafe(fmt.Sprint(r[2])),
			Low:    parseFloatSafe(fmt.Sprint(r[3])),
			Close:  parseFloatSafe(fmt.Sprint(r[4])),
			Volume: parseFloatSafe(fmt.Sprint(r[5])),
		})
	}
	return bars, nil
}

func (b *BinanceFuturesAdapter) FetchQuote(ctx context.Context, symbol string) (*market.QuoteSnapshot, error) {
	url := fmt.Sprintf("%s/ticker/price?symbol=%s", binanceFuturesAPI, symbol)
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	resp, err := b.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("binance_futures FetchQuote: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("binance_futures: HTTP %d", resp.StatusCode)
	}

	var result struct {
		Symbol string `json:"symbol"`
		Price  string `json:"price"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("binance_futures decode: %w", err)
	}
	return &market.QuoteSnapshot{
		Symbol:    symbol,
		Last:      parseFloatSafe(result.Price),
		Name:      symbol,
		Timestamp: time.Now().UnixMilli(),
	}, nil
}

func (b *BinanceFuturesAdapter) HealthCheck(ctx context.Context) error {
	_, err := b.FetchQuote(ctx, "BTCUSDT")
	return err
}
