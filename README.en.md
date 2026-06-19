<p align="center">
  <img src="https://img.shields.io/badge/QuantFlow-Terminal-black?style=for-the-badge" alt="QuantFlow">
</p>

<h1 align="center">QuantFlow Terminal</h1>

<p align="center">
  <strong>Dual-Mode Quantitative Finance Terminal — Bloomberg-Style Panels + Visual Workflow Orchestration</strong>
</p>

<p align="center">
  <a href="#-project-status"><img src="https://img.shields.io/badge/phase-9%20complete-success?style=flat-square" alt="Phase 9"></a>
  <a href="#-project-status"><img src="https://img.shields.io/badge/nodes-54-blue?style=flat-square" alt="54 Nodes"></a>
  <a href="#-project-status"><img src="https://img.shields.io/badge/panels-18-blue?style=flat-square" alt="18 Panels"></a>
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

> **Phases 1–9 All Complete** — 54 workflow nodes · 18 frontend panels · 9 development phases

| Component | Status | Details |
|-----------|--------|---------|
| Workflow Engine (DAG + goroutine parallel) | ✅ | 54 node types, Kahn topological sort, breakpoint recovery |
| Desktop Shell (Wails v3 + Vue 3 + TypeScript) | ✅ | Dual-mode UI: Terminal panels + Workflow canvas |
| Trading Engine (OMS + Paper/Live) | ✅ | Order management, order matching, risk pipeline |
| MarketDataHub (Go channel pub/sub) | ✅ | 14 data adapters, FallbackChain resilience |
| Backtesting Engine (CN/US/HK/CRYPTO) | ✅ | T+1, price limits, stamp duty, market-specific rules |
| Python gRPC Sidecar | ✅ | 25 Alpha factors, 15 AI skills, 4 LLM providers |
| AI Agent System (ReAct loop) | ✅ | Streaming SSE events, 4 AgentProfiles, 10+ capabilities |
| Broker Integration | ✅ | Binance REST API live + Futu stub |
| Portfolio & Risk Management | ✅ | VaR/CVaR/Sharpe/Sortino/MaxDD/Calmar |
| Notifications & Scheduler | ✅ | Telegram/in-app + robfig/cron scheduler |
| Theme System (dark/light + 3 densities) | ✅ | CSS Variables driven, localStorage persistence |
| Internationalization (zh/en) | ✅ | vue-i18n, ~80 translation keys per language |
| SQLite WAL Storage | ✅ | 9 migrations (001-009), single-file zero-config |
| Frontend Panels | ✅ | 18 Bloomberg-style panels |

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
│ Pinia (7 stores)             │     │ Trading Engine (OMS + Paper/Live)│
│ Monaco Editor                │◄─IPC►│ MarketDataHub (14 adapters)      │
│ Terminal Mode (18 panels)     │     │ AI Agent (ReAct + 4 LLM)         │
│ Workflow Mode (54 node types) │     │ Portfolio · Risk · Notification  │
│ Dark/Light Theme + i18n      │     │ Scheduler (robfig/cron)           │
└──────────────────────────────┘     │ SQLite WAL (9 migrations)         │
                                     │ gRPC ──► Python Sidecar           │
                                     │   (25 Factors / 15 AI Skills)     │
                                     └──────────────────────────────────┘
```

---

## Core Features

### Workflow Nodes (54 nodes, 14 categories)

| Category | Nodes | Count |
|----------|-------|-------|
| **Data** | DataLoader, Merge, Filter, Resample | 4 |
| **Indicator** | SMA, MACD, RSI, EMA, BollingerBands | 5 |
| **Alpha Factor** | pct_change, delta, std_dev, rank, scale, cross_over, compare, bool_combine, rolling_maxmin, rolling_zscore, arithmetic, if_else | 12 |
| **Signal** | CrossSignal, ThresholdSignal, SignalCombine, rank_select, hold_signal, rebalance, entry_signal, exit_signal | 8 |
| **Strategy** | StrategyNode (sma_cross, rsi_threshold, momentum, custom) | 1 |
| **Backtest** | BacktestNode (CN/US/HK/CRYPTO markets) | 1 |
| **Trading** | PlaceOrder, CancelOrder, PositionQuery, OrderQuery | 4 |
| **Portfolio** | PortfolioSummary, RiskMetrics, Allocation | 3 |
| **Risk** | StopLoss, PositionSizer | 2 |
| **Notify** | Notify, Alert | 2 |
| **Schedule** | Schedule, Wait | 2 |
| **Control** | Loop, if_condition, sub_workflow | 3 |
| **AI** | FactorNode, AgentNode | 2 |
| **Utility** | HTTPRequest, MathOperation, JSONParse, LogOutput, chart_data | 5 |

### Frontend Panels (18 panels)

| Panel | Category | Description |
|-------|----------|-------------|
| WatchlistPanel | Market | Watchlist with real-time quotes |
| QuoteDetailPanel | Market | Security detail with depth data |
| CandlestickPanel | Chart | Candlestick chart with ECharts |
| PortfolioSummary | Portfolio | Holdings summary, P&L analysis |
| PositionDetail | Portfolio | Position details, cost analysis |
| RiskDashboard | Risk | VaR/CVaR/Sharpe ratio dashboard |
| TradeHistory | Trading | Trade/order history, CSV export |
| OrderEntryPanel | Trading | Order entry with broker selection |
| BrokerConfig | Trading | Broker configuration, API key management |
| BacktestResultPanel | Backtest | Equity curve, drawdown chart, metrics grid |
| FactorAnalysisPanel | Research | 25-factor catalog with search and filter |
| AIChatPanel | AI | SSE streaming chat, tool call visualization |
| SchedulePanel | Schedule | Scheduled task management |
| NotifyPanel | Notifications | Notification history, channel management |
| SettingsPanel | Settings | 9 config sections: theme, language, data, display, etc. |
| NewsPanel | News | News summaries |
| PositionPanel | Positions | Position overview |
| SystemMonitorPanel | System | System resource monitoring |

### Broker Support

| Broker | Market | Status | Details |
|--------|--------|--------|---------|
| Binance | Crypto | ✅ Live | REST API spot orders, account, positions |
| Futu (富途) | CN/HK/US | 🔧 Stub | Interface defined, awaiting live integration |

### AI Agent System

- **ReAct Loop**: think → act → observe, with timeout and max-step limits
- **4 LLM Providers**: OpenAI, Anthropic, DeepSeek, Ollama (local deployment)
- **15 AI Skills**: Across 5 categories (technical analysis, fundamental analysis, risk management, trading strategies, market microstructure)
- **4 AgentProfiles**: general, quant_analyst, trader, research_assistant
- **Streaming Output**: SSE events → frontend Markdown rendering + tool call visualization

### Market Data Hub

14 data source adapters, automatic market detection, FallbackChain resilience:

| Market | Adapters | Fallback Chain |
|--------|----------|----------------|
| CN (A-Share) | Mootdx, TuShare, AKShare, EastMoney, Sina, Tencent, Baidu | 7 sources |
| US | Yahoo, Polygon | 2 sources |
| Crypto | Binance, OKX, CoinGecko | 3 sources |

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

# Go backend tests
go test ./internal/... -v -count=1

# Frontend tests
cd frontend && npx vitest run

# Python sidecar tests
cd python && python -m pytest tests/ -x -q

# Full check before commit
go vet ./... && go test ./...
cd frontend && npx vue-tsc --noEmit && npx vitest run
cd python && python -m pytest tests/ -x -q
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

---

## Directory Structure

```
quantflow/
├── main.go                    # Wails application entry point
├── app.go                     # Exported Go functions (frontend bindings)
├── go.mod / go.sum            # Go module definition
├── internal/
│   ├── workflow/              # Workflow engine (node, dag, engine)
│   │   └── nodes/             # 54 node implementations
│   ├── trading/               # Trading engine (OMS + matching)
│   │   └── brokers/           # Binance/Futu broker adapters
│   ├── market/                # Market data hub
│   │   └── adapters/          # 14 data source adapters
│   ├── backtest/              # Backtesting engine (CN/US/HK/CRYPTO)
│   ├── ai/                    # AI Agent system
│   │   └── capabilities/      # 10+ agent capabilities
│   ├── portfolio/             # Portfolio management & risk computation
│   ├── notify/                # Notification engine (Telegram/in-app)
│   ├── schedule/              # robfig/cron scheduler
│   ├── storage/               # SQLite WAL + migration framework
│   │   └── migrations/        # 9 migrations up to 009
│   ├── python/                # gRPC bridge to Python
│   ├── config/                # YAML configuration
│   └── logging/               # slog wrapper
├── frontend/                  # Vue 3 frontend
│   └── src/
│       ├── terminal/          # Terminal Mode components
│       │   ├── panels/        # 18 Bloomberg-style panels
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
| CN (A-Share) | T+1 | Price limits ±10%/±20%, stamp duty 0.05% | EastMoney, AKShare, TuShare, Mootdx, Sina, Tencent, Baidu | Futu (stub) |
| HK | T+2 | Stock Connect, T+2 settlement | Futu, Sina | Futu (stub), IBKR |
| US | T+2 | PDT rule, wash sale | Yahoo, Polygon, Alpaca | Alpaca, IBKR, Tradier |
| Crypto | Instant | Perpetual funding rate, liquidation | Binance, OKX, CoinGecko | Binance (live), OKX, Bybit |

---

## License

[AGPL-3.0](LICENSE) © 2024-2026 QuantFlow Contributors
