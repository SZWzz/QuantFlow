package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"quantflow/internal/market"
)

// AKShareAdapter fetches data via Tencent Finance HTTP API.
// Named "akshare" for the fallback chain. Uses Tencent's free HTTP interface
// which provides CN A-shares, HK stocks, and US (limited) — no API key required.
type AKShareAdapter struct {
	client *http.Client
	hkAvailable bool
	hkMu        sync.RWMutex
}

// NewAKShareAdapter creates a new AKShare adapter (Tencent-backed).
func NewAKShareAdapter() *AKShareAdapter {
	return &AKShareAdapter{
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

func (a *AKShareAdapter) Name() string      { return "akshare" }
func (a *AKShareAdapter) Markets() []string  { return []string{"CN", "HK", "US"} }
func (a *AKShareAdapter) RequiresAuth() bool { return false }

func (a *AKShareAdapter) IsAvailable(ctx context.Context) bool {
	req, _ := http.NewRequestWithContext(ctx, "GET",
		"http://qt.gtimg.cn/q=sh600519", nil)
	resp, err := a.client.Do(req)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func (a *AKShareAdapter) FetchQuote(ctx context.Context, symbol string) (*market.QuoteSnapshot, error) {
	code := toTencentCode(symbol)
	url := fmt.Sprintf("http://qt.gtimg.cn/q=%s", code)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("akshare: %w", err)
	}

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, market.NewTransientErrorf("akshare: http error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("akshare: HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		return nil, fmt.Errorf("akshare: read error: %w", err)
	}

	return parseTencentQuote(symbol, string(body))
}

func (a *AKShareAdapter) FetchOHLCV(ctx context.Context, symbol string, interval string, fqfactor string, start, end int64) ([]market.OHLCVBar, error) {
	code := toTencentCode(symbol)
	// Determine period: day/week/month
	period := "day"
	switch interval {
	case "1wk", "1w", "week":
		period = "week"
	case "1mo", "1M", "month":
		period = "month"
	}

	// Map fqfactor to Tencent adjustment param:
	//   qfq → 前复权 (forward-adjusted)
	//   hfq → 后复权 (backward-adjusted)
	//   ""  → 不复权 (raw prices)
	// Default to hfq for backtesting to avoid look-ahead bias.
	adjust := "hfq"
	switch strings.ToLower(fqfactor) {
	case "qfq":
		adjust = "qfq"
	case "":
		adjust = ""
	}

	// Tencent K-line API via proxy.finance.qq.com (works for both CN and HK from China).
	url := fmt.Sprintf("https://proxy.finance.qq.com/ifzqgtimg/appstock/app/newkline/newkline?param=%s,%s,,,320,%s", code, period, adjust)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("akshare OHLCV: %w", err)
	}
	req.Header.Set("Referer", "http://gu.qq.com/"+code)
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, market.NewTransientErrorf("akshare OHLCV: http error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("akshare OHLCV: HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<18)) // 256KB
	if err != nil {
		return nil, fmt.Errorf("akshare OHLCV: read error: %w", err)
	}

	return parseTencentKline(symbol, body)
}

// parseTencentKline parses Tencent's K-line JSON response.
// Format: {"code":0,"msg":"","data":{"hk00700":{"day":[...]|"week":[...]|"month":[...]}}}
func parseTencentKline(symbol string, body []byte) ([]market.OHLCVBar, error) {
	var result struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data map[string]struct {
			Day   [][]string `json:"day"`
			Week  [][]string `json:"week"`
			Month [][]string `json:"month"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("akshare OHLCV parse: %w", err)
	}
	if result.Code != 0 {
		return nil, fmt.Errorf("akshare OHLCV: API error code %d: %s", result.Code, result.Msg)
	}

	// Find the first stock entry
	var klines [][]string
	for _, v := range result.Data {
		if len(v.Day) > 0 {
			klines = v.Day
		} else if len(v.Week) > 0 {
			klines = v.Week
		} else if len(v.Month) > 0 {
			klines = v.Month
		}
		break
	}
	if len(klines) == 0 {
		return nil, fmt.Errorf("akshare OHLCV: no K-line data for %s", symbol)
	}

	bars := make([]market.OHLCVBar, 0, len(klines))
	for _, row := range klines {
		if len(row) < 6 {
			continue
		}
		// Tencent K-line format: [date, open, close, high, low, volume]
		date := strings.Trim(row[0], "\"")
		open := parseFloatSafe(row[1])
		high := parseFloatSafe(row[3])
		low := parseFloatSafe(row[4])
		closeV := parseFloatSafe(row[2])
		volume := parseFloatSafe(row[5])

		if open == 0 && closeV == 0 {
			continue
		}

		bars = append(bars, market.OHLCVBar{
			Symbol: symbol,
			Date:   date,
			Open:   open,
			High:   high,
			Low:    low,
			Close:  closeV,
			Volume: volume,
		})
	}

	if len(bars) == 0 {
		return nil, fmt.Errorf("akshare OHLCV: empty data after parse for %s", symbol)
	}
	return bars, nil
}


func (a *AKShareAdapter) HealthCheck(ctx context.Context) error {
	_, err := a.FetchQuote(ctx, "600519")
	return err
}

// toTencentCode converts symbol to Tencent format.
//
//	A-shares: sh600519, sz000001
//	HK: r_hk00700 → hk00700
func toTencentCode(symbol string) string {
	if len(symbol) > 3 && symbol[len(symbol)-3:] == ".HK" {
		return "hk" + symbol[:len(symbol)-3]
	}
	// A-shares
	s := symbol
	s = stripSuffix(s, ".SH")
	s = stripSuffix(s, ".SZ")
	if len(s) >= 2 && (s[0] == '6' || s[0] == '5' || s[0] == '9') {
		return "sh" + s
	}
	return "sz" + s
}

func stripSuffix(s, suffix string) string {
	if len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix {
		return s[:len(s)-len(suffix)]
	}
	return s
}

// parseTencentQuote parses Tencent's format.
//
//	Example: v_sh600519="1~茅台~600519~1950.00~1945.00~..."
func parseTencentQuote(symbol, body string) (*market.QuoteSnapshot, error) {
	// Tencent returns: v_<code>="<fields separated by ~>"
	start := 0
	for start < len(body) {
		if body[start] == '"' {
			start++
			break
		}
		start++
	}
	if start >= len(body) {
		return nil, fmt.Errorf("akshare: unexpected format")
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

	if len(fields) < 35 {
		// Try JSON parsing as fallback
		var result struct {
			Last      float64 `json:"last"`
			Open      float64 `json:"open"`
			High      float64 `json:"high"`
			Low       float64 `json:"low"`
			Volume    float64 `json:"volume"`
			Change    float64 `json:"change"`
			ChangePct float64 `json:"change_pct"`
		}
		if err := json.Unmarshal([]byte(content), &result); err != nil {
			return nil, fmt.Errorf("akshare: insufficient fields (%d)", len(fields))
		}
		return &market.QuoteSnapshot{
			Symbol:    symbol,
			Last:      result.Last,
			Open:      result.Open,
			High:      result.High,
			Low:       result.Low,
			Volume:    result.Volume,
			Change:    result.Change,
			ChangePct: result.ChangePct,
			Exchange:  "CN",
			Timestamp: time.Now().UnixMilli(),
		}, nil
	}

	// Tencent field mapping:
	// [0]=unknown, [1]=name, [2]=code, [3]=last, [4]=prevClose, [5]=open,
	// [6]=volume(手), [31]=high, [32]=low, [33]=high (alt), [34]=low (alt),
	// [37]=turnover/amount(元), [38]=turnoverRate(%), [39]=pe
	last := parseFloatSafe(fields[3])
	open := parseFloatSafe(fields[5])
	high := parseFloatSafe(fields[33])
	low := parseFloatSafe(fields[34])
	volume := parseFloatSafe(fields[6]) * 100
	prevClose := parseFloatSafe(fields[4])

	var turnover float64
	if len(fields) > 37 {
		turnover = parseFloatSafe(fields[37])
	}

	change := last - prevClose
	changePct := 0.0
	if prevClose > 0 {
		changePct = (change / prevClose) * 100
	}

	// Extract MarketCap and PE from Tencent response (available for CN stocks).
	// Field mapping (validated 2026-07):
	//   [44] = total market cap (亿, unit: 1e8 RMB)
	//   [39] = PE (TTM)
	var marketCap, pe float64
	if len(fields) > 44 {
		marketCap = parseFloatSafe(fields[44]) * 1e8 // 亿 → 元
	}
	if len(fields) > 39 {
		pe = parseFloatSafe(fields[39])
	}

	return &market.QuoteSnapshot{
		Symbol:    symbol,
		Name:      cleanName(fields[1]),
		Last:      last,
		Open:      open,
		High:      high,
		Low:       low,
		PrevClose: prevClose,
		Volume:    volume,
		Turnover:  turnover,
		Change:    change,
		ChangePct: changePct,
		MarketCap: marketCap,
		Pe:        pe,
		Exchange:  "CN",
		Timestamp: time.Now().UnixMilli(),
	}, nil
}

func splitTencent(s, sep string) []string {
	var result []string
	for {
		idx := 0
		for idx < len(s) {
			if idx+len(sep) <= len(s) && s[idx:idx+len(sep)] == sep {
				break
			}
			idx++
		}
		if idx > len(s) {
			result = append(result, s)
			break
		}
		result = append(result, s[:idx])
		if idx+len(sep) <= len(s) {
			s = s[idx+len(sep):]
		} else {
			break
		}
	}
	return result
}
