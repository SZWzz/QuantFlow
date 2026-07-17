package market

import (
	"context"
)

// StyleQuadrant positions an index on the size×style plane.
type StyleQuadrant struct {
	Index    string  `json:"index"`
	Size     float64 `json:"size"`
	Style    float64 `json:"style"`
	Return1M float64 `json:"return_1m"`
}

// MarketSentimentGauge wraps sentiment indicators.
type MarketSentimentGauge struct {
	LimitUp       int     `json:"limit_up"`
	LimitDown     int     `json:"limit_down"`
	Turnover      float64 `json:"turnover"`
	NorthboundCum float64 `json:"northbound_cum"`
}

// StyleService computes market style quadrants and sentiment.
type StyleService struct {
	reg *AdapterRegistry
}

// NewStyleService creates a new StyleService.
func NewStyleService(reg *AdapterRegistry) *StyleService {
	return &StyleService{reg: reg}
}

// GetStyleQuadrant returns index positions in the size×style space.
func (s *StyleService) GetStyleQuadrant(ctx context.Context, market string) ([]StyleQuadrant, error) {
	// Default quadrants using known index sizing
	quadrants := map[string][]StyleQuadrant{
		"CN": {
			{Index: "上证50", Size: 0.9, Style: 0.2, Return1M: 0},
			{Index: "沪深300", Size: 0.7, Style: 0.3, Return1M: 0},
			{Index: "中证500", Size: 0.3, Style: 0.5, Return1M: 0},
			{Index: "创业板指", Size: 0.1, Style: 0.9, Return1M: 0},
			{Index: "科创50", Size: 0.15, Style: 0.95, Return1M: 0},
		},
	}
	if q, ok := quadrants[market]; ok {
		return q, nil
	}
	return []StyleQuadrant{}, nil
}

// GetSentiment returns market sentiment gauges.
func (s *StyleService) GetSentiment(ctx context.Context, market string) (*MarketSentimentGauge, error) {
	// Sentiment data comes from MarketOverview IPC, aggregated on frontend
	// This service returns a stub — real data flows through GetMarketOverview IPC
	return &MarketSentimentGauge{}, nil
}
