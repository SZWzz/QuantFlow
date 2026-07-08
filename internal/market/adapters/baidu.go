package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"quantflow/internal/market"
	"quantflow/internal/normalize"
)

const (
	baiduQuoteURL = "https://finance.pae.baidu.com/selfselect/openapi?channel=cs&srcid=5353&code=%s&type=0&finClientType=pc"
	baiduKlineURL = "https://finance.pae.baidu.com/selfselect/getstockquotation"
)

// BaiduAdapter fetches A-share data from Baidu Finance (free, no auth).
// Provides real-time quotes + daily K-line with built-in MA5/MA10/MA20.
type BaiduAdapter struct {
	client *http.Client
}

func NewBaiduAdapter() *BaiduAdapter {
	return &BaiduAdapter{client: &http.Client{Timeout: 10 * time.Second}}
}

func (a *BaiduAdapter) Name() string      { return "baidu" }
func (a *BaiduAdapter) Markets() []string  { return []string{"CN"} }
func (a *BaiduAdapter) RequiresAuth() bool { return false }

func (a *BaiduAdapter) IsAvailable(ctx context.Context) bool {
	req, _ := http.NewRequestWithContext(ctx, "HEAD", "https://finance.pae.baidu.com/selfselect/openapi/v2", nil)
	_, err := a.client.Do(req)
	return err == nil
}

// ── Quote ──────────────────────────────────────────────────────────────────────

func (a *BaiduAdapter) FetchQuote(ctx context.Context, symbol string) (*market.QuoteSnapshot, error) {
	code := toBaiduCode(symbol)
	url := fmt.Sprintf(baiduQuoteURL, code)

	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	req.Header.Set("Accept", "application/vnd.finance-web.v1+json")
	req.Header.Set("Referer", "https://gushitong.baidu.com/")
	resp, err := a.client.Do(req)
	if err != nil {
		return nil, market.NewTransientErrorf("baidu: http error: %v", err)
	}
	defer resp.Body.Close()

	var result baiduQuoteResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("baidu: parse error: %w", err)
	}

	if !resultCodeOK(result.ResultCode) {
		return nil, fmt.Errorf("baidu: ResultCode indicates failure: %v", result.ResultCode)
	}
	if result.Result == nil || len(result.Result.Data) == 0 {
		return nil, fmt.Errorf("baidu: no data for %s", symbol)
	}

	d := result.Result.Data[0]
	return &market.QuoteSnapshot{
		Symbol:    symbol,
		Last:      d.Price,
		Change:    d.Change,
		ChangePct: d.ChangeRatio,
		High:      d.High,
		Low:       d.Low,
		Open:      d.Open,
		Volume:    normalize.NormalizeVolume(a.Name(), d.Volume),
		Timestamp: time.Now().UnixMilli(),
	}, nil
}

// ── K-line ─────────────────────────────────────────────────────────────────────

func (a *BaiduAdapter) FetchOHLCV(ctx context.Context, symbol string, interval string, _ string, start, end int64) ([]market.OHLCVBar, error) {
	if strings.ToUpper(interval) != "1D" {
		return nil, fmt.Errorf("baidu: only 1D interval supported, got %s", interval)
	}

	code := toBaiduCode(symbol)
	plain := strings.TrimLeft(code, "shsz")

	params := fmt.Sprintf(
		"all=1&isIndex=false&isBk=false&isBlock=false&isFutures=false&isStock=true&newFormat=1&group=quotation_kline_ab&finClientType=pc&code=%s&ktype=1",
		plain,
	)
	req, _ := http.NewRequestWithContext(ctx, "GET", baiduKlineURL+"?"+params, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("Accept", "application/vnd.finance-web.v1+json")
	req.Header.Set("Origin", "https://gushitong.baidu.com")
	req.Header.Set("Referer", "https://gushitong.baidu.com/")

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, market.NewTransientErrorf("baidu kline: http error: %v", err)
	}
	defer resp.Body.Close()

	var result baiduKlineResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("baidu kline: parse error: %w", err)
	}

	if !resultCodeOK(result.ResultCode) {
		return nil, fmt.Errorf("baidu kline: ResultCode indicates failure: %v", result.ResultCode)
	}

	md := result.Result.NewMarketData
	if md == nil || md.Keys == nil || md.MarketData == "" {
		return nil, fmt.Errorf("baidu kline: no data for %s", symbol)
	}

	// Build column index map
	colIdx := make(map[string]int)
	for i, k := range md.Keys {
		colIdx[strings.ToLower(strings.TrimSpace(k))] = i
	}

	startDate := time.Unix(start, 0)
	endDate := time.Unix(end, 0)

	bars := make([]market.OHLCVBar, 0)
	for _, line := range strings.Split(md.MarketData, ";") {
		if line == "" {
			continue
		}
		values := strings.Split(line, ",")
		if len(values) < len(md.Keys) {
			continue
		}

		// Parse time
		timeStr := colVal(values, colIdx, "time")
		if timeStr == "" {
			continue
		}
		t, err := time.Parse("2006-01-02 15:04:05", timeStr)
		if err != nil {
			t, err = time.Parse("2006-01-02", timeStr)
			if err != nil {
				continue
			}
		}

		if t.Before(startDate) || t.After(endDate) {
			continue
		}

		bars = append(bars, market.OHLCVBar{
			Symbol: symbol,
			Date:   t.Format("2006-01-02"),
			Open:   colFloat(values, colIdx, "open"),
			High:   colFloat(values, colIdx, "high"),
			Low:    colFloat(values, colIdx, "low"),
			Close:  colFloat(values, colIdx, "close"),
			Volume: normalize.NormalizeVolume(a.Name(), colFloat(values, colIdx, "volume")),
		})
	}

	return bars, nil
}

func (a *BaiduAdapter) HealthCheck(ctx context.Context) error { return nil }

// ── Response types ─────────────────────────────────────────────────────────────

type baiduQuoteResponse struct {
	Result     *baiduQuoteResult `json:"Result"`
	ResultCode interface{}       `json:"ResultCode"`
}

// resultCodeOK checks whether Baidu's ResultCode indicates success.
// The field is inconsistently typed (sometimes int, sometimes string),
// so we must accept both forms. See a-stock-data SKILL FAQ.
func resultCodeOK(rc interface{}) bool {
	if rc == nil {
		return true // no ResultCode field = success (older responses)
	}
	switch v := rc.(type) {
	case float64:
		return v == 0
	case string:
		return v == "0" || v == ""
	default:
		return true // unknown type, assume OK
	}
}

type baiduQuoteResult struct {
	Data []baiduQuoteData `json:"data"`
}

type baiduQuoteData struct {
	Code        string  `json:"code"`
	Name        string  `json:"name"`
	Price       float64 `json:"price"`
	Change      float64 `json:"change"`
	ChangeRatio float64 `json:"changeRatio"`
	High        float64 `json:"high"`
	Low         float64 `json:"low"`
	Open        float64 `json:"open"`
	Volume      float64 `json:"volume"`
}

type baiduKlineResponse struct {
	Result     baiduKlineResult `json:"Result"`
	ResultCode interface{}      `json:"ResultCode"`
}

type baiduKlineResult struct {
	NewMarketData *baiduMarketData `json:"newMarketData"`
}

type baiduMarketData struct {
	Keys       []string `json:"keys"`
	MarketData string   `json:"marketData"`
}

// ── Helpers ────────────────────────────────────────────────────────────────────

func toBaiduCode(symbol string) string {
	s := symbol
	s = strings.TrimSuffix(s, ".SH")
	s = strings.TrimSuffix(s, ".SZ")
	if strings.HasPrefix(s, "6") || strings.HasPrefix(s, "5") || strings.HasPrefix(s, "9") {
		return "sh" + s
	}
	return "sz" + s
}

func colVal(values []string, idx map[string]int, col string) string {
	i, ok := idx[col]
	if !ok || i >= len(values) {
		return ""
	}
	return values[i]
}

func colFloat(values []string, idx map[string]int, col string) float64 {
	s := colVal(values, idx, col)
	if s == "" || s == "--" {
		return 0
	}
	return parseFloatSafe(s)
}
