<p align="center">
  <img src="https://img.shields.io/badge/QuantFlow-Terminal-black?style=for-the-badge" alt="QuantFlow">
</p>

<h1 align="center">QuantFlow Terminal</h1>

<p align="center">
  <strong>双模式量化金融终端 — 融合彭博式面板终端 + 可视化工作流编排</strong>
</p>

<p align="center">
  <a href="#-project-status"><img src="https://img.shields.io/badge/version-2026.6.18-blue?style=flat-square" alt="Version 2026.6.18"></a>
  <a href="#-project-status"><img src="https://img.shields.io/badge/phase-11%20完成-success?style=flat-square" alt="Phase 11"></a>
  <a href="#-project-status"><img src="https://img.shields.io/badge/nodes-59-blue?style=flat-square" alt="59 Nodes"></a>
  <a href="#-project-status"><img src="https://img.shields.io/badge/panels-22-blue?style=flat-square" alt="22 Panels"></a>
  <a href="#-project-status"><img src="https://img.shields.io/badge/tests-frontend%2076%20%7C%20go%20251%20%7C%20py%20120-brightgreen?style=flat-square" alt="Tests"></a>
  <a href="https://golang.org"><img src="https://img.shields.io/badge/go-1.22+-00ADD8?style=flat-square&logo=go" alt="Go"></a>
  <a href="https://vuejs.org"><img src="https://img.shields.io/badge/vue-3.x-4FC08D?style=flat-square&logo=vue.js" alt="Vue"></a>
  <a href="https://www.sqlite.org"><img src="https://img.shields.io/badge/sqlite-WAL-003B57?style=flat-square&logo=sqlite" alt="SQLite"></a>
  <a href="https://www.python.org"><img src="https://img.shields.io/badge/python-3.12+-blue?style=flat-square&logo=python" alt="Python"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-AGPL--3.0-blue.svg?style=flat-square" alt="License"></a>
</p>

<p align="center">
  <a href="README.en.md">English</a> ·
  <a href="#-项目状态">项目状态</a> ·
  <a href="#-快速开始">快速开始</a> ·
  <a href="#-架构">架构</a> ·
  <a href="#-核心功能">核心功能</a> ·
  <a href="#-路线图">路线图</a>
</p>

---

## 项目状态

> **Phase 1-11 完成** — 59 个工作流节点 · 22 个前端面板 · 11 个开发阶段 · 447 个测试 (Frontend 76 + Go 251 + Python 120)

| 组件 | 状态 | 说明 |
|------|------|------|
| 工作流引擎 (DAG + goroutine 并行执行) | ✅ | 59 节点类型，Kahn 拓扑排序，断点恢复 |
| 桌面壳 (Wails v3 + Vue 3 + TypeScript) | ✅ | 双模式 UI：Terminal 面板 + Workflow 画布 |
| 交易引擎 (OMS + Paper/Live) | ✅ | 订单管理、撮合匹配、风控管线 |
| 行情数据中心 (Go channel pub/sub) | ✅ | 14 数据源适配器，FallbackChain 容灾 |
| 回测引擎 (CN/US/HK/CRYPTO) | ✅ | T+1、涨跌停、印花税等市场规则 |
| Python gRPC Sidecar | ✅ | 25 Alpha 因子、15 AI 技能、4 LLM 提供商 |
| AI Agent 系统 (ReAct 循环) | ✅ | 流式 SSE 事件、4 种 AgentProfile、10+ 能力 |
| 券商集成 | ✅ | Binance REST API 实盘 + Futu 存根 |
| 组合与风险管理 | ✅ | VaR/CVaR/Sharpe/Sortino/MaxDD/Calmar |
| 通知与调度 | ✅ | Telegram/应用内通知 + robfig/cron 调度器 |
| 主题系统 (dark/light + 3 密度) | ✅ | CSS Variables 驱动，localStorage 持久化 |
| 国际化 (zh/en) | ✅ | vue-i18n，~80 翻译键每种语言 |
| SQLite WAL 存储 | ✅ | 10 个迁移 (001-010)，单文件零配置 |
| 前端面板 | ✅ | 18 个彭博式面板 |

---

## 项目定位

QuantFlow 将彭博终端的即时数据面板与可视化工作流编排引擎融为一体。

```
┌─────────────────────────────────────────────────────────┐
│                    QuantFlow Desktop                     │
│  ┌─────────────────────┐    ┌─────────────────────────┐ │
│  │   TERMINAL MODE      │    │    WORKFLOW MODE         │ │
│  │   彭博式面板终端       │◄──►│    可视化工作流画布       │ │
│  │  [AAPL] [Port] [News]│    │  [数据]→[因子]→[策略]→[回测] │ │
│  │  [K线]  [研究] [订单] │    │      └→[AI]→[信号]→[下单]  │ │
│  └─────────────────────┘    └─────────────────────────┘ │
│  共享底层：Go 引擎 · SQLite · 统一数据总线                  │
└─────────────────────────────────────────────────────────┘
```

- **Terminal → Workflow**：面板 `[⊕]` → 生成为工作流节点
- **Workflow → Terminal**：执行结果 `[固定到终端]` → 实时监控面板

---

## 架构

```
Frontend (Vue 3 + Wails v3)          Go Backend (单二进制)
┌──────────────────────────────┐     ┌──────────────────────────────────┐
│ vue-flow · ECharts           │     │ Workflow Engine (Kahn + goroutine)│
│ Pinia (7 stores)             │     │ Trading Engine (OMS + Paper/Live)│
│ Monaco Editor                │◄─IPC►│ MarketDataHub (14 adapters)      │
│ Terminal Mode (22 面板)       │     │ AI Agent (ReAct + 4 LLM)         │
│ Workflow Mode (59 节点类型)    │     │ Portfolio · Risk · Notification  │
│ Dark/Light Theme + i18n      │     │ Scheduler (robfig/cron)           │
└──────────────────────────────┘     │ SQLite WAL (10 migrations)         │
                                     │ gRPC ──► Python Sidecar           │
                                     │   (25 Factors / 15 AI Skills)     │
                                     └──────────────────────────────────┘
```

---

## 核心功能

### 工作流节点 (59 个，16 类别)

| 类别 | 节点 | 数量 |
|------|------|------|
| **数据** | DataLoader, Merge, Filter, Resample | 4 |
| **指标** | SMA, MACD, RSI, EMA, BollingerBands | 5 |
| **Alpha 因子** | pct_change, delta, std_dev, rank, scale, cross_over, compare, bool_combine, rolling_maxmin, rolling_zscore, arithmetic, if_else | 12 |
| **信号** | CrossSignal, ThresholdSignal, SignalCombine, rank_select, hold_signal, rebalance, entry_signal, exit_signal | 8 |
| **策略** | StrategyNode (sma_cross, rsi_threshold, momentum, custom) | 1 |
| **回测** | BacktestNode (CN/US/HK/CRYPTO 市场) | 1 |
| **交易** | PlaceOrder, CancelOrder, PositionQuery, OrderQuery | 4 |
| **组合** | PortfolioSummary, RiskMetrics, Allocation | 3 |
| **风控** | StopLoss, PositionSizer | 2 |
| **通知** | Notify, Alert | 2 |
| **调度** | Schedule, Wait | 2 |
| **控制流** | Loop, if_condition, sub_workflow | 3 |
| **AI** | FactorNode, AgentNode | 2 |
| **ML** | FeatureEngineer, TrainModel, Predict, EvaluateModel, AlphaMining, RLEnv, RLTrain, RLPredict | 8 |
| **工具** | HTTPRequest, MathOperation, JSONParse, LogOutput, chart_data | 5 |
| **风控模型** | RiskModel (GARCH/GJR-GARCH/EGARCH/Covariance) | 1 |

### 前端面板 (22 个)

| 面板 | 类别 | 说明 |
|------|------|------|
| WatchlistPanel | 行情 | 自选股列表，实时报价 |
| QuoteDetailPanel | 行情 | 个股详情，深度数据 |
| CandlestickPanel | 图表 | K 线图，ECharts 渲染 |
| PortfolioSummary | 组合 | 持仓汇总，盈亏分析 |
| PositionDetail | 组合 | 持仓明细，成本分析 |
| RiskDashboard | 风控 | VaR/CVaR/夏普比率仪表盘，GARCH 波动率 |
| TradeHistory | 交易 | 成交/委托记录，CSV 导出 |
| OrderEntryPanel | 交易 | 下单面板，券商选择 |
| BrokerConfig | 交易 | 券商配置，API 密钥管理 |
| BacktestResultPanel | 回测 | 权益曲线，回撤图，指标网格 |
| FactorAnalysisPanel | 研究 | 25 因子分类浏览 |
| AIChatPanel | AI | SSE 流式对话，工具调用可视化 |
| SchedulePanel | 调度 | 定时任务管理 |
| NotifyPanel | 通知 | 通知历史，渠道管理 |
| SettingsPanel | 设置 | 9 配置区，主题/语言/数据/显示等 |
| NewsPanel | 资讯 | 新闻摘要 |
| PositionPanel | 持仓 | 仓位概览 |
| SystemMonitorPanel | 系统 | 系统资源监控 |
| ModelRegistryPanel | ML | 模型注册，CRUD，筛选搜索 |
| PredictionDashboardPanel | ML | 预测仪表盘，IC 时间线 |
| AlphaMiningWorkspacePanel | ML | 因子挖掘工作区 |
| RLMonitorPanel | ML | RL 训练监控，奖励/夏普曲线 |

### 券商支持

| 券商 | 市场 | 状态 | 说明 |
|------|------|------|------|
| Binance | 加密 | ✅ 实盘 | REST API 现货交易、账户查询、持仓 |
| Futu (富途) | A股/港股/美股 | 🔧 存根 | 接口已定义，待实盘接入 |

### AI Agent 系统

- **ReAct 循环**：think → act → observe，带超时和最大步数限制
- **4 个 LLM 提供商**：OpenAI、Anthropic、DeepSeek、Ollama（本地部署）
- **15 个 AI 技能**：分 5 类别（技术分析、基本面分析、风险管理、交易策略、市场微观结构）
- **4 种 AgentProfile**：general、quant_analyst、trader、research_assistant
- **流式输出**：SSE 事件 → 前端 Markdown 渲染 + 工具调用可视化

### 行情数据中心

14 个数据源适配器，自动市场检测，FallbackChain 容灾：

| 市场 | 适配器 | 容灾链 |
|------|--------|--------|
| A股 | Mootdx, TuShare, AKShare, EastMoney, Sina, Tencent, Baidu | 7 源 |
| 美股 | Yahoo, Polygon | 2 源 |
| 加密 | Binance, OKX, CoinGecko | 3 源 |

### 25 Alpha 因子

| 类别 | 因子数 | 示例 |
|------|--------|------|
| 动量 | 5 | momentum_1m, momentum_3m, momentum_6m, momentum_12m, rsi_alpha |
| 趋势 | 5 | ma_cross, macd_divergence, trend_strength, adx_alpha, price_channel |
| 波动率 | 5 | volatility_20d, volatility_60d, atr_alpha, bollinger_position, parkinson_vol |
| 成交量 | 5 | volume_ratio, volume_trend, obv_alpha, mfi_alpha, vwap_deviation |
| 横截面 | 5 | size_factor, sector_neutral_momentum, industry_relative, turnover_alpha, amplitude_alpha |

---

## 快速开始

### 环境要求

- **Go** 1.22+
- **Node.js** 20+ (前端开发)
- **Python** 3.12+ (可选，ML/因子/LLM 功能需要)

### 开发模式

```bash
# 克隆仓库
git clone https://github.com/SZWzz/QuantFlow.git
cd QuantFlow

# 启动开发服务器 (热重载)
wails dev

# Go 后端测试
go test ./internal/... -v -count=1

# 前端测试
cd frontend && npx vitest run

# Python 侧车测试
cd python && python -m pytest tests/ -x -q

# 完整检查
go vet ./... && go test ./...
cd frontend && npx vue-tsc --noEmit && npx vitest run
cd python && python -m pytest tests/ -x -q
```

---

## 技术栈

| 层 | 选择 | 理由 |
|----|------|------|
| 后端 | **Go 1.22+** | goroutine DAG 并行，单二进制部署 |
| 桌面壳 | **Wails v3** | Go 原生，同进程零 IPC 开销 |
| 前端 | **Vue 3 + TypeScript** | vue-flow 画布，Pinia 状态管理 |
| 数据库 | **SQLite (WAL)** | 零配置，单文件，桌面级并发 |
| ML/AI | **Python 3.12+ (gRPC)** | pandas/numpy 生态，独立 sidecar 进程 |
| 国际化 | **vue-i18n** | 编译时优化，中文/英文双语 |
| 主题 | **CSS Variables** | 双主题 + 三密度，运行时切换 |

### 为什么不...

| 避免 | 原因 |
|------|------|
| PostgreSQL/Redis | 桌面应用，单用户 — SQLite WAL 足够 |
| Tauri | Go+Rust 双工具链 — Wails 同语言 |
| React | vue-flow 是 xyflow 官方 Vue 移植；Pinia 优于 Zustand |
| Docker | 桌面应用 — Go 单二进制，无需容器 |
| 印度市场 | 重点聚焦：A股 > 港股 > 美股 > 加密 |

---

## 路线图

| 阶段 | 目标 | 状态 |
|------|------|------|
| Phase 1 | 纯 Go 工作流引擎 + CLI + SQLite 存储 | ✅ 完成 |
| Phase 2 | Wails 桌面壳 + Vue 3 前端 + 交易引擎 + 行情数据中心 | ✅ 完成 |
| Phase 2.5 | 14 数据源适配器 + FallbackChain 容灾 | ✅ 完成 |
| Phase 3 | Python gRPC Sidecar + 25 Alpha 因子 + 回测引擎 | ✅ 完成 |
| Phase 4 | AI Agent 系统 (ReAct + 4 LLM + 15 技能) | ✅ 完成 |
| Phase 5 | 券商集成 + 组合/风控 + 通知 + 调度器 | ✅ 完成 |
| Phase 6 | 7 前端面板 + SSE + Pinia 商店扩展 | ✅ 完成 |
| Phase 7 | 主题系统 (dark/light + 3 密度) + i18n + 设置面板 | ✅ 完成 |
| Phase 8 | 节点扩展 (20 → 34) | ✅ 完成 |
| Phase 9 | 因子原子 + 信号工程 (34 → 54) | ✅ 完成 |
| Phase 10 | ML 引擎 + Alpha 挖掘 + RL 交易 + 风险建模 (54 → 59 节点, 18 → 22 面板) | ✅ 完成 |

---

## 目录结构

```
quantflow/
├── main.go                    # Wails 应用入口
├── app.go                     # 导出 Go 函数（前端绑定）
├── go.mod / go.sum            # Go 模块定义
├── internal/
│   ├── workflow/              # 工作流引擎 (node, dag, engine)
│   │   └── nodes/             # 59 个节点实现
│   ├── trading/               # 交易引擎 (OMS + 撮合)
│   │   └── brokers/           # Binance/Futu 券商适配器
│   ├── market/                # 行情数据中心
│   │   └── adapters/          # 14 个数据源适配器
│   ├── backtest/              # 回测引擎 (CN/US/HK/CRYPTO)
│   ├── ai/                    # AI Agent 系统
│   │   └── capabilities/      # 10+ Agent 能力
│   ├── portfolio/             # 组合管理与风险计算
│   ├── notify/                # 通知引擎 (Telegram/应用内)
│   ├── schedule/              # robfig/cron 调度器
│   ├── storage/               # SQLite WAL + 迁移框架
│   │   └── migrations/        # 010 之前的 10 个迁移
│   ├── python/                # gRPC 桥接到 Python
│   ├── config/                # YAML 配置
│   └── logging/               # slog 封装
├── frontend/                  # Vue 3 前端
│   └── src/
│       ├── terminal/          # Terminal Mode 组件
│       │   ├── panels/        # 22 个彭博式面板
│       │   └── DockView/      # 可停靠面板系统
│       ├── workflow/          # Workflow Mode 组件
│       │   └── canvas/        # vue-flow 画布
│       ├── stores/            # 7 个 Pinia 商店
│       └── lib/               # i18n、主题、格式化工具
├── python/                    # Python gRPC Sidecar
│   ├── src/
│   │   ├── factor/            # 25 Alpha 因子
│   │   ├── ml/                # ML 模型 (rl/, risk/, alpha_mining/)
│   │   ├── llm/               # LLM 推理 (4 提供商)
│   │   │   └── providers/     # OpenAI/Anthropic/DeepSeek/Ollama
│   │   ├── skills/            # 15 AI 技能
│   │   └── data/              # 数据获取脚本
│   ├── proto/                 # gRPC 服务定义
│   └── tests/                 # Python 测试
├── resources/                 # 图标、模板、Agent Profile
├── docs/
│   └── specs/                 # Spec 文档（每个变更一篇）
├── examples/                  # 工作流示例
├── CHANGELOG.md               # 变更日志
├── CLAUDE.md                  # Claude Code 指引
└── LICENSE                    # AGPL-3.0
```

---

## 市场聚焦

| 市场 | 结算 | 关键规则 | 数据源 | 券商 |
|------|------|----------|--------|------|
| A 股 | T+1 | 涨跌停 ±10%/±20%, 印花税 0.05% | EastMoney, AKShare, TuShare, Mootdx, Sina, Tencent, Baidu | 富途 (存根) |
| 港股 | T+2 | 港股通, 交收 T+2 | Futu, 新浪 | 富途 (存根), IBKR |
| 美股 | T+2 | PDT 规则, wash sale | Yahoo, Polygon, Alpaca | Alpaca, IBKR, Tradier |
| 加密 | 即时 | 永续资金费率, 强平 | Binance, OKX, CoinGecko | Binance (实盘), OKX, Bybit |

---

## 许可证

[AGPL-3.0](LICENSE) © 2024-2026 QuantFlow Contributors
