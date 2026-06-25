# Fix Phase C: Remove All Remaining Panel Mock Data

> 扫描日期: 2026-06-25
> 前一阶段: Phase A (P0 金融正确性) + Phase B (13 面板 mock→API)
> 发现: 19 个面板仍有 mock 数据，全部未调用 Go API

## Motivation

昨天修复了 13 个核心面板的 mock 数据，但 BrokerStatusPanel 暴露了遗漏。全量扫描发现还有 **19 个面板**完全使用硬编码假数据，其中 BrokerStatusPanel 显示 Futu "connected" 但代码全是 stub。

## 分类与优先级

### P0：假状态误导用户（3 面板）

| 面板 | mock 内容 | Go API 状态 | 修复方案 |
|------|---------|------------|---------|
| BrokerStatusPanel | Futu"connected"、余额52.8万全假 | futu.go 全 stub | 移除所有 broker 假数据，显示真实状态（Paper=可用，其他=未实现/未配置） |
| SystemMonitorPanel | CPU/内存随机数 | 需新增 runtime stats | 用 Go 的 runtime.MemStats 提供真实指标 |
| BacktestResultPanel | 硬编码回测结果 | workflow store 已有 | 从 workflow execution result 取真实数据 |

### P1：已有 API 但未接线（6 面板）

| 面板 | 需要的 API | 已有？ | 修复 |
|------|-----------|:--:|------|
| MarketOverviewPanel | GetMarketOverview | ✅ | 接线 |
| HeatmapPanel | GetIndustryRanks | ✅ | 接线 |
| OrderBlotterPanel | GetOrders | ✅ | 接线 |
| OrderEntryPanel | PlaceOrder ✅，但价格估算 mock | 半 | 用 GetQuote 取实时价 |
| ExecutionPanel | GetTrades | ✅ | 接线 |
| RLMonitorPanel | 训练状态 | ❌ | 标注"需要 Python bridge" |

### P2：研究/数据面板，API 存在但走 mock（10 面板）

| 面板 | Go API | 修复 |
|------|--------|------|
| AnalystEstimatesPanel | GetStockResearch("estimates") | 接线 researchStore |
| CongressTradingPanel | GetCongressTrades | 接线 researchStore |
| FinancialsPanel | GetStockResearch("financials") | 接线 researchStore |
| InsiderTradingPanel | GetStockResearch("insider") | 接线 researchStore |
| PeerComparisonPanel | GetStockResearch("peers") | 接线 researchStore |
| SentimentPanel | GetSentiment | 接线 researchStore |
| GeopoliticsPanel | GetGeopoliticsRisks | 已有 Go API，接线 |
| GovDataPanel | GetEconomicIndicators | 已有 Go API，接线 |
| PredictionMarketPanel | GetPredictionMarkets | 已有 Go API，接线 |
| SatellitePanel | GetSatelliteSnapshots | 已有 Go API，接线 |

## 修复策略

由于 P2 的 10 个面板对应 Go API 都已存在但前端没接线（researchStore 有 mock fallback），统一方案：

1. **P0**：BrokerStatusPanel/SystemMonitor/BacktestResult — 直接修掉假数据
2. **P1**：MarketOverview/Heatmap/OrderBlotter/OrderEntry/Execution/RLMonitor — 接线
3. **P2**：10 个研究面板 — 移除前端 mock，仅保留 API 失败时的"暂无数据"提示

## 实施顺序

- Task C1: P0 三个面板（BrokerStatus + SystemMonitor + BacktestResult）
- Task C2: P1 六个面板（接线已有 API）
- Task C3: P2 研究面板（移除 mock，接线已有 API）

## 验收标准

- [ ] BrokerStatus 不再显示虚假 Futu connected
- [ ] 19 个面板全部移除硬编码 mock 数据
- [ ] vue-tsc 零新增错误
- [ ] go build 通过
