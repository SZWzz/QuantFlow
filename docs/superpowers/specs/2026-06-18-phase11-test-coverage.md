# Phase 11: Test Coverage Hardening

> **Priority**: Frontend (A) → Go deep (C) → Python (B)

## Motivation

Phase 1-10 built 59 workflow nodes, 22 frontend panels, 8 Pinia stores, 13 market data adapters, and 5 ML engines. Current test coverage is uneven and insufficient for production readiness:

- **Frontend**: Zero test files. 37 Vue components, 8 Pinia stores, 3 lib modules — none tested.
- **Go**: 176 test functions across 41 test files for 141 source files. But `market/adapters/` (13 files, 0 tests), `ai/capabilities/` (3 files, 0 tests), and several packages have only 1-2 tests.
- **Python**: 82 test functions for 39 source files. Factor modules (volatility, volume, cross_sectional), LLM providers (4), and data fetcher are untested.

This phase adds a testing safety net before further feature development.

## Design

### Sub-Phase A: Frontend Test Infrastructure + Store Tests + Panel Smoke Tests

**Goal**: Every store has unit tests. Every panel renders without crashing. vitest + jsdom infrastructure is already configured in `vite.config.ts`.

**Store tests** (TDD, no Wails mock needed — stores are pure Pinia):
- `data.ts` — quote CRUD, OHLCV cache, offline toggle
- `workflow.ts` — node/edge CRUD, undo/redo, serialization, execution state
- `terminal.ts` — panel open/close/active, mode switch
- `session.ts` — login state, token management
- `settings.ts` — settings CRUD, persistence
- `notify.ts` — notification add/dismiss/clear
- `portfolio.ts` — position CRUD, P&L computation
- `ml.ts` — model CRUD, training state, RL episode tracking, risk result

**Panel smoke tests** (mount with stubs, verify render):
- Terminal panels: Watchlist, QuoteDetail, Candlestick, OrderEntry, Position, News, AIChat, SystemMonitor, BacktestResult, FactorAnalysis, PortfolioSummary, PositionDetail, RiskDashboard, TradeHistory, Schedule, Notify, BrokerConfig, Settings, ModelRegistry, PredictionDashboard, AlphaMiningWorkspace, RLMonitor
- Workflow components: NodePalette, PropertyPanel, ExecutionLog, WorkflowCanvas, CustomNode, CommandBar, StatusBar, PushPinBar
- DockView components: DockView, DockContainer, DockSplitter, DockTab

### Sub-Phase C: Go Deep — Market Adapters + AI Capabilities

**Goal**: Test the data ingestion layer (13 adapters) and AI capabilities (3 files).

**Market adapter tests** (table-driven with httptest mocks):
- Each adapter's `FetchRealtime()` and its symbol normalization
- `FetchHistory()` time range parsing
- JSON response parsing (mock HTTP responses)
- Error handling (timeout, malformed response, empty body)
- `ParseCode()` / symbol conversion functions

**AI capability tests**:
- `factor.go` — Factor analysis prompt construction
- `quote.go` — Multi-source quote synthesis prompt
- `skills.go` — Skill routing and dispatch

**Also strengthen**:
- `internal/storage/db_test.go` — expand from 1 to 5+ tests
- `internal/config/config_test.go` — expand from 2 to 5+ tests
- `internal/schedule/scheduler_test.go` — expand from 2 to 5+ tests
- `internal/notify/manager_test.go` — expand from 2 to 5+ tests

### Sub-Phase B: Python Factor + Provider Tests

**Goal**: Cover untested factor modules and LLM providers.

**New test files**:
- `tests/test_factor_volatility.py` — Volatility factor computation
- `tests/test_factor_volume.py` — Volume factor computation
- `tests/test_factor_cross_sectional.py` — Cross-sectional factors
- `tests/test_llm_providers.py` — Provider instantiation, prompt formatting, error handling

**Strengthen existing**:
- `tests/test_ml_service.py` — Add streaming RLTrain test, RiskModel test
- `tests/test_alpha_mining.py` — Add evaluator edge cases

## Acceptance Criteria

### A: Frontend
- [ ] `vitest run` passes with ≥40 test files
- [ ] All 8 Pinia stores have ≥80% action coverage
- [ ] All 22 panels mount without crashing (smoke test)
- [ ] Workflow components (NodePalette, PropertyPanel, ExecutionLog, CustomNode) mount without crashing
- [ ] DockView components (DockView, DockContainer, DockSplitter, DockTab) mount without crashing

### C: Go Deep
- [ ] All 13 market adapters have table-driven tests (mock HTTP)
- [ ] All 3 AI capability files have tests
- [ ] `storage`, `config`, `schedule`, `notify` packages each have ≥5 test functions
- [ ] `go test ./...` count reaches ≥240 (from 176)

### B: Python
- [ ] Factor volatility, volume, cross_sectional each have ≥3 tests
- [ ] LLM providers (4) each have ≥2 tests
- [ ] Python test count reaches ≥110 (from 82)

## Risks / Trade-offs

- **Adapter tests with httptest**: Mock HTTP is reliable but doesn't catch real API changes. Acceptable — integration tests against live APIs are a separate concern.
- **Vue panel smoke tests are shallow**: They verify mount-ability, not business logic. This is intentional for Phase 11 — deep interaction tests (button clicks, form input) would blow up scope.
- **vitest 2.x + Vite 6**: Already installed and compatible. `test.globals: true` means `describe`/`it`/`expect` are available without imports.
- **Pinia requires `setActivePinia(createPinia())` per test**: Standard pattern, documented in Pinia testing guide.
