# Wire All Data Source Adapters to Services

## Motivation

`startup()` creates 7 Phase 2/3 adapters but only 3 are actually called by services. Six additional adapters have constructors and passing tests but are never instantiated. Result: every panel except Sentiment shows mock data.

This spec fixes the adapter→service→frontend data pipeline for all 13 research adapters.

## Design

### New Services (4 new files)

| Service | Adapter | Methods |
|---------|---------|---------|
| `CapitalService` | `EastMoneyCapitalAdapter` | `GetMarginTrading`, `GetBlockTrades`, `GetHolderChanges`, `GetDividendHistory` |
| `FundFlowService` | `EastMoneyFundFlowAdapter` | `GetMinuteFlow`, `GetDailyFlow` |
| `NorthboundService` | `THSNorthboundAdapter` | `GetMinuteFlow`, `GetHistory` |
| `AnnouncementService` | `CninfoAdapter` | `GetAnnouncements` |

### Fixed Existing Services (3 modified files)

| Service | Adapter Added | Change |
|---------|--------------|--------|
| `FinancialsService` | `SinaFinancialsAdapter` (already has it!) | `GetFinancials()` calls `FetchIncomeStatement`+`FetchBalanceSheet` to build real `FinancialData` |
| `AnalystEstimatesService` | `EastMoneyReportAdapter` + `THSConsensusAdapter` | `GetEstimates()` calls `FetchReports` for real ratings; `GetConsensusEPS()` new method |
| `PeerComparisonService` | `EastMoneyConceptAdapter` (already has) | `GetPeers()` fills MarketCap/PE/ROE from concept data when available |

### New Wiring (app.go)

```
New adapters to create in startup():
  EastMoneyFundFlowAdapter   → FundFlowService  
  THSNorthboundAdapter       → NorthboundService
  EastMoneyReportAdapter     → AnalystEstimatesService
  THSConsensusAdapter        → AnalystEstimatesService
  CninfoAdapter              → AnnouncementService
  EastMoneyGlobalNewsAdapter → SentimentEngine (secondary news source)

Existing orphan:
  EastMoneyCapitalAdapter    → CapitalService
```

### New package-level variables (research_deps.go)

```go
var capitalService *research.CapitalService
var fundFlowService *research.FundFlowService
var northboundService *research.NorthboundService
var announcementService *research.AnnouncementService
```

## Data Flow

```
Adapter (HTTP fetch)
  → Service (parse + transform to domain model)
    → App method (Wails-exported) — optional, for direct panel access
      → Frontend Panel (.vue)
```

Panels can access data via:
1. `App.GetStockResearch()` — aggregates financials, peers, estimates, insider, sentiment
2. Direct App methods (e.g., `GetCapitalData`, `GetFundFlow`) — for specialized panels
3. Workflow nodes — for drag-and-drop composition

## Files Changed

### New
- `internal/research/capital_service.go`
- `internal/research/fundflow_service.go`
- `internal/research/northbound_service.go`
- `internal/research/announcement_service.go`

### Modified
- `internal/research/financials_service.go` — real data from Sina
- `internal/research/analyst_estimates_service.go` — real data from report + consensus
- `internal/research/peer_comparison_service.go` — fill 0-value fields
- `internal/workflow/nodes/research_deps.go` — new package-level vars
- `app.go` — create all adapters + services, new exported methods
- `CHANGELOG.md`

## Acceptance Criteria

- [ ] `FinancialsService.GetFinancials()` calls `SinaFinancialsAdapter` when available, returns parsed data
- [ ] `AnalystEstimatesService.GetEstimates()` calls `EastMoneyReportAdapter` for real reports
- [ ] `AnalystEstimatesService` has new `GetConsensusEPS()` using `THSConsensusAdapter`
- [ ] `CapitalService` wraps `EastMoneyCapitalAdapter` with all 4 data methods
- [ ] `FundFlowService` wraps `EastMoneyFundFlowAdapter`
- [ ] `NorthboundService` wraps `THSNorthboundAdapter`
- [ ] `AnnouncementService` wraps `CninfoAdapter`
- [ ] `EastMoneyGlobalNewsAdapter` is created and passed to `SentimentEngine`
- [ ] All services degrade gracefully to mock when adapter is nil or API fails
- [ ] `go vet ./internal/research/...` passes
- [ ] `go test ./internal/research/...` passes

## Risks / Trade-offs

- **Sina parser brittleness**: Sina's JSON response has Chinese item titles (e.g., "营业总收入"). We match by keyword; field names may change.
- **Rate limiting**: EastMoney adapters share a global limiter; adding more calls means slower startup. Services should cache results.
- **THS HTML parsing**: THSConsensusAdapter parses HTML tables — fragile if THS changes page structure.
- **Cninfo startup latency**: First call loads orgId map (6198 stocks, ~500KB JSON). Use `sync.Once` (already implemented).
