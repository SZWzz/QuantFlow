<p align="center">
  <img src="https://img.shields.io/badge/QuantFlow-Terminal-111827?style=for-the-badge&logo=quantum" alt="QuantFlow">
</p>

<h1 align="center">QuantFlow Terminal</h1>

<p align="center">
  <strong>双模式量化金融终端 — 彭博式面板终端 × 可视化工作流编排</strong>
</p>

<p align="center">
  <a href="#-项目状态"><img src="https://img.shields.io/badge/版本-2026.6.19-3b82f6?style=flat-square" alt="Version"></a>
  <a href="#-项目状态"><img src="https://img.shields.io/badge/阶段-11-success?style=flat-square" alt="Phase"></a>
  <a href="#-项目状态"><img src="https://img.shields.io/badge/节点-54+-3b82f6?style=flat-square" alt="Nodes"></a>
  <a href="#-项目状态"><img src="https://img.shields.io/badge/面板-46-8b5cf6?style=flat-square" alt="Panels"></a>
  <a href="#-项目状态"><img src="https://img.shields.io/badge/适配器-25+-f59e0b?style=flat-square" alt="Adapters"></a>
  <a href="#-项目状态"><img src="https://img.shields.io/badge/测试-476-brightgreen?style=flat-square" alt="Tests"></a>
  <br>
  <a href="https://golang.org"><img src="https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat-square&logo=go" alt="Go"></a>
  <a href="https://vuejs.org"><img src="https://img.shields.io/badge/Vue-3.x-4FC08D?style=flat-square&logo=vue.js" alt="Vue"></a>
  <a href="https://www.sqlite.org"><img src="https://img.shields.io/badge/SQLite-WAL-003B57?style=flat-square&logo=sqlite" alt="SQLite"></a>
  <a href="https://www.python.org"><img src="https://img.shields.io/badge/Python-3.12+-3776AB?style=flat-square&logo=python" alt="Python"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/许可-AGPL--3.0-blue?style=flat-square" alt="License"></a>
</p>

---

## 📊 项目状态

> **Phase 1–11 完成** — 54+ 工作流节点 · 46 前端面板 · 25+ 数据适配器 · 476 测试 (前端 164 + Go 192 + Python 120)

| 组件 | 状态 | 说明 |
|------|:----:|------|
| 工作流引擎 | ✅ | DAG + goroutine 并行 + Kahn 拓扑排序 |
| 桌面壳 (Wails v3 + Vue 3) | ✅ | Terminal/Workflow 双模式一键切换 |
| 交易引擎 (OMS) | ✅ | Paper/Live 双模式，Alpaca/Binance 实盘 |
| 行情数据中心 | ✅ | 25+ 适配器，4 市场全覆盖，Fallback 容灾 |
| 回测引擎 | ✅ | CN/US/HK/CRYPTO 市场规则 |
| Python gRPC Sidecar | ✅ | 因子/ML/LLM/NLP 独立进程 |
| AI Agent 系统 | ✅ | ReAct 循环 + 4 LLM + 4 AgentProfile |
| 券商集成 | ✅ | Alpaca(美股) + Binance(加密) + Futu(港股) |
| 组合与风险管理 | ✅ | VaR/CVaR/Sharpe/MaxDD/Calmar |
| 通知 + 定时调度 | ✅ | Telegram/应用内 + cron 引擎 |
| 主题系统 | ✅ | 暗色/亮色 + 3 种密度 |
| 国际化 | ✅ | 中文/英文双语 |
| SQLite WAL 存储 | ✅ | 单文件零配置，10+ 迁移 |

---

## 🎯 核心理念

```
┌──────────────────────────────────────────────────────────┐
│                    QuantFlow Desktop                      │
│  ┌──────────────────────┐  ┌───────────────────────────┐ │
│  │   TERMINAL 模式       │  │    WORKFLOW 模式           │ │
│  │   彭博式面板终端       │◄─►│    可视化策略编排           │ │
│  │  [行情] [K线] [研究]   │  │  [数据]→[因子]→[策略]→[回测] │ │
│  │  [订单] [组合] [风控]  │  │      └→[AI]→[信号]→[下单]  │ │
│  └──────────────────────┘  └───────────────────────────┘ │
│         共享底层：Go 引擎 · SQLite · 统一数据总线            │
└──────────────────────────────────────────────────────────┘
```

- **Terminal → Workflow**：面板 `[⊕]` → 一键生成工作流节点
- **Workflow → Terminal**：执行结果 `[固定到终端]` → 实时监控面板

---

## 🏗️ 架构

```
前端 (Vue 3)                      Go 后端 (单二进制)
┌─────────────────────────┐       ┌──────────────────────────────┐
│ vue-flow · ECharts       │       │ 工作流引擎 (Kahn + goroutine) │
│ Pinia (8 stores)         │       │ 交易引擎 (OMS + Paper/Live)  │
│ Terminal Mode (46 面板)   │◄─IPC─►│ 行情中心 (25 适配器)          │
│ Workflow Mode (54 节点)   │       │ AI Agent (ReAct + 4 LLM)    │
│ 暗色/亮色主题 + i18n      │       │ 组合 · 风控 · 通知 · 调度    │
└─────────────────────────┘       │ SQLite WAL (10+ 迁移)          │
                                  │ gRPC ──► Python Sidecar      │
                                  │   (因子/ML/NLP/LLM)          │
                                  └──────────────────────────────┘
```

---

## 📦 核心功能

### 工作流节点 (54+, 16 类别)

| 类别 | 数量 | 代表节点 |
|------|:----:|---------|
| 数据加载 | 4 | DataLoader, Merge, Filter, Resample |
| 技术指标 | 5 | SMA, MACD, RSI, EMA, BollingerBands |
| Alpha 因子 | 12 | pct_change, rank, zscore, cross_over, if_else |
| 信号工程 | 8 | CrossSignal, Threshold, hold_signal, entry/exit |
| 策略构建 | 1 | StrategyNode (sma_cross/rsi/momentum/custom) |
| 回测执行 | 1 | BacktestNode (CN/US/HK/CRYPTO) |
| 交易执行 | 4 | PlaceOrder, CancelOrder, Position/OrderQuery |
| 组合管理 | 3 | PortfolioSummary, RiskMetrics, Allocation |
| 风控 | 2 | StopLoss, PositionSizer |
| ML 引擎 | 8 | FeatureEngineer, Train/Predict/Evaluate, RL×3 |
| AI Agent | 2 | FactorNode, AgentNode |
| 通知 | 2 | Notify, Alert |
| 控制流 | 3 | Loop, if_condition, sub_workflow |
| 调度 | 2 | Schedule, Wait |
| 研究分析 | 6 | Sentiment, StockResearch, Financials, Peers, Estimates, Insider |
| 工具 | 2 | HTTPRequest, MathOperation, JSONParse, chart_data |

### 前端面板 (46 个)

| 类别 | 面板 |
|------|------|
| **行情** (6) | Watchlist, QuoteDetail, Candlestick, MarketOverview, MarketDepth, Heatmap |
| **滚动** (1) | TickerTape |
| **交易** (8) | OrderEntry, OrderBlotter, Execution, BasketOrder, Position, PositionDetail, BrokerConfig, BrokerStatus |
| **组合** (3) | PortfolioSummary, Rebalance, RiskDashboard |
| **研究** (7) | StockResearch, Financials, Sentiment, PeerComparison, AnalystEstimates, InsiderTrading, CongressTrading |
| **图表** (5) | EquityCurve, Correlation, Distribution, MonteCarlo, SurfaceChart |
| **AI/ML** (5) | AIChat, ModelRegistry, PredictionDashboard, AlphaMining, RLMonitor |
| **回测** (1) | BacktestResult |
| **因子** (1) | FactorAnalysis |
| **加密** (1) | CryptoOverview |
| **资讯** (1) | News |
| **工具** (2) | Drawing, ActionCenter |
| **系统** (4) | Schedule, Notify, Settings, SystemMonitor |

### 数据适配器 (25+, 4 市场全覆盖)

| 市场 | 适配器 | 可用性 |
|------|--------|:------:|
| **A股** | mootdx(通达信) · sina · eastmoney · tencent · baidu · akshare · tushare · ths · cninfo · iwencai | 10 源容灾 |
| **港股** | sina · akshare/tencent · yahoo | 3 源 |
| **美股** | yahoo(v8) · finnhub · polygon · alpaca | 4 源 |
| **加密** | gateio · binance · okx · coingecko | 4 源 |
| **专项** | eastmoney_news/global_news/capital/concept/fundflow/signals/report · sina_financials · ths_hot/consensus/northbound | 11 专项 |

### 券商支持

| 券商 | 市场 | 状态 | 说明 |
|------|------|:----:|------|
| **Alpaca** | 美股 | ✅ 实盘 | Paper/Live REST API，订单/持仓/账户 |
| **Binance** | 加密 | ✅ 实盘 | 现货 REST API，订单/持仓/账户 |
| **Futu (富途)** | 港股 | 🔧 存根 | 接口已定义，待实盘接入 |

### AI Agent 系统

- **ReAct 循环**：think → act → observe，带超时和步数限制
- **4 LLM 提供商**：OpenAI · Anthropic · DeepSeek · Ollama (本地)
- **4 AgentProfile**：general · quant_analyst · trader · research_assistant
- **10+ 内置能力**：quote_lookup, search_symbol, compute_factor, run_backtest...
- **流式 SSE** → 前端 Markdown 渲染 + 工具调用可视化

---

## 🚀 快速开始

### 环境要求

- **Go** 1.22+
- **Node.js** 20+
- **Python** 3.12+ (可选，ML/因子/LLM 需要)

### 开发

```bash
git clone https://github.com/SZWzz/QuantFlow.git
cd QuantFlow

# 启动开发服务器 (热重载)
wails dev

# 完整检查
go vet ./... && go test ./...                                  # Go: 192 tests
cd frontend && npx vue-tsc --noEmit && npx vitest run          # 前端: 164 tests
cd python && python -m pytest tests/ -x -q                      # Python: 120 tests
```

---

## 🗺️ 技术选型

| 层 | 选择 | 理由 |
|----|------|------|
| 后端 | **Go 1.22+** | goroutine DAG 并行，单二进制部署 |
| 桌面 | **Wails v3** | Go 原生壳，同进程零开销 |
| 前端 | **Vue 3 + TypeScript** | vue-flow 画布，Pinia 状态管理 |
| 数据库 | **SQLite WAL** | 零配置，单文件，桌面级并发 |
| ML | **Python 3.12+ (gRPC)** | pandas/numpy 生态，独立 sidecar |
| 图表 | **ECharts** | 金融图表全覆盖，GL 3D 支持 |
| 主题 | **CSS Variables** | 双主题 + 三密度，运行时切换 |

### 为什么不...

| 不用 | 原因 |
|------|------|
| PostgreSQL/Redis | 桌面应用单用户 — SQLite WAL 足够 |
| Tauri | Go+Rust 双工具链 — Wails 同语言 |
| React | vue-flow 是 xyflow 官方 Vue 移植 |
| Docker | Go 单二进制，无需容器 |
| 印度市场 | 聚焦：A股 > 港股 > 美股 > 加密 |

---

## 📋 路线图

| 阶段 | 目标 | 状态 |
|------|------|:----:|
| Phase 1 | 纯 Go 工作流引擎 + CLI + SQLite | ✅ |
| Phase 2 | Wails 桌面壳 + Vue 3 + 交易引擎 + 行情中心 | ✅ |
| Phase 2.5 | 14 数据源适配器 + FallbackChain 容灾 | ✅ |
| Phase 3 | Python gRPC + 25 Alpha 因子 + 回测引擎 | ✅ |
| Phase 4 | AI Agent 系统 (ReAct + 4 LLM) | ✅ |
| Phase 5 | 券商 + 组合/风控 + 通知 + 调度 | ✅ |
| Phase 6 | 7 前端面板 + SSE + Pinia 商店 | ✅ |
| Phase 7 | 主题系统 + i18n + 设置面板 | ✅ |
| Phase 8 | 节点扩展 20→34 | ✅ |
| Phase 9 | 因子原子 + 信号工程 34→54 | ✅ |
| Phase 10 | ML 引擎 + Alpha 挖掘 + RL + 风险建模 | ✅ |
| Phase 11 | 测试覆盖 + 数据源补强 + 面板扩展 | ✅ |
| **Phase 12** | **多市场数据源完善 + 更多券商 + 面板补齐** | 🔜 |

---

## 📁 目录结构

```
quantflow/
├── main.go                       # Wails 入口
├── app.go                        # Go 导出函数(前端绑定)
├── internal/
│   ├── workflow/                 # 工作流引擎 + 54 节点
│   ├── trading/                  # OMS + 券商适配器
│   │   └── brokers/              # Alpaca / Binance / Futu
│   ├── market/                   # 行情中心 + 25 适配器
│   │   └── adapters/             # 全部数据源实现
│   ├── research/                 # 研究分析服务 (9 Service)
│   ├── ai/                       # AI Agent 系统
│   ├── portfolio/                # 组合管理 + 风控
│   ├── notify/                   # 通知引擎 (Telegram/InApp)
│   ├── schedule/                 # cron 调度器
│   ├── storage/                  # SQLite WAL + 迁移
│   └── python/                   # gRPC 桥接
├── frontend/                     # Vue 3 前端
│   └── src/
│       ├── terminal/panels/      # 46 面板
│       ├── workflow/             # vue-flow 画布
│       ├── stores/               # 8 Pinia 商店
│       └── lib/                  # i18n · 主题 · stats
├── python/                       # Python gRPC Sidecar
│   └── src/ (factor/ml/llm/research/data)
├── docs/specs/                   # Spec 文档
├── resources/                    # 图标 · 模板
├── CHANGELOG.md                  # 变更日志 (中文)
└── LICENSE                       # AGPL-3.0
```

---

## 🌍 市场聚焦

| 市场 | 结算 | 关键规则 | 主要数据源 | 券商 |
|------|------|----------|-----------|------|
| A 股 | T+1 | 涨跌停 ±10%/±20%, 印花税 0.05% | mootdx/东财/新浪/腾讯/百度/同花顺 | — |
| 港股 | T+2 | 港股通, T+2 交收 | 新浪/腾讯/AkShare/Yahoo | 富途(存根) |
| 美股 | T+2 | PDT 规则, wash sale | Yahoo/Finnhub/Polygon | Alpaca(实盘) |
| 加密 | 即时 | 永续资金费率, 强平 | Gate.io/Binance/OKX | Binance(实盘) |

---

## 📄 许可证

[AGPL-3.0](LICENSE) © 2024–2026 QuantFlow Contributors
