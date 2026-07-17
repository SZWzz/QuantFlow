package main

import (
	"context"
	"fmt"

	"quantflow/internal/market"
)

// GetStyleQuadrant returns index positions in the size×style space.
func (a *App) GetStyleQuadrant(ctx context.Context, market string) ([]market.StyleQuadrant, error) {
	if a.styleSvc == nil {
		return nil, fmt.Errorf("style service not initialized")
	}
	return a.styleSvc.GetStyleQuadrant(ctx, market)
}

// GetMarketSentiment returns sentiment gauges for a market.
func (a *App) GetMarketSentiment(ctx context.Context, market string) (*market.MarketSentimentGauge, error) {
	if a.styleSvc == nil {
		return nil, fmt.Errorf("style service not initialized")
	}
	return a.styleSvc.GetSentiment(ctx, market)
}
