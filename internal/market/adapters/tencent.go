package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"quantflow/internal/market"
	"quantflow/internal/normalize"
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

	// Tencent returns GBK-encoded content for Chinese text; decode to UTF-8.
	bodyStr, err := decodeGBK(body)
	if err != nil {
		if utf8.Valid(body) {
			bodyStr = string(body)
		} else {
			return nil, fmt.Errorf("tencent: encoding decode failed for %s: %w", symbol, err)
		}
	}
	return parseTencentQuote(symbol, bodyStr)
}

// ── K-line ─────────────────────────────────────────────────────────────────────

var tencentIntervalMap = map[string]string{
	"1D": "day",
	"1W": "week",
	"1M": "month",
}

func (a *TencentAdapter) FetchOHLCV(ctx context.Context, symbol string, interval string, fqfactor string, start, end int64) ([]market.OHLCVBar, error) {
	period, ok := tencentIntervalMap[strings.ToUpper(interval)]
	if !ok {
		return nil, fmt.Errorf("tencent: unsupported interval %s (supported: 1D, 1W, 1M)", interval)
	}

	// Map fqfactor to Tencent API param:
	//   qfq → 前复权 (forward-adjusted)
	//   hfq → 后复权 (backward-adjusted)
	//   ""  → 不复权 (no adjustment)
	fqParam := ""
	fqPrefix := ""
	switch strings.ToLower(fqfactor) {
	case "hfq":
		fqParam = "hfq"
		fqPrefix = "hfq"
	case "qfq":
		fqParam = "qfq"
		fqPrefix = "qfq"
	default:
		// No adjustment factor — fetch raw prices
		fqParam = ""
		fqPrefix = ""
	}

	code := toTencentCode(symbol)
	params := fmt.Sprintf("param=%s,%s,,,2000,%s", code, period, fqParam)
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

	// Navigate: data → code → ... → kline rows.
	// Handles both map format (old): {"hk00700":{"day":[[...]],"hfqday":[[...]]}}
	// and array format (new): [{"hk00700":{"day":[[...]]}}] or [["date",o,c,h,l,v],...]
	var rawRows []any

	// Try map format
	var mapData map[string]map[string]interface{}
	if err := json.Unmarshal(result.Data, &mapData); err == nil {
		if stockData, ok := mapData[code]; ok {
			if fqPrefix != "" {
				if ql, ok := stockData[fqPrefix+period]; ok {
					rawRows, _ = ql.([]any)
				}
			}
			if rawRows == nil {
				if ql, ok := stockData[period]; ok {
					rawRows, _ = ql.([]any)
				}
			}
		}
	}

	// Try array of maps format
	if rawRows == nil {
		var arrData []map[string]map[string]interface{}
		if err := json.Unmarshal(result.Data, &arrData); err == nil && len(arrData) > 0 {
			for _, m := range arrData {
				if stockData, ok := m[code]; ok {
					if fqPrefix != "" {
						if ql, ok := stockData[fqPrefix+period]; ok {
							rawRows, _ = ql.([]any)
						}
					}
					if rawRows == nil {
						if ql, ok := stockData[period]; ok {
							rawRows, _ = ql.([]any)
						}
					}
					if rawRows != nil {
						break
					}
				}
			}
		}
	}

	// Try flat array format: [["date",o,c,h,l,v],...]
	if rawRows == nil {
		var flat [][]any
		if err := json.Unmarshal(result.Data, &flat); err == nil && len(flat) > 0 && len(flat[0]) >= 6 {
			rawRows = make([]any, len(flat))
			for i, r := range flat {
				rawRows[i] = r
			}
		}
	}

	if rawRows == nil {
		return nil, fmt.Errorf("tencent kline: no data for %s", symbol)
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
			Volume: normalize.NormalizeVolume(a.Name(), toFloatVal(r[5])),
		})
	}

	return bars, nil
}

// FetchDepth returns 5-level order book depth for a symbol.
// Tencent API returns bid/ask levels in its quote response so we reuse it.
func (a *TencentAdapter) FetchDepth(ctx context.Context, symbol string) (*market.DepthSnapshot, error) {
	code := toTencentCode(symbol)
	url := fmt.Sprintf(tencentQuoteURL, code)

	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	req.Header.Set("Referer", "https://gu.qq.com/")
	resp, err := a.client.Do(req)
	if err != nil {
		return nil, market.NewTransientErrorf("tencent depth: http error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tencent depth: HTTP %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))

	bodyStr, err := decodeGBK(body)
	if err != nil {
		if utf8.Valid(body) {
			bodyStr = string(body)
		} else {
			return nil, fmt.Errorf("tencent depth: encoding decode failed for %s: %w", symbol, err)
		}
	}
	return parseTencentDepth(symbol, bodyStr)
}

func (a *TencentAdapter) HealthCheck(ctx context.Context) error {
	_, err := a.FetchQuote(ctx, "600519")
	return err
}

// ── Industry Ranks ──────────────────────────────────────────────────────────────

const tencentHKRankingURL = "http://web.ifzq.gtimg.cn/appstock/app/HK/hkranking"

// tencentHKBaseResp represents the common fields in a Tencent API response.
type tencentHKBaseResp struct {
	Code int             `json:"code"`
	Msg  string          `json:"msg"`
	Data json.RawMessage `json:"data"`
}

// tencentHKIndustryItem represents a single industry ranking item.
type tencentHKIndustryItem struct {
	Name      string  `json:"name"`
	ChangePct float64 `json:"change_pct"`
	UpCount   int     `json:"up_count"`
	DownCount int     `json:"down_count"`
	Leader    string  `json:"leader"`
}

// FetchIndustryRanks returns HK industry rankings via Tencent Finance.
// Only supports market="HK"; for other markets returns empty slice.
func (a *TencentAdapter) FetchIndustryRanks(ctx context.Context, mkt string, topN int) ([]market.IndustryRank, error) {
	if mkt != "HK" {
		return []market.IndustryRank{}, nil
	}
	if topN <= 0 {
		topN = 20
	}

	u := fmt.Sprintf("%s?type=industry", tencentHKRankingURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("tencent: create request: %w", err)
	}

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("tencent: fetch HK ranking: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tencent: HK ranking status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("tencent: read body: %w", err)
	}

	var base tencentHKBaseResp
	if err := json.Unmarshal(body, &base); err != nil {
		return nil, fmt.Errorf("tencent: parse HK ranking: %w", err)
	}
	if base.Code != 0 {
		return nil, fmt.Errorf("tencent: HK ranking API error code=%d: %s", base.Code, base.Msg)
	}

	var items []tencentHKIndustryItem
	if err := json.Unmarshal(base.Data, &items); err != nil {
		return nil, fmt.Errorf("tencent: parse HK ranking data: %w", err)
	}

	ranks := make([]market.IndustryRank, 0, min(topN, len(items)))
	for i, item := range items {
		if i >= topN {
			break
		}
		ranks = append(ranks, market.IndustryRank{
			Rank:      i + 1,
			Name:      item.Name,
			ChangePct: item.ChangePct,
			UpCount:   item.UpCount,
			DownCount: item.DownCount,
			Leader:    item.Leader,
		})
	}
	return ranks, nil
}

// ── Response types ─────────────────────────────────────────────────────────────

type tencentKlineResponse struct {
	Code int             `json:"code"`
	Data json.RawMessage `json:"data"`
}

// parseTencentDepth parses 5-level bid/ask from Tencent's quote response.
//
// Tencent field mapping for bid/ask (0-indexed after ~ split):
//
//	[7]=bid1_price, [8]=ask1_price, [9]=bid1_vol(手), [10]=ask1_vol(手)
//	[11]=bid2_price, [12]=ask2_price, [13]=bid2_vol, [14]=ask2_vol
//	[15]=bid3_price, [16]=ask3_price, [17]=bid3_vol, [18]=ask3_vol
//	[19]=bid4_price, [20]=ask4_price, [21]=bid4_vol, [22]=ask4_vol
//	[23]=bid5_price, [24]=ask5_price, [25]=bid5_vol, [26]=ask5_vol
func parseTencentDepth(symbol, body string) (*market.DepthSnapshot, error) {
	start := 0
	for start < len(body) {
		if body[start] == '"' {
			start++
			break
		}
		start++
	}
	if start >= len(body) {
		return nil, fmt.Errorf("tencent depth: unexpected format")
	}
	end := start
	for end < len(body) {
		if body[end] == '"' {
			break
		}
		end++
	}
	content := body[start:end]
	fields := splitTencent(content, "~")

	if len(fields) < 27 {
		return nil, fmt.Errorf("tencent depth: insufficient fields (%d)", len(fields))
	}

	bids := make([]market.DepthLevel, 5)
	asks := make([]market.DepthLevel, 5)
	for i := 0; i < 5; i++ {
		bidPriceIdx := 7 + i*4
		askPriceIdx := 8 + i*4
		bidVolIdx := 9 + i*4
		askVolIdx := 10 + i*4

		bids[4-i] = market.DepthLevel{ // reverse so bids[0] is best (highest price)
			Price: parseFloatSafe(fields[bidPriceIdx]),
			Size:  normalize.NormalizeVolume("tencent", parseFloatSafe(fields[bidVolIdx])),
		}
		asks[i] = market.DepthLevel{ // asks[0] is best (lowest price)
			Price: parseFloatSafe(fields[askPriceIdx]),
			Size:  normalize.NormalizeVolume("tencent", parseFloatSafe(fields[askVolIdx])),
		}
	}

	return &market.DepthSnapshot{
		Symbol:    symbol,
		Bids:      bids,
		Asks:      asks,
		Timestamp: time.Now().UnixMilli(),
	}, nil
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
