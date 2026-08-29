package adapters

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"quantflow/internal/market"
	"sync"
	"time"
)

const qosMinuteCooldown = 3 * time.Second

type QOSMinuteAdapter struct {
	client    *http.Client
	apiKey    string
	baseURL   string
	mu        sync.Mutex
	lastFetch map[string]time.Time
}

type QOSConfig struct {
	APIKey  string
	BaseURL string
}

func NewQOSMinuteAdapter(cfg QOSConfig) *QOSMinuteAdapter {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "https://api.qos.hk"
	}
	return &QOSMinuteAdapter{
		client:    &http.Client{Timeout: 10 * time.Second},
		apiKey:    cfg.APIKey,
		baseURL:   baseURL,
		lastFetch: make(map[string]time.Time),
	}
}

func (a *QOSMinuteAdapter) Name() string         { return "qos" }
func (a *QOSMinuteAdapter) Markets() []string    { return []string{"HK"} }
func (a *QOSMinuteAdapter) RequiresAuth() bool   { return true }
func (a *QOSMinuteAdapter) SetAPIKey(key string) { a.apiKey = key }

func (a *QOSMinuteAdapter) IsAvailable(ctx context.Context) bool {
	return a.apiKey != ""
}

func (a *QOSMinuteAdapter) FetchQuote(ctx context.Context, symbol string) (*market.QuoteSnapshot, error) {
	return nil, fmt.Errorf("qos: FetchQuote not implemented")
}

func (a *QOSMinuteAdapter) FetchOHLCV(ctx context.Context, symbol, interval, fqfactor string, start, end int64) ([]market.OHLCVBar, error) {
	return nil, fmt.Errorf("qos: FetchOHLCV not implemented")
}

func (a *QOSMinuteAdapter) HealthCheck(ctx context.Context) error {
	if a.apiKey == "" {
		return fmt.Errorf("qos: API key not configured")
	}
	return nil
}

type qosKlineReq struct {
	Key       string         `json:"key"`
	KlineReqs []qosSingleReq `json:"kline_reqs"`
}

type qosSingleReq struct {
	Code   string `json:"c"`  // "HK:700"
	Count  int    `json:"co"` // 240 = number of 1-min bars
	Adjust int    `json:"a"`  // 0 = no adjust
	Ktype  int    `json:"kt"` // 1 = 1min
}

type qosKlineRespItem struct {
	T string  `json:"t"` // time "0910" (HHMM)
	O float64 `json:"o"`
	H float64 `json:"h"`
	L float64 `json:"l"`
	C float64 `json:"c"`
	V float64 `json:"v"` // volume
}

func (a *QOSMinuteAdapter) FetchMinuteLine(symbol string) ([]market.MinuteTick, error) {
	if a.apiKey == "" {
		return nil, fmt.Errorf("qos: API key not configured")
	}

	a.mu.Lock()
	if last, ok := a.lastFetch[symbol]; ok && time.Since(last) < qosMinuteCooldown {
		a.mu.Unlock()
		return nil, nil
	}
	a.lastFetch[symbol] = time.Now()
	a.mu.Unlock()

	qosSymbol := toQOSSymbol(symbol)

	reqBody := qosKlineReq{
		Key: a.apiKey,
		KlineReqs: []qosSingleReq{
			{Code: qosSymbol, Count: 240, Adjust: 0, Ktype: 1},
		},
	}

	bodyBytes, _ := json.Marshal(reqBody)
	url := a.baseURL + "/kline"

	httpReq, err := http.NewRequest("POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("qos: create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("qos: http error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("qos: HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("qos: read error: %w", err)
	}

	var qosResp struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data []struct {
			K []qosKlineRespItem `json:"k"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &qosResp); err != nil {
		return nil, fmt.Errorf("qos: parse error: %w", err)
	}
	if qosResp.Code != 0 {
		return nil, fmt.Errorf("qos: API error %d: %s", qosResp.Code, qosResp.Msg)
	}
	if len(qosResp.Data) == 0 {
		return nil, fmt.Errorf("qos: no data for %s", symbol)
	}

	ticks := make([]market.MinuteTick, 0, len(qosResp.Data[0].K))
	for _, k := range qosResp.Data[0].K {
		t := k.T
		if len(t) == 4 {
			t = t[:2] + ":" + t[2:]
		}
		ticks = append(ticks, market.MinuteTick{
			Time:   t,
			Price:  k.C,
			Volume: k.V,
		})
	}
	return ticks, nil
}

func toQOSSymbol(symbol string) string {
	i := 0
	for i < len(symbol) && symbol[i] == '0' {
		i++
	}
	num := symbol[i:]
	if num == "" {
		num = "0"
	}
	return "HK:" + num
}
