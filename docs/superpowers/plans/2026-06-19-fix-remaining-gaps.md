# Fix All Remaining Gaps — Implementation Plan

## Task 1: mootdx Finance/F10 (Gap 1)

### 1a: Python — Add finance + F10 fetchers
**File**: `python/src/data/fetcher.py` (MODIFY)
- Add `_fetch_mootdx_finance(symbols)` → returns 37-field quarterly snapshot
- Add `_fetch_mootdx_f10(symbols, category)` → returns F10 text
- Add `data_type == "finance"` and `data_type == "f10"` dispatch in `_handle_mootdx()`

### 1b: Go — Add FetchFinance + FetchF10 to MootdxAdapter
**File**: `internal/market/adapters/mootdx.go` (MODIFY)
- Add `MootdxFinance` struct (key 37 fields)
- Add `FetchFinance(ctx, symbol)` method
- Add `FetchF10(ctx, symbol, category)` method
- Categories constant: 最新提示/公司概况/财务分析/股东研究/股本结构/资本运作/业内点评/行业分析/公司大事

### 1c: Go — Integrate into FinancialsService
**File**: `internal/research/financials_service.go` (MODIFY)
- Accept optional `*adapters.MootdxAdapter`
- Try mootdx Finance first, fallback to Sina

## Task 2: EastMoney stock_info (Gap 2)

### 2a: Add FetchStockInfo to EastMoneyAdapter
**File**: `internal/market/adapters/eastmoney.go` (MODIFY)
- Add `EastMoneyStockInfo` struct
- Add `FetchStockInfo(ctx, symbol)` method using push2 API
- Fields: f57(code), f58(name), f127(industry), f84(total_shares), f85(float_shares), f116(mcap), f117(float_mcap), f189(list_date)

### 2b: Use in GetStockResearch overview
**File**: `app.go` (MODIFY)
- In `GetStockResearch("overview")`, call `a.eastmoneyAdpt.FetchStockInfo()` if available

## Task 3: Frontend expose 6 services (Gap 3)

### 3a: Add 8 exported methods to app.go
**File**: `app.go` (MODIFY)
- `GetCapitalData(symbol)` → returns MarginTrading + BlockTrades + HolderChanges + Dividends
- `GetFundFlow(symbol, flowType)` → returns minute or daily flow
- `GetNorthboundFlow()` → returns minute flow + history
- `GetAnnouncements(symbol, pageSize)` → returns announcement list
- `GetDragonTiger(symbol, endDate, lookBack)` → returns dragon tiger records + seats
- `GetDailyDragonTiger(date, minNetBuy)` → returns market-wide dragon tiger
- `GetLockupExpiry(symbol)` → returns history + upcoming
- `GetIndustryRanks(topN)` → returns industry ranking
- `GetConceptBlocks(symbol)` → returns concept/industry blocks

### 3b: Add capital service methods to nodes
**File**: `internal/workflow/nodes/research_deps.go` (MODIFY — only if methods missing)

## Task 4: Fix Baidu JSON bug (Gap 4)

### 4a: Fix ResultCode type handling in FetchOHLCV
**File**: `internal/market/adapters/baidu.go` (MODIFY)
- Add flexible ResultCode parsing: accept both string and number
- Use `json.RawMessage` or `interface{}` for the ResultCode field

## Task 5: Fix THS HTML parse (Gap 5)

### 5a: Fix table finding logic — handle GBK properly
**File**: `internal/market/adapters/ths_consensus.go` (MODIFY)
- Decode body with GBK before HTML parsing (currently reads bytes as UTF-8)
- Improve table finding: search for "一致预期" or "预测" in addition to "每股收益"
- Add debug logging for table count and content

## Execution Order

Task 4 → Task 5 → Task 2 → Task 1 → Task 3
(easy fixes first, complex features later)
