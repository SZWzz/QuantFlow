package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"quantflow/internal/market"
	"time"
)

// OKXAdapter fetches crypto data from OKX (free, no auth).
type OKXAdapter struct {
	client *http.Client
}

func NewOKXAdapter() *OKXAdapter {
	return &OKXAdapter{client: &http.Client{Timeout: 10 * time.Second}}
}

func (a *OKXAdapter) Name() string       { return "okx" }
func (a *OKXAdapter) Markets() []string  { return []string{"CRYPTO"} }
func (a *OKXAdapter) RequiresAuth() bool { return false }

func (a *OKXAdapter) IsAvailable(ctx context.Context) bool {
	req, _ := http.NewRequestWithContext(ctx, "GET", "https://www.okx.com/api/v5/market/ticker?instId=BTC-USDT", nil)
	resp, err := a.client.Do(req)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func (a *OKXAdapter) FetchQuote(ctx context.Context, symbol string) (*market.QuoteSnapshot, error) {
	instID := toOKXInstID(symbol)
	url := fmt.Sprintf("https://www.okx.com/api/v5/market/ticker?instId=%s", instID)

	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	resp, err := a.client.Do(req)
	if err != nil {
		return nil, market.NewTransientErrorf("okx: http error: %v", err)
	}
	defer resp.Body.Close()

	var result struct {
		Code string `json:"code"`
		Data []struct {
			Last    string `json:"last"`
			Open24h string `json:"open24h"`
			High24h string `json:"high24h"`
			Low24h  string `json:"low24h"`
			Vol24h  string `json:"vol24h"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("okx: parse error: %v", err)
	}
	if result.Code != "0" || len(result.Data) == 0 {
		return nil, fmt.Errorf("okx: API error code=%s", result.Code)
	}

	d := result.Data[0]
	last := parseFloat(d.Last)
	open := parseFloat(d.Open24h)
	change := last - open
	changePct := 0.0
	if open > 0 {
		changePct = change / open * 100
	}

	return &market.QuoteSnapshot{
		Symbol:    symbol,
		Last:      last,
		Open:      open,
		High:      parseFloat(d.High24h),
		Low:       parseFloat(d.Low24h),
		Volume:    parseFloat(d.Vol24h),
		Change:    change,
		ChangePct: changePct,
		Timestamp: time.Now().UnixMilli(),
	}, nil
}

func (a *OKXAdapter) FetchOHLCV(ctx context.Context, symbol string, interval string, _ string, start, end int64) ([]market.OHLCVBar, error) {
	instID := toOKXInstID(symbol)
	// OKX candles API uses Unix millisecond timestamps for after/before.
	url := fmt.Sprintf("https://www.okx.com/api/v5/market/candles?instId=%s&bar=%s&limit=100&after=%d000&before=%d000",
		instID, interval, start, end)

	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	resp, err := a.client.Do(req)
	if err != nil {
		return nil, market.NewTransientErrorf("okx: http error: %v", err)
	}
	defer resp.Body.Close()

	var result struct {
		Code string     `json:"code"`
		Data [][]string `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("okx: parse error: %v", err)
	}

	endDate := time.Unix(end, 0)
	bars := make([]market.OHLCVBar, 0, len(result.Data))
	for i := len(result.Data) - 1; i >= 0; i-- {
		k := result.Data[i]
		if len(k) < 6 {
			continue
		}
		ts := parseFloat(k[0])
		barTime := time.Unix(int64(ts)/1000, 0)
		// Post-filter to enforce date range (belt-and-suspenders with API params).
		if barTime.After(endDate) {
			continue
		}
		bars = append(bars, market.OHLCVBar{
			Symbol: symbol,
			Date:   barTime.Format("2006-01-02 15:04"),
			Open:   parseFloat(k[1]),
			High:   parseFloat(k[2]),
			Low:    parseFloat(k[3]),
			Close:  parseFloat(k[4]),
			Volume: parseFloat(k[5]),
		})
	}
	return bars, nil
}

func (a *OKXAdapter) HealthCheck(ctx context.Context) error {
	_, err := a.FetchQuote(ctx, "BTC-USDT")
	return err
}

func toOKXInstID(symbol string) string {
	// BTCUSDT → BTC-USDT
	if len(symbol) > 4 && symbol[len(symbol)-4:] == "USDT" {
		return symbol[:len(symbol)-4] + "-USDT"
	}
	return symbol
}
