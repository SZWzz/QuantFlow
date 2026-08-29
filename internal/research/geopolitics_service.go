package research

import (
	"context"
	"log/slog"
	"quantflow/internal/market/adapters"
	"sync"
	"time"
)

// GeopoliticsService provides geopolitical risk data with TTL caching
// and signal extraction. Degrades gracefully to mock data when the adapter
// is nil or API calls fail.
type GeopoliticsService struct {
	adapter adapters.GeopoliticsAdapter
	topics  []adapters.TopicQuery
	mu      sync.RWMutex
	cache   map[string]*geoCachedResult
}

type geoCachedResult struct {
	risks     []TopicRisk
	expiresAt time.Time
}

// TopicRisk represents the risk assessment for a single geopolitical topic.
type TopicRisk struct {
	ID         string  `json:"id"`
	Title      string  `json:"title"`
	TitleCN    string  `json:"title_cn"`
	RiskLevel  string  `json:"risk_level"`  // high / medium / low
	Tone       float64 `json:"tone"`        // current average tone
	ToneChange float64 `json:"tone_change"` // 7-day tone change
	VolChange  float64 `json:"vol_change"`  // 7-day volume change (%)
	Associated string  `json:"associated"`  // assets affected
	UpdatedAt  int64   `json:"updated_at"`
}

// NewGeopoliticsService creates a new geopolitics service.
// adapter may be nil for mock-only mode.
func NewGeopoliticsService(adapter adapters.GeopoliticsAdapter) *GeopoliticsService {
	return &GeopoliticsService{
		adapter: adapter,
		topics:  buildTopicList(),
		cache:   make(map[string]*geoCachedResult),
	}
}

// isAvailable returns true if the adapter is reachable.
// Results are cached for 5 minutes to avoid redundant HTTP calls.
func (s *GeopoliticsService) isAvailable(ctx context.Context) bool {
	if s.adapter == nil {
		return false
	}
	return s.adapter.IsAvailable(ctx)
}

// GetTopicRisks returns risk assessments for all 10 pre-defined geopolitical
// topics. Results are cached for 5 minutes. Falls back to mock data on error.
func (s *GeopoliticsService) GetTopicRisks(ctx context.Context) ([]TopicRisk, error) {
	cacheKey := "all_risks"

	s.mu.RLock()
	if entry, ok := s.cache[cacheKey]; ok && time.Now().Before(entry.expiresAt) {
		defer s.mu.RUnlock()
		return entry.risks, nil
	}
	s.mu.RUnlock()

	var risks []TopicRisk

	if s.isAvailable(ctx) {
		// Fetch tone and volume for all topics, compute risk levels
		risks = make([]TopicRisk, 0, len(s.topics))
		for _, tq := range s.topics {
			risk, err := s.computeTopicRisk(ctx, tq)
			if err != nil {
				slog.Warn("geopolitics: risk computation failed, using mock for topic", "topic", tq.ID, "error", err)
				risk = mockTopicRisk(tq)
			}
			risks = append(risks, risk)
		}
	} else {
		// Mock fallback
		risks = mockAllTopicRisks()
	}

	// Update cache
	s.mu.Lock()
	s.cache[cacheKey] = &geoCachedResult{
		risks:     risks,
		expiresAt: time.Now().Add(5 * time.Minute),
	}
	s.mu.Unlock()

	return risks, nil
}

// GetTopicDetail returns volume + tone time series for a single topic.
func (s *GeopoliticsService) GetTopicDetail(ctx context.Context, topicID, timespan string) (map[string]interface{}, error) {
	if timespan == "" {
		timespan = "7d"
	}

	var volumes []adapters.VolumePoint
	var tones []adapters.TonePoint
	var err error

	if s.isAvailable(ctx) {
		volumes, err = s.adapter.FetchTopicVolume(ctx, topicID, timespan)
		if err != nil {
			slog.Warn("geopolitics: volume fetch failed, using mock", "topic", topicID, "error", err)
		}
		tones, err = s.adapter.FetchTopicTone(ctx, topicID, timespan)
		if err != nil {
			slog.Warn("geopolitics: tone fetch failed, using mock", "topic", topicID, "error", err)
		}
	}

	if volumes == nil {
		volumes = mockVolumePoints(topicID, timespan)
	}
	if tones == nil {
		tones = mockTonePoints(topicID, timespan)
	}

	return map[string]interface{}{
		"volumes":      volumes,
		"tones":        tones,
		"topic_id":     topicID,
		"timespan":     timespan,
		"generated_at": time.Now().UnixMilli(),
	}, nil
}

// ExtractRiskSignals scans all topics for coverage and tone anomalies
// that exceed the threshold. Returns only topics with elevated risk.
// minVolChange is the minimum volume change percentage to trigger a signal (default 50).
func (s *GeopoliticsService) ExtractRiskSignals(ctx context.Context, minVolChange float64) ([]TopicRisk, error) {
	if minVolChange <= 0 {
		minVolChange = 50
	}

	risks, err := s.GetTopicRisks(ctx)
	if err != nil {
		return nil, err
	}

	// Filter to topics that meet the criteria
	signals := make([]TopicRisk, 0)
	for _, r := range risks {
		if r.VolChange >= minVolChange && r.RiskLevel != "low" {
			signals = append(signals, r)
		}
	}

	if len(signals) == 0 {
		// Return at least the highest-risk topic if no threshold met
		var highest *TopicRisk
		for i := range risks {
			if highest == nil || risks[i].VolChange > highest.VolChange {
				highest = &risks[i]
			}
		}
		if highest != nil {
			signals = append(signals, *highest)
		}
	}

	return signals, nil
}

// computeTopicRisk fetches tone and volume data for a single topic and
// computes the risk level from the change metrics.
func (s *GeopoliticsService) computeTopicRisk(ctx context.Context, tq adapters.TopicQuery) (TopicRisk, error) {
	volumes, err := s.adapter.FetchTopicVolume(ctx, tq.ID, "7d")
	if err != nil {
		return TopicRisk{}, err
	}
	tones, err := s.adapter.FetchTopicTone(ctx, tq.ID, "7d")
	if err != nil {
		return TopicRisk{}, err
	}

	// Compute avg tone from the last 7 days
	currentTone := 0.0
	if len(tones) > 0 {
		sum := 0.0
		for _, tp := range tones {
			sum += tp.Tone
		}
		currentTone = sum / float64(len(tones))
	}

	// Compute vol change as % change from first to last point
	volChange := 0.0
	if len(volumes) >= 2 {
		first := volumes[0].Value
		last := volumes[len(volumes)-1].Value
		if first > 0 {
			volChange = ((last - first) / first) * 100
		}
	}

	// Compute tone change
	toneChange := 0.0
	if len(tones) >= 2 {
		toneChange = tones[len(tones)-1].Tone - tones[0].Tone
	}

	riskLevel := computeRiskLevel(volChange, currentTone)

	return TopicRisk{
		ID:         tq.ID,
		Title:      tq.Title,
		TitleCN:    topicTitleCN(tq.ID),
		RiskLevel:  riskLevel,
		Tone:       currentTone,
		ToneChange: toneChange,
		VolChange:  volChange,
		Associated: tq.Associated,
		UpdatedAt:  time.Now().UnixMilli(),
	}, nil
}

// computeRiskLevel determines the risk level from volume change and tone.
// vol_change >50% + tone < -2 → "high"
// vol_change >20% or tone < 0 → "medium"
// else "low"
func computeRiskLevel(volChange, tone float64) string {
	if volChange > 50 && tone < -2 {
		return "high"
	}
	if volChange > 20 || tone < 0 {
		return "medium"
	}
	return "low"
}

// topicTitleCN returns the Chinese name for a topic ID.
func topicTitleCN(topicID string) string {
	names := map[string]string{
		"middle-east":    "中东局势",
		"taiwan-strait":  "台海紧张",
		"ukraine-war":    "俄乌战争",
		"trade-tariffs":  "贸易关税",
		"north-korea":    "朝鲜半岛",
		"fed-policy":     "美联储政策",
		"europe-energy":  "欧洲能源",
		"terrorism":      "恐怖主义",
		"china-economy":  "中国经济",
		"semiconductors": "半导体",
	}
	if cn, ok := names[topicID]; ok {
		return cn
	}
	return topicID
}

// buildTopicList returns the 10 pre-defined geopolitical topics as a slice.
func buildTopicList() []adapters.TopicQuery {
	tqMap := adapters.NewGDELTAdapter().TopicQueries
	topics := make([]adapters.TopicQuery, 0, len(tqMap))
	order := []string{
		"middle-east", "taiwan-strait", "ukraine-war", "trade-tariffs",
		"north-korea", "fed-policy", "europe-energy", "terrorism",
		"china-economy", "semiconductors",
	}
	for _, id := range order {
		if tq, ok := tqMap[id]; ok {
			topics = append(topics, tq)
		}
	}
	return topics
}

// ── Mock data ─────────────────────────────────────────────────────

func mockAllTopicRisks() []TopicRisk {
	order := []string{
		"middle-east", "taiwan-strait", "ukraine-war", "trade-tariffs",
		"north-korea", "fed-policy", "europe-energy", "terrorism",
		"china-economy", "semiconductors",
	}
	risks := make([]TopicRisk, 0, len(order))
	for _, id := range order {
		tqMap := adapters.NewGDELTAdapter().TopicQueries
		tq := tqMap[id]
		risks = append(risks, mockTopicRisk(tq))
	}
	return risks
}

func mockTopicRisk(tq adapters.TopicQuery) TopicRisk {
	// Realistic mock data for each topic
	mockData := map[string]struct {
		tone      float64
		volChange float64
		riskLevel string
	}{
		"middle-east":    {tone: -3.5, volChange: 65, riskLevel: "high"},
		"taiwan-strait":  {tone: -1.8, volChange: 45, riskLevel: "medium"},
		"ukraine-war":    {tone: -2.2, volChange: 30, riskLevel: "medium"},
		"trade-tariffs":  {tone: -1.5, volChange: 55, riskLevel: "medium"},
		"north-korea":    {tone: -0.8, volChange: 15, riskLevel: "low"},
		"fed-policy":     {tone: 0.5, volChange: 25, riskLevel: "medium"},
		"europe-energy":  {tone: -2.8, volChange: 70, riskLevel: "high"},
		"terrorism":      {tone: -4.2, volChange: 80, riskLevel: "high"},
		"china-economy":  {tone: -1.2, volChange: 35, riskLevel: "medium"},
		"semiconductors": {tone: -0.3, volChange: 10, riskLevel: "low"},
	}

	data, ok := mockData[tq.ID]
	if !ok {
		data = struct {
			tone      float64
			volChange float64
			riskLevel string
		}{tone: 0, volChange: 0, riskLevel: "low"}
	}

	toneChange := 0.0
	switch {
	case data.tone < -2:
		toneChange = -1.5
	case data.tone > 0:
		toneChange = 0.8
	default:
		toneChange = -0.5
	}

	return TopicRisk{
		ID:         tq.ID,
		Title:      tq.Title,
		TitleCN:    topicTitleCN(tq.ID),
		RiskLevel:  data.riskLevel,
		Tone:       data.tone,
		ToneChange: toneChange,
		VolChange:  data.volChange,
		Associated: tq.Associated,
		UpdatedAt:  time.Now().UnixMilli(),
	}
}

func mockVolumePoints(topicID, timespan string) []adapters.VolumePoint {
	now := time.Now()
	days := 7
	switch timespan {
	case "30d":
		days = 30
	case "60d":
		days = 60
	case "90d":
		days = 90
	}

	points := make([]adapters.VolumePoint, days)
	base := 100.0
	if topicID == "terrorism" || topicID == "europe-energy" {
		base = 80.0
	}
	for i := 0; i < days; i++ {
		date := now.Add(-time.Duration(days-i) * 24 * time.Hour).Format("2006-01-02")
		// Trending upward for high-risk topics
		trend := float64(i) / float64(days) * 50.0
		points[i] = adapters.VolumePoint{
			Date:  date,
			Value: base + trend,
			Query: topicID,
		}
	}
	return points
}

func mockTonePoints(topicID, timespan string) []adapters.TonePoint {
	now := time.Now()
	days := 7
	switch timespan {
	case "30d":
		days = 30
	case "60d":
		days = 60
	case "90d":
		days = 90
	}

	// Realistic tone baselines per topic
	toneBases := map[string]float64{
		"middle-east":    -3.5,
		"taiwan-strait":  -1.8,
		"ukraine-war":    -2.2,
		"trade-tariffs":  -1.5,
		"north-korea":    -0.8,
		"fed-policy":     0.5,
		"europe-energy":  -2.8,
		"terrorism":      -4.2,
		"china-economy":  -1.2,
		"semiconductors": -0.3,
	}
	base, ok := toneBases[topicID]
	if !ok {
		base = 0
	}

	points := make([]adapters.TonePoint, days)
	for i := 0; i < days; i++ {
		date := now.Add(-time.Duration(days-i) * 24 * time.Hour).Format("2006-01-02")
		// Add slight random-like variation around base
		variation := (float64(i%5) - 2.0) * 0.3
		points[i] = adapters.TonePoint{
			Date:  date,
			Tone:  base + variation,
			Query: topicID,
		}
	}
	return points
}
