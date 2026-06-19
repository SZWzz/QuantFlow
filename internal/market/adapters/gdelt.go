// internal/market/adapters/gdelt.go
package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

const gdeltBaseURL = "https://api.gdeltproject.org/api/v2/doc/doc"

// GeopoliticsAdapter defines the interface for geopolitical data sources.
// Separate from market.Adapter because geopolitics data carries topic-based
// coverage volume and sentiment tone time series, not financial quotes.
type GeopoliticsAdapter interface {
	Name() string
	IsAvailable(ctx context.Context) bool

	// FetchTopicVolume returns coverage volume time series for a topic.
	FetchTopicVolume(ctx context.Context, topicID, timespan string) ([]VolumePoint, error)

	// FetchTopicTone returns average tone time series for a topic.
	FetchTopicTone(ctx context.Context, topicID, timespan string) ([]TonePoint, error)
}

// VolumePoint represents a single coverage volume observation from GDELT.
type VolumePoint struct {
	Date  string  `json:"date"`
	Value float64 `json:"value"` // article count or % of global coverage
	Query string  `json:"query"`
}

// TonePoint represents a single sentiment tone observation from GDELT.
// Tone ranges from -10 (extreme negative) to +10 (extreme positive).
type TonePoint struct {
	Date  string  `json:"date"`
	Tone  float64 `json:"tone"` // -10 to +10
	Query string  `json:"query"`
}

// TopicQuery defines a pre-configured geopolitical topic with its GDELT query.
type TopicQuery struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	Query      string `json:"query"`      // URL-encoded GDELT query string
	Associated string `json:"associated"` // assets affected by this topic
}

// TopicQueries contains the 10 pre-defined geopolitical topics with
// URL-encoded boolean queries ready for the GDELT DOC 2.0 API.
var TopicQueries = map[string]string{
	"middle-east":    "%22middle%20east%22%20OR%20israel%20OR%20gaza%20OR%20iran",
	"taiwan-strait":  "%22taiwan%20strait%22%20OR%20%22south%20china%20sea%22%20OR%20taiwan",
	"ukraine-war":    "%22ukraine%20war%22%20OR%20russia%20ukraine",
	"trade-tariffs":  "tariffs%20OR%20%22trade%20war%22%20OR%20%22supply%20chain%22%20OR%20sanctions",
	"north-korea":    "%22north%20korea%22%20OR%20pyongyang%20OR%20%22missile%20launch%22",
	"fed-policy":     "%22federal%20reserve%22%20OR%20fomc%20OR%20%22rate%20hike%22%20OR%20%22rate%20cut%22",
	"europe-energy":  "%22europe%20energy%22%20OR%20%22natural%20gas%22%20OR%20%22energy%20crisis%22",
	"terrorism":      "terrorism%20OR%20%22terrorist%20attack%22%20OR%20extremism",
	"china-economy":  "%22china%20economy%22%20OR%20%22china%20gdp%22%20OR%20%22china%20property%22",
	"semiconductors": "semiconductor%20OR%20chips%20OR%20%22chip%20ban%22%20OR%20tsmc%20OR%20%22export%20control%22",
}

// GDELTAdapter fetches geopolitical data from the GDELT DOC 2.0 API.
// The API is free, requires no API key, and returns JSON.
type GDELTAdapter struct {
	client       *http.Client
	TopicQueries map[string]TopicQuery
}

// Compile-time interface check.
var _ GeopoliticsAdapter = (*GDELTAdapter)(nil)

// NewGDELTAdapter creates a new GDELT adapter with pre-configured topics.
func NewGDELTAdapter() *GDELTAdapter {
	topics := buildTopicQueries()
	return &GDELTAdapter{
		client:       &http.Client{Timeout: 30 * time.Second},
		TopicQueries: topics,
	}
}

func (a *GDELTAdapter) Name() string { return "gdelt" }

// IsAvailable performs a quick HEAD request to check GDELT API reachability.
func (a *GDELTAdapter) IsAvailable(ctx context.Context) bool {
	req, _ := http.NewRequestWithContext(ctx, "HEAD", gdeltBaseURL+"?query=test&mode=TimelineVol&timespan=1d&format=json", nil)
	resp, err := a.client.Do(req)
	if err != nil {
		slog.Debug("gdelt availability check failed", "error", err)
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// FetchTopicVolume returns coverage volume time series for a geopolitical topic.
// timespan: "7d", "30d", "60d", "90d", "180d", "365d".
func (a *GDELTAdapter) FetchTopicVolume(ctx context.Context, topicID, timespan string) ([]VolumePoint, error) {
	if timespan == "" {
		timespan = "7d"
	}

	queryStr := topicID
	if tq, ok := a.TopicQueries[topicID]; ok {
		queryStr = tq.Query
	}

	url := fmt.Sprintf("%s?query=%s&mode=TimelineVol&timespan=%s&format=json", gdeltBaseURL, queryStr, timespan)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("gdelt: %w", err)
	}

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("gdelt: http error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("gdelt: HTTP %d: %s", resp.StatusCode, string(body))
	}

	var raw struct {
		Timeline []struct {
			Date      string  `json:"date"`
			Value     float64 `json:"value"`
			NormValue float64 `json:"normvalue"`
		} `json:"timeline"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("gdelt: parse error: %w", err)
	}

	points := make([]VolumePoint, 0, len(raw.Timeline))
	for _, p := range raw.Timeline {
		points = append(points, VolumePoint{
			Date:  p.Date,
			Value: p.Value,
			Query: queryStr,
		})
	}
	return points, nil
}

// FetchTopicTone returns average tone time series for a geopolitical topic.
// Tone ranges from -10 (extreme negative) to +10 (extreme positive).
// timespan: "7d", "30d", "60d", "90d", "180d", "365d".
func (a *GDELTAdapter) FetchTopicTone(ctx context.Context, topicID, timespan string) ([]TonePoint, error) {
	if timespan == "" {
		timespan = "7d"
	}

	queryStr := topicID
	if tq, ok := a.TopicQueries[topicID]; ok {
		queryStr = tq.Query
	}

	url := fmt.Sprintf("%s?query=%s&mode=TimelineTone&timespan=%s&format=json", gdeltBaseURL, queryStr, timespan)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("gdelt: %w", err)
	}

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("gdelt: http error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("gdelt: HTTP %d: %s", resp.StatusCode, string(body))
	}

	var raw struct {
		Timeline []struct {
			Date  string  `json:"date"`
			Value float64 `json:"value"` // average tone -10 to +10
		} `json:"timeline"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("gdelt: parse error: %w", err)
	}

	points := make([]TonePoint, 0, len(raw.Timeline))
	for _, p := range raw.Timeline {
		points = append(points, TonePoint{
			Date:  p.Date,
			Tone:  p.Value,
			Query: queryStr,
		})
	}
	return points, nil
}

// buildTopicQueries constructs the pre-defined topic query map with metadata.
func buildTopicQueries() map[string]TopicQuery {
	return map[string]TopicQuery{
		"middle-east": {
			ID: "middle-east", Title: "Middle East",
			Query:      TopicQueries["middle-east"],
			Associated: "原油",
		},
		"taiwan-strait": {
			ID: "taiwan-strait", Title: "Taiwan Strait",
			Query:      TopicQueries["taiwan-strait"],
			Associated: "A股/港股",
		},
		"ukraine-war": {
			ID: "ukraine-war", Title: "Ukraine War",
			Query:      TopicQueries["ukraine-war"],
			Associated: "能源/粮食",
		},
		"trade-tariffs": {
			ID: "trade-tariffs", Title: "Trade Tariffs",
			Query:      TopicQueries["trade-tariffs"],
			Associated: "全球",
		},
		"north-korea": {
			ID: "north-korea", Title: "North Korea",
			Query:      TopicQueries["north-korea"],
			Associated: "韩国/日元",
		},
		"fed-policy": {
			ID: "fed-policy", Title: "Fed Policy",
			Query:      TopicQueries["fed-policy"],
			Associated: "美股/美元",
		},
		"europe-energy": {
			ID: "europe-energy", Title: "Europe Energy",
			Query:      TopicQueries["europe-energy"],
			Associated: "天然气",
		},
		"terrorism": {
			ID: "terrorism", Title: "Terrorism",
			Query:      TopicQueries["terrorism"],
			Associated: "全球",
		},
		"china-economy": {
			ID: "china-economy", Title: "China Economy",
			Query:      TopicQueries["china-economy"],
			Associated: "A股/港股",
		},
		"semiconductors": {
			ID: "semiconductors", Title: "Semiconductors",
			Query:      TopicQueries["semiconductors"],
			Associated: "科技股",
		},
	}
}
