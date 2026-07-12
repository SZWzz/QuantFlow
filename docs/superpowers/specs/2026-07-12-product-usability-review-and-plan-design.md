# QuantFlow Product Usability — Review & Execution Plan

> **Date**: 2026-07-12
> **Type**: Comprehensive Review + A+B Hybrid Execution Plan
> **Priority**: Product Usability (产品可用性)

---

## 1. Executive Summary

QuantFlow has 65K lines of Go, 33K lines of Vue (93 panels), 39K lines of Python, with **981/983 Go tests passing (99.8%)** and **198/198 frontend tests passing (100%)**. The architecture is sound, the code is clean (4 TODOs total), and **88% of panels (82/93) are wired to real data**.

However, **three critical usability breaks** prevent the product from being a daily-driver:

1. **OrderEntryPanel broker routing is non-functional** — all orders go to paper engine regardless of broker selection
2. **OrderEntryPanel hardcodes CN market** — US/HK/Crypto order entry gets broken price quotes
3. **Review panels show paper-only data** — no real broker position/trade/portfolio sync

Fixing these three breaks + eliminating 9 mock panels will make QuantFlow "daily usable."

---

## 2. Current State Assessment

### 2.1 Panel Wiring Status (93 panels total)

| Classification | Count | Description |
|----------------|-------|-------------|
| Wired (Stores) | 25 | Use Pinia stores (data/portfolio/research/ml/settings/terminal/notify) |
| Wired (Direct Go) | 23 | Call Go backend via `window.go.main.App.*` or composables |
| Wired (panelCache) | 33 | Use `usePanelCache` to call Go backend (structurally wired) |
| Mock | 9 | Hardcoded/mock data for primary content |
| Static/Stub | 1 | RLMonitorPanel (placeholder) |

### 2.2 Go Backend (139 exported methods)

All `*App` methods are auto-exposed to frontend via Wails v3 `application.NewService`. Frontend calls via `window.go.main.App.MethodName(...)` Proxy shim. **No registration step needed** — adding a Go method makes it instantly callable from any panel.

### 2.3 Core Flow Health

| Flow | Status | Key Breaks |
|------|--------|------------|
| Flow 1: 看盘 (Search→Watchlist→K-line→Indicators) | 🟢 Fully wired | IndicatorPanel symbol not linked to active symbol (UX only) |
| Flow 2: 下单 (Order→Position→Risk→Broker) | 🔴 3 breaks | Broker routing, CN hardcode, BrokerStatus hardcode |
| Flow 3: 复盘 (Backtest→Trades→Portfolio) | 🟡 2 breaks | RunBacktest not wired, paper-only data |

### 2.4 Known Defects

| Severity | Issue | File(s) |
|----------|-------|---------|
| 🔴 | Python tests: 0 collected (conftest.py missing + import path) | python/tests/ |
| 🟡 | GDELT tests: HTTP 429 rate limit | gdelt_test.go |
| 🟡 | Implementation status doc: says 22 panels/54 nodes, actual 93/196 | proposal-implementation-status.md |

---

## 3. Execution Plan

### Phase 1: Fix Order Flow (Week 1)

**Goal**: OrderEntryPanel works with real broker routing.

#### Task 1.1: Fix market detection in OrderEntryPanel

**Current**: `app.GetQuote('CN', symbol)` — hardcoded to CN.
**Fix**: Detect market from symbol format or symbolContext active symbol.
**Files**: `frontend/src/terminal/panels/OrderEntryPanel.vue`
**Lines**: `fetchQuote()` function (~line 26-43)

```typescript
// Before:
const quote = await app.GetQuote('CN', symbol.value)

// After:
const market = detectMarket(symbol.value) // 'CN'|'HK'|'US'|'CRYPTO'
const quote = await app.GetQuote(market, symbol.value)
```

#### Task 1.2: Add broker routing to PlaceOrder

**Current**: `PlaceOrder(symbol, side, orderType, qty, price)` — no broker param, always routes to paper engine.
**Fix**: Add `brokerName string` parameter to Go method. OMS routes to broker if name != "paper".
**Files**: `app_trading.go`, `internal/trading/oms.go`, `OrderEntryPanel.vue`

```go
// app_trading.go
func (a *App) PlaceOrder(symbol, side, orderType, brokerName string, qty, price float64) (*trading.Order, error) {
    return a.oms.PlaceOrder(symbol, side, orderType, brokerName, qty, price)
}
```

#### Task 1.3: Wire BrokerStatusPanel to real brokers

**Current**: Returns hardcoded `[{Name: "paper", Connected: true}]`.
**Fix**: Probe all registered brokers via `Broker.IsConnected()`.
**Files**: `app_trading.go`

#### Task 1.4: End-to-end verification

- Paper order → position update → trade history
- Verify market auto-detection for CN/HK/US/Crypto symbols
- Verify BrokerStatusPanel shows actual broker states

---

### Phase 2: Fix Review Flow (Week 2)

**Goal**: Backtest and trade review panels show meaningful data.

#### Task 2.1: Resolve RunBacktest dead end

**Current**: `RunBacktest()` returns `"not yet wired"`.
**Fix**: Remove the standalone `RunBacktest` Go method and frontend button. Unify on workflow-based backtesting (which already works).
**Files**: `app_trading.go`, `BacktestPanel.vue` (if it calls RunBacktest directly)

#### Task 2.2: Portfolio/Trade panels support real broker data

**Current**: `GetPositions()`/`GetOrders()`/`GetTrades()` return paper OMS data only.
**Fix**: Add optional `brokerName` filter parameter. When specified, delegate to broker implementation.
**Files**: `app_trading.go`, `internal/trading/oms.go`

#### Task 2.3: Backtest→Review data chain verification

- Run a workflow backtest → verify persistence to SQLite
- Verify BacktestPanel list loads stored results
- Verify equity curve and trade markers render correctly

---

### Phase 3: Quality Cleanup (Week 3-4)

**Goal**: Eliminate known defects, mock panels, and outdated docs.

#### Task 3.1: Fix Python test collection 🔴

**Root cause**: Missing `conftest.py`, import path mismatch.
**Fix**: Create `python/conftest.py` with `sys.path.insert(0, 'src')`. Fix any import errors in 20 test files.
**Verification**: `python -m pytest tests/ -x -q` collects and runs all tests.

#### Task 3.2: Fix GDELT test failures

**Root cause**: GDELT API rate limits (HTTP 429).
**Fix**: Add rate-limit detection in test helpers; auto-skip when rate-limited.
**Files**: `internal/market/adapters/gdelt_test.go`

#### Task 3.3: Eliminate 9 mock panels

| Panel | Fix |
|-------|-----|
| AIChatPanel.vue | Wire `send` to `app.Chat()` (Go method exists) |
| BasketOrderPanel.vue | Wire order submission to `app.PlaceOrder()` |
| BrokerConfig.vue | Wire save/load to `app.SaveCredential()`/`app.ListCredentials()` |
| ChanlunPanel.vue | Wire to `app.GetChanlun()` (Go method exists) |
| EconomicCalendarPanel.vue | Remove mock fallback, use `app.GetEconomicIndicators()` only |
| FactorAnalysisPanel.vue | Wire to Python factor registry sync |
| IndicatorPanel.vue | Wire `compute` to `app.ComputeIndicator()` (Go method exists) |
| StockScannerPanel.vue | Wire `scan` to `app.ScanStocks()` (Go method exists) |
| MonteCarloPanel.vue | Functional as-is (client-side math); optional Go acceleration |

#### Task 3.4: Update implementation status document

Update `docs/specs/2026-06-18-proposal-implementation-status.md`:
- Panel count: 22 → 93
- Node count: 54 → 196
- Reflect actual wiring status

---

## 4. Data Flow Reference

### How frontend calls Go backend

```
Vue Panel
  → window.go.main.App.MethodName(args...)  // Proxy shim (wails.ts:85-108)
  → Call.ByName("main.App.MethodName", args...)  // Wails v3 IPC
  → App.MethodName(ctx, args...)  // Go exported method
  → Internal service / adapter chain
```

### How real-time data flows

```
QuotePoller (Go, goroutine)
  → adapter.FetchQuote() → data source (eastmoney/tencent/binance/...)
  → ws.Hub.Broadcast(topic, data)  // Go WebSocket hub
  → Browser WebSocket client
  → useWebSocket composable → panel reactive state
```

---

## 5. Risks & Mitigations

| Risk | Prob | Impact | Mitigation |
|------|------|--------|------------|
| Broker interface methods not fully implemented | Medium | High | Test with Paper first; Binance testnet second; real brokers last |
| Python test fix reveals many broken tests | Low | Medium | Fix conftest first, then activate tests one file at a time |
| Adding broker routing breaks paper trading | Low | High | Paper is default; broker routing is opt-in via parameter |
| Frontend wiring triggers regression (revert pattern) | Medium | Low | Each task = independent commit; easy to revert singly |

---

## 6. Acceptance Criteria

- [ ] OrderEntryPanel correctly detects market from symbol (CN/HK/US/CRYPTO)
- [ ] PlaceOrder accepts brokerName parameter and routes to Paper by default
- [ ] BrokerStatusPanel shows real broker connection states (not hardcoded)
- [ ] RunBacktest dead-end resolved (either wired or removed)
- [ ] Python tests: `pytest tests/` collects and runs ≥ 80% of test files
- [ ] GDELT tests pass or auto-skip (no CI failures)
- [ ] 9 mock panels → ≤ 2 remaining (MonteCarlo exempt as client-side math)
- [ ] Implementation status doc reflects actual panel/node counts
- [ ] All existing Go tests (981) and frontend tests (198) continue to pass
- [ ] CHANGELOG updated for all changes

---

## 7. Files Summary

### Will modify
- `app_trading.go` — PlaceOrder signature, GetBrokerStatuses, RunBacktest removal
- `internal/trading/oms.go` — broker routing logic
- `frontend/src/terminal/panels/OrderEntryPanel.vue` — market detection
- `frontend/src/terminal/panels/BrokerStatusPanel.vue` — wire to real data
- `frontend/src/terminal/panels/BrokerConfig.vue` — wire to credential store
- `frontend/src/terminal/panels/AIChatPanel.vue` — wire to app.Chat()
- `frontend/src/terminal/panels/EconomicCalendarPanel.vue` — remove mock fallback
- `frontend/src/terminal/panels/IndicatorPanel.vue` — wire compute button
- `frontend/src/terminal/panels/StockScannerPanel.vue` — wire scan button
- `frontend/src/terminal/panels/FactorAnalysisPanel.vue` — sync with Python
- `frontend/src/terminal/panels/BasketOrderPanel.vue` — wire order submission
- `frontend/src/terminal/panels/ChanlunPanel.vue` — wire to app.GetChanlun()
- `internal/market/adapters/gdelt_test.go` — rate-limit handling
- `python/conftest.py` — create (missing)
- `docs/specs/2026-06-18-proposal-implementation-status.md` — update counts
- `CHANGELOG.md` — record all changes

### Will NOT modify
- `internal/workflow/` — Flow 1 is fully functional
- `internal/market/` — adapter chain works correctly
- `frontend/src/stores/` — stores are correctly structured
- `frontend/src/terminal/DockView/` — docking system works
- `python/src/` — sidecar code is structurally sound
