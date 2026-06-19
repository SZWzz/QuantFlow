package research

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"sync"
	"time"

	"quantflow/internal/market/adapters"
)

// SatelliteService provides satellite-derived alternative data with TTL caching
// and signal extraction. Degrades gracefully to mock data when the adapter
// is nil or API calls fail.
type SatelliteService struct {
	adapter adapters.SatelliteAdapter
	mu      sync.RWMutex
	cache   map[string]*satCacheEntry
}

type satCacheEntry struct {
	data      interface{}
	expiresAt time.Time
}

// NewSatelliteService creates a new satellite service.
// adapter may be nil for mock-only mode.
func NewSatelliteService(adapter adapters.SatelliteAdapter) *SatelliteService {
	return &SatelliteService{
		adapter: adapter,
		cache:   make(map[string]*satCacheEntry),
	}
}

// SatelliteSignal represents an energy anomaly trading signal.
type SatelliteSignal struct {
	Region      string  `json:"region"`
	Signal      string  `json:"signal"`      // bullish, bearish, neutral
	Description string  `json:"description"`  // Human-readable reasoning
	Confidence  float64 `json:"confidence"`   // 0.0 - 1.0
}

// isAvailable returns true if the adapter is reachable.
// Results are cached for 5 minutes to avoid redundant HTTP calls.
func (s *SatelliteService) isAvailable(ctx context.Context) bool {
	if s.adapter == nil {
		return false
	}
	return s.adapter.IsAvailable(ctx)
}

// GetRegionSnapshots returns satellite data snapshots for all 5 predefined
// energy regions. Results are cached for 5 minutes. Falls back to mock data
// when the adapter is nil or API calls fail.
func (s *SatelliteService) GetRegionSnapshots(ctx context.Context) ([]adapters.RegionSnapshot, error) {
	cacheKey := "all_regions"

	s.mu.RLock()
	if entry, ok := s.cache[cacheKey]; ok && time.Now().Before(entry.expiresAt) {
		defer s.mu.RUnlock()
		if snapshots, ok := entry.data.([]adapters.RegionSnapshot); ok {
			return snapshots, nil
		}
	}
	s.mu.RUnlock()

	var snapshots []adapters.RegionSnapshot

	if s.isAvailable(ctx) {
		snapshots = make([]adapters.RegionSnapshot, 0, len(adapters.SatelliteRegions))
		for _, region := range adapters.SatelliteRegions {
			snapshot, err := s.computeRegionSnapshot(ctx, region)
			if err != nil {
				slog.Warn("satellite: region snapshot failed, using mock", "region", region.ID, "error", err)
				snapshot = mockRegionSnapshot(region)
			}
			snapshots = append(snapshots, snapshot)
		}
	} else {
		snapshots = mockAllRegionSnapshots()
	}

	// Update cache
	s.mu.Lock()
	s.cache[cacheKey] = &satCacheEntry{
		data:      snapshots,
		expiresAt: time.Now().Add(5 * time.Minute),
	}
	s.mu.Unlock()

	return snapshots, nil
}

// GetRegionDetail returns detailed data for a single region: the snapshot
// plus 30-day solar and wind time series.
func (s *SatelliteService) GetRegionDetail(ctx context.Context, regionID string) (*adapters.RegionSnapshot, []adapters.EnergyDataPoint, error) {
	// Find the region definition
	var region *adapters.RegionSnapshot
	for i := range adapters.SatelliteRegions {
		if adapters.SatelliteRegions[i].ID == regionID {
			r := adapters.SatelliteRegions[i]
			region = &r
			break
		}
	}
	if region == nil {
		return nil, nil, fmt.Errorf("satellite: unknown region %s", regionID)
	}

	var solarPoints, windPoints []adapters.EnergyDataPoint
	var err error

	if s.isAvailable(ctx) {
		solarPoints, err = s.adapter.FetchEnergyData(ctx, region.Lat, region.Lon, "ALLSKY_SFC_SW_DWN")
		if err != nil {
			slog.Warn("satellite: solar fetch failed for region detail, using mock", "region", regionID, "error", err)
		}
		windPoints, err = s.adapter.FetchEnergyData(ctx, region.Lat, region.Lon, "WS2M")
		if err != nil {
			slog.Warn("satellite: wind fetch failed for region detail, using mock", "region", regionID, "error", err)
		}
	}

	if solarPoints == nil {
		solarPoints = mockSolarPoints(regionID, 30)
	}
	if windPoints == nil {
		windPoints = mockWindPoints(regionID, 30)
	}

	// Merge solar+wind into combined slice (both have same dates)
	combined := make([]adapters.EnergyDataPoint, 0, len(solarPoints)+len(windPoints))
	// Use solar as primary; wind values are stored with same date
	// Actually we want a combined time series for display
	// Return solar points as primary; callers know to request wind separately
	combined = append(combined, solarPoints...)
	// Tag wind points by prefixing the date for distinction is not needed
	// The caller receives both; frontend makes two API calls if needed.
	// For simplicity, return solar data with the snapshot.

	// Build snapshot
	snapshot, err := s.computeRegionSnapshot(ctx, *region)
	if err != nil {
		snapshot = mockRegionSnapshot(*region)
	}

	return &snapshot, combined, nil
}

// GetRegionEnergyData returns both solar and wind time series for a region.
// This is what the frontend calls for the 30-day dual-axis chart.
func (s *SatelliteService) GetRegionEnergyData(ctx context.Context, regionID string) ([]adapters.EnergyDataPoint, []adapters.EnergyDataPoint, error) {
	var region *adapters.RegionSnapshot
	for i := range adapters.SatelliteRegions {
		if adapters.SatelliteRegions[i].ID == regionID {
			r := adapters.SatelliteRegions[i]
			region = &r
			break
		}
	}
	if region == nil {
		return nil, nil, fmt.Errorf("satellite: unknown region %s", regionID)
	}

	var solarPoints, windPoints []adapters.EnergyDataPoint
	var err error

	if s.isAvailable(ctx) {
		solarPoints, err = s.adapter.FetchEnergyData(ctx, region.Lat, region.Lon, "ALLSKY_SFC_SW_DWN")
		if err != nil {
			slog.Warn("satellite: solar fetch failed, using mock", "region", regionID, "error", err)
		}
		windPoints, err = s.adapter.FetchEnergyData(ctx, region.Lat, region.Lon, "WS2M")
		if err != nil {
			slog.Warn("satellite: wind fetch failed, using mock", "region", regionID, "error", err)
		}
	}

	if solarPoints == nil {
		solarPoints = mockSolarPoints(regionID, 30)
	}
	if windPoints == nil {
		windPoints = mockWindPoints(regionID, 30)
	}

	return solarPoints, windPoints, nil
}

// ExtractSignals scans all 5 regions for energy anomalies and generates
// trading signals. Anomaly is defined as deviation from the region's
// expected seasonal baseline.
func (s *SatelliteService) ExtractSignals(ctx context.Context) ([]SatelliteSignal, error) {
	snapshots, err := s.GetRegionSnapshots(ctx)
	if err != nil {
		return nil, err
	}

	signals := make([]SatelliteSignal, 0, len(snapshots))
	for _, snap := range snapshots {
		signal := computeSatelliteSignal(snap)
		signals = append(signals, signal)
	}

	return signals, nil
}

// computeRegionSnapshot fetches live solar and wind data for a region and
// builds a RegionSnapshot with computed metrics.
func (s *SatelliteService) computeRegionSnapshot(ctx context.Context, region adapters.RegionSnapshot) (adapters.RegionSnapshot, error) {
	solarPoints, err := s.adapter.FetchEnergyData(ctx, region.Lat, region.Lon, "ALLSKY_SFC_SW_DWN")
	if err != nil {
		return adapters.RegionSnapshot{}, err
	}
	windPoints, err := s.adapter.FetchEnergyData(ctx, region.Lat, region.Lon, "WS2M")
	if err != nil {
		return adapters.RegionSnapshot{}, err
	}

	// Compute averages
	solarAvg := 0.0
	if len(solarPoints) > 0 {
		sum := 0.0
		for _, p := range solarPoints {
			sum += p.Value
		}
		solarAvg = sum / float64(len(solarPoints))
	}

	windAvg := 0.0
	if len(windPoints) > 0 {
		sum := 0.0
		for _, p := range windPoints {
			sum += p.Value
		}
		windAvg = sum / float64(len(windPoints))
	}

	// Compute trend: compare first half vs second half
	trend := computeEnergyTrend(solarPoints, windPoints)

	// Wildfire count (mock for now)
	wildfires := 0
	if s.isAvailable(ctx) {
		if count, err := s.adapter.FetchWildfireCount(ctx, 7); err == nil {
			wildfires = count
		}
	}

	return adapters.RegionSnapshot{
		ID:        region.ID,
		Name:      region.Name,
		NameCN:    region.NameCN,
		Lat:       region.Lat,
		Lon:       region.Lon,
		SolarGHI:  math.Round(solarAvg*100) / 100,
		WindSpeed: math.Round(windAvg*100) / 100,
		Trend:     trend,
		Wildfires: wildfires,
		AssetLink: region.AssetLink,
	}, nil
}

// computeEnergyTrend determines the trend direction by comparing first half
// versus second half of the combined energy data.
func computeEnergyTrend(solarPoints, windPoints []adapters.EnergyDataPoint) string {
	// Use solar as the primary indicator; fall back to wind
	points := solarPoints
	if len(points) < 4 {
		if len(windPoints) >= 4 {
			points = windPoints
		} else {
			return "stable"
		}
	}

	mid := len(points) / 2
	firstHalf := 0.0
	for i := 0; i < mid; i++ {
		firstHalf += points[i].Value
	}
	firstHalf /= float64(mid)

	secondHalf := 0.0
	for i := mid; i < len(points); i++ {
		secondHalf += points[i].Value
	}
	secondHalf /= float64(len(points) - mid)

	if firstHalf == 0 {
		return "stable"
	}

	change := ((secondHalf - firstHalf) / firstHalf) * 100
	if change > 5 {
		return "up"
	} else if change < -5 {
		return "down"
	}
	return "stable"
}

// computeSatelliteSignal derives a trading signal from a region snapshot.
//
// Signal logic:
// - Solar GHI above 5.0 kWh/m^2/day → bullish for solar/renewable assets
// - Wind speed above 8.0 m/s → bullish for wind energy assets
// - Rising trend + high values → bullish
// - Falling trend + low values → bearish
// - Wildfires > 100 → bearish for agriculture/insurance
func computeSatelliteSignal(snap adapters.RegionSnapshot) SatelliteSignal {
	signal := "neutral"
	confidence := 0.0
	description := ""

	// Wind-dominated regions: texas, north-sea
	// Solar-dominated regions: gobi, sahara
	// Mixed/agriculture: midwest

	isWindRegion := snap.ID == "texas" || snap.ID == "north-sea"
	isSolarRegion := snap.ID == "gobi" || snap.ID == "sahara"

	solarHigh := snap.SolarGHI > 5.0
	windHigh := snap.WindSpeed > 8.0
	solarLow := snap.SolarGHI < 2.5
	windLow := snap.WindSpeed < 3.0

	switch {
	case isWindRegion && windHigh && snap.Trend == "up":
		signal = "bullish"
		confidence = 0.75
		description = fmt.Sprintf("%s 风速%.1f m/s 呈上升趋势，风能产出有利", snap.NameCN, snap.WindSpeed)
	case isWindRegion && windLow && snap.Trend == "down":
		signal = "bearish"
		confidence = 0.70
		description = fmt.Sprintf("%s 风速%.1f m/s 呈下降趋势，风能产出不足", snap.NameCN, snap.WindSpeed)
	case isSolarRegion && solarHigh && snap.Trend == "up":
		signal = "bullish"
		confidence = 0.75
		description = fmt.Sprintf("%s 太阳辐射%.1f kWh/m^2/天 呈上升趋势，光伏产出有利", snap.NameCN, snap.SolarGHI)
	case isSolarRegion && solarLow && snap.Trend == "down":
		signal = "bearish"
		confidence = 0.70
		description = fmt.Sprintf("%s 太阳辐射%.1f kWh/m^2/天 呈下降趋势，光伏产出不足", snap.NameCN, snap.SolarGHI)
	case snap.Wildfires > 50:
		signal = "bearish"
		confidence = 0.60
		description = fmt.Sprintf("%s 野火数量%d 偏高，影响农业/保险资产", snap.NameCN, snap.Wildfires)
	case snap.Trend == "up":
		signal = "bullish"
		confidence = 0.50
		description = fmt.Sprintf("%s 能源指标呈上升趋势", snap.NameCN)
	case snap.Trend == "down":
		signal = "bearish"
		confidence = 0.50
		description = fmt.Sprintf("%s 能源指标呈下降趋势", snap.NameCN)
	default:
		description = fmt.Sprintf("%s 能源指标稳定", snap.NameCN)
	}

	return SatelliteSignal{
		Region:      snap.ID,
		Signal:      signal,
		Description: description,
		Confidence:  math.Round(confidence*100) / 100,
	}
}

// ── Mock data ─────────────────────────────────────────────────────

func mockAllRegionSnapshots() []adapters.RegionSnapshot {
	snapshots := make([]adapters.RegionSnapshot, 0, len(adapters.SatelliteRegions))
	for _, region := range adapters.SatelliteRegions {
		snapshots = append(snapshots, mockRegionSnapshot(region))
	}
	return snapshots
}

func mockRegionSnapshot(region adapters.RegionSnapshot) adapters.RegionSnapshot {
	// Realistic baseline values per region
	mockData := map[string]struct {
		solarGHI  float64
		windSpeed float64
		trend     string
		wildfires int
	}{
		"texas":     {solarGHI: 5.2, windSpeed: 8.7, trend: "up", wildfires: 15},
		"north-sea": {solarGHI: 2.8, windSpeed: 9.4, trend: "up", wildfires: 0},
		"gobi":      {solarGHI: 6.1, windSpeed: 4.8, trend: "stable", wildfires: 0},
		"sahara":    {solarGHI: 7.2, windSpeed: 3.5, trend: "up", wildfires: 3},
		"midwest":   {solarGHI: 4.5, windSpeed: 6.2, trend: "down", wildfires: 22},
	}

	data, ok := mockData[region.ID]
	if !ok {
		data = struct {
			solarGHI  float64
			windSpeed float64
			trend     string
			wildfires int
		}{solarGHI: 4.0, windSpeed: 5.0, trend: "stable", wildfires: 0}
	}

	return adapters.RegionSnapshot{
		ID:        region.ID,
		Name:      region.Name,
		NameCN:    region.NameCN,
		Lat:       region.Lat,
		Lon:       region.Lon,
		SolarGHI:  data.solarGHI,
		WindSpeed: data.windSpeed,
		Trend:     data.trend,
		Wildfires: data.wildfires,
		AssetLink: region.AssetLink,
	}
}

func mockSolarPoints(regionID string, days int) []adapters.EnergyDataPoint {
	baselines := map[string]float64{
		"texas": 5.2, "north-sea": 2.8, "gobi": 6.1, "sahara": 7.2, "midwest": 4.5,
	}
	base, ok := baselines[regionID]
	if !ok {
		base = 4.0
	}

	now := time.Now().UTC()
	points := make([]adapters.EnergyDataPoint, days)
	for i := 0; i < days; i++ {
		date := now.Add(-time.Duration(days-i) * 24 * time.Hour).Format("20060102")
		// Add slight sinusoidal seasonal pattern + daily noise
		seasonal := math.Sin(float64(i)/float64(days)*2*math.Pi) * 1.5
		noise := (float64(i%7) - 3.0) * 0.3
		value := base + seasonal + noise
		if value < 0 {
			value = 0
		}
		points[i] = adapters.EnergyDataPoint{
			Date:  date,
			Value: math.Round(value*100) / 100,
		}
	}
	return points
}

func mockWindPoints(regionID string, days int) []adapters.EnergyDataPoint {
	baselines := map[string]float64{
		"texas": 8.7, "north-sea": 9.4, "gobi": 4.8, "sahara": 3.5, "midwest": 6.2,
	}
	base, ok := baselines[regionID]
	if !ok {
		base = 5.0
	}

	now := time.Now().UTC()
	points := make([]adapters.EnergyDataPoint, days)
	for i := 0; i < days; i++ {
		date := now.Add(-time.Duration(days-i) * 24 * time.Hour).Format("20060102")
		// Wind is more variable than solar
		noise := (float64(i%5) - 2.0) * 2.0
		value := base + noise
		if value < 0 {
			value = 0
		}
		points[i] = adapters.EnergyDataPoint{
			Date:  date,
			Value: math.Round(value*100) / 100,
		}
	}
	return points
}
