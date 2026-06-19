# Geopolitics Module — 4-Task Backend Implementation Report

**Date**: 2026-06-19
**Spec**: `docs/superpowers/specs/2026-06-19-geopolitics-design.md`
**Status**: All 4 backend tasks complete

## Task Summary

| Task | File(s) | Lines | Commit |
|------|---------|-------|--------|
| 1. GDELT Adapter | `internal/market/adapters/gdelt.go` + `_test.go` | 367 | `26cc89a` |
| 2. GeopoliticsService | `internal/research/geopolitics_service.go` | 417 | `0b770e0` |
| 3. Geopolitics Node | `internal/workflow/nodes/geopolitics.go` | 141 | `85545c4` |
| 4. Wire everything | `research_deps.go`, `register.go`, `app.go` | 38 | `33a9656` |

## Task 1: GDELT Adapter

- `GeopoliticsAdapter` interface with `FetchTopicVolume` and `FetchTopicTone`
- `GDELTAdapter` struct: `*http.Client` (30s timeout), `TopicQueries map[string]TopicQuery`
- 10 pre-defined topic queries (URL-encoded for GDELT DOC 2.0 API)
- `TopicQuery` struct: ID, Title, Query, Associated
- `IsAvailable` via HEAD request
- Tests skip gracefully when API unreachable

## Task 2: GeopoliticsService

- Wraps GDELT adapter with TTL cache (5min)
- `GetTopicRisks()` — returns all 10 TopicRisk with risk levels
- `GetTopicDetail()` — returns volume + tone time series
- `ExtractRiskSignals(minVolChange)` — detects coverage + tone anomalies
- Risk scoring: vol>50% + tone<-2 = high / vol>20% or tone<0 = medium / else low
- Mock data for all 10 topics with realistic risk profiles
- Chinese title mapping (`topicTitleCN`)

## Task 3: Geopolitics Node

- NodeType: `geopolitics`, Category: `alternative_data`
- Inputs: topic (string), region (string)
- Outputs: risk_signal (Signal), risk_score (number), tone (number)
- Params: topic, min_vol_change (default 50)
- Signal action: high→sell (0.8), medium→sell (0.4), low→hold (0.1)
- Degrades to mock when service not configured

## Task 4: Wiring

- `research_deps.go`: Added `geopoliticsService` var + `SetGeopoliticsService()`
- `register.go`: Registered `geopolitics` node in `alternative_data` category
- `app.go`: 
  - Fields: `geopoliticsAdpt`, `geopoliticsSvc`
  - startup(): Init GDELT adapter → GeopoliticsService → wire to nodes
  - `GetGeopoliticsRisks()` Wails method
  - `GetGeopoliticsDetail(topicID, timespan)` Wails method

## Verification

- `go build ./internal/...` — passes
- `go vet ./internal/...` — only pre-existing backtest warning, no new issues
- The `go build ./...` fails due to pre-existing `main.go` frontend embed issue (unrelated)

## Remaining (Frontend)

The spec also calls for a `GeopoliticsPanel` Vue component (card grid, ECharts, risk filter). This is frontend work and not part of this backend batch.
