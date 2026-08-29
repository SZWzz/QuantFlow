package adapters

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"quantflow/internal/market"
	"quantflow/internal/normalize"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

// SinaAdapter fetches A-share and HK stock quotes from Sina Finance (free, no auth).
// Uses different endpoints and field mappings for CN vs HK stocks.
type SinaAdapter struct {
	client *http.Client
}

func NewSinaAdapter() *SinaAdapter {
	return &SinaAdapter{client: &http.Client{Timeout: 10 * time.Second}}
}

func (a *SinaAdapter) Name() string       { return "sina" }
func (a *SinaAdapter) Markets() []string  { return []string{"CN", "HK", "US"} }
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

	// Sina returns GBK-encoded content; convert to UTF-8 for proper Chinese name display.
	bodyStr, err := decodeGBK(body)
	if err != nil {
		if utf8.Valid(body) {
			bodyStr = string(body) // already valid UTF-8, use as-is
		} else {
			// Invalid GBK and not UTF-8 — return error so caller falls through to next adapter
			return nil, fmt.Errorf("sina: encoding decode failed for %s: %w", symbol, err)
		}
	}

	// Detect market in the response: "hk" = HK, "gb_" = US, else CN
	if strings.HasPrefix(code, "hk") {
		return parseSinaHKQuote(symbol, bodyStr)
	}
	if strings.HasPrefix(code, "gb_") {
		return parseSinaUSQuote(symbol, bodyStr)
	}
	return parseSinaQuote(symbol, bodyStr)
}

func (a *SinaAdapter) FetchDepth(ctx context.Context, symbol string) (*market.DepthSnapshot, error) {
	code := toSinaCode(symbol)

	url := fmt.Sprintf("http://hq.sinajs.cn/list=%s", code)
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	req.Header.Set("Referer", "https://finance.sina.com.cn/")
	resp, err := a.client.Do(req)
	if err != nil {
		return nil, market.NewTransientErrorf("sina depth: http error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("sina depth: HTTP %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))

	bodyStr, err := decodeGBK(body)
	if err != nil {
		if utf8.Valid(body) {
			bodyStr = string(body)
		} else {
			return nil, fmt.Errorf("sina depth: encoding decode failed for %s: %w", symbol, err)
		}
	}

	if strings.HasPrefix(code, "hk") {
		return parseSinaHKDepth(symbol, bodyStr)
	}
	if strings.HasPrefix(code, "gb_") {
		return nil, fmt.Errorf("sina depth: US depth not available")
	}
	return parseSinaDepth(symbol, bodyStr)
}

// parseSinaDepth parses 5-level bid/ask from Sina's quote response.
//
// Sina CN field mapping (0-indexed):
//
//	Level 1:      [6]=bid1_price, [7]=ask1_price, [10]=bid1_vol, [20]=ask1_vol
//	Bid levels 2-5: vol at [12,14,16,18], price at [13,15,17,19]
//	Ask levels 2-5: vol at [22,24,26,28], price at [23,25,27,29]
//	[11]=bid1_price (repeated, unused)
func parseSinaDepth(symbol, body string) (*market.DepthSnapshot, error) {
	idx := strings.Index(body, "\"")
	if idx == -1 {
		return nil, fmt.Errorf("sina depth: unexpected format")
	}
	content := body[idx+1:]
	endIdx := strings.LastIndex(content, "\"")
	if endIdx == -1 {
		return nil, fmt.Errorf("sina depth: unexpected format")
	}
	content = content[:endIdx]

	fields := strings.Split(content, ",")
	if len(fields) < 30 {
		return nil, fmt.Errorf("sina depth: insufficient fields (%d)", len(fields))
	}

	bids := make([]market.DepthLevel, 5)
	asks := make([]market.DepthLevel, 5)

	// Level 1
	bids[0] = market.DepthLevel{Price: parseFloatSafe(fields[6]), Size: normalize.NormalizeVolume("sina", parseFloatSafe(fields[10]))}
	asks[0] = market.DepthLevel{Price: parseFloatSafe(fields[7]), Size: normalize.NormalizeVolume("sina", parseFloatSafe(fields[20]))}

	// Bid levels 2-5: [vol, price] pairs
	for i := 1; i < 5; i++ {
		volIdx := 12 + (i-1)*2
		priceIdx := volIdx + 1
		bids[i] = market.DepthLevel{Price: parseFloatSafe(fields[priceIdx]), Size: normalize.NormalizeVolume("sina", parseFloatSafe(fields[volIdx]))}
	}

	// Ask levels 2-5: [vol, price] pairs
	for i := 1; i < 5; i++ {
		volIdx := 22 + (i-1)*2
		priceIdx := volIdx + 1
		asks[i] = market.DepthLevel{Price: parseFloatSafe(fields[priceIdx]), Size: normalize.NormalizeVolume("sina", parseFloatSafe(fields[volIdx]))}
	}

	return &market.DepthSnapshot{
		Symbol:    symbol,
		Bids:      bids,
		Asks:      asks,
		Timestamp: time.Now().UnixMilli(),
	}, nil
}

// parseSinaHKDepth parses depth from Sina's HK quote response.
func parseSinaHKDepth(symbol, body string) (*market.DepthSnapshot, error) {
	idx := strings.Index(body, "\"")
	if idx == -1 {
		return nil, fmt.Errorf("sina HK depth: unexpected format")
	}
	content := body[idx+1:]
	endIdx := strings.LastIndex(content, "\"")
	if endIdx == -1 {
		return nil, fmt.Errorf("sina HK depth: unexpected format")
	}
	content = content[:endIdx]

	fields := strings.Split(content, ",")
	if len(fields) < 13 {
		return nil, fmt.Errorf("sina HK depth: insufficient fields (%d)", len(fields))
	}

	// HK only has single bid/ask at [9]=bid, [10]=ask, no 5-level
	bid := parseFloatSafe(fields[9])
	ask := parseFloatSafe(fields[10])
	if bid <= 0 || ask <= 0 || ask <= bid {
		return nil, fmt.Errorf("sina HK depth: invalid bid/ask")
	}

	bids := make([]market.DepthLevel, 5)
	asks := make([]market.DepthLevel, 5)
	step := (ask - bid) / 5
	for i := 0; i < 5; i++ {
		bids[i] = market.DepthLevel{Price: bid - float64(i)*step, Size: 0}
		asks[i] = market.DepthLevel{Price: ask + float64(i)*step, Size: 0}
	}

	return &market.DepthSnapshot{
		Symbol:    symbol,
		Bids:      bids,
		Asks:      asks,
		Timestamp: time.Now().UnixMilli(),
	}, nil
}

func (a *SinaAdapter) FetchOHLCV(ctx context.Context, symbol string, interval string, _ string, start, end int64) ([]market.OHLCVBar, error) {
	// Sina doesn't have a public OHLCV endpoint. Use Tencent K-line via AkShare adapter instead.
	return nil, fmt.Errorf("sina: OHLCV not supported (real-time quotes only, use akshare or yahoo for historical data)")
}

func (a *SinaAdapter) HealthCheck(ctx context.Context) error {
	_, err := a.FetchQuote(ctx, "600519")
	return err
}

// ── Sina US-specific helpers ───────────────────────────────────────────

// toSinaUSCode converts a US stock symbol to Sina's US format.
// "AAPL" → "gb_aapl", "MSFT" → "gb_msft"
func toSinaUSCode(symbol string) string {
	return "gb_" + strings.ToLower(symbol)
}

// parseSinaUSQuote parses Sina's US stock response format.
//
// Example: var hq_str_gb_aapl="Apple,294.51,0.07,2026-06-24 22:00:41,0.21,295.35,296.47,293.20,...";
//
// US field mapping (0-indexed):
//
//	[0]=name, [1]=last, [2]=changePct, [3]=time,
//	[4]=change, [5]=open, [6]=high, [7]=low,
//	[10]=volume
func parseSinaUSQuote(symbol, body string) (*market.QuoteSnapshot, error) {
	idx := strings.Index(body, "\"")
	if idx == -1 {
		return nil, fmt.Errorf("sina US: unexpected response format")
	}
	content := body[idx+1:]
	endIdx := strings.LastIndex(content, "\"")
	if endIdx == -1 {
		return nil, fmt.Errorf("sina US: unexpected response format")
	}
	content = content[:endIdx]

	fields := strings.Split(content, ",")
	if len(fields) < 11 {
		return nil, fmt.Errorf("sina US: insufficient fields: got %d", len(fields))
	}

	last := parseFloatSafe(fields[1])
	changePct := parseFloatSafe(fields[2])
	change := parseFloatSafe(fields[4])
	open := parseFloatSafe(fields[5])
	high := parseFloatSafe(fields[6])
	low := parseFloatSafe(fields[7])
	volume := parseFloatSafe(fields[10])

	return &market.QuoteSnapshot{
		Symbol:    symbol,
		Name:      fields[0],
		Last:      last,
		Open:      open,
		High:      high,
		Low:       low,
		Volume:    volume,
		Change:    change,
		ChangePct: changePct,
		Exchange:  "US",
		Timestamp: time.Now().UnixMilli(),
	}, nil
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
	turnover := parseFloatSafe(fields[11])
	volume := parseFloatSafe(fields[12])

	// If last is 0 (suspended/not trading), fall back to prevClose
	if last == 0 {
		last = prevClose
	}

	return &market.QuoteSnapshot{
		Symbol:    symbol,
		Name:      cleanName(fields[1]), // Chinese name
		Last:      last,
		Open:      open,
		High:      high,
		Low:       low,
		PrevClose: prevClose,
		Bid:       bid,
		Ask:       ask,
		Volume:    volume,
		Turnover:  turnover,
		Change:    change,
		ChangePct: changePct,
		Exchange:  "CN",
		Timestamp: time.Now().UnixMilli(),
	}, nil
}

// cleanName validates a stock name is valid UTF-8 and returns it.
// Invalid UTF-8 bytes are replaced with U+FFFD to avoid garbled display.
func cleanName(name string) string {
	if utf8.ValidString(name) {
		return name
	}
	var b strings.Builder
	b.Grow(len(name))
	for _, r := range name {
		if r == utf8.RuneError {
			b.WriteRune('?')
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// decodeGBK converts a GBK/GB2312-encoded byte slice to a UTF-8 string.
// Sina Finance returns content in GBK encoding for Chinese stock names.
func decodeGBK(data []byte) (string, error) {
	reader := transform.NewReader(strings.NewReader(string(data)), simplifiedchinese.GBK.NewDecoder())
	decoded, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}
	out := string(decoded)
	if !utf8.ValidString(out) {
		return "", fmt.Errorf("GBK decoded output is still invalid UTF-8")
	}
	return out, nil
}
