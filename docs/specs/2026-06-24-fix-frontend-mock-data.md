# Fix Frontend Mock Data — Wire All Panels to Backend APIs

> 父 spec: 无（独立专项）
> 前序: Phase A (P0 金融正确性) 已完成
> 扫描日期: 2026-06-24

## Motivation

前端扫描发现 13 个面板使用硬编码 mock 数据、9 个 store 在 API 不可用时兜底 mock。Go 后端行情适配器链（7 级 fallback）完全可用，但前端大量面板从未调用后端——用户看到的行情/盘口/K 线/组合/自选股全是假数据。

核心问题不是后端缺 API，是**前端没接线**。

## Design

### 前端→后端映射矩阵

Go 后端已导出 40+ 方法，以下对照 13 个假面板的缺口：

#### 已有后端 API，仅需前端接线（5 面板）

| 面板 | 后端 API | 改动范围 |
|------|---------|---------|
| WatchlistPanel | `GetQuote("CN", symbol)` | 替换 mockQuotes → 调 API |
| CandlestickPanel | `FetchOHLCV("CN", symbol, "1d")` | 替换随机游走 → 调 API |
| QuoteDetailPanel | `GetQuote("CN", symbol)` | 替换 mock snapshot |
| PortfolioSummary | `GetPortfolioSummary()` | 接入已有的 portfolio store/API |
| TradeHistory | `GetTrades()` + `GetOrders()` | 替换硬编码数据 |

#### 需新增 Go API 的方法（8 面板 + 2 聚合）

| 面板 | 所需数据 | Go 实现策略 |
|------|---------|-----------|
| TickerTapePanel | 多标的实时行情滚动条 | 新增 `GetMarketSnapshot`，复用 `GetQuote` 批量调用 |
| MarketDepthPanel | 5 档盘口 + 逐笔成交 | A 股无免费盘口 API；用 OHLCV 模拟买卖价（卖=High，买=Low），标记为"模拟盘口" |
| ActionCenterPanel | 风控事件列表 | 新增 `GetRiskEvents`，从 `portfolio.RiskManager` 或 OMS 事件日志获取 |
| CryptoOverviewPanel | 加密货币行情 | 已有 crypto 适配器(binance/gate)，新增 `GetCryptoOverview` |
| RebalancePanel | 再平衡建议 | 新增 `GetRebalanceSuggestions`，从 portfolio.Service 计算 |
| SurfaceChartPanel | 波动率曲面 | 新增 `GetVolatilitySurface`，在 Go 端用 SVI 模型拟合（从 OHLCV 计算期权隐含波动率→简化：用历史波动率构造曲面） |
| CorrelationPanel | 相关矩阵 | 新增 `GetCorrelationMatrix`，Go 端计算 Pearson 相关 |
| DistributionPanel | 收益率分布 | 新增 `GetReturnDistribution`，Go 端从 OHLCV 计算日收益直方图 |
| MarketOverview(聚合) | 指数快照 + 市场宽度 | 新增 `GetMarketOverview`，聚合主要指数 quote + 涨跌家数 |
| OMS 事件日志 | 订单/成交/持仓变更 | OMS 已有 `notifyTrade`/`notifyOrder` 回调链，只需暴露历史事件 |

### store 兜底修复（4 个 store）

| Store | 当前行为 | 修复 |
|-------|---------|------|
| data.ts | 指数 mock、板块调 API 成功但 indexData 永远 mock | 新增 `GetMarketOverview` 调用，移除 `generateMockIndices` |
| portfolio.ts | `generateMockOrders/Trades/EquityCurve` 兜底 | 调 `GetOrders/GetTrades/GetPortfolioSummary` 后不再兜底 |
| research.ts | 情绪/研究/国会交易 API 失败时 mock | 已有 API 且正常时不应兜底；仅在 API 返回 error 时提示而不是静默替换 |
| symbolSearch.ts | 搜索建议硬编码 | 已有 `SearchSymbols` API，移除 mock fallback |

### 文件范围

**Go 后端新增**（约 8 个新方法）：
- `app.go`：新增 `GetMarketOverview`, `GetMarketSnapshot`, `GetRiskEvents`, `GetCryptoOverview`, `GetRebalanceSuggestions`, `GetVolatilitySurface`, `GetCorrelationMatrix`, `GetReturnDistribution`

**Go 后端辅助计算**（约 4 个新文件）：
- `internal/market/market_snapshot.go`：多 symbol 批量行情聚合
- `internal/portfolio/analytics.go`：相关矩阵、收益率分布、波动率曲面计算
- `internal/portfolio/rebalance.go`：再平衡建议引擎

**前端面板修复**（13 个面板）：
- WatchlistPanel, CandlestickPanel, QuoteDetailPanel, PortfolioSummary, TradeHistory（已有 API，直接接线）
- TickerTapePanel, MarketDepthPanel, ActionCenterPanel, CryptoOverviewPanel, RebalancePanel, SurfaceChartPanel, CorrelationPanel, DistributionPanel（需等 Go API 新增后接线）

**前端 store 修复**（4 个）：
- data.ts, portfolio.ts, research.ts, symbolSearch.ts

### 实施优先级

按"用户能立即看到变化"排序：

1. **P0：核心行情线**（WatchlistPanel + QuoteDetailPanel + CandlestickPanel + TickerTapePanel）——这 4 个面板直接决定用户打开后第一眼看到的是真数据还是假数据
2. **P1：组合/交易线**（PortfolioSummary + TradeHistory + ActionCenterPanel + RebalancePanel）
3. **P2：市场分析线**（MarketOverview/data.ts + MarketDepthPanel + CorrelationPanel + DistributionPanel + SurfaceChartPanel + CryptoOverviewPanel）
4. **P3：store 兜底修补**（data.ts, portfolio.ts, research.ts, symbolSearch.ts）

## 验收标准

- [ ] 自选股输入 `600519`，WatchlistPanel 显示真实行情（价格与涨跌幅非 mock）
- [ ] 点击自选股，CandlestickPanel 加载真实历史 K 线（非随机游走）
- [ ] QuoteDetailPanel 显示真实报价（exchange 非 "MOCK"）
- [ ] TickerTapePanel 滚动条显示真实 A 股行情
- [ ] PortfolioSummary 显示真实组合数据（非定时器模拟）
- [ ] TradeHistory 显示真实成交记录
- [ ] MarketOverview 显示真实指数快照（非 generateMockIndices）
- [ ] 前端无任何 `generateMock` 函数在生产路径中执行
- [ ] 所有修复后 `frontend: vue-tsc --noEmit` 通过、`vitest` 通过
- [ ] CHANGELOG 记录本次修复范围

## 风险 / 权衡

| 风险 | 缓解 |
|------|------|
| MarketDepthPanel：A 股无免费 Level 2 盘口 API | 用 OHLCV 的 High/Low 模拟买卖报价，UI 标注"模拟盘口" |
| SurfaceChartPanel：真实波动率曲面需要期权链数据 | v1 用历史波动率构造常曲面（不同期限用不同窗口），标注"历史波动率曲面" |
| CryptoOverviewPanel：需要 binance/gate API key | binance 公网接口无需 key，优先使用 |
| 批量行情调用可能触发 API 限流 | 单个 symbol 查询走 fallback 链已有 jitter；批量查询加节流(200ms间隔) |
| 部分 Go 方法签名变更影响已生成的前端 binding | 新增方法不影响已有 binding；需重新 `wails generate module` |
