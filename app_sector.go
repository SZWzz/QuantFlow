package main

import (
	"context"
	"sort"

	"quantflow/internal/market"
)

// GetSectorHeatmap returns industry heatmap data using the existing GetIndustryRanks.
func (a *App) GetSectorHeatmap(ctx context.Context, mkt string) ([]market.SectorHeat, error) {
	ranks, err := a.GetIndustryRanks(mkt, 31)
	if err != nil {
		return nil, err
	}

	result := make([]market.SectorHeat, 0, len(ranks))
	for _, r := range ranks {
		result = append(result, market.SectorHeat{
			Name:      r.Name,
			ChangePct: r.ChangePct,
		})
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].ChangePct > result[j].ChangePct
	})
	return result, nil
}

// GetSectorValuation returns industry valuation data.
func (a *App) GetSectorValuation(ctx context.Context, mkt string) ([]market.SectorValuation, error) {
	ranks, err := a.GetIndustryRanks(mkt, 31)
	if err != nil {
		return nil, err
	}

	result := make([]market.SectorValuation, 0, len(ranks))
	for _, r := range ranks {
		result = append(result, market.SectorValuation{
			Name: r.Name,
		})
	}
	return result, nil
}
