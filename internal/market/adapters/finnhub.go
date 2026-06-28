package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"quantflow/internal/market"
)

const finnhubBaseURL = "https://finnhub.io/api/v1"

// FinnhubAdapter fetches US stock data from Finnhub (free tier: 60 req/min).
// Requires FINNHUB_API_KEY env var (free registration at finnhub.io).
// Provides real-time quotes and OHLCV for US equities.
type FinnhubAdapter struct {
	client *http.Client
	apiKey string
}

// NewFinnhubAdapter creates a new Finnhub adapter.
// Reads FINNHUB_API_KEY from environment. Returns adapter even without key
// (IsAvailable will return false gracefully).
func NewFinnhubAdapter() *FinnhubAdapter {
	return &FinnhubAdapter{
		client: &http.Client{Timeout: 10 * time.Second},
		apiKey: os.Getenv("FINNHUB_API_KEY"),
	}
}

// SetAPIKey updates the Finnhub API key from app config.
func (a *FinnhubAdapter) SetAPIKey(key string) {
	if key != "" {
		a.apiKey = key
	}
}

func (a *FinnhubAdapter) Name() string      { return "finnhub" }
func (a *FinnhubAdapter) Markets() []string  { return []string{"US"} }
func (a *FinnhubAdapter) RequiresAuth() bool { return true }

func (a *FinnhubAdapter) IsAvailable(ctx context.Context) bool {
	if a.apiKey == "" {
		return false
	}
	req, _ := http.NewRequestWithContext(ctx, "GET",
		finnhubBaseURL+"/quote?symbol=AAPL&token="+a.apiKey, nil)
	resp, err := a.client.Do(req)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// FetchQuote returns real-time US stock quote from Finnhub.
func (a *FinnhubAdapter) FetchQuote(ctx context.Context, symbol string) (*market.QuoteSnapshot, error) {
	if a.apiKey == "" {
		return nil, fmt.Errorf("finnhub: API key not configured (set FINNHUB_API_KEY)")
	}

	url := fmt.Sprintf("%s/quote?symbol=%s&token=%s", finnhubBaseURL, symbol, a.apiKey)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("finnhub: %w", err)
	}

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, market.NewTransientErrorf("finnhub: http error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, market.NewTransientErrorf("finnhub: rate limited (60/min free tier)")
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("finnhub: HTTP %d: %s", resp.StatusCode, string(body))
	}

	var q struct {
		Current     float64 `json:"c"`
		High        float64 `json:"h"`
		Low         float64 `json:"l"`
		Open        float64 `json:"o"`
		PrevClose   float64 `json:"pc"`
		Timestamp   int64   `json:"t"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&q); err != nil {
		return nil, fmt.Errorf("finnhub: parse error: %w", err)
	}

	change := q.Current - q.PrevClose
	changePct := 0.0
	if q.PrevClose > 0 {
		changePct = (change / q.PrevClose) * 100
	}

	return &market.QuoteSnapshot{
		Symbol:    symbol,
		Last:      q.Current,
		Open:      q.Open,
		High:      q.High,
		Low:       q.Low,
		Volume:    0, // Finnhub quote endpoint doesn't include volume
		Change:    change,
		ChangePct: changePct,
		Timestamp: q.Timestamp * 1000, // Finnhub uses Unix seconds
	}, nil
}

// FetchOHLCV fetches historical OHLCV from Finnhub's stock/candle endpoint.
func (a *FinnhubAdapter) FetchOHLCV(ctx context.Context, symbol string, interval string, _ string, start, end int64) ([]market.OHLCVBar, error) {
	if a.apiKey == "" {
		return nil, fmt.Errorf("finnhub: API key not configured (set FINNHUB_API_KEY)")
	}

	// Map interval to Finnhub resolution
	resolution := "D"
	switch interval {
	case "1m", "5m", "15m", "30m", "60m", "1h":
		resolution = interval
	case "1wk", "1w", "week":
		resolution = "W"
	case "1mo", "1M", "month":
		resolution = "M"
	}

	url := fmt.Sprintf("%s/stock/candle?symbol=%s&resolution=%s&from=%d&to=%d&token=%s",
		finnhubBaseURL, symbol, resolution, start, end, a.apiKey)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("finnhub OHLCV: %w", err)
	}

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, market.NewTransientErrorf("finnhub OHLCV: http error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("finnhub OHLCV: HTTP %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Close     []float64 `json:"c"`
		High      []float64 `json:"h"`
		Low       []float64 `json:"l"`
		Open      []float64 `json:"o"`
		Volume    []float64 `json:"v"`
		Timestamp []int64   `json:"t"`
		Status    string    `json:"s"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("finnhub OHLCV parse: %w", err)
	}
	if result.Status == "no_data" {
		return nil, fmt.Errorf("finnhub OHLCV: no data for %s", symbol)
	}

	bars := make([]market.OHLCVBar, 0, len(result.Timestamp))
	for i, ts := range result.Timestamp {
		if i >= len(result.Open) || i >= len(result.Close) {
			break
		}
		bars = append(bars, market.OHLCVBar{
			Symbol: symbol,
			Date:   time.Unix(ts, 0).Format("2006-01-02"),
			Open:   result.Open[i],
			High:   result.High[i],
			Low:    result.Low[i],
			Close:  result.Close[i],
			Volume: result.Volume[i],
		})
	}
	return bars, nil
}

func (a *FinnhubAdapter) HealthCheck(ctx context.Context) error {
	_, err := a.FetchQuote(ctx, "AAPL")
	return err
}

// ShortInterestData represents Finnhub short interest snapshot for a US stock.
type ShortInterestData struct {
	Symbol         string  `json:"symbol"`
	Date           string  `json:"date"`
	ShortInterest  float64 `json:"short_interest"`
	AvgDailyVolume float64 `json:"avg_daily_volume"`
	DaysToCover    float64 `json:"days_to_cover"`
	ShortPercent   float64 `json:"short_pct"`
}

// FetchShortInterest returns short interest data for a US stock symbol.
// Calls GET /stock/short-interest?symbol=X.
// Free tier returns the most recent 12 monthly snapshots.
func (a *FinnhubAdapter) FetchShortInterest(ctx context.Context, symbol string) ([]ShortInterestData, error) {
	if a.apiKey == "" {
		return nil, fmt.Errorf("finnhub: API key not configured")
	}
	url := fmt.Sprintf("%s/stock/short-interest?symbol=%s&token=%s", finnhubBaseURL, symbol, a.apiKey)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("finnhub short_interest: %w", err)
	}
	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("finnhub short_interest: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("finnhub short_interest: HTTP %d", resp.StatusCode)
	}
	var result struct {
		Data []struct {
			Symbol         string  `json:"symbol"`
			Date           string  `json:"date"`
			ShortInterest  float64 `json:"shortInterest"`
			AvgDailyVolume float64 `json:"avgDailyVolume"`
			DaysToCover    float64 `json:"daysToCover"`
			ShortPercent   float64 `json:"shortPercent"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("finnhub short_interest parse: %w", err)
	}
	out := make([]ShortInterestData, 0, len(result.Data))
	for _, d := range result.Data {
		out = append(out, ShortInterestData{
			Symbol:         d.Symbol,
			Date:           d.Date,
			ShortInterest:  d.ShortInterest,
			AvgDailyVolume: d.AvgDailyVolume,
			DaysToCover:    d.DaysToCover,
			ShortPercent:   d.ShortPercent,
		})
	}
	return out, nil
}

// ── Option Chain ─────────────────────────────────────────────────────

// OptionChainItem holds a single option chain contract.
type OptionChainItem struct {
	Expiry       string  `json:"expiry"`
	Strike       float64 `json:"strike"`
	Type         string  `json:"type"`
	Bid          float64 `json:"bid"`
	Ask          float64 `json:"ask"`
	Last         float64 `json:"last"`
	Volume       int     `json:"volume"`
	OpenInterest int     `json:"open_interest"`
	ImpliedVol   float64 `json:"implied_vol"`
	Delta        float64 `json:"delta"`
	Gamma        float64 `json:"gamma"`
	Theta        float64 `json:"theta"`
	Vega         float64 `json:"vega"`
}

func (a *FinnhubAdapter) FetchOptionChain(ctx context.Context, symbol string) ([]OptionChainItem, error) {
	if a.apiKey == "" {
		return nil, fmt.Errorf("finnhub: API key not configured")
	}
	url := fmt.Sprintf("%s/stock/option-chain?symbol=%s&token=%s", finnhubBaseURL, symbol, a.apiKey)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("finnhub option: %w", err)
	}
	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("finnhub option: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("finnhub option: HTTP %d", resp.StatusCode)
	}
	var result struct {
		Data []struct {
			Expiry       string  `json:"expirationDate"`
			Strike       float64 `json:"strike"`
			Type         string  `json:"optionType"`
			Bid          float64 `json:"bid"`
			Ask          float64 `json:"ask"`
			Last         float64 `json:"lastPrice"`
			Volume       int     `json:"volume"`
			OpenInterest int     `json:"openInterest"`
			ImpliedVol   float64 `json:"impliedVolatility"`
			Delta        float64 `json:"delta"`
			Gamma        float64 `json:"gamma"`
			Theta        float64 `json:"theta"`
			Vega         float64 `json:"vega"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("finnhub option parse: %w", err)
	}
	out := make([]OptionChainItem, 0, len(result.Data))
	for _, o := range result.Data {
		out = append(out, OptionChainItem{
			Expiry:       o.Expiry,
			Strike:       o.Strike,
			Type:         o.Type,
			Bid:          o.Bid,
			Ask:          o.Ask,
			Last:         o.Last,
			Volume:       o.Volume,
			OpenInterest: o.OpenInterest,
			ImpliedVol:   o.ImpliedVol,
			Delta:        o.Delta,
			Gamma:        o.Gamma,
			Theta:        o.Theta,
			Vega:         o.Vega,
		})
	}
	return out, nil
}

// ── SEC Filings ──────────────────────────────────────────────────────

// FinnhubSECFiling holds a single SEC filing record from Finnhub.
type FinnhubSECFiling struct {
	Symbol      string `json:"symbol"`
	Form        string `json:"form"`
	Date        string `json:"date"`
	Description string `json:"description"`
	URL         string `json:"url"`
	Filer       string `json:"filer"`
}

func (a *FinnhubAdapter) FetchSECFilings(ctx context.Context, symbol string) ([]FinnhubSECFiling, error) {
	if a.apiKey == "" {
		return nil, fmt.Errorf("finnhub: API key not configured")
	}
	url := fmt.Sprintf("%s/stock/filings?symbol=%s&token=%s", finnhubBaseURL, symbol, a.apiKey)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("finnhub filings: %w", err)
	}
	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("finnhub filings: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("finnhub filings: HTTP %d", resp.StatusCode)
	}
	var result []struct {
		Symbol      string `json:"symbol"`
		Form        string `json:"form"`
		Date        string `json:"filedDate"`
		Description string `json:"description"`
		URL         string `json:"reportUrl"`
		Filer       string `json:"filingOwner"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("finnhub filings parse: %w", err)
	}
	out := make([]FinnhubSECFiling, 0, len(result))
	for _, f := range result {
		out = append(out, FinnhubSECFiling{
			Symbol:      f.Symbol,
			Form:        f.Form,
			Date:        f.Date,
			Description: f.Description,
			URL:         f.URL,
			Filer:       f.Filer,
		})
	}
	return out, nil
}

// ── Earnings Calendar ────────────────────────────────────────────────

type EarningsEvent struct {
	Symbol          string  `json:"symbol"`
	Date            string  `json:"date"`
	Hour            string  `json:"hour"`  // bmo=before market open, amc=after close
	Quarter         int     `json:"quarter"`
	Year            int     `json:"year"`
	EPSActual       float64 `json:"eps_actual"`
	EPSEstimate     float64 `json:"eps_estimate"`
	RevenueActual   float64 `json:"revenue_actual"`
	RevenueEstimate float64 `json:"revenue_estimate"`
}

// FetchEarningsCalendar returns upcoming earnings events (max ~3-6 months on free tier).
// Calls GET /calendar/earnings with optional from/to dates (YYYY-MM-DD).
func (a *FinnhubAdapter) FetchEarningsCalendar(ctx context.Context, from, to string) ([]EarningsEvent, error) {
	if a.apiKey == "" {
		return nil, fmt.Errorf("finnhub: API key not configured")
	}
	url := fmt.Sprintf("%s/calendar/earnings?token=%s", finnhubBaseURL, a.apiKey)
	if from != "" {
		url += "&from=" + from
	}
	if to != "" {
		url += "&to=" + to
	}
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("finnhub earnings: %w", err)
	}
	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("finnhub earnings: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("finnhub earnings: HTTP %d", resp.StatusCode)
	}
	var result struct {
		EarningsCalendar []struct {
			Symbol          string  `json:"symbol"`
			Date            string  `json:"date"`
			Hour            string  `json:"hour"`
			Quarter         int     `json:"quarter"`
			Year            int     `json:"year"`
			EPSActual       float64 `json:"epsActual"`
			EPSEstimate     float64 `json:"epsEstimate"`
			RevenueActual   float64 `json:"revenueActual"`
			RevenueEstimate float64 `json:"revenueEstimate"`
		} `json:"earningsCalendar"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("finnhub earnings parse: %w", err)
	}
	out := make([]EarningsEvent, 0, len(result.EarningsCalendar))
	for _, e := range result.EarningsCalendar {
		out = append(out, EarningsEvent{
			Symbol:          e.Symbol,
			Date:            e.Date,
			Hour:            e.Hour,
			Quarter:         e.Quarter,
			Year:            e.Year,
			EPSActual:       e.EPSActual,
			EPSEstimate:     e.EPSEstimate,
			RevenueActual:   e.RevenueActual,
			RevenueEstimate: e.RevenueEstimate,
		})
	}
	return out, nil
}
