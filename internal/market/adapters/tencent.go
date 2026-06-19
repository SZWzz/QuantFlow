package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"quantflow/internal/market"
)

const (
	tencentQuoteURL = "http://qt.gtimg.cn/q=%s"
	tencentKlineURL = "https://web.ifzq.gtimg.cn/appstock/app/fqkline/get"
)

// TencentAdapter fetches A/HK market data from Tencent Finance (free, no auth).
// Real-time quotes via qt.gtimg.cn, K-line via web.ifzq.gtimg.cn.
type TencentAdapter struct {
	client *http.Client
}

func NewTencentAdapter() *TencentAdapter {
	return &TencentAdapter{client: &http.Client{Timeout: 10 * time.Second}}
}

func (a *TencentAdapter) Name() string      { return "tencent" }
func (a *TencentAdapter) Markets() []string  { return []string{"CN", "HK"} }
func (a *TencentAdapter) RequiresAuth() bool { return false }

func (a *TencentAdapter) IsAvailable(ctx context.Context) bool {
	req, _ := http.NewRequestWithContext(ctx, "GET",
		fmt.Sprintf(tencentQuoteURL, "sh600519"), nil)
	resp, err := a.client.Do(req)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// ── Quote ──────────────────────────────────────────────────────────────────────

func (a *TencentAdapter) FetchQuote(ctx context.Context, symbol string) (*market.QuoteSnapshot, error) {
	code := toTencentCode(symbol)
	url := fmt.Sprintf(tencentQuoteURL, code)

	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	req.Header.Set("Referer", "https://gu.qq.com/")
	resp, err := a.client.Do(req)
	if err != nil {
		return nil, market.NewTransientErrorf("tencent: http error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tencent: HTTP %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	return parseTencentQuote(symbol, string(body))
}

// ── K-line ─────────────────────────────────────────────────────────────────────

var tencentIntervalMap = map[string]string{
	"1D": "day",
	"1W": "week",
	"1M": "month",
}

func (a *TencentAdapter) FetchOHLCV(ctx context.Context, symbol string, interval string, start, end int64) ([]market.OHLCVBar, error) {
	period, ok := tencentIntervalMap[strings.ToUpper(interval)]
	if !ok {
		return nil, fmt.Errorf("tencent: unsupported interval %s (supported: 1D, 1W, 1M)", interval)
	}

	code := toTencentCode(symbol)
	params := fmt.Sprintf("param=%s,%s,,,2000,qfq", code, period)
	req, _ := http.NewRequestWithContext(ctx, "GET", tencentKlineURL+"?"+params, nil)
	req.Header.Set("Referer", "https://gu.qq.com/")
	req.Header.Set("Accept", "*/*")

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, market.NewTransientErrorf("tencent kline: http error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("tencent kline: HTTP %d: %s", resp.StatusCode, string(body))
	}

	var result tencentKlineResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("tencent kline: parse error: %w", err)
	}

	if result.Code != 0 {
		return nil, fmt.Errorf("tencent kline: API error code=%d", result.Code)
	}

	// Navigate: data → code → qfqday/qfqweek/qfqmonth → [[time,open,close,high,low,volume],...]
	stockData := result.Data[code]
	ql := stockData["qfq"+period]
	if ql == nil {
		// Try without adj prefix
		ql = stockData[period]
	}
	if ql == nil {
		return nil, fmt.Errorf("tencent kline: no data for %s", symbol)
	}

	rawRows, ok := ql.([]any)
	if !ok {
		return nil, fmt.Errorf("tencent kline: unexpected data format for %s", symbol)
	}

	startDate := time.Unix(start, 0)
	endDate := time.Unix(end, 0)

	bars := make([]market.OHLCVBar, 0, len(rawRows))
	for _, row := range rawRows {
		r, ok := row.([]any)
		if !ok || len(r) < 6 {
			continue
		}

		ts := parseTencentKlineTime(fmt.Sprint(r[0]))
		if ts.IsZero() {
			continue
		}

		// Filter to date range
		if ts.Before(startDate) || ts.After(endDate) {
			continue
		}

		bars = append(bars, market.OHLCVBar{
			Symbol: symbol,
			Date:   ts.Format("2006-01-02"),
			Open:   toFloatVal(r[1]),
			Close:  toFloatVal(r[2]),
			High:   toFloatVal(r[3]),
			Low:    toFloatVal(r[4]),
			Volume: toFloatVal(r[5]),
		})
	}

	return bars, nil
}

func (a *TencentAdapter) HealthCheck(ctx context.Context) error {
	_, err := a.FetchQuote(ctx, "600519")
	return err
}

// ── Response types ─────────────────────────────────────────────────────────────

type tencentKlineResponse struct {
	Code int                               `json:"code"`
	Data map[string]map[string]interface{} `json:"data"`
}

// ── Helpers ────────────────────────────────────────────────────────────────────

func parseTencentKlineTime(raw string) time.Time {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}
	}
	for _, layout := range []string{
		"2006-01-02 15:04:05", "2006-01-02 15:04", "2006-01-02",
		"2006/01/02", "20060102",
	} {
		if t, err := time.Parse(layout, raw); err == nil {
			return t
		}
	}
	return time.Time{}
}

func toFloatVal(v any) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case string:
		return parseFloatSafe(val)
	case json.Number:
		f, _ := val.Float64()
		return f
	}
	return 0
}
