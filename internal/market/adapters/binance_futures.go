package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"quantflow/internal/market"
	"time"
)

const binanceFuturesAPI = "https://fapi.binance.com/fapi/v1"

// BinanceFuturesAdapter queries Binance USDⓈ-M perpetual futures.
type BinanceFuturesAdapter struct {
	client *http.Client
}

func NewBinanceFuturesAdapter() *BinanceFuturesAdapter {
	return &BinanceFuturesAdapter{client: &http.Client{Timeout: 10 * time.Second}}
}

func (b *BinanceFuturesAdapter) Name() string       { return "binance_futures" }
func (b *BinanceFuturesAdapter) Markets() []string  { return []string{"CRYPTO"} }
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

// FundingRateData represents Binance USDⓈ-M perpetual funding rate snapshot.
type FundingRateData struct {
	Symbol          string  `json:"symbol"`
	MarkPrice       float64 `json:"mark_price"`
	IndexPrice      float64 `json:"index_price"`
	FundingRate     float64 `json:"funding_rate"`
	NextFundingTime int64   `json:"next_funding_time"`
}

// FetchFundingRates returns funding rates for the given perpetual symbols.
// Calls GET /fapi/v1/premiumIndex for each symbol.
func (b *BinanceFuturesAdapter) FetchFundingRates(ctx context.Context, symbols []string) ([]FundingRateData, error) {
	if len(symbols) == 0 {
		symbols = []string{"BTCUSDT", "ETHUSDT", "BNBUSDT", "SOLUSDT", "XRPUSDT", "ADAUSDT", "DOGEUSDT", "DOTUSDT", "AVAXUSDT", "LINKUSDT"}
	}
	results := make([]FundingRateData, 0, len(symbols))
	for _, sym := range symbols {
		url := fmt.Sprintf("%s/premiumIndex?symbol=%s", binanceFuturesAPI, sym)
		req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
		resp, err := b.client.Do(req)
		if err != nil {
			continue
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			continue
		}
		var raw struct {
			Symbol          string `json:"symbol"`
			MarkPrice       string `json:"markPrice"`
			IndexPrice      string `json:"indexPrice"`
			LastFundingRate string `json:"lastFundingRate"`
			NextFundingTime int64  `json:"nextFundingTime"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
			resp.Body.Close()
			continue
		}
		resp.Body.Close()
		results = append(results, FundingRateData{
			Symbol:          raw.Symbol,
			MarkPrice:       parseFloatSafe(raw.MarkPrice),
			IndexPrice:      parseFloatSafe(raw.IndexPrice),
			FundingRate:     parseFloatSafe(raw.LastFundingRate),
			NextFundingTime: raw.NextFundingTime,
		})
	}
	return results, nil
}

// LiquidationData represents a single forced liquidation order.
type LiquidationData struct {
	Symbol    string  `json:"symbol"`
	Side      string  `json:"side"`
	Price     float64 `json:"price"`
	Qty       float64 `json:"qty"`
	Amount    float64 `json:"amount"`
	Time      int64   `json:"time"`
	OrderSide string  `json:"order_side"`
}

// FetchLiquidations returns historical liquidation orders for a symbol.
// Calls GET /fapi/v1/allForceOrders with optional symbol and limit.
func (b *BinanceFuturesAdapter) FetchLiquidations(ctx context.Context, symbol string, limit int) ([]LiquidationData, error) {
	if limit <= 0 || limit > 100 {
		limit = 100
	}
	url := fmt.Sprintf("%s/allForceOrders?limit=%d", binanceFuturesAPI, limit)
	if symbol != "" {
		url += "&symbol=" + symbol
	}
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	resp, err := b.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("binance_futures FetchLiquidations: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("binance_futures: HTTP %d", resp.StatusCode)
	}
	var raw []struct {
		Symbol    string `json:"symbol"`
		Side      string `json:"side"`
		Price     string `json:"price"`
		Qty       string `json:"qty"`
		Amount    string `json:"amount"`
		Time      int64  `json:"time"`
		OrderSide string `json:"orderSide"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("binance_futures decode: %w", err)
	}
	results := make([]LiquidationData, 0, len(raw))
	for _, r := range raw {
		results = append(results, LiquidationData{
			Symbol:    r.Symbol,
			Side:      r.Side,
			Price:     parseFloatSafe(r.Price),
			Qty:       parseFloatSafe(r.Qty),
			Amount:    parseFloatSafe(r.Amount),
			Time:      r.Time,
			OrderSide: r.OrderSide,
		})
	}
	return results, nil
}
