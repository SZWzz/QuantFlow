# Plan: Symbol Normalizer + Symbol Search

## Task 1: SymbolIdentity + NormalizeCN (core, no deps)

**File**: `internal/market/symbol.go`

Create the canonical symbol normalizer:

```go
package market

import (
    "fmt"
    "strings"
)

// SymbolIdentity holds the canonical identifier for a CN A-share stock.
type SymbolIdentity struct {
    Raw    string // original input
    Code   string // 6-digit code, e.g. "600519"
    Market string // "SH", "SZ", or "BJ"
}

// NormalizeCN accepts any common CN stock identifier format and returns
// the canonical 6-digit code + market. Supported inputs:
//   - "600519", "600519.SH", "600519.SS", "sh600519", "SH600519"
//   - "000001", "000001.SZ", "sz000001"
//   - "830799", "830799.BJ", "bj830799"
func NormalizeCN(input string) (*SymbolIdentity, error) { ... }

// Converter methods — each returns the adapter-specific format.
func (s *SymbolIdentity) ToEastMoney() string { ... }  // "1.600519"
func (s *SymbolIdentity) ToTencent() string   { ... }  // "sh600519"
func (s *SymbolIdentity) ToSina() string      { ... }  // "sh600519"
func (s *SymbolIdentity) ToBaidu() string     { ... }  // "600519"
func (s *SymbolIdentity) ToMootdx() string    { ... }  // "600519"
func (s *SymbolIdentity) ToYahoo() string     { ... }  // "600519.SS"
func (s *SymbolIdentity) ToPlain() string     { ... }  // "600519"
func (s *SymbolIdentity) MarketCode() string  { ... }  // "1" (SH) / "0" (SZ/BJ)
```

**Test file**: `internal/market/symbol_test.go` — 30+ cases covering all formats above.

**Commit**: `market: add SymbolIdentity + NormalizeCN unified CN symbol handler`

---

## Task 2: Remove duplicated helpers, redirect to SymbolIdentity

**Files**: `internal/market/adapters/parsers.go`

- Remove `toSinaCode()`, `toTencentCode()`, `stripSuffix()` from parsers.go
- Add thin wrappers that call `market.NormalizeCN().ToXxx()` for backward compat during transition

Actually, the cleanest approach: keep old functions but mark as deprecated wrappers redirecting to SymbolIdentity. This avoids breaking the akshare.go copy of toTencentCode.

Better approach: replace the function bodies with delegations to SymbolIdentity.

**Commit**: `adapters: redirect deprecated toXxxCode helpers to SymbolIdentity`

---

## Task 3: Refactor all adapters to use SymbolIdentity directly

**Files** (mechanical change, one file at a time):
- `eastmoney.go` — `toEastMoneySecID(symbol)` → `id, _ := market.NormalizeCN(symbol); id.ToEastMoney()`
- `eastmoney_fundflow.go` — same pattern
- `eastmoney_concept.go` — inline marketCode logic → `id.MarketCode()`
- `eastmoney_capital.go` — use `id.ToPlain()`
- `eastmoney_signals.go` — use `id.ToPlain()` for filter building
- `eastmoney_report.go` — use `id.ToPlain()`
- `eastmoney_news.go` — use `id.ToPlain()`
- `tencent.go` — `toTencentCode(symbol)` → `id.ToTencent()`
- `akshare.go` — same
- `sina.go` — `toSinaCode(symbol)` → `id.ToSina()`
- `sina_financials.go` — inline prefix logic → `id.ToSina()`  
- `baidu.go` — `toBaiduCode(symbol)` → `id.ToBaidu()`
- `mootdx.go` — normalize symbol before passing to Python (the Python side also normalizes, but Go side doing it first avoids round-trips on bad input)
- `cninfo.go` — use `id.ToPlain()`
- `ths_consensus.go` — use `id.ToPlain()`

For adapters that receive `symbol string` from the `market.Adapter` interface: keep the public signature, normalize internally. No API change.

**Verify**: `go test ./internal/market/adapters/...` — all 107 tests must pass.

**Commit**: `adapters: refactor all code converters to use SymbolIdentity`

---

## Task 4: Stock list fetcher

**File**: `internal/market/stock_list.go`

```go
package market

type StockEntry struct {
    Code   string `json:"code"`
    Name   string `json:"name"`  
    Market string `json:"market"`
}

// FetchCNStockList fetches the full A-share stock list from EastMoney.
// Returns ~5500 entries. Call once at startup.
func FetchCNStockList(ctx context.Context) ([]StockEntry, error)
```

Uses EastMoney push2 API: `fs=m:0+t:6,m:0+t:80,m:1+t:2,m:1+t:23&fields=f12,f14`.

**Commit**: `market: add FetchCNStockList from EastMoney push2`

---

## Task 5: Pinyin index + SymbolSearchService

**File**: `internal/market/symbol_search.go`

Pinyin mapping: embed a minimal map of ~400 common Chinese chars → pinyin. Generate pinyin abbreviation for each stock name (first letter of each char's pinyin).

```go
type SymbolSearchService struct {
    entries []StockEntry
}

func NewSymbolSearchService(ctx context.Context) (*SymbolSearchService, error)
// Fetches stock list, generates pinyin, builds in-memory index.

func (s *SymbolSearchService) Search(query string, limit int) []StockEntry
// Matches: exact code prefix > name contains > pinyin prefix.
// Sorted by relevance score.
```

Search algorithm:
1. `query` is numeric → code prefix match only
2. `query` is ASCII → pinyin prefix match (e.g., "gzmt" → 贵州茅台)
3. `query` is Chinese → name contains match (e.g., "茅台")

**Test file**: `internal/market/symbol_search_test.go`

**Commit**: `market: add SymbolSearchService with pinyin fuzzy search`

---

## Task 6: Wire into app.go

**Changes**:
- Add `searchSvc *market.SymbolSearchService` to App struct
- Init in startup(): `a.searchSvc, err = market.NewSymbolSearchService(context.Background())`
- Add exported method:
```go
func (a *App) SearchSymbols(query string) ([]market.StockEntry, error) {
    if a.searchSvc == nil {
        return nil, fmt.Errorf("symbol search not initialized")
    }
    return a.searchSvc.Search(query, 20), nil
}
```

**Commit**: `app: wire SymbolSearchService + SearchSymbols Wails method`

---

## Task 7: Frontend composable + component

**Files**:
- `frontend/src/lib/symbolSearch.ts` — useSymbolSearch composable with debounced API call
- `frontend/src/components/SymbolSearch.vue` — input + dropdown + keyboard nav

**Component behavior**:
- Debounced input (200ms)
- Dropdown shows: `[SH] 600519  贵州茅台`
- Arrow keys navigate, Enter selects, Escape closes
- Selected value = 6-digit code
- Loading state while searching
- Empty state: "未找到匹配股票"

**Commit**: `frontend: add SymbolSearch autocomplete component`

---

## Execution Order & Estimated Commits

```
1. symbol.go + symbol_test.go          (core, self-contained)
2. Remove dupes in parsers.go          (cleanup)
3. Refactor 15 adapters                (mechanical, bulk)
4. stock_list.go                       (new capability)
5. symbol_search.go + test             (search engine)
6. app.go wiring                       (Wails export)
7. frontend component                  (UI)
8. CHANGELOG update                    (docs)
```

**Risk mitigations**:
- Task 3 is the riskiest (15 files). Run `go test ./internal/market/adapters/... -count=1` after each file change.
- If time is short, Tasks 4-7 (search + frontend) can be deferred to a follow-up spec while still delivering the normalizer (Tasks 1-3) which is the core value.
