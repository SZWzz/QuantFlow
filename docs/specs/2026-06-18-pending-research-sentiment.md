# [待开发] 研究分析 + 情绪分析模块

> **Status**: PENDING — 后续开发
> **Proposal ref**: NEW_PROJECT_PROPOSAL.md §4.1.4 (研究分析模块)
> **Priority**: 🔴 高

## Motivation

规划中 FinceptTerminal 的研究分析功能（EquityResearchScreen 7 标签页）和股票情绪分析（SentimentNode）目前完全未实现。这直接影响了用户做基本面研究、新闻情绪分析的能力。

## 缺失组件清单

### Backend: `internal/research/` — 全新包

| 组件 | 状态 | 说明 |
|------|------|------|
| `internal/research/` 目录 | 📋 未创建 | 新建整个 research 包 |
| StockResearchNode (7 tab) | 📋 | 股票研究节点，覆盖概览/财务/技术/同行/情绪 |
| SentimentNode | 📋 | 情绪分析节点 — 新闻/社交媒体情绪→信号 |
| SentimentEngine | 📋 | 后端情绪计算引擎（NLP pipeline） |
| FinancialsService | 📋 | 财务数据获取与比率计算 |
| PeerComparisonService | 📋 | 同行对比分析 |
| AnalystEstimatesService | 📋 | 分析师评级/目标价聚合 |
| InsiderTradingService | 📋 | 内部交易监控 |
| CongressTradingService | 📋 | 国会交易监控 |
| ResearchRepo (SQLite) | 📋 | 研究数据持久化 |

### Python Sidecar: 新闻 NLP + 情绪

| 组件 | 状态 | 说明 |
|------|------|------|
| `python/src/research/` | 📋 | 新建 Python research 包 |
| NLP pipeline | 📋 | 新闻解析→实体识别→情感打分 |
| 情绪聚合器 | 📋 | 多源情绪聚合（新闻/社交媒体/监管披露） |
| Sentiment gRPC service | 📋 | proto + 服务实现 |
| 财务数据爬取 | 📋 | 财报/分析师数据抓取 |

### Frontend Panels

| 面板 | 状态 | 说明 |
|------|------|------|
| **SentimentPanel** | 📋 | 情绪仪表盘：情感曲线、关键词云、情绪分布 |
| StockResearchPanel | 📋 | 7 标签页研究面板 |
| FinancialsPanel | 📋 | 财务数据面板 |
| PeerComparisonPanel | 📋 | 同行对比面板 |
| AnalystEstimatesPanel | 📋 | 分析师预期面板 |
| InsiderTradingPanel | 📋 | 内部交易面板 |
| CongressTradingPanel | 📋 | 国会交易面板 |

### Workflow Nodes (注册到 NodeRegistry)

| 节点 | 类型 | 说明 |
|------|------|------|
| `stock_research` | research | 股票全维度研究节点 |
| `sentiment` | research | 情绪分析→信号输出 |
| `financials` | research | 财务数据加载 |
| `peer_compare` | research | 同行对比 |
| `analyst_estimates` | research | 分析师数据 |
| `insider_trades` | research | 内部交易 |

## Data Flow

```
[SentimentPanel/Frontend]
    ↓ Wails IPC
[App.GetSentiment(symbol)]
    ↓
[SentimentEngine (Go)]
    ├→ [SQLite: sentiment_cache]
    └→ [Python gRPC: NLP pipeline]
         └→ [外部 API: 新闻/社交媒体]
    ↓
[SentimentNode → StrategyNode/TradingNode]
```

## Acceptance Criteria

- [ ] `internal/research/` 包存在，包含 SentimentNode 等 6+ 节点
- [ ] 前端 7 个研究面板可渲染，显示 mock 数据
- [ ] Python 情绪分析 gRPC 服务可计算正面/负面/中性评分
- [ ] SentimentNode 输出可连接下游策略/交易节点
- [ ] 注册到 NodeRegistry，CommandBar 可搜索
- [ ] 76 前端测试 + 现有 Go 测试全部通过

## 工作量估算

- `internal/research/` 后端: ~3 天
- Python NLP pipeline: ~3 天
- 前端面板 (7个): ~4 天
- 工作流节点 (6个): ~2 天
- 测试 + 联调: ~2 天
- **合计: ~14 天**

## Risks / Trade-offs

- 外部新闻 API 需要 API key/license（可先用 mock 数据开发）
- 中文 NLP 情绪准确度有限（A 股新闻特有术语）
- 依赖 Python sidecar，sidecar 未启动时需优雅降级
