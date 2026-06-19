package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"quantflow/internal/market"
)

// CoinGeckoAdapter fetches crypto data from CoinGecko (free, no auth for basic tier).
type CoinGeckoAdapter struct {
	client *http.Client
}

func NewCoinGeckoAdapter() *CoinGeckoAdapter {
	return &CoinGeckoAdapter{client: &http.Client{Timeout: 15 * time.Second}}
}

func (a *CoinGeckoAdapter) Name() string      { return "coingecko" }
func (a *CoinGeckoAdapter) Markets() []string  { return []string{"CRYPTO"} }
func (a *CoinGeckoAdapter) RequiresAuth() bool { return false }

func (a *CoinGeckoAdapter) IsAvailable(ctx context.Context) bool {
	req, _ := http.NewRequestWithContext(ctx, "GET", "https://api.coingecko.com/api/v3/ping", nil)
	resp, err := a.client.Do(req)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func (a *CoinGeckoAdapter) FetchQuote(ctx context.Context, symbol string) (*market.QuoteSnapshot, error) {
	id := toCoinGeckoID(symbol)
	url := fmt.Sprintf("https://api.coingecko.com/api/v3/simple/price?ids=%s&vs_currencies=usd&include_24hr_change=true&include_24hr_vol=true", id)

	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	req.Header.Set("Accept", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, market.NewTransientErrorf("coingecko: http error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, fmt.Errorf("coingecko: rate limited")
	}

	var result map[string]struct {
		USD       float64 `json:"usd"`
		USD24hChg float64 `json:"usd_24h_change"`
		USD24hVol float64 `json:"usd_24h_vol"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("coingecko: parse error: %v", err)
	}

	data, ok := result[id]
	if !ok {
		return nil, fmt.Errorf("coingecko: no data for %s", id)
	}

	return &market.QuoteSnapshot{
		Symbol:    symbol,
		Last:      data.USD,
		Change:    data.USD24hChg,
		Volume:    data.USD24hVol,
		Timestamp: time.Now().UnixMilli(),
	}, nil
}

func (a *CoinGeckoAdapter) FetchOHLCV(ctx context.Context, symbol string, interval string, start, end int64) ([]market.OHLCVBar, error) {
	// CoinGecko free tier doesn't support OHLCV via simple API
	return nil, fmt.Errorf("coingecko: OHLCV not supported on free tier")
}

func (a *CoinGeckoAdapter) HealthCheck(ctx context.Context) error {
	return nil
}

func toCoinGeckoID(symbol string) string {
	ids := map[string]string{
		"BTCUSDT": "bitcoin", "ETHUSDT": "ethereum",
		"SOLUSDT": "solana", "BNBUSDT": "binancecoin",
	}
	if id, ok := ids[symbol]; ok {
		return id
	}
	return symbol
}
