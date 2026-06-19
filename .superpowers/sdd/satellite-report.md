# Satellite Alternative Data Module — Implementation Report

**Date**: 2026-06-19
**Branch**: main
**Commits**: 3

## Summary

Implemented the complete Satellite alternative data module for QuantFlow, integrating NASA POWER (solar/wind energy) and NASA FIRMS (wildfire) data. All 6 tasks completed across Go backend, workflow engine, and Vue frontend.

## Commits

### 1. `fa257e7` — feat: add Satellite adapter (NASA POWER + FIRMS)

**Files**:
- `internal/market/adapters/satellite.go` (new) — SatelliteAPI HTTP adapter
- `internal/market/adapters/satellite_test.go` (new) — 5 tests

**Details**:
- `SatelliteAdapter` interface with `Name()`, `IsAvailable()`, `FetchEnergyData()`, `FetchWildfireCount()`
- NASA POWER API integration: `https://power.larc.nasa.gov/api/temporal/daily/point`
  - Parameters: `ALLSKY_SFC_SW_DWN` (solar GHI) and `WS2M` (wind speed)
  - 30-day lookback, no API key required
- `RegionSnapshot` struct + 5 predefined `SatelliteRegions` (Texas, North Sea, Gobi, Sahara, Midwest)
- FIRMS wildfire: gracefully returns mock (real data requires MAP_KEY from NASA Earthdata)
- Tests: Name, IsAvailable, FetchEnergyData (solar + wind), FetchWildfireCount, SatelliteRegions validation

### 2. `857a37b` — feat: add SatelliteService with 5 energy regions and anomaly detection

**Files**:
- `internal/research/satellite_service.go` (new)

**Details**:
- `SatelliteService` wrapping `SatelliteAdapter` with 5-min TTL cache
- `GetRegionSnapshots()` — fetches all 5 regions with live/mock data
- `GetRegionDetail()` — single region detail with 30-day solar + wind time series
- `GetRegionEnergyData()` — separate solar and wind time series for dual-axis charts
- `ExtractSignals()` — energy anomaly detection returning `SatelliteSignal` (bullish/bearish/neutral)
- Signal logic: wind regions (Texas/North Sea) use wind speed >8 m/s threshold; solar regions (Gobi/Sahara) use solar GHI >5.0 kWh/m^2/day; trend comparison between first and second halves
- Region-specific mock data with realistic baselines (e.g., Gobi: 6.1 kWh/m^2, North Sea: 9.4 m/s)

### 3. `b08f1b6` — feat: add SatellitePanel Vue component + registry + tests

**Files**:
- `frontend/src/terminal/panels/SatellitePanel.vue` (new)
- `frontend/src/terminal/panels/registry.ts` (modified)
- `frontend/src/terminal/panels/__tests__/SatellitePanel.test.ts` (new)
- `internal/workflow/nodes/satellite.go` (new)
- `internal/workflow/nodes/research_deps.go` (modified)
- `internal/workflow/nodes/register.go` (modified)
- `app.go` (modified)
- `CHANGELOG.md` (modified)

**Details**:
- **SatellitePanel.vue**: 5 region cards in responsive grid, each showing Chinese name, solar GHI gauge (kWh/m^2/day), wind speed gauge (m/s), trend arrow (up/down/stable), wildfire count, and asset link badge. Click expands detail panel with ECharts 30-day solar/wind dual-axis line chart. Wails IPC or mock data fallback.
- **SatelliteNode**: workflow node with region input, energy_signal/solar_ghi/wind_speed outputs, anomaly_threshold param. Registered as "satellite" in "alternative_data" category.
- **Wiring**: `satelliteService` var + `SetSatelliteService()` in research_deps.go; node registration in register.go; adapter/service fields + startup initialization + `GetSatelliteSnapshots()` and `GetSatelliteDetail()` Wails methods in app.go.
- **Tests**: 5 vitest tests (panel header, region cards, trend badges, card click expand, data-panel-id) — all pass.

## Verification

| Check | Result |
|-------|--------|
| `go build ./internal/...` | PASS |
| `go vet ./internal/market/adapters/` | PASS |
| `go vet ./internal/research/` | PASS |
| `go vet ./internal/workflow/nodes/` | PASS |
| `go test ./internal/market/adapters/ -run TestSatellite` | 5/5 PASS |
| `go test ./internal/workflow/nodes/ -run TestRegister` | PASS |
| `cd frontend && npx vitest run` | 184/184 PASS |
| CHANGELOG updated | DONE |

## Architecture

```
NASA POWER API ──→ SatelliteHTTPAdapter ──→ SatelliteService ──→ SatelliteNode
  (free, no key)     (adapters/)              (research/)         (workflow/nodes/)
                                                      │
                                                      ▼
                                               app.go (Wails IPC)
                                                      │
                                                      ▼
                                              SatellitePanel.vue
                                                (Vue 3 + ECharts)
```

## Data Flow

1. `SatelliteHTTPAdapter.FetchEnergyData(lat, lon, parameter)` → NASA POWER JSON
2. `SatelliteService.GetRegionSnapshots()` → fetches all 5 regions → caches 5 min
3. `SatelliteService.ExtractSignals()` → computes trend + anomaly → SatelliteSignal[]
4. Frontend calls `GetSatelliteSnapshots()` / `GetSatelliteDetail(regionID)` via Wails IPC
5. `SatellitePanel.vue` renders cards + ECharts dual-axis chart

## Pre-defined Regions

| ID | Name (CN) | Lat/Lon | Asset Link | Solar (mock) | Wind (mock) |
|----|-----------|---------|------------|-------------|------------|
| texas | 德州风能走廊 | 32.8, -100.1 | 天然气/电力 | 5.2 | 8.7 |
| north-sea | 北海风电场 | 56.0, 3.0 | 欧洲电力/天然气 | 2.8 | 9.4 |
| gobi | 戈壁太阳能基地 | 40.5, 100.0 | 中国新能源/多晶硅 | 6.1 | 4.8 |
| sahara | 撒哈拉太阳能带 | 23.0, 13.0 | 欧洲碳配额 | 7.2 | 3.5 |
| midwest | 美国中西部农业带 | 41.0, -93.0 | 玉米/大豆/小麦期货 | 4.5 | 6.2 |
