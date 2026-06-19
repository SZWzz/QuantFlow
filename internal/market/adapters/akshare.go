package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"quantflow/internal/market"
)

// AKShareAdapter fetches data via Tencent Finance HTTP API.
// Named "akshare" for the fallback chain. Uses Tencent's free HTTP interface
// which provides independent data redundancy from Sina/EastMoney.
type AKShareAdapter struct {
	client *http.Client
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

func (a *AKShareAdapter) FetchOHLCV(ctx context.Context, symbol string, interval string, start, end int64) ([]market.OHLCVBar, error) {
	return nil, fmt.Errorf("akshare: OHLCV not supported (real-time quotes only via Tencent backend, use tushare or yahoo for historical data)")
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
			Timestamp: time.Now().UnixMilli(),
		}, nil
	}

	// Tencent field mapping:
	// [0]=unknown, [1]=name, [2]=code, [3]=last, [4]=prevClose, [5]=open,
	// [6]=volume(手), [31]=high, [32]=low, [33]=high (alt), [34]=low (alt)
	last := parseFloatSafe(fields[3])
	open := parseFloatSafe(fields[5])
	high := parseFloatSafe(fields[33])
	low := parseFloatSafe(fields[34])
	volume := parseFloatSafe(fields[6]) * 100
	prevClose := parseFloatSafe(fields[4])

	change := last - prevClose
	changePct := 0.0
	if prevClose > 0 {
		changePct = (change / prevClose) * 100
	}

	return &market.QuoteSnapshot{
		Symbol:    symbol,
		Last:      last,
		Open:      open,
		High:      high,
		Low:       low,
		Volume:    volume,
		Change:    change,
		ChangePct: changePct,
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
