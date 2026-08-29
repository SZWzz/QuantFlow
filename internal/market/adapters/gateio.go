package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"quantflow/internal/market"
	"strings"
	"time"
)

const gateioBaseURL = "https://api.gateio.ws/api/v4/spot"

// GateIOAdapter fetches crypto market data from Gate.io (free, no auth).
// Chosen because api.gateio.ws is accessible from mainland China where
// Binance/OKX/CoinGecko are blocked.
type GateIOAdapter struct {
	client *http.Client
}

// NewGateIOAdapter creates a new Gate.io adapter.
func NewGateIOAdapter() *GateIOAdapter {
	return &GateIOAdapter{
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

func (a *GateIOAdapter) Name() string       { return "gateio" }
func (a *GateIOAdapter) Markets() []string  { return []string{"CRYPTO"} }
func (a *GateIOAdapter) RequiresAuth() bool { return false }

func (a *GateIOAdapter) IsAvailable(ctx context.Context) bool {
	req, _ := http.NewRequestWithContext(ctx, "GET",
		gateioBaseURL+"/tickers?currency_pair=BTC_USDT", nil)
	resp, err := a.client.Do(req)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// FetchQuote returns real-time crypto quote from Gate.io.
// Symbol format: "BTCUSDT" (auto-converted to "BTC_USDT").
func (a *GateIOAdapter) FetchQuote(ctx context.Context, symbol string) (*market.QuoteSnapshot, error) {
	pair := toGateIOPair(symbol)
	url := fmt.Sprintf("%s/tickers?currency_pair=%s", gateioBaseURL, pair)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("gateio: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, market.NewTransientErrorf("gateio: http error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("gateio: HTTP %d: %s", resp.StatusCode, string(body))
	}

	var tickers []gateioTicker
	if err := json.NewDecoder(resp.Body).Decode(&tickers); err != nil {
		return nil, fmt.Errorf("gateio: parse error: %w", err)
	}
	if len(tickers) == 0 {
		return nil, fmt.Errorf("gateio: no ticker for %s", symbol)
	}

	t := tickers[0]
	changePct := parseFloat(t.ChangePercentage)

	return &market.QuoteSnapshot{
		Symbol:    symbol,
		Last:      parseFloat(t.Last),
		Open:      0, // Not in ticker endpoint; use OHLCV for open
		High:      parseFloat(t.High24h),
		Low:       parseFloat(t.Low24h),
		Volume:    parseFloat(t.BaseVolume),
		Change:    parseFloat(t.Last) - parseFloat(t.Last)*(1/(1+changePct/100)), // approximate
		ChangePct: changePct,
		Timestamp: time.Now().UnixMilli(),
	}, nil
}

// FetchOHLCV fetches OHLCV bars from Gate.io candlestick endpoint.
func (a *GateIOAdapter) FetchOHLCV(ctx context.Context, symbol string, interval string, _ string, start, end int64) ([]market.OHLCVBar, error) {
	pair := toGateIOPair(symbol)
	gateInterval := toGateIOInterval(interval)

	// Gate.io returns candles from newest to oldest; request enough to cover the range
	limit := 1000
	url := fmt.Sprintf("%s/candlesticks?currency_pair=%s&interval=%s&limit=%d",
		gateioBaseURL, pair, gateInterval, limit)
	if start > 0 {
		url += fmt.Sprintf("&from=%d", start)
	}
	if end > 0 {
		url += fmt.Sprintf("&to=%d", end)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("gateio OHLCV: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, market.NewTransientErrorf("gateio OHLCV: http error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("gateio OHLCV: HTTP %d: %s", resp.StatusCode, string(body))
	}

	// Gate.io returns: [[timestamp, volume_quote, close, high, low, open, volume_base, is_complete]]
	var raw [][]any
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("gateio OHLCV parse: %w", err)
	}

	bars := make([]market.OHLCVBar, 0, len(raw))
	for i := len(raw) - 1; i >= 0; i-- { // Reverse to chronological
		k := raw[i]
		if len(k) < 7 {
			continue
		}
		ts := int64(toFloat(k[0]))
		bars = append(bars, market.OHLCVBar{
			Symbol: symbol,
			Date:   time.Unix(ts, 0).Format("2006-01-02"),
			Open:   toFloat(k[5]), // open at index 5
			High:   toFloat(k[3]), // high at index 3
			Low:    toFloat(k[4]), // low at index 4
			Close:  toFloat(k[2]), // close at index 2
			Volume: toFloat(k[6]), // base volume at index 6
		})
	}

	if len(bars) == 0 {
		return nil, fmt.Errorf("gateio OHLCV: no data for %s", symbol)
	}
	return bars, nil
}

func (a *GateIOAdapter) HealthCheck(ctx context.Context) error {
	_, err := a.FetchQuote(ctx, "BTCUSDT")
	return err
}

// --- helpers ---

type gateioTicker struct {
	CurrencyPair     string `json:"currency_pair"`
	Last             string `json:"last"`
	LowestAsk        string `json:"lowest_ask"`
	HighestBid       string `json:"highest_bid"`
	ChangePercentage string `json:"change_percentage"`
	BaseVolume       string `json:"base_volume"`
	QuoteVolume      string `json:"quote_volume"`
	High24h          string `json:"high_24h"`
	Low24h           string `json:"low_24h"`
}

// toGateIOPair converts QuantFlow symbol to Gate.io format.
func toGateIOPair(symbol string) string {
	// "BTCUSDT" → "BTC_USDT"
	// Common quote currencies: USDT, USDC, BTC, ETH, BUSD
	quotes := []string{"USDT", "USDC", "BTC", "ETH", "BUSD", "USD"}
	upper := strings.ToUpper(symbol)
	for _, q := range quotes {
		if strings.HasSuffix(upper, q) && len(upper) > len(q)+1 {
			base := upper[:len(upper)-len(q)]
			return base + "_" + q
		}
	}
	// Fallback: insert underscore before last known quote-like segment
	return upper
}

// toGateIOInterval converts QuantFlow interval to Gate.io format.
func toGateIOInterval(interval string) string {
	switch interval {
	case "1m", "5m", "15m", "30m":
		return interval
	case "1h", "4h", "8h":
		return interval
	case "1d", "1w", "1M":
		return interval
	default:
		return "1d"
	}
}
