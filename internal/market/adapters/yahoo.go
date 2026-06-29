package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"sync"
	"time"

	"quantflow/internal/market"
)

const (
	yahooChartURL = "https://query2.finance.yahoo.com/v8/finance/chart"
	yahooCrumbURL = "https://query2.finance.yahoo.com/v1/test/getcrumb"
	// Fallback endpoint when query2 is unreachable (common from China).
	yahooFallbackURL = "https://query1.finance.yahoo.com/v8/finance/chart"
)

// YahooAdapter fetches market data from Yahoo Finance (free, no auth).
// Handles crumb-based cookie auth required by Yahoo's v8 chart API.
type YahooAdapter struct {
	client  *http.Client
	crumb   string
	crumbMu sync.RWMutex
	crumbAt time.Time
}

// NewYahooAdapter creates a new Yahoo Finance adapter.
func NewYahooAdapter() *YahooAdapter {
	jar, _ := cookiejar.New(nil)
	return &YahooAdapter{
		client: &http.Client{
			Timeout: 10 * time.Second,
			Jar:     jar,
		},
	}
}

func (a *YahooAdapter) Name() string      { return "yahoo" }
func (a *YahooAdapter) Markets() []string  { return []string{"US", "HK", "CN"} }
func (a *YahooAdapter) RequiresAuth() bool { return false }

// IsAvailable checks if Yahoo Finance is reachable.
func (a *YahooAdapter) IsAvailable(ctx context.Context) bool {
	req, err := http.NewRequestWithContext(ctx, "GET", yahooChartURL+"/AAPL?interval=1d&range=1d", nil)
	if err != nil {
		return false
	}
	a.setHeaders(req)
	resp, err := a.client.Do(req)
	if err != nil {
		// Try fallback
		req2, _ := http.NewRequestWithContext(ctx, "GET", yahooFallbackURL+"/AAPL?interval=1d&range=1d", nil)
		a.setHeaders(req2)
		resp2, err2 := a.client.Do(req2)
		if err2 != nil {
			return false
		}
		resp2.Body.Close()
		return resp2.StatusCode == http.StatusOK
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// getCrumb fetches a fresh crumb from Yahoo. Crumb expires after ~1 hour.
func (a *YahooAdapter) getCrumb(ctx context.Context) (string, error) {
	a.crumbMu.RLock()
	if a.crumb != "" && time.Since(a.crumbAt) < 30*time.Minute {
		defer a.crumbMu.RUnlock()
		return a.crumb, nil
	}
	a.crumbMu.RUnlock()

	a.crumbMu.Lock()
	defer a.crumbMu.Unlock()

	// Double-check after acquiring write lock
	if a.crumb != "" && time.Since(a.crumbAt) < 30*time.Minute {
		return a.crumb, nil
	}

	req, err := http.NewRequestWithContext(ctx, "GET", yahooCrumbURL, nil)
	if err != nil {
		return "", err
	}
	a.setHeaders(req)

	resp, err := a.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("crumb request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 256))
	if err != nil {
		return "", err
	}
	crumb := strings.TrimSpace(string(body))
	if crumb == "" || resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("crumb unavailable (HTTP %d): %s", resp.StatusCode, string(body))
	}

	a.crumb = crumb
	a.crumbAt = time.Now()
	return crumb, nil
}

// setHeaders adds the required headers for Yahoo Finance API requests.
func (a *YahooAdapter) setHeaders(req *http.Request) {
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Referer", "https://finance.yahoo.com/")
}

// FetchQuote builds a quote from recent OHLCV data.
func (a *YahooAdapter) FetchQuote(ctx context.Context, symbol string) (*market.QuoteSnapshot, error) {
	bars, err := a.FetchOHLCV(ctx, symbol, "1d", "", time.Now().AddDate(0, 0, -5).Unix(), time.Now().Unix())
	if err != nil {
		return nil, fmt.Errorf("yahoo: %w", err)
	}
	if len(bars) < 2 {
		return nil, fmt.Errorf("yahoo: insufficient data for %s", symbol)
	}

	last := bars[len(bars)-1]
	prev := bars[len(bars)-2]
	change := last.Close - prev.Close
	changePct := 0.0
	if prev.Close > 0 {
		changePct = (change / prev.Close) * 100
	}

	return &market.QuoteSnapshot{
		Symbol:    symbol,
		Last:      last.Close,
		Open:      last.Open,
		High:      last.High,
		Low:       last.Low,
		PrevClose: prev.Close,
		Volume:    last.Volume,
		Change:    change,
		ChangePct: changePct,
		Exchange:  "US",
		Timestamp: time.Now().UnixMilli(),
	}, nil
}

// FetchOHLCV fetches OHLCV bars from Yahoo Finance v8 chart API.
func (a *YahooAdapter) FetchOHLCV(ctx context.Context, symbol string, interval string, _ string, start, end int64) ([]market.OHLCVBar, error) {
	// Normalize symbol for Yahoo: Yahoo uses .HK suffix for Hong Kong stocks.
	// If symbol is a 5-digit HK code (e.g., "00700"), convert to "0700.HK".
	yahooSymbol := normalizeYahooSymbol(symbol)

	crumb, err := a.getCrumb(ctx)
	if err != nil {
		// Crumb is optional — some endpoints work without it. Log and continue.
		crumb = ""
	}

	bars, err := a.fetchWithBase(ctx, yahooChartURL, yahooSymbol, interval, start, end, crumb)
	if err != nil {
		// Fallback to query1
		bars, err = a.fetchWithBase(ctx, yahooFallbackURL, yahooSymbol, interval, start, end, crumb)
		if err != nil {
			return nil, fmt.Errorf("yahoo: %w", err)
		}
	}
	return bars, nil
}

func (a *YahooAdapter) fetchWithBase(ctx context.Context, baseURL, symbol, interval string, start, end int64, crumb string) ([]market.OHLCVBar, error) {
	url := fmt.Sprintf("%s/%s?interval=%s&period1=%d&period2=%d&includePrePost=false&events=history",
		baseURL, symbol, interval, start, end)
	if crumb != "" {
		url += "&crumb=" + crumb
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("yahoo: %w", err)
	}
	a.setHeaders(req)

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, market.NewTransientErrorf("yahoo: http error: %v", err)
	}
	defer resp.Body.Close()

	// Yahoo sometimes returns HTML error pages with 200 OK.
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1MB max
	if isHTMLBody(body) {
		return nil, fmt.Errorf("yahoo: received HTML response (likely geo-blocked or rate-limited), try alternate endpoint")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("yahoo: HTTP %d: %s", resp.StatusCode, truncate(body, 200))
	}

	var result yahooChartResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("yahoo: parse error: %w (body preview: %s)", err, truncate(body, 100))
	}

	if result.Chart.Error != nil {
		return nil, fmt.Errorf("yahoo: API error: %s", result.Chart.Error.Description)
	}

	if len(result.Chart.Result) == 0 {
		return nil, fmt.Errorf("yahoo: no results for %s", symbol)
	}

	r := result.Chart.Result[0]
	timestamps := r.Timestamp
	quote := r.Indicators.Quote[0]
	opens := quote.Open
	highs := quote.High
	lows := quote.Low
	closes := quote.Close
	volumes := quote.Volume

	bars := make([]market.OHLCVBar, 0, len(timestamps))
	for i, ts := range timestamps {
		if i >= len(opens) || i >= len(closes) {
			break
		}
		if opens[i] == 0 && closes[i] == 0 && highs[i] == 0 && lows[i] == 0 {
			continue
		}
		bars = append(bars, market.OHLCVBar{
			Symbol: symbol,
			Date:   time.Unix(ts, 0).Format("2006-01-02"),
			Open:   safeFloat(opens, i),
			High:   safeFloat(highs, i),
			Low:    safeFloat(lows, i),
			Close:  safeFloat(closes, i),
			Volume: safeFloat(volumes, i),
		})
	}

	return bars, nil
}

func (a *YahooAdapter) HealthCheck(ctx context.Context) error {
	_, err := a.FetchQuote(ctx, "AAPL")
	return err
}

// --- helpers ---

// normalizeYahooSymbol converts QuantFlow symbols to Yahoo Finance format.
// "00700" → "0700.HK", "00005" → "0005.HK"
func normalizeYahooSymbol(symbol string) string {
	// Already has suffix: AAPL, TSLA, 0700.HK, 600519.SS
	if strings.Contains(symbol, ".") {
		return symbol
	}
	// 5-digit HK stock code → strip leading zeros, add .HK
	if len(symbol) == 5 && isDigit(symbol) {
		return strings.TrimLeft(symbol, "0") + ".HK"
	}
	// 6-digit CN stock code
	if len(symbol) == 6 && isDigit(symbol) {
		switch {
		case symbol[0] == '6':
			return symbol + ".SS" // Shanghai
		case symbol[0] == '0' || symbol[0] == '3':
			return symbol + ".SZ" // Shenzhen
		default:
			return symbol + ".SS"
		}
	}
	return symbol
}

func isDigit(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func isHTMLBody(body []byte) bool {
	s := strings.TrimSpace(string(body))
	return strings.HasPrefix(s, "<!") || strings.HasPrefix(s, "<html")
}

func safeFloat(arr []float64, i int) float64 {
	if i < len(arr) {
		return arr[i]
	}
	return 0
}

func truncate(body []byte, n int) string {
	s := string(body)
	if len(s) > n {
		return s[:n] + "..."
	}
	return s
}

// --- response types ---

type yahooChartResponse struct {
	Chart yahooChart `json:"chart"`
}

type yahooChart struct {
	Result []yahooResult `json:"result"`
	Error  *yahooError   `json:"error"`
}

type yahooResult struct {
	Timestamp  []int64         `json:"timestamp"`
	Indicators yahooIndicators `json:"indicators"`
}

type yahooIndicators struct {
	Quote []yahooQuote `json:"quote"`
}

type yahooQuote struct {
	Open   []float64 `json:"open"`
	High   []float64 `json:"high"`
	Low    []float64 `json:"low"`
	Close  []float64 `json:"close"`
	Volume []float64 `json:"volume"`
}

type yahooError struct {
	Code        string `json:"code"`
	Description string `json:"description"`
}
