// internal/market/adapters/polymarket.go
package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const polymarketBaseURL = "https://gamma-api.polymarket.com"

// PolymarketAdapter defines the interface for prediction market data sources.
// Separate from market.Adapter because prediction markets don't have
// quotes/OHLCV — they carry event probabilities and price histories.
type PolymarketAdapter interface {
	Name() string
	IsAvailable(ctx context.Context) bool
	RequiresAuth() bool

	// FetchEvents returns prediction market events, optionally filtered by category.
	// category can be "" to return all active markets. limit caps results (max 100).
	FetchEvents(ctx context.Context, category string, limit int) ([]PredictionEvent, error)

	// FetchEvent returns a single event with full outcome details.
	FetchEvent(ctx context.Context, id string) (*PredictionEvent, error)

	// FetchPriceHistory returns historical prices for an outcome.
	// interval: "1h", "6h", "1d".
	FetchPriceHistory(ctx context.Context, outcomeID string, interval string, limit int) ([]PricePoint, error)
}

// PolymarketHTTPAdapter fetches prediction market data from Polymarket's
// public Gamma API (free, no API key required).
type PolymarketHTTPAdapter struct {
	client *http.Client
}

// Compile-time interface check.
var _ PolymarketAdapter = (*PolymarketHTTPAdapter)(nil)

// NewPolymarketAdapter creates a new Polymarket HTTP adapter.
func NewPolymarketAdapter() *PolymarketHTTPAdapter {
	return &PolymarketHTTPAdapter{
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

func (a *PolymarketHTTPAdapter) Name() string       { return "polymarket" }
func (a *PolymarketHTTPAdapter) RequiresAuth() bool { return false }

func (a *PolymarketHTTPAdapter) IsAvailable(ctx context.Context) bool {
	req, _ := http.NewRequestWithContext(ctx, "GET",
		polymarketBaseURL+"/markets?limit=1", nil)
	req.Header.Set("Accept", "application/json")
	resp, err := a.client.Do(req)
	if err != nil {
		slog.Debug("polymarket availability check failed", "error", err)
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// FetchEvents returns prediction market events from Polymarket.
// category maps to Polymarket's "tag" query parameter.
func (a *PolymarketHTTPAdapter) FetchEvents(ctx context.Context, category string, limit int) ([]PredictionEvent, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	url := fmt.Sprintf("%s/markets?limit=%d&active=true&closed=false", polymarketBaseURL, limit)
	if category != "" {
		url += "&tag=" + category
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("polymarket: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("polymarket: http error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("polymarket: HTTP %d: %s", resp.StatusCode, string(body))
	}

	// Polymarket Gamma API returns a JSON array of markets directly.
	var rawEvents []polymarketMarket
	if err := json.NewDecoder(resp.Body).Decode(&rawEvents); err != nil {
		return nil, fmt.Errorf("polymarket: parse error: %w", err)
	}

	events := make([]PredictionEvent, 0, len(rawEvents))
	for _, m := range rawEvents {
		event, err := convertPolymarketMarket(m)
		if err != nil {
			slog.Debug("polymarket: skipping malformed market", "id", m.ID, "error", err)
			continue
		}
		events = append(events, event)
	}
	return events, nil
}

// FetchEvent returns a single event by Polymarket numeric ID.
func (a *PolymarketHTTPAdapter) FetchEvent(ctx context.Context, id string) (*PredictionEvent, error) {
	url := fmt.Sprintf("%s/markets/%s", polymarketBaseURL, id)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("polymarket: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("polymarket: http error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("polymarket: HTTP %d", resp.StatusCode)
	}

	var raw polymarketMarket
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("polymarket: parse error: %w", err)
	}

	event, err := convertPolymarketMarket(raw)
	if err != nil {
		return nil, fmt.Errorf("polymarket: convert error: %w", err)
	}
	return &event, nil
}

// FetchPriceHistory returns historical price data for an outcome.
// Note: Polymarket's Gamma API does not currently expose a public price
// history endpoint; this returns an empty slice with no error.
func (a *PolymarketHTTPAdapter) FetchPriceHistory(ctx context.Context, outcomeID string, interval string, limit int) ([]PricePoint, error) {
	if interval == "" {
		interval = "1d"
	}
	if limit <= 0 {
		limit = 30
	}
	if limit > 365 {
		limit = 365
	}

	// Try the price history endpoint; many deployments return 404.
	url := fmt.Sprintf("%s/markets/%s/prices?interval=%s&limit=%d", polymarketBaseURL, outcomeID, interval, limit)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("polymarket prices: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("polymarket prices: http error: %w", err)
	}
	defer resp.Body.Close()

	// If the endpoint is not available (404), return empty data gracefully.
	if resp.StatusCode == http.StatusNotFound {
		slog.Debug("polymarket prices endpoint not available (404)")
		return []PricePoint{}, nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("polymarket prices: HTTP %d", resp.StatusCode)
	}

	// Price response format: {"history": [{"t": timestamp_ms, "p": price}, ...]}
	var priceResponse struct {
		History []polymarketPricePoint `json:"history"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&priceResponse); err != nil {
		return nil, fmt.Errorf("polymarket prices: parse error: %w", err)
	}

	prices := make([]PricePoint, 0, len(priceResponse.History))
	for _, pp := range priceResponse.History {
		prices = append(prices, PricePoint{
			Timestamp: pp.T,
			Price:     pp.P,
		})
	}
	return prices, nil
}

// ── Polymarket API wire types (private) ──────────────────────────

// polymarketMarket mirrors Polymarket's Gamma API JSON shape.
// The API returns numeric fields like volume/liquidity as strings,
// and outcomes/prices as JSON-encoded string arrays.
type polymarketMarket struct {
	ID               string  `json:"id"`
	Question         string  `json:"question"`
	Slug             string  `json:"slug"`
	VolumeNum        float64 `json:"volumeNum"`
	LiquidityNum     float64 `json:"liquidityNum"`
	EndDateIso       string  `json:"endDateIso"`
	EndDate          string  `json:"endDate"`
	Closed           bool    `json:"closed"`
	Active           bool    `json:"active"`
	OutcomesRaw      string  `json:"outcomes"`
	OutcomePricesRaw string  `json:"outcomePrices"`
	Description      string  `json:"description"`
	UpdatedAt        string  `json:"updatedAt"`
}

type polymarketPricePoint struct {
	T int64   `json:"t"`
	P float64 `json:"p"`
}

// ── Public domain types ──────────────────────────────────────────

// PredictionEvent represents a single prediction market event.
type PredictionEvent struct {
	ID          string              `json:"id"`
	Title       string              `json:"title"`
	Category    string              `json:"category"`
	Volume      float64             `json:"volume"`
	Liquidity   float64             `json:"liquidity"`
	EndDate     string              `json:"end_date"`
	Status      string              `json:"status"`
	Outcomes    []PredictionOutcome `json:"outcomes"`
	Description string              `json:"description"`
	UpdatedAt   int64               `json:"updated_at"`
}

// PredictionOutcome represents an outcome within a prediction market.
type PredictionOutcome struct {
	ID        string  `json:"id"`
	Label     string  `json:"label"`
	Price     float64 `json:"price"`
	Change24h float64 `json:"change_24h"`
}

// PricePoint represents a historical price observation.
type PricePoint struct {
	Timestamp int64   `json:"timestamp"`
	Price     float64 `json:"price"`
	Volume    float64 `json:"volume,omitempty"`
}

// ── Conversion ───────────────────────────────────────────────────

// parseStringSlice parses a JSON-encoded string array like "[\"Yes\",\"No\"]".
func parseStringSlice(raw string) ([]string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	var result []string
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return nil, err
	}
	return result, nil
}

// parseFloatSlice parses a JSON-encoded float64 array like "[\"0.51\",\"0.49\"]".
// Polymarket prices are quoted as strings within the JSON array.
func parseFloatSlice(raw string) ([]float64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	// Prices may be quoted as strings or as numbers; try string slice first.
	var strSlice []string
	if err := json.Unmarshal([]byte(raw), &strSlice); err == nil {
		result := make([]float64, 0, len(strSlice))
		for _, s := range strSlice {
			f, err := strconv.ParseFloat(s, 64)
			if err != nil {
				return nil, fmt.Errorf("parse price %q: %w", s, err)
			}
			result = append(result, f)
		}
		return result, nil
	}
	// Fall back to raw float64 slice.
	var result []float64
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return nil, err
	}
	return result, nil
}

// categoryKeywords maps tokens found in tags, slugs, and question text to
// our category enum. Order is significant: more specific matches first.
var categoryKeywords = map[string]string{
	// Economics (order: most specific first)
	"inflation": "economics", "cpi": "economics", "gdp": "economics",
	"fed": "economics", "fomc": "economics", "interest-rate": "economics",
	"rate-cut": "economics", "rate-hike": "economics", "unemployment": "economics",
	"treasury": "economics", "recession": "economics", "tariff": "economics",
	// Crypto
	"bitcoin": "crypto", "btc": "crypto", "ethereum": "crypto", "eth": "crypto",
	"crypto": "crypto", "solana": "crypto", "defi": "crypto", "blockchain": "crypto",
	"altcoin": "crypto", "stablecoin": "crypto",
	// Politics
	"election": "politics", "trump": "politics", "biden": "politics",
	"republican": "politics", "democrat": "politics", "congress": "politics",
	"president": "politics", "senate": "politics", "supreme-court": "politics",
	// Sports
	"nfl": "sports", "nba": "sports", "mlb": "sports", "super-bowl": "sports",
	"playoffs": "sports", "championship": "sports",
	// Tech
	"ai": "tech", "artificial-intelligence": "tech", "apple": "tech",
	"google": "tech", "microsoft": "tech", "tesla": "tech",
	// Entertainment
	"oscar": "entertainment", "grammy": "entertainment", "movie": "entertainment",
	"album": "entertainment", "tv": "entertainment",
	// Science
	"nasa": "science", "spacex": "science", "climate": "science",
}

// inferCategory extracts the best category from the market's slug, question
// text, and optional tags. Falls back to "other" when nothing matches.
// The current Gamma API does not return tags; we infer from text content.
func inferCategory(slug, question string) string {
	text := strings.ToLower(slug + " " + question)

	// Tokenize: replace common separators with spaces for simpler matching.
	text = strings.NewReplacer("-", " ", "_", " ", "?", " ", ":", " ").Replace(text)
	tokens := strings.Fields(text)

	// Score each category by the number of keyword matches.
	scores := make(map[string]int)
	for _, token := range tokens {
		if cat, ok := categoryKeywords[token]; ok {
			scores[cat]++
		}
	}

	bestCat := "other"
	bestScore := 0
	for cat, score := range scores {
		if score > bestScore {
			bestScore = score
			bestCat = cat
		}
	}
	return bestCat
}

func convertPolymarketMarket(m polymarketMarket) (PredictionEvent, error) {
	status := "open"
	if m.Closed {
		status = "closed"
	}

	// Use Question as the title (API uses "question", not "title").
	title := m.Question
	if title == "" {
		title = m.Slug
	}

	// Parse outcomes string -> []string, outcomePrices -> []float64.
	labels, _ := parseStringSlice(m.OutcomesRaw)
	prices, _ := parseFloatSlice(m.OutcomePricesRaw)

	// Determine the end date: prefer endDateIso, fall back to endDate.
	endDate := m.EndDateIso
	if endDate == "" {
		endDate = m.EndDate
	}

	// Build the timestamp from updatedAt if available.
	updatedAt := time.Now().UnixMilli()
	if m.UpdatedAt != "" {
		if t, err := time.Parse(time.RFC3339Nano, m.UpdatedAt); err == nil {
			updatedAt = t.UnixMilli()
		}
	}

	// Build outcomes, matching labels with their prices.
	outcomes := make([]PredictionOutcome, 0, max(len(labels), 1))
	for i, label := range labels {
		price := 0.0
		if i < len(prices) {
			price = prices[i]
		}
		outcomes = append(outcomes, PredictionOutcome{
			ID:    label,
			Label: label,
			Price: price,
		})
	}
	// If no outcomes were parsed, create a placeholder.
	if len(outcomes) == 0 {
		outcomes = append(outcomes, PredictionOutcome{
			ID:    "unknown",
			Label: "Unknown",
			Price: 0,
		})
	}

	return PredictionEvent{
		ID:          m.ID,
		Title:       title,
		Category:    inferCategory(m.Slug, m.Question),
		Volume:      m.VolumeNum,
		Liquidity:   m.LiquidityNum,
		EndDate:     endDate,
		Status:      status,
		Outcomes:    outcomes,
		Description: m.Description,
		UpdatedAt:   updatedAt,
	}, nil
}
