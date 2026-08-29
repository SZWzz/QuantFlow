package research

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"quantflow/internal/market/adapters"
	"sync"
	"time"
)

// PredictionMarketService provides prediction market data with TTL caching
// and signal extraction. Degrades gracefully to mock data when the adapter
// is nil or API calls fail.
type PredictionMarketService struct {
	adapter        adapters.PolymarketAdapter
	cache          map[string]*cacheEntry // keyed by category
	mu             sync.RWMutex
	available      bool      // cached availability check
	availCheckedAt time.Time // when availability was last checked
}

type cacheEntry struct {
	events    []adapters.PredictionEvent
	expiresAt time.Time
}

// NewPredictionMarketService creates a new prediction market service.
// adapter may be nil for mock-only mode.
func NewPredictionMarketService(adapter adapters.PolymarketAdapter) *PredictionMarketService {
	return &PredictionMarketService{
		adapter: adapter,
		cache:   make(map[string]*cacheEntry),
	}
}

// SignalOutput is the signal extracted from prediction market data.
type SignalOutput struct {
	Category    string                     `json:"category"`
	Events      []adapters.PredictionEvent `json:"events"`
	Signal      SignalSummary              `json:"signal"`
	GeneratedAt time.Time                  `json:"generated_at"`
}

// isAvailable returns true if the adapter is reachable.
// Results are cached for 5 minutes to avoid redundant HTTP calls.
func (s *PredictionMarketService) isAvailable(ctx context.Context) bool {
	if s.adapter == nil {
		return false
	}
	s.mu.RLock()
	if time.Since(s.availCheckedAt) < 5*time.Minute {
		avail := s.available
		s.mu.RUnlock()
		return avail
	}
	s.mu.RUnlock()

	avail := s.adapter.IsAvailable(ctx)
	s.mu.Lock()
	s.available = avail
	s.availCheckedAt = time.Now()
	s.mu.Unlock()
	return avail
}

// SignalSummary describes the extracted trading signal.
type SignalSummary struct {
	Action      string  `json:"action"`      // "buy", "sell", "hold"
	Confidence  float64 `json:"confidence"`  // 0.0 - 1.0
	Description string  `json:"description"` // Human-readable reasoning
}

// GetEvents returns prediction market events for a category.
// Results are cached for 5 minutes. Falls back to mock data on error.
func (s *PredictionMarketService) GetEvents(ctx context.Context, category string, limit int) ([]adapters.PredictionEvent, error) {
	if limit <= 0 {
		limit = 20
	}

	// Check cache (5min TTL)
	cacheKey := category
	if cacheKey == "" {
		cacheKey = "all"
	}
	s.mu.RLock()
	if entry, ok := s.cache[cacheKey]; ok && time.Now().Before(entry.expiresAt) {
		defer s.mu.RUnlock()
		events := entry.events
		if len(events) > limit {
			events = events[:limit]
		}
		return events, nil
	}
	s.mu.RUnlock()

	// Fetch from adapter
	var events []adapters.PredictionEvent
	var err error
	if s.isAvailable(ctx) {
		events, err = s.adapter.FetchEvents(ctx, category, limit)
		if err != nil {
			slog.Warn("polymarket: fetch failed, falling back to mock", "error", err)
		}
	}

	// Fall back to mock
	if events == nil {
		events = mockPredictionEvents(category, limit)
	}

	// Update cache
	s.mu.Lock()
	s.cache[cacheKey] = &cacheEntry{
		events:    events,
		expiresAt: time.Now().Add(5 * time.Minute),
	}
	s.mu.Unlock()

	if len(events) > limit {
		events = events[:limit]
	}
	return events, nil
}

// GetEventDetail returns a single event with full outcome details.
func (s *PredictionMarketService) GetEventDetail(ctx context.Context, id string) (*adapters.PredictionEvent, error) {
	if s.isAvailable(ctx) {
		event, err := s.adapter.FetchEvent(ctx, id)
		if err != nil {
			slog.Warn("polymarket: event detail fetch failed", "id", id, "error", err)
		} else {
			return event, nil
		}
	}
	// Mock fallback: return a single mock event matching the id pattern
	return mockSingleEvent(id), nil
}

// GetPriceHistory returns historical price data for an event.
func (s *PredictionMarketService) GetPriceHistory(ctx context.Context, eventID string, interval string, limit int) ([]adapters.PricePoint, error) {
	if limit <= 0 {
		limit = 30
	}
	if s.isAvailable(ctx) {
		prices, err := s.adapter.FetchPriceHistory(ctx, eventID, interval, limit)
		if err != nil {
			slog.Warn("polymarket: price history fetch failed", "event_id", eventID, "error", err)
		} else if len(prices) > 0 {
			return prices, nil
		}
	}
	// Mock fallback: generate plausible price history
	return mockPriceHistory(eventID, limit), nil
}

// ExtractSignals scans prediction market events for probability shifts
// that exceed the threshold and generates trading signals.
// minProbChange is the minimum absolute probability change (0.0-1.0) to trigger a signal.
func (s *PredictionMarketService) ExtractSignals(ctx context.Context, category string, minProbChange float64) (*SignalOutput, error) {
	if minProbChange <= 0 {
		minProbChange = 0.05 // default 5% threshold
	}

	events, err := s.GetEvents(ctx, category, 50)
	if err != nil {
		return nil, err
	}

	action := "hold"
	confidence := 0.0
	description := "no significant probability movement detected"

	// Scan for events where Yes price has moved > threshold
	for _, e := range events {
		for _, o := range e.Outcomes {
			absChange := math.Abs(o.Change24h)
			if absChange > minProbChange && e.Volume > 100000 {
				if o.Change24h > 0 && o.Price > 0.5 {
					action = "buy"
				} else if o.Change24h < 0 && o.Price < 0.5 {
					action = "sell"
				}
				if action != "hold" {
					confidence = absChange * 2 // scale for 0-1 range
					if confidence > 1.0 {
						confidence = 1.0
					}
					description = fmt.Sprintf("%s: %s probability changed %.1f%% to %.1f%%",
						e.Title, o.Label, o.Change24h*100, o.Price*100)
					break
				}
			}
		}
		if action != "hold" {
			break
		}
	}

	return &SignalOutput{
		Category: category,
		Events:   events,
		Signal: SignalSummary{
			Action:      action,
			Confidence:  confidence,
			Description: description,
		},
		GeneratedAt: time.Now(),
	}, nil
}

// ── Mock data ─────────────────────────────────────────────────────

func mockPredictionEvents(category string, limit int) []adapters.PredictionEvent {
	allMock := []adapters.PredictionEvent{
		{
			ID: "fed-rate-cut-july-2026", Title: "Fed cuts rates by July 2026?",
			Category: "economics", Volume: 2_500_000, Liquidity: 1_800_000,
			EndDate: "2026-07-31T23:59:59Z", Status: "open",
			Outcomes: []adapters.PredictionOutcome{
				{ID: "yes-1", Label: "Yes", Price: 0.35, Change24h: 0.03},
				{ID: "no-1", Label: "No", Price: 0.65, Change24h: -0.03},
			},
			Description: "Market predicts probability of a Federal Reserve rate cut by July 2026 FOMC meeting.",
		},
		{
			ID: "bitcoin-100k-q3-2026", Title: "Bitcoin breaks $100K by Q3 2026?",
			Category: "crypto", Volume: 4_200_000, Liquidity: 3_100_000,
			EndDate: "2026-09-30T23:59:59Z", Status: "open",
			Outcomes: []adapters.PredictionOutcome{
				{ID: "yes-2", Label: "Yes", Price: 0.28, Change24h: -0.05},
				{ID: "no-2", Label: "No", Price: 0.72, Change24h: 0.05},
			},
			Description: "Will Bitcoin price exceed $100,000 before the end of Q3 2026?",
		},
		{
			ID: "cpi-above-3pct-q2-2026", Title: "CPI inflation above 3.0% in Q2 2026?",
			Category: "economics", Volume: 1_800_000, Liquidity: 1_500_000,
			EndDate: "2026-06-30T23:59:59Z", Status: "open",
			Outcomes: []adapters.PredictionOutcome{
				{ID: "yes-3", Label: "Yes", Price: 0.42, Change24h: 0.02},
				{ID: "no-3", Label: "No", Price: 0.58, Change24h: -0.02},
			},
			Description: "Will US CPI year-over-year inflation remain above 3.0% in Q2 2026?",
		},
		{
			ID: "ethereum-etf-approval-2026", Title: "Ethereum ETF options approved by end of 2026?",
			Category: "crypto", Volume: 1_100_000, Liquidity: 900_000,
			EndDate: "2026-12-31T23:59:59Z", Status: "open",
			Outcomes: []adapters.PredictionOutcome{
				{ID: "yes-4", Label: "Yes", Price: 0.55, Change24h: 0.08},
				{ID: "no-4", Label: "No", Price: 0.45, Change24h: -0.08},
			},
			Description: "SEC approves options trading on spot Ethereum ETFs by end of 2026?",
		},
		{
			ID: "china-gdp-below-4pct-2026", Title: "China GDP growth below 4% in 2026?",
			Category: "economics", Volume: 950_000, Liquidity: 700_000,
			EndDate: "2026-12-31T23:59:59Z", Status: "open",
			Outcomes: []adapters.PredictionOutcome{
				{ID: "yes-5", Label: "Yes", Price: 0.18, Change24h: 0.01},
				{ID: "no-5", Label: "No", Price: 0.82, Change24h: -0.01},
			},
			Description: "Will China's 2026 GDP growth rate fall below 4%?",
		},
	}

	if category == "" || category == "all" {
		if limit > 0 && limit < len(allMock) {
			return allMock[:limit]
		}
		return allMock
	}

	filtered := make([]adapters.PredictionEvent, 0)
	for _, e := range allMock {
		if e.Category == category {
			filtered = append(filtered, e)
		}
	}
	if limit > 0 && limit < len(filtered) {
		filtered = filtered[:limit]
	}
	return filtered
}

func mockSingleEvent(id string) *adapters.PredictionEvent {
	all := mockPredictionEvents("", 0)
	for _, e := range all {
		if e.ID == id {
			return &e
		}
	}
	return &adapters.PredictionEvent{
		ID: id, Title: id, Category: "other", Status: "open",
		Outcomes: []adapters.PredictionOutcome{
			{ID: "yes", Label: "Yes", Price: 0.50},
		},
	}
}

func mockPriceHistory(eventID string, limit int) []adapters.PricePoint {
	now := time.Now()
	points := make([]adapters.PricePoint, limit)
	for i := 0; i < limit; i++ {
		ts := now.Add(-time.Duration(limit-i) * 24 * time.Hour).UnixMilli()
		// Generate a slightly noisy random walk from 0.4 to 0.6
		base := 0.5
		noise := float64(i-limit/2) / float64(limit) * 0.2
		points[i] = adapters.PricePoint{Timestamp: ts, Price: base + noise}
	}
	return points
}
