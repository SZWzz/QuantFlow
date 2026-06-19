# Symbol Normalizer + Symbol Search with Frontend Autocomplete

## Motivation

### Problem 1: No unified symbol format

The project has 15 data adapters, each converting symbols differently:

```
同一只贵州茅台:
  eastmoney → "1.600519"      tencent → "sh600519"
  sina      → "sh600519"      baidu   → "600519"
  mootdx    → "600519"        yahoo   → "600519.SS"
```

Conversion logic is duplicated across 6+ files (`stripSuffix` appears in `tencent.go` and `akshare.go`, `toTencentCode` in `tencent.go` and `akshare.go`). Adding a new adapter means copy-pasting the same prefix/suffix logic again. There is no single place to answer "what market does this code belong to?"

### Problem 2: Users can only type raw codes

Frontend panels require exact 6-digit codes. Users can't type "茅台", "贵州茅台", "Moutai", or "600519.SH" — only `600519` works. For a terminal-style product, instant symbol lookup by name, pinyin, or partial code is a baseline expectation (Bloomberg, Wind, Tongdaxin all have it).

## Design

### Layer 1: SymbolNormalizer (Go, `internal/market/symbol.go`)

A single source of truth for CN A-share symbol identity.

```go
type SymbolIdentity struct {
    Raw     string // original input, e.g. "600519.SH"
    Code    string // 6-digit code,    e.g. "600519"
    Market  string // "SH" | "SZ" | "BJ"
}

// NormalizeCN accepts any common format and returns canonical identity.
func NormalizeCN(input string) (*SymbolIdentity, error)

// Converter methods on SymbolIdentity:
func (s *SymbolIdentity) ToEastMoney() string  // "1.600519"
func (s *SymbolIdentity) ToTencent() string    // "sh600519"  
func (s *SymbolIdentity) ToSina() string       // "sh600519"
func (s *SymbolIdentity) ToBaidu() string      // "600519"
func (s *SymbolIdentity) ToMootdx() string     // "600519"
func (s *SymbolIdentity) ToYahoo() string      // "600519.SS"
func (s *SymbolIdentity) ToCninfo() string     // "600519" (plain)
func (s *SymbolIdentity) MarketCode() string   // "1" for SH, "0" for SZ/BJ
```

**Input formats accepted**:
| Input | Result |
|-------|--------|
| `"600519"` | `{Code:"600519", Market:"SH"}` |
| `"600519.SH"` | `{Code:"600519", Market:"SH"}` |
| `"600519.SS"` | `{Code:"600519", Market:"SH"}` |
| `"sh600519"` | `{Code:"600519", Market:"SH"}` |
| `"000001.SZ"` | `{Code:"000001", Market:"SZ"}` |
| `"sz000001"` | `{Code:"000001", Market:"SZ"}` |
| `"830799.BJ"` | `{Code:"830799", Market:"BJ"}` |

**Market detection rules**:
- `6xxxxx` → SH (Shanghai)
- `0xxxxx` / `3xxxxx` → SZ (Shenzhen)  
- `8xxxxx` / `4xxxxx` → BJ (Beijing)
- Suffix `.SH` / `.SS` → SH, `.SZ` → SZ, `.BJ` → BJ
- Prefix `sh` → SH, `sz` → SZ, `bj` → BJ

### Layer 2: SymbolSearchService (Go, `internal/market/symbol_search.go`)

Indexes the full A-share stock list for code + name + pinyin search.

```go
type StockEntry struct {
    Code   string `json:"code"`   // "600519"
    Name   string `json:"name"`   // "贵州茅台"
    Market string `json:"market"` // "SH"
    Pinyin string `json:"pinyin"` // "gzmt"
}

type SymbolSearchService struct {
    entries []StockEntry  // ~5500 stocks, in-memory
}

func NewSymbolSearchService() (*SymbolSearchService, error)
// Fetches stock list from EastMoney push2 API on init.

func (s *SymbolSearchService) Search(query string, limit int) []StockEntry
// Matches by: code prefix, name contains, pinyin prefix.
// "茅台" → 贵州茅台, "gzmt" → 贵州茅台, "6005" → 6005xx stocks
// Results sorted by relevance: exact code > code prefix > name match > pinyin match.
```

**Data source**: EastMoney `push2.eastmoney.com/api/qt/clist/get?fs=m:0+t:6,m:0+t:80,m:1+t:2,m:1+t:23&fields=f12,f14` — returns 5534 stocks.

**Pinyin generation**: Use a minimal mapping table for ~400 common Chinese surname/financial characters, sufficient for stock name abbreviation lookup. (No heavy NLP dependency.)

### Layer 3: Wails Exported Method (app.go)

```go
// SearchSymbols searches A-share stocks by code, name, or pinyin.
func (a *App) SearchSymbols(query string) ([]market.StockEntry, error)
```

### Layer 4: Frontend SymbolSearch Component (Vue)

A composable + component:
- `useSymbolSearch()` composable — debounced search against `App.SearchSymbols`
- `<SymbolSearch>` component — input with dropdown, used by all symbol-input panels

```vue
<SymbolSearch v-model="symbol" placeholder="输入代码/名称/拼音..." />
```

Dropdown shows: `600519  贵州茅台  SH` with matching characters highlighted.

## Data Flow

```
User types "茅台"
  → Frontend SymbolSearch (debounce 200ms)
    → App.SearchSymbols("茅台")
      → SymbolSearchService.Search("茅台", 10)
        → in-memory index match → [{Code:"600519", Name:"贵州茅台", Market:"SH"}]
      ← JSON response
    ← update dropdown
  ← User clicks "贵州茅台"
  → symbol = "600519" (normalized)
  → panel fetches data with canonical code
```

## Files

### New
- `internal/market/symbol.go` — SymbolIdentity + NormalizeCN + converters
- `internal/market/symbol_test.go` — table-driven tests
- `internal/market/symbol_search.go` — SymbolSearchService
- `internal/market/symbol_search_test.go` — search tests
- `internal/market/stock_list.go` — EastMoney stock list fetcher
- `frontend/src/lib/symbolSearch.ts` — useSymbolSearch composable
- `frontend/src/components/SymbolSearch.vue` — search component

### Modified
- `internal/market/adapters/parsers.go` — remove toSinaCode, toTencentCode, stripSuffix; redirect to symbol.go
- `internal/market/adapters/eastmoney.go` — Use SymbolIdentity.ToEastMoney()
- `internal/market/adapters/eastmoney_fundflow.go` — Use SymbolIdentity
- `internal/market/adapters/eastmoney_concept.go` — Use SymbolIdentity
- `internal/market/adapters/tencent.go` — Use SymbolIdentity.ToTencent()
- `internal/market/adapters/akshare.go` — Use SymbolIdentity.ToTencent()
- `internal/market/adapters/sina.go` — Use SymbolIdentity.ToSina()
- `internal/market/adapters/sina_financials.go` — Use SymbolIdentity.ToSina()
- `internal/market/adapters/baidu.go` — Use SymbolIdentity.ToBaidu()
- `internal/market/adapters/mootdx.go` — Normalize symbol before passing to Python
- `internal/market/adapters/eastmoney_capital.go` — Use SymbolIdentity
- `internal/market/adapters/eastmoney_signals.go` — Use SymbolIdentity
- `internal/market/adapters/eastmoney_report.go` — Use SymbolIdentity
- `internal/market/adapters/cninfo.go` — Use SymbolIdentity
- `internal/market/adapters/ths_consensus.go` — Use SymbolIdentity
- `internal/market/adapters/eastmoney_news.go` — Use SymbolIdentity
- `app.go` — Add SearchSymbols method, init SymbolSearchService
- `CHANGELOG.md`

## Acceptance Criteria

- [ ] `NormalizeCN("600519")` → `{Code:"600519", Market:"SH"}`
- [ ] `NormalizeCN("000001.SZ")` → `{Code:"000001", Market:"SZ"}`
- [ ] `NormalizeCN("sh600519")` → `{Code:"600519", Market:"SH"}`
- [ ] `NormalizeCN("600519.SS")` → `{Code:"600519", Market:"SH"}`
- [ ] `NormalizeCN("830799.BJ")` → `{Code:"830799", Market:"BJ"}`
- [ ] All adapter `toXxxCode()` calls replaced with `SymbolIdentity` methods
- [ ] `SymbolSearchService.Search("茅台")` returns `[{Code:"600519", Name:"贵州茅台"}]`
- [ ] `SymbolSearchService.Search("gzmt")` returns `[{Code:"600519", Name:"贵州茅台"}]`
- [ ] `SymbolSearchService.Search("6005")` returns 6005xx stocks
- [ ] Frontend `<SymbolSearch>` renders dropdown with code + name + market
- [ ] Typing "茅台" in a panel's symbol input shows autocomplete dropdown
- [ ] `go test ./internal/market/...` passes
- [ ] `go test ./internal/market/adapters/...` passes (107 tests, no regressions)

## Risks / Trade-offs

- **Pinyin mapping**: A minimal table covers ~400 chars; rare chars will fall through (no match, not wrong match). Acceptable for v1 — can add chars on demand.
- **Stock list freshness**: EastMoney API is fetched once at startup. Newly listed stocks won't appear until restart. Acceptable for desktop app — can add periodic refresh later.
- **Memory**: ~5500 entries × ~200 bytes = ~1MB. Negligible.
- **Adapter refactor scope**: Touches 15+ files. Risk of regression mitigated by existing 107 adapter tests.
