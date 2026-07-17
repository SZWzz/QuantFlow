package main

import (
	"context"
	"fmt"

	"quantflow/internal/market"
)

// GetSectorHeatmap returns industry heatmap data for the sector dashboard.
func (a *App) GetSectorHeatmap(ctx context.Context, market string) ([]market.SectorHeat, error) {
	if a.sectorSvc == nil {
		return nil, fmt.Errorf("sector service not initialized")
	}
	return a.sectorSvc.GetSectorHeatmap(ctx, market)
}

// GetSectorValuation returns industry valuation data with PE/PB percentiles.
func (a *App) GetSectorValuation(ctx context.Context, market string) ([]market.SectorValuation, error) {
	if a.sectorSvc == nil {
		return nil, fmt.Errorf("sector service not initialized")
	}
	return a.sectorSvc.GetSectorValuation(ctx, market)
}
