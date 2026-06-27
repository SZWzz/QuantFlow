<p align="center">
  <img src="https://img.shields.io/badge/QuantFlow-Terminal-black?style=for-the-badge" alt="QuantFlow">
</p>

<h1 align="center">QuantFlow Terminal</h1>

<p align="center">
  <strong>Dual-Mode Quantitative Finance Terminal — Bloomberg-Style Panels + Visual Workflow Orchestration</strong>
</p>

<p align="center">
  <a href="#-project-status"><img src="https://img.shields.io/badge/phase-13%20complete-success?style=flat-square" alt="Phase 13"></a>
  <a href="#-project-status"><img src="https://img.shields.io/badge/nodes-93-blue?style=flat-square" alt="93 Nodes"></a>
  <a href="#-project-status"><img src="https://img.shields.io/badge/panels-64-blue?style=flat-square" alt="64 Panels"></a>
  <a href="https://golang.org"><img src="https://img.shields.io/badge/go-1.22+-00ADD8?style=flat-square&logo=go" alt="Go"></a>
  <a href="https://vuejs.org"><img src="https://img.shields.io/badge/vue-3.x-4FC08D?style=flat-square&logo=vue.js" alt="Vue"></a>
  <a href="https://www.sqlite.org"><img src="https://img.shields.io/badge/sqlite-WAL-003B57?style=flat-square&logo=sqlite" alt="SQLite"></a>
  <a href="https://www.python.org"><img src="https://img.shields.io/badge/python-3.12+-blue?style=flat-square&logo=python" alt="Python"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-AGPL--3.0-blue.svg?style=flat-square" alt="License"></a>
</p>

<p align="center">
  <a href="README.md">中文</a> ·
  <a href="#-project-status">Project Status</a> ·
  <a href="#-quick-start">Quick Start</a> ·
  <a href="#-architecture">Architecture</a> ·
  <a href="#-core-features">Core Features</a> ·
  <a href="#-roadmap">Roadmap</a>
</p>

---

## Project Status

> **Phases 1–13 All Complete** — 93 workflow nodes · 64 frontend panels · 37 data adapters · 832 tests

| Component | Status | Details |
|-----------|--------|---------|
| Workflow Engine (DAG + goroutine parallel) | ✅ | 93 node types, Kahn topological sort, breakpoint recovery |
| Desktop Shell (Wails v3 + Vue 3 + TypeScript) | ✅ | Dual-mode UI: Terminal panels + Workflow canvas |
| Trading Engine (OMS + Paper/Live) | ✅ | Order management, order matching, risk pipeline |
| MarketDataHub (Go channel pub/sub) | ✅ | 37 data adapters, MAC protocol, FallbackChain resilience |
| Backtesting Engine (CN/US/HK/CRYPTO) | ✅ | T+1, price limits, stamp duty, market-specific rules |
| Python gRPC Sidecar | ✅ | 25 Alpha factors, 15 AI skills, 4 LLM providers |
| AI Agent System (ReAct loop) | ✅ | Streaming SSE events, 4 AgentProfiles, 10+ capabilities |
| Broker Integration | ✅ | Binance REST API live + Futu stub |
| Portfolio & Risk Management | ✅ | VaR/CVaR/Sharpe/Sortino/MaxDD/Calmar |
| Notifications & Scheduler | ✅ | Telegram/in-app + robfig/cron scheduler |
| Theme System (dark/light + 3 densities) | ✅ | CSS Variables driven, localStorage persistence |
| Internationalization (zh/en) | ✅ | vue-i18n, ~350 translation keys per language |
| SQLite WAL Storage | ✅ | 12+ migrations, single-file zero-config |
| Frontend Panels | ✅ | 64 Bloomberg-style panels |

---

## What is QuantFlow?

QuantFlow merges the instant data access of a Bloomberg Terminal with the visual pipeline orchestration of a workflow engine — in a single desktop application.

```
┌─────────────────────────────────────────────────────────┐
│                    QuantFlow Desktop                     │
│  ┌─────────────────────┐    ┌─────────────────────────┐ │
│  │   TERMINAL MODE      │    │    WORKFLOW MODE         │ │
│  │  Bloomberg-style      │◄──►│    Visual DAG Editor     │ │
│  │  Dockable Panels      │    │    Drag-and-Drop Nodes   │ │
│  │                       │    │                          │ │
│  │  [AAPL] [Port] [News] │    │  [Data]→[Factor]→[Strategy] │
│  │  [Chart][Research]    │    │     └→[AI]→[Signal]→[Trade] │
│  └─────────────────────┘    └─────────────────────────┘ │
│  Shared Core: Go Engine · SQLite · Unified Data Bus      │
└─────────────────────────────────────────────────────────┘
```

**Bidirectional Flow**:

- **Terminal → Workflow**: Any panel's `[⊕]` button generates a workflow node
- **Workflow → Terminal**: Execution results `[Pin to Terminal]` create live monitoring panels

---

## Architecture

```
Frontend (Vue 3 + Wails v3)          Go Backend (Single Binary)
┌──────────────────────────────┐     ┌──────────────────────────────────┐
│ vue-flow · ECharts           │     │ Workflow Engine (Kahn + goroutine)│
│ Pinia (8 stores)             │     │ Trading Engine (OMS + Paper/Live)│
│ ECharts · Monaco Editor      │◄─IPC►│ MarketDataHub (37 adapters)      │
│ Terminal Mode (64 panels)    │     │ AI Agent (ReAct + 4 LLM)         │
│ Workflow Mode (93 node types)│     │ Portfolio · Risk · Notification  │
│ Dark/Light Theme + i18n      │     │ Scheduler (robfig/cron)           │
└──────────────────────────────┘     │ SQLite WAL (12+ migrations)       │
                                     │ gRPC ──► Python Sidecar           │
                                     │   (25 Factors / 15 AI Skills)     │
                                     └──────────────────────────────────┘
```

---

## Core Features

### Workflow Nodes (93 nodes, 18 categories)

| Category | Nodes | Count |
|----------|-------|-------|
| **Data** | DataLoader, Merge, Filter, Resample | 4 |
| **Indicator** | SMA, MACD, RSI, EMA, Bollinger, OBV, MFI, PSY, Aroon, ASI, WR, CCI, ROC, BIAS, Chaikin, Keltner, Donchian, TRIX, MassIndex, Vortex | 20 |
| **Chanlun** | ChanlunBi, ChanlunDuan, ChanlunZhongshu, ChanlunMaiDian, ChanlunLeixing | 5 |
| **Alpha Factor** | pct_change, delta, std_dev, rank, scale, cross_over, compare, bool_combine, rolling_maxmin, rolling_zscore, arithmetic, if_else | 12 |
| **Signal** | CrossSignal, ThresholdSignal, SignalCombine, rank_select, hold_signal, rebalance, entry_signal, exit_signal | 8 |
| **Strategy** | StrategyNode (sma_cross, rsi_threshold, momentum, custom) | 1 |
| **Backtest** | BacktestNode (CN/US/HK/CRYPTO markets) | 1 |
| **Slippage** | FixedSlippage, PercentageSlippage, VolumeSlippage | 3 |
| **Trading** | PlaceOrder, CancelOrder, PositionQuery, OrderQuery | 4 |
| **Portfolio** | PortfolioSummary, RiskMetrics, Allocation | 3 |
| **Risk** | StopLoss, PositionSizer | 2 |
| **Notify** | Notify, Alert | 2 |
| **Schedule** | Schedule, Wait | 2 |
| **Control** | Loop, if_condition, sub_workflow | 3 |
| **AI** | FactorNode, AgentNode | 2 |
| **Research** | Sentiment, StockResearch, Financials, Peers, Estimates, Insider | 6 |
| **ML** | FeatureEngineer, TrainModel, PredictModel, EvaluateModel, RL×3 | 8 |
| **Utility** | HTTPRequest, MathOperation, JSONParse, LogOutput, chart_data, fqfactor | 6 |

### Frontend Panels (64 panels)

| Category | Panels |
|----------|--------|
| **Market** (6) | Watchlist, QuoteDetail, Candlestick, MarketOverview, MarketDepth, Heatmap |
| **Ticker** (1) | TickerTape |
| **Trading** (8) | OrderEntry, OrderBlotter, Execution, BasketOrder, Position, PositionDetail, BrokerConfig, BrokerStatus |
| **Portfolio** (3) | PortfolioSummary, Rebalance, RiskDashboard |
| **Research** (8) | StockResearch, Financials, Sentiment, PeerComparison, AnalystEstimates, InsiderTrading, CongressTrading, FactorAnalysis |
| **Chart** (5) | EquityCurve, Correlation, Distribution, MonteCarlo, SurfaceChart |
| **AI/ML** (5) | AIChat, ModelRegistry, PredictionDashboard, AlphaMining, RLMonitor |
| **Backtest** (1) | BacktestResult |
| **Chanlun** (3) | ChanlunBi, ChanlunDuan, ChanlunZhongshu |
| **Crypto** (1) | CryptoOverview |
| **News** (1) | News |
| **Tools** (3) | Drawing, ActionCenter, MACProtocol |
| **System** (4) | Schedule, Notify, Settings, SystemMonitor |

### Broker Support

| Broker | Market | Status | Details |
|--------|--------|--------|---------|
| Alpaca | US Stocks | ✅ Live | Paper/Live REST API, orders/positions/accounts |
| Binance | Crypto | ✅ Live | REST API spot orders, account, positions |
| Futu (富途) | CN/HK/US | 🔧 Stub | Interface defined, awaiting live integration |

### AI Agent System

- **ReAct Loop**: think → act → observe, with timeout and max-step limits
- **4 LLM Providers**: OpenAI, Anthropic, DeepSeek, Ollama (local deployment)
- **15 AI Skills**: Across 5 categories (technical analysis, fundamental analysis, risk management, trading strategies, market microstructure)
- **4 AgentProfiles**: general, quant_analyst, trader, research_assistant
- **Streaming Output**: SSE events → frontend Markdown rendering + tool call visualization

### Market Data Hub

37 data source adapters, automatic market detection, MAC protocol direct TCP, FallbackChain resilience:

| Market | Adapters | Fallback Chain |
|--------|----------|----------------|
| CN (A-Share) | Mootdx, MAC Protocol, Sina, EastMoney, Tencent, Baidu, AKShare, TuShare, THS, cninfo, iwencai | 11 sources |
| HK | Sina, AKShare/Tencent, Yahoo | 3 sources |
| US | Yahoo(v8), Finnhub, Polygon, Alpaca | 4 sources |
| Crypto | Gate.io, Binance, OKX, CoinGecko | 4 sources |
| Specialized | EastMoney_news/global/capital/concept/fundflow/signals/report, Sina_financials, THS_hot/consensus/northbound, MAC golden/master/sector | 15 sources |

### 25 Alpha Factors

| Category | Count | Examples |
|----------|-------|----------|
| Momentum | 5 | momentum_1m, momentum_3m, momentum_6m, momentum_12m, rsi_alpha |
| Trend | 5 | ma_cross, macd_divergence, trend_strength, adx_alpha, price_channel |
| Volatility | 5 | volatility_20d, volatility_60d, atr_alpha, bollinger_position, parkinson_vol |
| Volume | 5 | volume_ratio, volume_trend, obv_alpha, mfi_alpha, vwap_deviation |
| Cross-sectional | 5 | size_factor, sector_neutral_momentum, industry_relative, turnover_alpha, amplitude_alpha |

---

## Quick Start

### Prerequisites

- **Go** 1.22+
- **Node.js** 20+ (frontend development)
- **Python** 3.12+ (optional, needed for ML/factors/LLM)

### Development

```bash
# Clone the repository
git clone https://github.com/SZWzz/QuantFlow.git
cd QuantFlow

# Start dev server (hot reload)
wails dev

# Go backend tests (383 tests)
go test ./... -v -count=1

# Frontend tests
cd frontend && npx vitest run

# Python sidecar tests
cd python && python -m pytest tests/ -x -q

# Full check before commit
go vet ./... && go test ./...         # Go: 383 tests
cd frontend && npx vue-tsc --noEmit && npx vitest run         # Frontend: 304 tests
cd python && python -m pytest tests/ -x -q                    # Python: 145 tests
```

---

## Tech Stack

| Layer | Choice | Rationale |
|-------|--------|-----------|
| Backend | **Go 1.22+** | Goroutines for DAG parallelism, single binary deploy |
| Desktop Shell | **Wails v3** | Go-native, same process, zero IPC overhead |
| Frontend | **Vue 3 + TypeScript** | vue-flow canvas, Pinia state management |
| Database | **SQLite (WAL)** | Zero-config, single-file, desktop-grade concurrency |
| ML/AI | **Python 3.12+ (gRPC)** | pandas/numpy ecosystem, independent sidecar process |
| i18n | **vue-i18n** | Compile-time optimized, Chinese/English bilingual |
| Theming | **CSS Variables** | Dual theme + 3 densities, runtime switching |

### Why Not...

| Avoided | Reason |
|---------|--------|
| PostgreSQL/Redis | Desktop app, single-user — SQLite WAL suffices |
| Tauri | Go+Rust dual toolchain — Wails uses same Go language |
| React | vue-flow is the official xyflow Vue port; Pinia > Zustand |
| Docker | Desktop app — single Go binary, no containers needed |
| Indian Market | Focus: CN > HK > US > Crypto |

---

## Roadmap

| Phase | Goal | Status |
|-------|------|--------|
| Phase 1 | Pure-Go workflow engine + CLI + SQLite storage | ✅ Complete |
| Phase 2 | Wails desktop shell + Vue 3 frontend + trading engine + market data | ✅ Complete |
| Phase 2.5 | 14 data source adapters + FallbackChain resilience | ✅ Complete |
| Phase 3 | Python gRPC sidecar + 25 Alpha factors + backtesting engine | ✅ Complete |
| Phase 4 | AI Agent system (ReAct + 4 LLM + 15 skills) | ✅ Complete |
| Phase 5 | Broker integration + portfolio/risk + notifications + scheduler | ✅ Complete |
| Phase 6 | 7 frontend panels + SSE + Pinia store expansion | ✅ Complete |
| Phase 7 | Theme system (dark/light + 3 densities) + i18n + settings panel | ✅ Complete |
| Phase 8 | Node expansion (20 → 34) | ✅ Complete |
| Phase 9 | Factor atoms + signal engineering (34 → 54) | ✅ Complete |
| Phase 10 | ML engine + Alpha mining + RL + Risk modeling | ✅ Complete |
| Phase 11 | Test coverage + Data source hardening + Panel expansion | ✅ Complete |
| Phase 12 | easy-tdx deep integration + Frontend quality + P0 correctness | ✅ Complete |
| Phase 13 | Chanlun/Indicator nodes + MAC protocol + Minute chart + Stock names | ✅ Complete |

---

## Directory Structure

```
quantflow/
├── main.go                    # Wails application entry point
├── app.go                     # Exported Go functions (frontend bindings)
├── go.mod / go.sum            # Go module definition
├── internal/
│   ├── workflow/              # Workflow engine (node, dag, engine)
│   │   └── nodes/             # 93 node implementations
│   ├── trading/               # Trading engine (OMS + matching)
│   │   └── brokers/           # Binance/Futu broker adapters
│   ├── market/                # Market data hub
│   │   └── adapters/          # 37 data source adapters
│   ├── backtest/              # Backtesting engine (CN/US/HK/CRYPTO)
│   ├── ai/                    # AI Agent system
│   │   └── capabilities/      # 10+ agent capabilities
│   ├── portfolio/             # Portfolio management & risk computation
│   ├── notify/                # Notification engine (Telegram/in-app)
│   ├── schedule/              # robfig/cron scheduler
│   ├── storage/               # SQLite WAL + migration framework
│   │   └── migrations/        # 12+ migrations up to 012
│   ├── python/                # gRPC bridge to Python
│   ├── config/                # YAML configuration
│   └── logging/               # slog wrapper
├── frontend/                  # Vue 3 frontend
│   └── src/
│       ├── terminal/          # Terminal Mode components
│       │   ├── panels/        # 64 Bloomberg-style panels
│       │   └── DockView/      # Docking panel system
│       ├── workflow/          # Workflow Mode components
│       │   └── canvas/        # vue-flow canvas
│       ├── stores/            # 7 Pinia stores
│       └── lib/               # i18n, theme, formatting utilities
├── python/                    # Python gRPC Sidecar
│   ├── src/
│   │   ├── factor/            # 25 Alpha factors
│   │   ├── ml/                # ML models (Qlib/PyTorch)
│   │   ├── llm/               # LLM inference (4 providers)
│   │   │   └── providers/     # OpenAI/Anthropic/DeepSeek/Ollama
│   │   ├── skills/            # 15 AI skills
│   │   └── data/              # Data fetching scripts
│   ├── proto/                 # gRPC service definitions
│   └── tests/                 # Python tests
├── resources/                 # Icons, templates, agent profiles
├── docs/
│   └── specs/                 # Spec documents (one per change)
├── examples/                  # Workflow examples
├── CHANGELOG.md               # Changelog
├── CLAUDE.md                  # Claude Code guidance
└── LICENSE                    # AGPL-3.0
```

---

## Market Coverage

| Market | Settlement | Key Rules | Data Sources | Brokers |
|--------|-----------|-----------|-------------|---------|
| CN (A-Share) | T+1 | Price limits ±10%/±20%, stamp duty 0.05% | Mootdx, MAC Protocol, Sina, EastMoney, Tencent, Baidu, AKShare, TuShare | Futu (stub) |
| HK | T+2 | Stock Connect, T+2 settlement | Futu, Sina | Futu (stub), IBKR |
| US | T+2 | PDT rule, wash sale | Yahoo, Polygon, Alpaca | Alpaca, IBKR, Tradier |
| Crypto | Instant | Perpetual funding rate, liquidation | Binance, OKX, CoinGecko | Binance (live), OKX, Bybit |

---

## License

[AGPL-3.0](LICENSE) © 2024-2026 QuantFlow Contributors
