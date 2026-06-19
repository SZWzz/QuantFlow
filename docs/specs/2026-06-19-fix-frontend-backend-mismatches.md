# Fix Frontend/Backend Data Mismatches

## Motivation

Analysis revealed 5 data contract mismatches between Go backend models and Vue frontend panels that prevent data from displaying correctly. Three are severe (panels never show data), two are field-level (specific columns show `--` instead of values).

## Design

### 1. Financials nested structure (SEVERE)

**Problem**: Go `StockResearchResult` serializes `financials` as flat `FinancialData` and `ratios` as separate key. Frontend expects `financials: { data: FinancialData, ratios: Record }` — a single nested object.

**Root**: `models.go:78-79` — `Financials` and `Ratios` are separate struct fields with separate JSON keys.

**Fix**: Introduce `FinancialsBundle` struct that nests `Data` and `Ratios`, replace both fields in `StockResearchResult` with a single `Financials *FinancialsBundle` field with JSON tag `"financials"`.

**Files**: `internal/research/models.go`, `internal/workflow/nodes/stock_research.go`, `internal/workflow/nodes/financials.go`

### 2. Insider trades key name (SEVERE)

**Problem**: Go serializes `insider_trades` but frontend reads `insider`.

**Root**: `models.go:83` — JSON tag is `"insider_trades"`.

**Fix**: Change JSON tag to `"insider"`.

**Files**: `internal/research/models.go`

### 3. CongressTrades not exported (SEVERE)

**Problem**: Frontend calls `app.GetCongressTrades()` but no such method exists on `App`.

**Root**: Service is initialized in `startup()` but never exposed to frontend.

**Fix**: Add `GetCongressTrades()` method to `App` struct.

**Files**: `app.go`

### 4. margin vs net_margin (field name)

**Problem**: Go serializes `net_margin` but `PeerComparisonPanel.vue:68` accesses `p.margin`.

**Root**: `models.go:49` — JSON tag is `"net_margin"`.

**Fix**: Change JSON tag to `"margin"` (frontend is the consumer; it defines the contract).

**Files**: `internal/research/models.go`

### 5. value field missing (field name)

**Problem**: `InsiderTradingPanel.vue:94` displays `t.value` but `InsiderTransaction` has no `Value` field.

**Root**: `models.go:64-71` — `InsiderTransaction` struct missing `Value` computed field.

**Fix**: Add `Value float64` with JSON tag `"value"`, computed as `Shares * Price` in service layer.

**Files**: `internal/research/models.go`, `internal/research/insider_trading_service.go`

## Data Flow

```
Go models.go (struct + JSON tags)
  → app.go GetStockResearch() / GetCongressTrades()
    → Wails IPC JSON serialization
      → Frontend research.ts (TypeScript interface)
        → Panel .vue (renders data)
```

## Acceptance Criteria

- [ ] FinancialsPanel displays income statement, balance sheet, cash flow, and ratios from real/mock data
- [ ] InsiderTradingPanel shows trades with all columns populated (Name, Role, Type, Shares, Price, Value, Date)
- [ ] CongressTradingPanel loads data via `GetCongressTrades` (no longer falls through to mock in catch block)
- [ ] PeerComparisonPanel shows margin values (not `--`)
- [ ] All existing tests continue to pass

## Risks / Trade-offs

- **FinancialsBundle change**: Any code referencing `result.Financials` or `result.Ratios` directly needs updating. The `GetStockResearch` method in `app.go` and the `StockResearchNode` must be updated.
- **JSON tag changes**: External consumers (if any) relying on old JSON keys would break. Since the frontend is the primary consumer and already expects the new keys, this is safe.
- **Backward compatibility**: These are breaking changes to the IPC contract, but since frontend/backend are compiled together, there is no versioning concern.
