package adapters

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"quantflow/internal/market"
)

// SinaAdapter fetches A-share and HK stock quotes from Sina Finance (free, no auth).
// Uses different endpoints and field mappings for CN vs HK stocks.
type SinaAdapter struct {
	client *http.Client
}

func NewSinaAdapter() *SinaAdapter {
	return &SinaAdapter{client: &http.Client{Timeout: 10 * time.Second}}
}

func (a *SinaAdapter) Name() string      { return "sina" }
func (a *SinaAdapter) Markets() []string  { return []string{"CN", "HK"} }
func (a *SinaAdapter) RequiresAuth() bool { return false }

func (a *SinaAdapter) IsAvailable(ctx context.Context) bool {
	req, _ := http.NewRequestWithContext(ctx, "GET", "http://hq.sinajs.cn/list=sh600519", nil)
	req.Header.Set("Referer", "https://finance.sina.com.cn/")
	resp, err := a.client.Do(req)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// FetchQuote fetches real-time quote for CN A-shares or HK stocks.
func (a *SinaAdapter) FetchQuote(ctx context.Context, symbol string) (*market.QuoteSnapshot, error) {
	code := toSinaCode(symbol)

	url := fmt.Sprintf("http://hq.sinajs.cn/list=%s", code)
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	req.Header.Set("Referer", "https://finance.sina.com.cn/")
	resp, err := a.client.Do(req)
	if err != nil {
		return nil, market.NewTransientErrorf("sina: http error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("sina: HTTP %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))

	// Detect HK vs CN in the response: HK codes start with "hk" prefix
	if strings.HasPrefix(code, "hk") {
		return parseSinaHKQuote(symbol, string(body))
	}
	return parseSinaQuote(symbol, string(body))
}

func (a *SinaAdapter) FetchOHLCV(ctx context.Context, symbol string, interval string, start, end int64) ([]market.OHLCVBar, error) {
	// Sina doesn't have a public OHLCV endpoint. Use Tencent K-line via AkShare adapter instead.
	return nil, fmt.Errorf("sina: OHLCV not supported (real-time quotes only, use akshare or yahoo for historical data)")
}

func (a *SinaAdapter) HealthCheck(ctx context.Context) error {
	_, err := a.FetchQuote(ctx, "600519")
	return err
}

// ── Sina HK-specific helpers ───────────────────────────────────────────

// toSinaHKCode converts an HK symbol to Sina's HK format.
// "00700" → "hk00700", "00700.HK" → "hk00700"
func toSinaHKCode(symbol string) string {
	s := strings.ToUpper(symbol)
	s = strings.TrimSuffix(s, ".HK")
	// Remove leading zeros for Sina HK format? No — Sina uses hk00700 with leading zeros.
	return "hk" + s
}

// parseSinaHKQuote parses Sina's HK stock response format.
//
// Example: var hq_str_hk00700="TENCENT,腾讯控股,440.000,445.400,446.200,435.600,440.200,-5.200,-1.167,440.00000,440.20001,13215630970,30119117,0,0,675.134,420.400,2026/06/18,16:08";
//
// HK field mapping:
//
//	[0]=engName, [1]=name, [2]=open, [3]=prevClose, [4]=high, [5]=low,
//	[6]=last, [7]=change, [8]=changePct, [9]=bid, [10]=ask,
//	[11]=turnover, [12]=volume, [13]=?, [14]=?, [15]=52wHigh, [16]=52wLow, [17]=date, [18]=time
func parseSinaHKQuote(symbol, body string) (*market.QuoteSnapshot, error) {
	idx := strings.Index(body, "\"")
	if idx == -1 {
		return nil, fmt.Errorf("sina HK: unexpected response format")
	}
	content := body[idx+1:]
	endIdx := strings.LastIndex(content, "\"")
	if endIdx == -1 {
		return nil, fmt.Errorf("sina HK: unexpected response format")
	}
	content = content[:endIdx]

	fields := strings.Split(content, ",")
	if len(fields) < 13 {
		return nil, fmt.Errorf("sina HK: insufficient fields: got %d", len(fields))
	}

	open := parseFloatSafe(fields[2])
	prevClose := parseFloatSafe(fields[3])
	high := parseFloatSafe(fields[4])
	low := parseFloatSafe(fields[5])
	last := parseFloatSafe(fields[6])
	change := parseFloatSafe(fields[7])
	changePct := parseFloatSafe(fields[8])
	bid := parseFloatSafe(fields[9])
	ask := parseFloatSafe(fields[10])
	volume := parseFloatSafe(fields[12])

	// If last is 0 (suspended/not trading), fall back to prevClose
	if last == 0 {
		last = prevClose
	}

	return &market.QuoteSnapshot{
		Symbol:    symbol,
		Last:      last,
		Open:      open,
		High:      high,
		Low:       low,
		Bid:       bid,
		Ask:       ask,
		Volume:    volume,
		Change:    change,
		ChangePct: changePct,
		Timestamp: time.Now().UnixMilli(),
	}, nil
}

