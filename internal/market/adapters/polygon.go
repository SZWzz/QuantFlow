package adapters

import (
	"context"
	"fmt"
	"os"

	"quantflow/internal/market"
)

// PolygonAdapter fetches US equity data from Polygon.io (requires API key).
type PolygonAdapter struct {
	apiKey string
}

func NewPolygonAdapter() *PolygonAdapter {
	return &PolygonAdapter{apiKey: os.Getenv("POLYGON_API_KEY")}
}

func (a *PolygonAdapter) Name() string      { return "polygon" }
func (a *PolygonAdapter) Markets() []string  { return []string{"US"} }
func (a *PolygonAdapter) RequiresAuth() bool { return true }

func (a *PolygonAdapter) IsAvailable(ctx context.Context) bool {
	return a.apiKey != ""
}

func (a *PolygonAdapter) FetchQuote(ctx context.Context, symbol string) (*market.QuoteSnapshot, error) {
	if a.apiKey == "" {
		return nil, fmt.Errorf("polygon: POLYGON_API_KEY not set")
	}
	// Full HTTP implementation deferred — requires API key
	return nil, fmt.Errorf("polygon: not implemented (requires POLYGON_API_KEY)")
}

func (a *PolygonAdapter) FetchOHLCV(ctx context.Context, symbol string, interval string, _ string, start, end int64) ([]market.OHLCVBar, error) {
	if a.apiKey == "" {
		return nil, fmt.Errorf("polygon: POLYGON_API_KEY not set")
	}
	return nil, fmt.Errorf("polygon: not implemented")
}

func (a *PolygonAdapter) HealthCheck(ctx context.Context) error { return nil }
