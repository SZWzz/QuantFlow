# Proposal Implementation Status Map

> **Purpose**: Map every module/feature in `NEW_PROJECT_PROPOSAL.md` to its implementation status.
> **Legend**: ✅ 完全实现 | 🔶 部分实现 | 📋 待开发 | ❌ 规划中未开始

---

## 1. 项目定位与技术选型

| 决策点 | 状态 | 说明 |
|--------|------|------|
| Go 1.22+ 后端 | ✅ | 已使用 |
| Python gRPC sidecar | ✅ | bridge/proto 全套 |
| Vue 3 + TypeScript 前端 | ✅ | 已使用 |
| Wails v3 桌面壳 | ✅ | 已使用 |
| ECharts 金融图表 | ✅ | vue-echarts 集成 |
| Pinia 状态管理 | ✅ | 7 stores |
| SQLite WAL 存储 | ✅ | storage 包 |
| AGPL-3.0 许可证 | ✅ | 已配置 |

---

## 2. 架构全景

| 组件 | 状态 | 说明 |
|------|------|------|
| Terminal Mode (彭博面板) | ✅ | DockView 系统 + 22 panels |
| Workflow Mode (vue-flow) | ✅ | 54 节点 + Canvas/Property/Log |
| Terminal ↔ Workflow 双向 | 🔶 | 工作流→终端可固定；终端→工作流 [⊕] 未实现 |
| 共享后端/SQLite/数据总线 | ✅ | App struct 统一管理 |
| 分屏递归 | ✅ | DockView + DockContainer + DockSplitter |
| 浮动窗口 | 📋 | Wails 多窗口 API 未接入 |

---

## 3. 工作流引擎 (核心)

### 3.1 引擎基础设施

| 组件 | 状态 | 说明 |
|------|------|------|
| BaseNode 接口 + Schema | ✅ | `internal/workflow/node.go` |
| NodeRegistry | ✅ | 54 节点已注册 |
| DAG 拓扑排序 (Kahn) | ✅ | `internal/workflow/dag.go` |
| goroutine 逐层并行 | ✅ | `internal/workflow/engine.go` |
| SQLite 工作流持久化 | ✅ | `internal/storage/workflow_repo.go` |
| LRU 节点缓存 | ✅ | `internal/workflow/cache.go` |
| 断点调试 | 📋 | `internal/workflow/debugger.go` 未实现 |

### 3.2 节点类型 (规划 200+, 已实现 54)

| 类别 | 规划 | 已实现 | 状态 | 具体节点 |
|------|------|--------|------|---------|
| 数据加载 (Data) | ~25 | 5 | 🔶 | data_loader, merge, filter, resample, http_request |
| Alpha 因子 (Alpha) | ~20 | 1 | 📋 | factor 节点(调用Python)，无GP进化 |
| 技术指标 (Indicator) | ~15 | 5 | 🔶 | sma, ema, macd, rsi, bollinger |
| 信号生成 (Signal) | ~10 | 5 | 🔶 | cross_signal, threshold_signal, signal_combine, entry_signal, exit_signal |
| 策略构建 (Strategy) | ~12 | 1 | 📋 | strategy 节点，无权重优化 |
| 回测执行 (Backtest) | ~10 | 1 | 🔶 | backtest 节点，CN+US 引擎 |
| 归因分析 (Attribution) | ~8 | 0 | 📋 | Brinson/因子分解均未实现 |
| 风控分析 (Risk) | ~8 | 2 | 🔶 | risk_metrics, stop_loss, risk_model |
| 交易执行 (Trading) | ~20 | 6 | 🔶 | place_order, cancel_order, position_query, order_query, position_sizer, portfolio_summary |
| 市场数据 (Market) | ~18 | 1 | 📋 | data_loader 已覆盖基本需求 |
| 投资组合 (Portfolio) | ~12 | 3 | 🔶 | portfolio_summary, risk_metrics, allocation, rebalance |
| 研究分析 (Research) | ~15 | 0 | 📋 | **全未实现** |
| AI 智能体 (Agent) | ~10 | 1 | 🔶 | agent 节点，Skills体系未完善 |
| 通知输出 (Notify) | ~8 | 2 | 🔶 | notify, alert |
| 控制流 (Control) | ~8 | 5 | 🔶 | loop, wait, schedule, if_condition, sub_workflow |
| 工具节点 (Utility) | ~15 | 7 | 🔶 | math_op, arithmetic, json_parse, compare, chart_data, if_else, delta |

| 因子原子节点 | 规划 | 已实现 | 说明 |
| pct_change | ~2 | 1 | ✅ |
| delta | ~2 | 1 | ✅ |
| std_dev | ~2 | 1 | ✅ |
| rank | ~2 | 1 | ✅ |
| scale | ~2 | 1 | ✅ |
| rolling_zscore | ~2 | 1 | ✅ |
| rolling_maxmin | ~2 | 1 | ✅ |
| cross_over | ~2 | 1 | ✅ |
| compare | ~2 | 1 | ✅ |
| bool_combine | ~2 | 1 | ✅ |
| arithmetic | ~2 | 1 | ✅ |
| if_else | ~2 | 1 | ✅ |
| rank_select | ~2 | 1 | ✅ |
| hold_signal | ~2 | 1 | ✅ |

### 3.3 工作流模板系统 (规划 50+, 已实现 0)

| 模板分类 | 状态 |
|---------|------|
| 量化研究 (10) | 📋 |
| 交易执行 (8) | 📋 |
| 投资组合管理 (6) | 📋 |
| AI 驱动 (8) | 📋 |
| 另类数据 (6) | 📋 |
| 监控告警 (6) | 📋 |
| 定时任务 (6) | 📋 |

---

## 4. 功能模块融合映射

### 4.1 市场数据模块

| FT 功能 | 状态 | QuantFlow 实现 |
|---------|------|---------------|
| MarketDataService (批量报价) | ✅ | MarketDataHub + 14 adapters |
| DataHub (发布/订阅) | ✅ | Go channel pub/sub |
| MarketSearchService | 📋 | 未实现 |
| 100+ 数据连接器 | 🔶 | 14/100 适配器 |
| 数据归一化 (7 Schema) | 📋 | 未实现 |
| 凭据管理 (AES-256-GCM) | 📋 | 未实现 |
| PythonWorker 守护进程 | ✅ | gRPC PythonBridge |

### 4.2 交易执行模块

| FT 功能 | 状态 | QuantFlow 实现 |
|---------|------|---------------|
| UnifiedTrading | ✅ | TradingHub (channel) |
| IBroker 接口 | ✅ | Broker interface, Futu + Binance |
| PaperTrading | ✅ | PaperEngine |
| OrderMatcher | ✅ | OrderMatcher |
| SmartOrderEngine | 🔶 | PositionSizer 节点，算法交易未实现 |
| ActionCenter (审批) | 📋 | 未实现 |
| WebhookListener | 📋 | 未实现 |
| AlgoEngine | 🔶 | 已合并入 WorkflowEngine |
| AccountManager | 📋 | 未实现 |
| ExchangeService (加密) | 🔶 | Binance adapter 仅 REST |

### 4.3 AI 智能体系统

| FT 功能 | 状态 | QuantFlow 实现 |
|---------|------|---------------|
| finagent_core | 🔶 | AgentOrchestrator 实现，Skills 体系待完善 |
| MCP (81 工具) | 📋 | mcp.go 未实现 |
| 37+ 智能体角色 | 🔶 | 4 profiles (general/quant/trader/research) |
| TerminalMcpBridge | 📋 | 未实现 |
| AgentService | 🔶 | `internal/ai/` 仅基础框架 |
| AI Chat Screen | ✅ | AIChatPanel (Markdown + 流式) |
| 89 Skills | 📋 | skills 目录未创建 |

### 4.4 研究分析模块 🔴 **核心未实现模块**

| FT 功能 | 状态 | 说明 |
|---------|------|------|
| EquityResearchScreen (7 tabs) | 📋 | 无 StockResearchPanel |
| **股票情绪分析** | 📋 | **无 SentimentNode/后端/Python** |
| 投资组合屏幕 | 🔶 | 仅 PortfolioSummary 面板 |
| QuantLib (590 端点) | 📋 | 未实现 |
| 曲面分析 (35 曲面) | 📋 | 未实现 |
| 回测 (6 提供商) | 🔶 | 仅 CN+US 引擎 |
| Vision Quant | 📋 | 未实现 |

### 4.5 AI Quant Lab

| FT 功能 | 状态 | 说明 |
|---------|------|------|
| Qlib (15 模型) | 📋 | 未实现 |
| 强化学习 (5 算法) | 🔶 | PPO/DQN/SAC Python 实现, RLMonitor 面板 |
| 高频交易 | 📋 | 未实现 |
| 特征工程 (16 指标) | 🔶 | FeatureEngineerNode 基础 |
| 高级回测 (执行算法) | 📋 | TWAP/VWAP 未实现 |
| 在线学习/元学习 | 📋 | 未实现 |

### 4.6 另类数据模块 🔴 **全未实现**

| FT 功能 | 状态 |
|---------|------|
| Polymarket/Kalshi | 📋 |
| 海事智能 | 📋 |
| 地缘政治 | 📋 |
| 政府数据 | 📋 |
| 卫星数据 (NASA/Sentinel) | 📋 |
| 新闻聚合 & NLP | 📋 |

### 4.7 工具与实用功能

| FT 功能 | 状态 |
|---------|------|
| 代码编辑器 (Monaco) | 📋 |
| Excel/Spreadsheet | 📋 |
| Report Builder | 📋 |
| 笔记 | 📋 |
| 文件管理器 | 📋 |
| 语音控制 | 📋 |
| 设置 (18 分区) | 🔶 SettingsPanel 基础 |
| 翻译 (12 语言) | 🔶 zh/en 已实现 |

### 4.8 AstockPursue 核心功能

| 功能 | 状态 |
|------|------|
| 工作流引擎 (58→200+ 节点) | 🔶 54 节点 |
| TradingEngine (bar-by-bar) | ✅ |
| Alpha Factory (450 因子) | 🔶 Go 调用 Python 因子计算，450+ 未全部迁移 |
| GP 进化引擎 | 📋 |
| AI Agent (89 Skills) | 📋 |
| 实验管道 | 📋 |
| 市场状态检测 | 📋 |
| 策略进化 | 📋 |
| 选股器 | 📋 |
| 归因分析 | 📋 |
| 通知系统 | 🔶 Telegram + InApp |
| 模拟交易 | ✅ PaperEngine |
| 策略市场 | 📋 |
| 定时任务 | ✅ Cron + ScheduleNode |

---

## 5. 数据架构

| 组件 | 状态 | 说明 |
|------|------|------|
| MarketDataHub (Go channel) | ✅ | `internal/market/hub.go` |
| ~100 数据连接器 | 🔶 | 14/100 |
| 数据归一化 | 📋 | 未实现 |
| SQLite WAL | ✅ | `internal/storage/db.go` |
| ohlcv_cache 表 | ✅ | Migration 005 |
| trades 表 | ✅ | Migration 004 |
| workflows 表 | ✅ | Migration 001-003 |
| user_config | 📋 | 未实现 |
| L0 内存缓存 (sync.Map) | 🔶 | 部分实现 |
| L1 SQLite 缓存 | ✅ | ohlcv_cache |
| L2 Python gRPC | ✅ | PythonBridge |
| 离线模式 | 📋 | 未实现 |

---

## 6. AI 智能体系统

| 组件 | 状态 | 说明 |
|------|------|------|
| Agent 循环 (ReAct) | 🔶 | Go 实现，但 Skills/MCP 未接入 |
| CapabilityRegistry | 🔶 | 10 capabilities |
| Agent Profile YAML | 🔶 | 4 profiles |
| LLM gRPC (OpenAI/Anthropic/DeepSeek/Ollama) | ✅ | Python LLM service |
| 统一 Tool/Skill 注册 | 📋 | 未将工作流节点注册为 LLM 可调用 tool |
| AgentNode 工作流集成 | ✅ | agent.go 节点 |
| AIChatPanel (SSE + Markdown) | ✅ | 前端面板 |

---

## 7. 前端设计

### 7.1 Terminal Mode

| 组件 | 状态 | 说明 |
|------|------|------|
| CommandBar (Ctrl+K) | ✅ | 搜索面板/命令/导航 |
| DockView 停靠系统 | ✅ | 递归分割 + 多标签 |
| 面板拖拽 | 📋 | 未实现 |
| 自动网格 (1/2/3/4) | 🔶 | 仅 DockView 预设按钮 |
| 撕下面板 (浮动窗口) | 📋 | 未实现 |
| Focus Mode (Ctrl+Shift+F) | 🔶 | 已声明但未完整实现 |
| 布局保存/恢复 | 🔶 | localStorage 不完善 |
| PushPinBar | ✅ | 自选/面板/工作流图钉 |
| StatusBar | ✅ | 连接/模式/时钟 |

#### 面板目录 (规划 50+, 已实现 22)

| 面板 | 状态 | 实现文件 |
|------|------|---------|
| WatchlistPanel | ✅ | `frontend/.../WatchlistPanel.vue` |
| QuoteDetailPanel | ✅ | `QuoteDetailPanel.vue` |
| CandlestickPanel | ✅ | `CandlestickPanel.vue` |
| NewsPanel | ✅ | `NewsPanel.vue` |
| OrderEntryPanel | ✅ | `OrderEntryPanel.vue` |
| PositionPanel | ✅ | `PositionPanel.vue` |
| AIChatPanel | ✅ | `AIChatPanel.vue` |
| SystemMonitorPanel | ✅ | `SystemMonitorPanel.vue` |
| BacktestResultPanel | ✅ | `BacktestResultPanel.vue` |
| FactorAnalysisPanel | ✅ | `FactorAnalysisPanel.vue` |
| PortfolioSummary | ✅ | `PortfolioSummary.vue` |
| PositionDetail | ✅ | `PositionDetail.vue` |
| RiskDashboard | ✅ | `RiskDashboard.vue` |
| TradeHistory | ✅ | `TradeHistory.vue` |
| SchedulePanel | ✅ | `SchedulePanel.vue` |
| NotifyPanel | ✅ | `NotifyPanel.vue` |
| BrokerConfig | ✅ | `BrokerConfig.vue` |
| SettingsPanel | ✅ | `SettingsPanel.vue` |
| ModelRegistryPanel | ✅ | `ModelRegistryPanel.vue` |
| PredictionDashboardPanel | ✅ | `PredictionDashboardPanel.vue` |
| AlphaMiningWorkspacePanel | ✅ | `AlphaMiningWorkspacePanel.vue` |
| RLMonitorPanel | ✅ | `RLMonitorPanel.vue` |
| MarketOverviewPanel | 📋 | 未实现 |
| MarketDepthPanel | 📋 | 未实现 |
| TickerTapePanel | 📋 | 未实现 |
| HeatmapPanel | 📋 | 未实现 |
| CryptoOverviewPanel | 📋 | 未实现 |
| EquityCurvePanel | 📋 | 可复用 ECharts 但无独立面板 |
| SurfaceChartPanel | 📋 | 未实现 |
| CorrelationPanel | 📋 | 未实现 |
| DistributionPanel | 📋 | 未实现 |
| DrawingPanel | 📋 | 未实现 |
| OrderBlotterPanel | 📋 | 未实现 |
| ExecutionPanel | 📋 | 未实现 |
| BasketOrderPanel | 📋 | 未实现 |
| BrokerStatusPanel | 📋 | 未实现 |
| ActionCenterPanel | 📋 | 未实现 |
| WebhookMonitorPanel | 📋 | 未实现 |
| StockResearchPanel | 📋 | 未实现 |
| FinancialsPanel | 📋 | 未实现 |
| AnalystEstimatesPanel | 📋 | 未实现 |
| PeerComparisonPanel | 📋 | 未实现 |
| SentimentPanel | 📋 | **未实现** |
| InsiderTradingPanel | 📋 | 未实现 |
| CongressTradingPanel | 📋 | 未实现 |
| PerformancePanel | 📋 | 未实现 |
| MonteCarloPanel | 📋 | 未实现 |
| RebalancePanel | 📋 | 未实现 |
| PredictionMarketPanel | 📋 | 未实现 |
| MaritimePanel | 📋 | 未实现 |
| GeopoliticsPanel | 📋 | 未实现 |
| SatellitePanel | 📋 | 未实现 |
| GovDataPanel | 📋 | 未实现 |
| SpreadsheetPanel | 📋 | 未实现 |
| CodeEditorPanel | 📋 | 未实现 |
| ReportPanel | 📋 | 未实现 |
| NotesPanel | 📋 | 未实现 |
| FileManagerPanel | 📋 | 未实现 |
| AlertPanel | 📋 | 未实现 |

### 7.2 Workflow Mode

| 组件 | 状态 | 说明 |
|------|------|------|
| vue-flow 画布 | ✅ | Canvas + MiniMap + Controls |
| CustomNode | ✅ | 彩色头部 + 手柄 + 状态指示器 |
| NodePalette | ✅ | 5 类可搜索节点 |
| PropertyPanel | ✅ | 动态属性编辑 + Pin to Terminal |
| ExecutionLog | ✅ | 可折叠日志 |
| TemplatesPanel | 📋 | 未实现 |
| [⊕ 添加到工作流] 按钮 | 📋 | 面板未实现此功能 |
| [固定到终端] | ✅ | 工作流节点右键固定 |

### 7.3 Pinia Stores

| Store | 状态 |
|-------|------|
| session | ✅ 主题/密度/语言/模式 |
| terminal | ✅ 面板布局/命令历史/图钉 |
| workflow | ✅ 画布节点/撤销重做/执行 |
| data | ✅ 行情/K线/数据源缓存 |
| portfolio | ✅ 组合/分配/持仓 |
| ml | ✅ 模型/训练/预测/RL |
| notify | ✅ 通知/未读 |
| settings | ✅ 应用配置 |

---

## 8. 项目目录结构

| 目录 | 状态 | 说明 |
|------|------|------|
| `internal/workflow/` | ✅ | 引擎 + 54 节点实现 |
| `internal/workflow/nodes/` | ✅ | 54 节点文件 |
| `internal/workflow/templates/` | 📋 | 未创建 |
| `internal/trading/` | ✅ | 交易引擎 |
| `internal/trading/brokers/` | 🔶 | Futu + Binance，缺 Alpaca/IBKR/OKX/长桥 |
| `internal/market/` | ✅ | Hub + 14 adapters |
| `internal/ai/` | 🔶 | 基础框架，缺 MCP/Skills |
| `internal/ai/profiles/` | 🔶 | 4 profiles |
| `internal/portfolio/` | ✅ | 组合服务 + 风控 |
| `internal/research/` | 📋 | **未创建 — 情绪分析/股票研究/新闻NLP 在此** |
| `internal/notify/` | ✅ | Telegram + InApp |
| `internal/schedule/` | ✅ | cron 引擎 |
| `internal/storage/` | ✅ | SQLite + migrations |
| `internal/python/` | ✅ | gRPC bridge + proto |
| `internal/config/` | ✅ | Viper |
| `internal/crypto/` | 📋 | 未创建 |
| `python/src/factor/` | ✅ | 因子计算 |
| `python/src/ml/` | ✅ | PPO/DQN/SAC |
| `python/src/llm/` | ✅ | 4 providers |
| `python/src/data/` | ✅ | 数据抓取(mootdx 等) |
| `resources/skills/` | 📋 | 未创建 |
| `resources/agent-profiles/` | 🔶 | 4 profiles 但仅基础 |

---

## 9. 实现路线图 — 完成度

| 阶段 | 内容 | 完成度 | 说明 |
|------|------|--------|------|
| Phase 1 | 核心骨架 | ✅ ~90% | 引擎/面板/工作流画布 |
| Phase 2 | 交易引擎+市场数据 | ✅ ~85% | OMS/PaperEngine/14 adapters |
| Phase 2.5 | 数据源加固 | ✅ | 14 adapters + fallback |
| Phase 3 | 因子+策略+回测 | 🔶 ~60% | CN+US引擎，450因子待迁移 |
| Phase 4 | AI 智能体 | 🔶 ~40% | Agent 框架，Skills/MCP 缺失 |
| Phase 5 | 券商+风控+通知+调度 | ✅ ~80% | Futu+Binance, Telegram, cron |
| Phase 6 | 前端面板+SSE+Stores | ✅ ~80% | 22 panels |
| Phase 7 | 主题+i18n+设置 | ✅ | 深色/浅色主题，zh/en |
| Phase 8 | 节点扩充 20→34 | ✅ | 实际 54 节点 |
| Phase 9 | 因子原子+信号工程 | ✅ | 12+5 节点 |
| Phase 10 | ML 全链路 | 🔶 | Train/Predict/Evaluate/Risk/AlphaMining/RL |
| Phase 11 | 测试覆盖 | ✅ | Go 251 + 前端 76 + Python 120 测试 |
| **Phase 未到** | **研究分析(情绪)** | 📋 **0%** | **SentimentNode + StockResearchPanel + NLP** |
| **Phase 未到** | **另类数据** | 📋 **0%** | **6 面板全部未实现** |
| **Phase 未到** | **工具面板(Spreadsheet/Code/Report/Notes)** | 📋 **0%** | **5 面板全部未实现** |

---

## 10. 关键设计决策 — 遵循情况

| 决策 | 遵循情况 | 说明 |
|------|---------|------|
| SQLite 唯一数据库 | ✅ | 无 PostgreSQL/Redis |
| 双模式 (Terminal+Workflow) | ✅ | 路由切换 |
| Wails v3 + Go 壳 | ✅ | 项目基于 Wails |
| Vue 3 + vue-flow | ✅ | 前端技术栈一致 |
| Python sidecar 可选 | 🔶 | 已实现但核心功能仍依赖 |
| 无印度市场 | ✅ | 无印度券商/数据 |
| TDD + spec-before-code | ✅ | CLAUDE.md 规则已执行 |

---

## 总结: 优先缺失项

### 🔴 高优先级未实现

1. **研究分析模块 (`internal/research/`)** — 情绪分析/新闻NLP/StockResearch
2. **AlternativeData 模块** — 6 种另类数据源
3. **58 缺失的工作流节点** (规划 200+ → 已有 54)
4. **36 缺失的前端面板** (规划 50+ → 已有 22)

### 🟡 中优先级

5. 工作流模板 (50+)
6. Skills 知识库 (89)
7. MCP Provider 集成
8. 浮窗面板
9. 离线模式
10. Broker 补齐 (Alpaca/IBKR/OKX)

### 🟢 低优先级

11. 语音控制
12. 策略市场
13. GP 进化引擎
14. 在线学习/元学习
