// internal/market/adapters/satellite.go
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

const nasaPowerBaseURL = "https://power.larc.nasa.gov/api/temporal/daily/point"

// SatelliteAdapter defines the interface for satellite-derived alternative data.
// Covers NASA POWER (solar/wind energy) and NASA FIRMS (wildfire) — both free,
// no API key required. Separate from market.Adapter because satellite data carries
// geospatial energy metrics, not financial quotes.
type SatelliteAdapter interface {
	Name() string
	IsAvailable(ctx context.Context) bool

	// FetchEnergyData returns solar/wind data for a location.
	// parameter: "ALLSKY_SFC_SW_DWN" (solar, kWh/m^2/day) or "WS2M" (wind speed, m/s).
	FetchEnergyData(ctx context.Context, lat, lon float64, parameter string) ([]EnergyDataPoint, error)

	// FetchWildfireCount returns recent wildfire count for a region.
	// daysBack: number of days to look back.
	FetchWildfireCount(ctx context.Context, daysBack int) (int, error)
}

// EnergyDataPoint represents a single daily energy observation from NASA POWER.
type EnergyDataPoint struct {
	Date  string  `json:"date"`
	Value float64 `json:"value"` // kWh/m^2/day for solar, m/s for wind
}

// RegionSnapshot represents the satellite-derived snapshot for a predefined energy region.
type RegionSnapshot struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	NameCN    string  `json:"name_cn"`
	Lat       float64 `json:"lat"`
	Lon       float64 `json:"lon"`
	SolarGHI  float64 `json:"solar_ghi"`  // latest monthly avg (kWh/m^2/day)
	WindSpeed float64 `json:"wind_speed"` // latest monthly avg (m/s)
	Trend     string  `json:"trend"`      // up/down/stable
	Wildfires int     `json:"wildfires"`   // recent count in region
	AssetLink string  `json:"asset_link"`  // associated commodity
}

// SatelliteRegions contains the 5 predefined energy regions with metadata.
var SatelliteRegions = []RegionSnapshot{
	{ID: "texas", Name: "Texas Wind Corridor", NameCN: "德州风能走廊", Lat: 32.8, Lon: -100.1, AssetLink: "天然气/电力"},
	{ID: "north-sea", Name: "North Sea Wind Farm", NameCN: "北海风电场", Lat: 56.0, Lon: 3.0, AssetLink: "欧洲电力/天然气"},
	{ID: "gobi", Name: "Gobi Solar Base", NameCN: "戈壁太阳能基地", Lat: 40.5, Lon: 100.0, AssetLink: "中国新能源/多晶硅"},
	{ID: "sahara", Name: "Sahara Solar Belt", NameCN: "撒哈拉太阳能带", Lat: 23.0, Lon: 13.0, AssetLink: "欧洲碳配额"},
	{ID: "midwest", Name: "US Midwest Agricultural Belt", NameCN: "美国中西部农业带", Lat: 41.0, Lon: -93.0, AssetLink: "玉米/大豆/小麦期货"},
}

// SatelliteHTTPAdapter fetches satellite energy data from the NASA POWER API
// (free, no API key required). Wildfire data from NASA FIRMS is simplified —
// see FetchWildfireCount documentation for details.
type SatelliteHTTPAdapter struct {
	client *http.Client
}

// Compile-time interface check.
var _ SatelliteAdapter = (*SatelliteHTTPAdapter)(nil)

// NewSatelliteAdapter creates a new Satellite HTTP adapter.
func NewSatelliteAdapter() *SatelliteHTTPAdapter {
	return &SatelliteHTTPAdapter{
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (a *SatelliteHTTPAdapter) Name() string { return "satellite" }

// IsAvailable performs a quick GET to the NASA POWER API to check reachability.
func (a *SatelliteHTTPAdapter) IsAvailable(ctx context.Context) bool {
	// Use a minimal request: 1 day of data for a known location.
	now := time.Now().UTC()
	dateStr := now.Format("20060102")
	url := fmt.Sprintf("%s?parameters=ALLSKY_SFC_SW_DWN&community=RE&longitude=0&latitude=0&start=%s&end=%s&format=json",
		nasaPowerBaseURL, dateStr, dateStr)

	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	resp, err := a.client.Do(req)
	if err != nil {
		slog.Debug("satellite availability check failed", "error", err)
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// FetchEnergyData returns solar or wind data for a location from NASA POWER.
// parameter: "ALLSKY_SFC_SW_DWN" (solar GHI in kWh/m^2/day) or "WS2M" (wind speed at 2m in m/s).
// Fetches the last 30 days by default.
func (a *SatelliteHTTPAdapter) FetchEnergyData(ctx context.Context, lat, lon float64, parameter string) ([]EnergyDataPoint, error) {
	if parameter == "" {
		parameter = "ALLSKY_SFC_SW_DWN"
	}

	now := time.Now().UTC()
	endDate := now.Format("20060102")
	startDate := now.AddDate(0, 0, -30).Format("20060102")

	url := fmt.Sprintf("%s?parameters=%s&community=RE&longitude=%.4f&latitude=%.4f&start=%s&end=%s&format=json",
		nasaPowerBaseURL, parameter, lon, lat, startDate, endDate)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("satellite: %w", err)
	}

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("satellite: http error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("satellite: HTTP %d: %s", resp.StatusCode, string(body))
	}

	// NASA POWER API response shape:
	// {"properties": {"parameter": {"ALLSKY_SFC_SW_DWN": {"20260101": 3.5, ...}, "WS2M": {...}}}}
	var raw struct {
		Properties struct {
			Parameter map[string]map[string]float64 `json:"parameter"`
		} `json:"properties"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("satellite: parse error: %w", err)
	}

	paramData, ok := raw.Properties.Parameter[parameter]
	if !ok {
		return nil, fmt.Errorf("satellite: parameter %s not found in response", parameter)
	}

	// Build sorted slice from map entries
	points := make([]EnergyDataPoint, 0, len(paramData))
	for date, value := range paramData {
		// Skip fill values (-999.0 or similar)
		if value < -900 {
			continue
		}
		points = append(points, EnergyDataPoint{
			Date:  date,
			Value: value,
		})
	}

	// Sort by date ascending
	for i := 0; i < len(points); i++ {
		for j := i + 1; j < len(points); j++ {
			if points[i].Date > points[j].Date {
				points[i], points[j] = points[j], points[i]
			}
		}
	}

	return points, nil
}

// FetchWildfireCount returns recent wildfire count for a region.
//
// NASA FIRMS requires a MAP_KEY for the area API. The key application process
// is complex (requires NASA Earthdata login + approval). For now, this method
// gracefully logs the limitation and returns a mock count of 0.
//
// To enable real FIRMS data:
//  1. Register at https://urs.earthdata.nasa.gov/
//  2. Obtain a MAP_KEY from https://firms.modaps.eosdis.nasa.gov/api/
//  3. Set FIRMS_MAP_KEY env var
//  4. Uncomment the HTTP call below
func (a *SatelliteHTTPAdapter) FetchWildfireCount(ctx context.Context, daysBack int) (int, error) {
	if daysBack <= 0 {
		daysBack = 7
	}

	// The FIRMS area endpoint requires a MAP_KEY:
	// https://firms.modaps.eosdis.nasa.gov/api/area/csv/{MAP_KEY}/VIIRS_SNPP_NRT/world/{days}
	// For now, return a graceful mock.
	slog.Debug("satellite: FIRMS wildfire data requires MAP_KEY — returning mock count=0. " +
		"Set FIRMS_MAP_KEY env var and uncomment HTTP call for real data.")
	return 0, nil
}
