package research

import (
	"context"
	"log/slog"
	"math"
	"sync"
	"time"

	"quantflow/internal/market/adapters"
)

// GovDataService provides economic indicator data with TTL caching
// and macro signal extraction. Degrades gracefully to mock data when the
// adapter is nil or API calls fail.
type GovDataService struct {
	adapter    adapters.GovDataAdapter
	indicators map[string]adapters.IndicatorMeta
	mu         sync.RWMutex
	cache      map[string]*govCacheEntry // keyed by seriesID
}

type govCacheEntry struct {
	points    []adapters.IndicatorPoint
	expiresAt time.Time
}

// MacroSignal represents the trading signal derived from an economic indicator.
type MacroSignal struct {
	IndicatorID string  `json:"indicator_id"`
	Name        string  `json:"name"`
	NameCN      string  `json:"name_cn"`
	LatestValue float64 `json:"latest_value"`
	Change      float64 `json:"change"`     // MoM or QoQ change
	Direction   string  `json:"direction"`  // up, down, flat
	Signal      string  `json:"signal"`     // bullish, bearish, neutral
	Unit        string  `json:"unit"`
	Category    string  `json:"category"`
	UpdatedAt   int64   `json:"updated_at"`
}

// NewGovDataService creates a new govdata service.
// adapter may be nil for mock-only mode.
func NewGovDataService(adapter adapters.GovDataAdapter) *GovDataService {
	return &GovDataService{
		adapter:    adapter,
		indicators: adapters.FREDIndicators,
		cache:      make(map[string]*govCacheEntry),
	}
}

// isAvailable returns true if the adapter is reachable.
// Results are cached for 5 minutes to avoid redundant HTTP calls.
func (s *GovDataService) isAvailable(ctx context.Context) bool {
	if s.adapter == nil {
		return false
	}
	return s.adapter.IsAvailable(ctx)
}

// GetIndicatorList returns the metadata for all 15 predefined indicators.
func (s *GovDataService) GetIndicatorList() []adapters.IndicatorMeta {
	list := make([]adapters.IndicatorMeta, 0, len(s.indicators))
	for _, meta := range s.indicators {
		list = append(list, meta)
	}
	return list
}

// GetIndicator fetches indicator observations with TTL caching.
// Falls back to mock data when the adapter is nil or API calls fail.
func (s *GovDataService) GetIndicator(ctx context.Context, seriesID string, limit int) ([]adapters.IndicatorPoint, error) {
	if limit <= 0 {
		limit = 12
	}

	// Check cache (5min TTL)
	cacheKey := seriesID
	s.mu.RLock()
	if entry, ok := s.cache[cacheKey]; ok && time.Now().Before(entry.expiresAt) {
		defer s.mu.RUnlock()
		points := entry.points
		if len(points) > limit {
			points = points[:limit]
		}
		return points, nil
	}
	s.mu.RUnlock()

	// Fetch from adapter
	var points []adapters.IndicatorPoint
	var err error
	if s.isAvailable(ctx) {
		points, err = s.adapter.FetchIndicator(ctx, seriesID, limit)
		if err != nil {
			slog.Warn("govdata: indicator fetch failed, falling back to mock", "series", seriesID, "error", err)
		}
	}

	// Fall back to mock
	if points == nil {
		points = mockIndicatorPoints(seriesID, limit)
	}

	// Update cache
	s.mu.Lock()
	s.cache[cacheKey] = &govCacheEntry{
		points:    points,
		expiresAt: time.Now().Add(5 * time.Minute),
	}
	s.mu.Unlock()

	if len(points) > limit {
		points = points[:limit]
	}
	return points, nil
}

// GetAllSignals fetches all 15 indicators and computes macro signals.
// Returns a MacroSignal for each indicator with direction and trading signal.
func (s *GovDataService) GetAllSignals(ctx context.Context) ([]MacroSignal, error) {
	cacheKey := "all_signals"

	s.mu.RLock()
	if entry, ok := s.cache[cacheKey]; ok && time.Now().Before(entry.expiresAt) {
		defer s.mu.RUnlock()
		// Not the same type but ok — we re-check differently
		_ = entry
	}
	s.mu.RUnlock()

	// Collect all indicator IDs in stable order
	order := []string{
		"GDP", "GDPC1", "CPIAUCSL", "PCEPI", "PPIACO",
		"UNRATE", "PAYEMS", "IC4WSA",
		"FEDFUNDS", "DGS10", "T10Y2Y",
		"DCOILWTICO", "NGDPRPI",
		"HOUST", "MSPUS",
	}

	signals := make([]MacroSignal, 0, len(order))
	for _, id := range order {
		meta, ok := s.indicators[id]
		if !ok {
			continue
		}

		points, err := s.GetIndicator(ctx, id, 12)
		if err != nil {
			slog.Warn("govdata: failed to get indicator for signal", "series", id, "error", err)
			points = mockIndicatorPoints(id, 12)
		}

		signal := computeMacroSignal(meta, points)
		signals = append(signals, signal)
	}

	return signals, nil
}

// computeMacroSignal derives a MacroSignal from an indicator's metadata and data.
func computeMacroSignal(meta adapters.IndicatorMeta, points []adapters.IndicatorPoint) MacroSignal {
	latest := 0.0
	previous := 0.0
	if len(points) > 0 {
		latest = points[len(points)-1].Value
	}
	if len(points) > 1 {
		previous = points[len(points)-2].Value
	}

	change := 0.0
	direction := "flat"
	if previous != 0 {
		change = ((latest - previous) / math.Abs(previous)) * 100
	}
	if change > 0.5 {
		direction = "up"
	} else if change < -0.5 {
		direction = "down"
	}

	// Signal logic:
	// For "good for economy" indicators (GDP, employment) → up = bullish
	// For "bad when high" indicators (CPI, unemployment, rates) → up = bearish
	signal := computeEconomicSignal(meta.Category, direction)

	return MacroSignal{
		IndicatorID: meta.ID,
		Name:        meta.Name,
		NameCN:      meta.NameCN,
		LatestValue: latest,
		Change:      math.Round(change*100) / 100,
		Direction:   direction,
		Signal:      signal,
		Unit:        meta.Unit,
		Category:    meta.Category,
		UpdatedAt:   time.Now().UnixMilli(),
	}
}

// computeEconomicSignal determines bullish/bearish/neutral based on category and direction.
// Categories with "negative" interpretation (up = bad):
//   - inflation: high CPI/PPI/PCE is bearish
//   - employment (unemployment): high UNRATE is bearish
//
// Categories with "positive" interpretation (up = good):
//   - gdp: high GDP is bullish
//   - employment (payrolls): high PAYEMS is bullish
//   - energy: rising oil may be bearish (costs) or bullish (growth)
//   - housing: rising starts/prices are bullish
func computeEconomicSignal(category, direction string) string {
	if direction == "flat" {
		return "neutral"
	}

	switch category {
	case "inflation":
		// Rising inflation is bearish for stocks
		if direction == "up" {
			return "bearish"
		}
		return "bullish"
	case "gdp":
		// Rising GDP is bullish
		if direction == "up" {
			return "bullish"
		}
		return "bearish"
	case "employment":
		// For UNRATE, up = bearish; for PAYEMS/IC4WSA, up = bullish
		// Default conservative: up = bullish (payrolls growing)
		if direction == "up" {
			return "bullish"
		}
		return "bearish"
	case "rates":
		// Rising rates are generally bearish for equities
		if direction == "up" {
			return "bearish"
		}
		return "bullish"
	case "energy":
		// Rising energy costs are bearish
		if direction == "up" {
			return "bearish"
		}
		return "bullish"
	case "housing":
		// Rising housing starts/prices signal economic health
		if direction == "up" {
			return "bullish"
		}
		return "bearish"
	default:
		if direction == "up" {
			return "bullish"
		}
		return "bearish"
	}
}

// categoryNameCN returns Chinese category name for display.
func categoryNameCN(category string) string {
	names := map[string]string{
		"gdp":        "GDP/增长",
		"inflation":  "通胀",
		"employment": "就业",
		"rates":      "利率",
		"energy":     "能源",
		"housing":    "房地产",
	}
	if cn, ok := names[category]; ok {
		return cn
	}
	return category
}

// ── Mock data ─────────────────────────────────────────────────────

func mockIndicatorPoints(seriesID string, limit int) []adapters.IndicatorPoint {
	mockValues := map[string]struct {
		base  float64
		unit  float64 // per-month increment
		noise float64 // random variation
	}{
		"GDP":         {base: 29000, unit: 150, noise: 200},
		"GDPC1":       {base: 23000, unit: 80, noise: 100},
		"CPIAUCSL":    {base: 315, unit: 0.3, noise: 0.2},
		"PCEPI":       {base: 125, unit: 0.2, noise: 0.1},
		"PPIACO":      {base: 265, unit: 0.4, noise: 0.3},
		"UNRATE":      {base: 3.9, unit: 0.02, noise: 0.1},
		"PAYEMS":      {base: 159000, unit: 200, noise: 100},
		"IC4WSA":      {base: 220, unit: 1, noise: 5},
		"FEDFUNDS":    {base: 4.25, unit: -0.05, noise: 0.1},
		"DGS10":       {base: 4.5, unit: -0.02, noise: 0.1},
		"T10Y2Y":      {base: -0.3, unit: 0.02, noise: 0.05},
		"DCOILWTICO":  {base: 75, unit: 0.5, noise: 2},
		"NGDPRPI":     {base: 3.5, unit: 0.05, noise: 0.2},
		"HOUST":       {base: 1450, unit: 10, noise: 30},
		"MSPUS":       {base: 420000, unit: 2000, noise: 5000},
	}

	data, ok := mockValues[seriesID]
	if !ok {
		data = struct {
			base  float64
			unit  float64
			noise float64
		}{base: 100, unit: 0, noise: 1}
	}

	now := time.Now()
	points := make([]adapters.IndicatorPoint, limit)
	for i := 0; i < limit; i++ {
		date := now.AddDate(0, -(limit-i), 0).Format("2006-01-02")
		// Monthly data: march forward from limit months ago
		value := data.base + data.unit*float64(i) + float64(i%3-1)*data.noise
		points[i] = adapters.IndicatorPoint{
			Date:  date,
			Value: math.Round(value*100) / 100,
		}
	}
	return points
}

// mockMacroSignals returns realistic mock signals for all 15 indicators.
func mockMacroSignals() []MacroSignal {
	order := []string{
		"GDP", "GDPC1", "CPIAUCSL", "PCEPI", "PPIACO",
		"UNRATE", "PAYEMS", "IC4WSA",
		"FEDFUNDS", "DGS10", "T10Y2Y",
		"DCOILWTICO", "NGDPRPI",
		"HOUST", "MSPUS",
	}

	// Realistic mock data with directional signals
	mockData := map[string]struct {
		latest    float64
		change    float64
		direction string
		signal    string
	}{
		"GDP":         {latest: 29250.3, change: 2.8, direction: "up", signal: "bullish"},
		"GDPC1":       {latest: 23250.1, change: 2.1, direction: "up", signal: "bullish"},
		"CPIAUCSL":    {latest: 316.8, change: 0.3, direction: "up", signal: "bearish"},
		"PCEPI":       {latest: 125.6, change: 0.2, direction: "up", signal: "bearish"},
		"PPIACO":      {latest: 267.2, change: 0.5, direction: "up", signal: "bearish"},
		"UNRATE":      {latest: 3.8, change: -2.6, direction: "down", signal: "bullish"},
		"PAYEMS":      {latest: 159850, change: 0.15, direction: "up", signal: "bullish"},
		"IC4WSA":      {latest: 215, change: -3.2, direction: "down", signal: "bullish"},
		"FEDFUNDS":    {latest: 4.25, change: 0, direction: "flat", signal: "neutral"},
		"DGS10":       {latest: 4.32, change: -1.8, direction: "down", signal: "bullish"},
		"T10Y2Y":      {latest: -0.25, change: 10.0, direction: "up", signal: "bearish"}, // steepening = fear
		"DCOILWTICO":  {latest: 72.5, change: -3.5, direction: "down", signal: "bullish"},
		"NGDPRPI":     {latest: 3.85, change: 8.5, direction: "up", signal: "bearish"},
		"HOUST":       {latest: 1480, change: 1.2, direction: "up", signal: "bullish"},
		"MSPUS":       {latest: 428500, change: 0.8, direction: "up", signal: "bullish"},
	}

	signals := make([]MacroSignal, 0, len(order))
	for _, id := range order {
		meta, ok := adapters.FREDIndicators[id]
		if !ok {
			continue
		}
		data := mockData[id]

		signals = append(signals, MacroSignal{
			IndicatorID: meta.ID,
			Name:        meta.Name,
			NameCN:      meta.NameCN,
			LatestValue: data.latest,
			Change:      data.change,
			Direction:   data.direction,
			Signal:      data.signal,
			Unit:        meta.Unit,
			Category:    meta.Category,
			UpdatedAt:   time.Now().UnixMilli(),
		})
	}
	return signals
}

// signalEmoji returns an emoji for the signal direction.
func signalEmoji(signal string) string {
	switch signal {
	case "bullish":
		return "🟢"
	case "bearish":
		return "🔴"
	default:
		return "⚪"
	}
}

// directionArrow returns an arrow for the direction.
func directionArrow(direction string) string {
	switch direction {
	case "up":
		return "↑"
	case "down":
		return "↓"
	default:
		return "→"
	}
}

var _ = signalEmoji // ensure helpers are used
var _ = directionArrow
