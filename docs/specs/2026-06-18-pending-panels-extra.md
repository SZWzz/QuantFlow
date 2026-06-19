# [待开发] 缺失前端面板

> **Status**: PENDING — 后续开发
> **Proposal ref**: NEW_PROJECT_PROPOSAL.md §7.3 (面板目录)
> **Priority**: 🟡 中

## Motivation

规划 50+ 面板，目前仅实现 22 个。缺失的 30+ 面板覆盖了交易执行、市场概览、研究分析、工具等多个领域，直接影响用户的工作效率。

## 缺失面板清单

### 市场类 (6)

| 面板 | 说明 | 优先级 |
|------|------|--------|
| MarketOverviewPanel | 市场总览：大盘指数/涨跌家数/板块轮动 | 🟡 |
| MarketDepthPanel | 深度行情：买卖盘口/逐笔成交 | 🟡 |
| TickerTapePanel | 滚动报价条 | 🟢 |
| HeatmapPanel | 市场热力图（市值权重+涨跌幅颜色） | 🟡 |
| CryptoOverviewPanel | 加密市场总览 | 🟢 |
| SparklinePanel | 迷你走势图集合 | 🟢 |

### 图表类 (5)

| 面板 | 说明 | 优先级 |
|------|------|--------|
| EquityCurvePanel | 净值曲线（可复用 ECharts） | 🔴 可与 BacktestResult 合并 |
| SurfaceChartPanel | 3D 曲面图（波动率曲面） | 🟡 |
| CorrelationPanel | 相关性矩阵 | 🟡 |
| DistributionPanel | 收益分布/回撤分布 | 🟡 |
| DrawingPanel | 画线工具面板 | 🟢 |

### 交易类 (6)

| 面板 | 说明 | 优先级 |
|------|------|--------|
| OrderBlotterPanel | 订单流水 | 🔴 |
| ExecutionPanel | 成交明细 | 🟡 |
| BasketOrderPanel | 篮子交易面板 | 🟡 |
| BrokerStatusPanel | 券商连接状态 | 🟡 |
| ActionCenterPanel | 交易审批中心 | 🟡 |
| WebhookMonitorPanel | Webhook 监听/日志 | 🟢 |

### 研究类 (7) — 另见 `pending-research-sentiment.md`

| 面板 | 说明 |
|------|------|
| StockResearchPanel | 7 标签页研究面板 |
| FinancialsPanel | 财务数据 |
| AnalystEstimatesPanel | 分析师预期 |
| PeerComparisonPanel | 同行对比 |
| SentimentPanel | 情绪分析 |
| InsiderTradingPanel | 内部交易 |
| CongressTradingPanel | 国会交易 |

### 另类数据 (5) — 另见 `pending-alternative-data.md`

| 面板 |
|------|
| PredictionMarketPanel |
| MaritimePanel |
| GeopoliticsPanel |
| SatellitePanel |
| GovDataPanel |

### 组合类 (2)

| 面板 | 说明 | 优先级 |
|------|------|--------|
| MonteCarloPanel | 蒙特卡洛模拟 | 🟡 |
| RebalancePanel | 组合再平衡面板 | 🟡 |

### 工具类 (5)

| 面板 | 说明 | 优先级 |
|------|------|--------|
| SpreadsheetPanel | 嵌入式电子表格 | 🟡 |
| CodeEditorPanel | Monaco 代码编辑器 | 🟡 |
| ReportPanel | 报表生成/导出 | 🟢 |
| NotesPanel | 工作笔记 | 🟢 |
| FileManagerPanel | 文件管理器 | 🟢 |

### 监控类 (1)

| 面板 | 说明 | 优先级 |
|------|------|--------|
| AlertPanel | 告警规则管理 | 🟢 |

## 实现要求

每个新面板需遵循现有模式:
- 接收 `panelId: string` + `params?: Record<string, any>` props
- 懒加载注册 (`defineAsyncComponent`) 到 `registry.ts`
- CommandBar 中可搜索打开
- 添加到 `registry.ts` 的 panel map
- 具有对应的 vitest 测试文件
- 支持主题变量 (CSS var)

## Acceptance Criteria

- [ ] 每个新面板在 `registry.ts` 中注册
- [ ] 面板可被 CommandBar 搜索打开
- [ ] 面板显示 mock 数据（后端未提供真实数据时）
- [ ] 现有 76 前端测试通过

## 工作量估算

- 每面板平均 ~0.5 天（含测试）
- 28 个面板 ≈ **14 天**

## Risks / Trade-offs

- 优先实现交易/市场类面板，工具类后置
- EquityCurvePanel 可合并入 BacktestResultPanel 不做独立面板
