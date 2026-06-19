# Fix All Remaining Data Source Gaps

## Motivation

SKILL.md 27 个端点审计发现：5 个未实现/有 bug 的端点需要补齐，6 个已有 Go Service 需要暴露前端方法。

## Gap Inventory

### Gap 1: mootdx Finance/F10 — 基础数据层核心缺失 🔴

**当前状态**: MootdxAdapter 只提供 Quote + OHLCV，财务快照（37字段季报: EPS/ROE/净利/营收/每股净资产等）和 F10（9类公司资料文本）完全没有暴露。

**数据流**:
```
GetStockResearch(symbol, ["financials"])
    → Python gRPC: Client.Finance(symbol)  → 37 fields
    → Python gRPC: Client.F10(symbol, name) → 9 categories
    → Go adapter 封装
    → FinancialsService 优先走 mootdx Finance（比新浪财报更快/更全）
```

**修改文件**:
- `internal/market/adapters/mootdx.go` — 新增 `FetchFinance()`, `FetchF10()`
- `python/src/data/fetcher.py` — 新增 `mootdx_finance()`, `mootdx_f10()` Python 函数
- `python/src/data/server.py` — 注册新 gRPC handler（如果有的话）
- `internal/research/financials_service.go` — 尝试先走 mootdx Finance，fallback 新浪

### Gap 2: EastMoney 个股信息缺失 🟡

**当前状态**: EastMoneyAdapter 只做行情。SKILL.md §6.3 的 `eastmoney_stock_info()` (push2 API: 行业/总股本/流通股/总市值/流通市值/上市日期) 未实现。

**数据流**:
```
GetStockResearch(symbol, ["overview"])
    → EastMoneyAdapter.FetchStockInfo(symbol)
    → 返回: code, name, industry, total_shares, float_shares, mcap, float_mcap, list_date, price
```

**修改文件**:
- `internal/market/adapters/eastmoney.go` — 新增 `StockInfo` type + `FetchStockInfo()` 方法
- `app.go` — `GetStockResearch("overview")` 使用真实 stock_info 替代 mock

### Gap 3: 6 个 Service 未暴露前端 🔴

**当前状态**: 这些 Go Service 已完整实现并 wired，但 `app.go` 没有对应的导出方法：

| Service | 已有方法 | 缺失的 app.go 导出 |
|---------|---------|-------------------|
| CapitalService | GetMarginTrading, GetBlockTrades, GetHolderChanges, GetDividendHistory | GetCapitalData(symbol) |
| FundFlowService | GetMinuteFlow, GetDailyFlow | GetFundFlow(symbol, type) |
| NorthboundService | GetMinuteFlow, GetHistory | GetNorthboundFlow() |
| AnnouncementService | GetAnnouncements | GetAnnouncements(symbol) |
| EastMoneySignalsAdapter | FetchDragonTigerStock, FetchDailyDragonTiger, FetchLockupExpiry | GetDragonTiger(symbol), GetLockupExpiry(symbol) |
| EastMoneyConceptAdapter | FetchConceptBlocks | GetConceptBlocks(symbol) |

### Gap 4: 百度 K线 JSON parse bug 🟡

**当前状态**: `baidu.go` 的 `FetchOHLCV` 有 JSON 解析 bug（ResultCode 类型不稳定: int vs string），之前推迟修复。

### Gap 5: 同花顺一致预期 HTML parse 🟡

**当前状态**: `ths_consensus.go` 的 HTML 表格解析逻辑有缺陷，部分股票返回空。

## Design

### 整体数据流

```
前端面板 (Vue)
    → app.go 导出方法
    → Research Service / Adapter
    → HTTP 直连 API / Python gRPC (mootdx)
```

### Gap 1 详细设计: mootdx Finance/F10

Python 端新增两个 data fetcher:

```python
# python/src/data/fetcher.py
def mootdx_finance(code: str) -> dict:
    """37字段季报快照"""
    client = Quotes.factory(market='std')
    market_code = 1 if code.startswith(("6","9")) else 0
    return client.finance(symbol=code, market=market_code)

def mootdx_f10(code: str, name: str) -> str:
    """9类F10文本: 最新提示/公司概况/财务分析/股东研究/股本结构/资本运作/业内点评/行业分析/公司大事"""
    client = Quotes.factory(market='std')
    return client.F10(symbol=code, name=name)
```

Go 端在 MootdxAdapter 新增:

```go
type MootdxFinance struct {
    // 37 fields from mootdx finance snapshot
    EPS      float64 `json:"eps"`
    BVPS     float64 `json:"bvps"`
    ROE      float64 `json:"roe"`
    Profit   float64 `json:"profit"`
    Income   float64 `json:"income"`
    // ... more fields
}

func (a *MootdxAdapter) FetchFinance(ctx context.Context, symbol string) (*MootdxFinance, error)
func (a *MootdxAdapter) FetchF10(ctx context.Context, symbol, category string) (string, error)
```

### Gap 3 详细设计: Frontend 暴露

```go
// app.go 新增方法

// Capital data
func (a *App) GetCapitalData(symbol string) (*CapitalDataResult, error)

// Fund flow
func (a *App) GetFundFlow(symbol string, flowType string) (*FundFlowResult, error)

// Northbound flow
func (a *App) GetNorthboundFlow() (*NorthboundResult, error)

// Announcements
func (a *App) GetAnnouncements(symbol string, pageSize int) ([]adapters.Announcement, error)

// Dragon tiger
func (a *App) GetDragonTiger(symbol string, endDate string, lookBack int) (*DragonTigerResult, error)

// Concept blocks
func (a *App) GetConceptBlocks(symbol string) (*ConceptBlocksResult, error)

// Lockup expiry
func (a *App) GetLockupExpiry(symbol string) (*LockupResult, error)

// Industry ranks
func (a *App) GetIndustryRanks(topN int) ([]adapters.IndustryRank, error)
```

## Acceptance Criteria

### Gap 1:
- [ ] `MootdxAdapter.FetchFinance("600519")` 返回 EPS/ROE/净利等 37 字段
- [ ] `MootdxAdapter.FetchF10("600519", "最新提示")` 返回文本
- [ ] `FinancialsService` 优先走 mootdx Finance，不可用时 fallback 新浪
- [ ] Python sidecar 新增 `mootdx_finance` / `mootdx_f10` 函数

### Gap 2:
- [ ] `EastMoneyAdapter.FetchStockInfo("600519")` 返回行业/总股本/流通股/市值/上市日期
- [ ] `GetStockResearch("600519", ["overview"])` 使用真实数据

### Gap 3:
- [ ] 8 个新 app.go 导出方法全部可调用
- [ ] 每个方法有优雅降级（adapter nil 时返回清晰错误）

### Gap 4:
- [ ] `BaiduAdapter.FetchOHLCV` 正确处理 ResultCode 为 int 或 string 的情况

### Gap 5:
- [ ] `THSConsensusAdapter.FetchConsensus` 正确解析 HTML 表格，返回 EPS 数据

## Risks / Trade-offs

- **mootdx TCP 需国内 IP** — 海外环境走新浪 fallback
- **东财风控** — stock_info 走 push2 API，已有限流保护
- **Python sidecar 依赖** — mootdx Finance/F10 必需 Python sidecar 运行中
