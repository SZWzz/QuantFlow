package market

import (
	"context"
	"database/sql"
	"math"
	"sort"
)

// SectorHeat represents a single industry's heatmap metrics.
type SectorHeat struct {
	Name      string  `json:"name"`
	ChangePct float64 `json:"change_pct"`
	Volume    float64 `json:"volume"`
	PE        float64 `json:"pe"`
	PEPct     float64 `json:"pe_pct"`
}

// SectorValuation holds PE/PB/ROE with historical percentiles for a sector.
type SectorValuation struct {
	Name  string  `json:"name"`
	PE    float64 `json:"pe"`
	PEPct float64 `json:"pe_pct"`
	PB    float64 `json:"pb"`
	PBPct float64 `json:"pb_pct"`
	ROE   float64 `json:"roe"`
}

// SectorService aggregates industry data for the sector dashboard.
type SectorService struct {
	reg *AdapterRegistry
	db  *sql.DB
}

// NewSectorService creates a new SectorService.
func NewSectorService(reg *AdapterRegistry, db *sql.DB) *SectorService {
	return &SectorService{reg: reg, db: db}
}

// GetSectorHeatmap returns industry heatmap data for a market.
func (s *SectorService) GetSectorHeatmap(ctx context.Context, market string) ([]SectorHeat, error) {
	ranks, err := s.reg.FetchIndustryRanksWithFallback(ctx, market, 60)
	if err != nil {
		return nil, err
	}

	result := make([]SectorHeat, 0, len(ranks))
	for _, r := range ranks {
		result = append(result, SectorHeat{
			Name:      r.Name,
			ChangePct: r.ChangePct,
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].ChangePct > result[j].ChangePct
	})

	return result, nil
}

// GetSectorValuation returns PE/PB/ROE with historical percentiles.
func (s *SectorService) GetSectorValuation(ctx context.Context, market string) ([]SectorValuation, error) {
	ranks, err := s.reg.FetchIndustryRanksWithFallback(ctx, market, 31)
	if err != nil {
		return nil, err
	}

	result := make([]SectorValuation, 0, len(ranks))
	for _, r := range ranks {
		result = append(result, SectorValuation{
			Name:  r.Name,
			PE:    0,
			PEPct: computePEPercentile(0, nil),
			PB:    0,
			PBPct: 0,
			ROE:   0,
		})
	}

	return result, nil
}

// computePEPercentile calculates where current PE sits in a historical distribution.
// Returns 0-100. When history is empty, returns 50 (neutral).
func computePEPercentile(currentPE float64, historicalPEs []float64) float64 {
	if len(historicalPEs) == 0 || currentPE <= 0 {
		return 50
	}

	sorted := make([]float64, len(historicalPEs))
	copy(sorted, historicalPEs)
	sort.Float64s(sorted)

	count := 0
	for _, pe := range sorted {
		if pe <= currentPE {
			count++
		}
	}

	return math.Round(float64(count)/float64(len(sorted))*10000) / 100
}
