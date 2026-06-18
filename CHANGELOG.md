# Changelog

All notable changes to QuantFlow Terminal will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [2026.6.18] - 2026-06-18

### Added
- [Frontend] researchStore: Pinia store for Research & Sentiment module with Wails bridge (GetSentiment, GetStockResearch, GetSentimentHistory) and frontend-mock fallback
- [Frontend] Added 5 research panel components: FinancialsPanel (income/balance/ratios card layout), PeerComparisonPanel (peer metrics comparison table), AnalystEstimatesPanel (analyst ratings table with consensus badge), InsiderTradingPanel (insider trades table with net activity indicator), CongressTradingPanel (congress trades table with party/chamber filters)
- [Python] SentimentService gRPC implementation — AnalyzeSentiment (single text via NLPPipeline with fallback to neutral on empty input) and BatchAnalyzeSentiment (concurrent fan-out across symbols). Errors returned in response, not raised as gRPC exceptions, matching the factor/engine.py pattern.
- [Docs] Created proposal implementation status map (`docs/specs/2026-06-18-proposal-implementation-status.md`) — annotated every module in NEW_PROJECT_PROPOSAL.md with ✅/🔶/📋 markers
- [Docs] Created 7 pending-development specs covering all unbuilt and partially-built modules: Research & Sentiment, Alternative Data, Missing Frontend Panels, Missing Workflow Nodes, Brokers & Trading, AI/MCP/Skills, and Misc enhancements — each with motivation, component list, acceptance criteria, and effort estimate
- [MarketData] Mootdx adapter: real TDX (通达信) TCP protocol via Python sidecar gRPC. Supports 1D/1W/1M + 1m/5m/15m/30m/1H K-line, 分时图, paginated fetching (800 bars/chunk). Best-IP auto-config from astockpursue ref. No registration, no API key, no IP-blocking risk — the best free A-share data source.
- [Python] DataService.FetchData: implemented mootdx support (ohlcv + quote data types) with mootdx library
- [Go] DataClient: gRPC client wrapper for Python DataService with timeout/retry
- [Python] Declared mootdx as an optional dependency (`pip install -e ".[data]"` or listed in requirements.txt) — the core sidecar still runs without it; the Go MootdxAdapter degrades to IsAvailable()==false when mootdx is absent.
- [Python] Added test_data_fetcher.py (6 tests): asserts the DataService forwards the requested K-line interval through to mootdx's `frequency=` arg (1W→week, 5m→5m, …), defaults to daily when the caller omits interval, errors clearly on unsupported intervals, caches the mootdx Quotes client across fetches, and rebuilds the client after a failure. Guards the silent-wrong-period regression and the per-fetch-setup() perf regression.
- [MarketData] Wired AdapterRegistry into App.startup: all 12 adapters registered (mootdx, sina, tushare, eastmoney, tencent, baidu, akshare, yfinance, polygon, okx, binance, coingecko). The CN fallback chain (mootdx first) now takes effect at runtime — previously the chain existed but `Get("mootdx")` returned nil so nothing was tried.
- [MarketData] Exposed Wails IPC methods `App.GetQuote(market, symbol)` and `App.FetchOHLCV(market, symbol, interval, start, end)` — the frontend/dataStore can now pull real quotes and K-line via the fallback chain (previously `ai/capabilities/quote.go` shipped placeholder data and no market IPC existed).
- [Go] Added app_test.go: verifies all 12 adapters register, mootdx reports IsAvailable()==false without a Python bridge (graceful degradation), and GetQuote/FetchOHLCV error cleanly when the registry is uninitialized.

### Changed
- [MarketData] MootdxAdapter.IsAvailable is now a cheap nil-check on the DataClient (no TDX round-trip). Previously it probed a live quote for `600519`, which doubled the TDX TCP connections per CN quote (the registry calls IsAvailable, then FetchQuote). The real liveness signal is FetchQuote itself; on failure the fallback chain moves on.
- [Python] The mootdx `Quotes` client is now cached at module level (double-checked locking via `threading.Lock`) instead of being rebuilt on every fetch — `mootdx_config.setup()` + `Quotes.factory()` (the expensive TDX-server probe) now run once. A broken client is reset on `bars()`/`minute()` failure and rebuilt on the next call.

### Fixed
- [MarketData] Mootdx OHLCV interval forwarding: `DataService._handle_mootdx` previously hardcoded `"1D"`, ignoring the `interval` passed by the Go adapter via `request.params` — so 1W/1m/5m/15m/30m/1H requests silently returned daily bars. Now reads `params["interval"]` (default `1D`) and routes it through `_FREQ_MAP`. Financial-correctness fix: wrong-period bars no longer flow into factor/P&L/backtest code.
- [MarketData] EastMoney FetchOHLCV: fixed wrong URL (now uses push2his.eastmoney.com K-line API) and discarded HTTP response — previously always returned a single fake bar
- [MarketData] Tencent adapter: added real K-line API via web.ifzq.gtimg.cn (supports A-share + HK, up to 2000 bars) — was incorrectly returning "not supported" error
- [MarketData] Baidu adapter: fixed broken quote parser + added real K-line API via finance.pae.baidu.com (daily K-line with MA5/MA10/MA20) — quote parser previously always returned error, K-line was reported as "not supported"
- [MarketData] Deleted old mootdx adapter that was wrapping Sina HTTP API (not real TDX)
- [MarketData] Sina/Tencent/AKShare/Baidu FetchOHLCV: now return honest errors for unsupported intervals instead of silently fabricating single-bar fake data
- [Workflow] Engine now passes node params to Execute instead of nil — user-configured parameters (e.g. sma period=5) now flow to nodes with upstream edges correctly
- [Workflow] Unified three incompatible OHLCVBar types (nodes/market/trading) — data_loader now outputs market.OHLCVBar, backtest converts to trading.OHLCVBar explicitly. The data_loader→backtest pipeline no longer returns "no OHLCV data"
- [App] Wired PythonBridge to workflow ML nodes (SetPythonBridge/SetModelRegistry) — train_model/predict/alpha_mining nodes no longer return "PythonBridge not set"
- [MarketData] Added NormalizeInterval() to normalize OHLCV interval case at registry entry — frontend lowercase "1d" now works for all A-share K-line adapters (Tencent/Baidu previously only accepted uppercase "1D")
- [MarketData] TuShare adapter now correctly parses data.fields+data.items parallel-array format — previously read a top-level items field the API never fills, causing empty results without error
- [Python] Added missing numpy import in ML engine.py — RLTrain used np.float32 but numpy was never imported, causing silent NameError swallowed by except Exception
- [Python] Factor engine now preserves NaN instead of converting to 0.0 — prevents look-ahead-like data contamination where warmup-period NaN became "zero momentum/volatility" in z-score/ML features
- [Storage] Migration execution now wrapped in transactions — SQL + version INSERT are atomic, preventing half-applied schema on process kill
- [Backtest] PnL now correctly deducts trading costs (commission + slippage + stamp duty) — was using gross price, systematically overstating win rate and Sharpe
- [Trading] OMS FillOrder: added sell validation (prevent negative positions), realized P&L tracking, and AvgPrice reset when position goes flat
- [Trading] Risk pipeline CheckStopLoss/CheckTakeProfit now handle short positions — was returning false for qty<0, leaving shorts with no risk protection
- [Trading] Market order risk check uses position market price instead of order.Price (always 0 for market orders) — 25% max position limit was silently bypassed
- [Trading] Paper engine take profit now has short-position branch — shorts previously had no take-profit, only stop-loss
- [App] GetPortfolioSummary now derives cash from actual trade history instead of hardcoded 100000
- [AI] Quote lookup capability now wired to AdapterRegistry for real market data — was returning $100 placeholder for every symbol
- [App] Chat method now uses 5-minute timeout context — was context.Background() with no deadline, could hang forever
- [Frontend] DockView now calls unwatch() in onUnmounted to prevent Pinia subscription memory leak
- [App] Added Shutdown() method for graceful cleanup (Python bridge close)
- [MarketData] Hub unsubscribe now properly deletes subscriber from map (channels left for GC to avoid close/send race)
- [MarketData] MarketForSymbol now recognizes .BJ suffix (Beijing Stock Exchange) → CN, and crypto detection loop cleaned up
- [Python] isTransient() now uses canonical gRPC status.Code instead of fragile string matching
- [Python] DataClient retry: added jitter and context cancellation check during backoff sleep
- [Config] ResolveDBPath() makes relative DB path work consistently across wails dev/build/.app bundles
- [Storage] SQLite DSN now includes _busy_timeout=5000ms to prevent immediate SQLITE_BUSY on WAL write conflicts
- [Frontend] PropertyPanel preserves numeric/boolean types when editing workflow params — was always converting to string
- [Workflow] CancelOrder node now propagates OMS error instead of silently reporting success
- [MarketData] Added guard tests (10) ensuring unsupported adapters never silently return fake OHLCV data
- [MarketData] EastMoney: post-filter OHLCV bars to end date (prevents look-ahead bias in backtesting); volume now ×100 to normalize 手→股 with other CN adapters
- [MarketData] OKX FetchOHLCV: now passes start/end as after/before query params (was always returning most recent 100 bars regardless of requested range)
- [App] DB connection now shared: opened once at startup with migrations run once; LoadWorkflow/SaveWorkflow/ListWorkflow reuse the connection (was open→migrate→close per call)
- [Python] ComputeFactorBatch: pre-decode OHLCV data once + parallel factor computation via asyncio.gather (was sequential O(N²) with redundant Arrow parsing)
- [Python] evaluator.py: replaced eval() with AST whitelist parser (arithmetic + comparison + 6 safe functions) — eliminates RCE vector through numpy/pandas object-model escape from the eval sandbox
- [Frontend] CandlestickPanel: fixed OHLCV index order in ECharts candlestick series — mock data stores [date,open,close,low,high,volume] but mapping passed [d[1],d[4],d[3],d[2]] (open,high,low,close) instead of [d[1],d[2],d[3],d[4]] (open,close,low,high), causing every candle body/wick to render incorrectly
- [Frontend] PortfolioSummary: positions table now passes `pos.symbol` to `fmtMoney()` — A股/BTCUSDT rows were always showing `$` instead of ¥/USDT respectively

### Added

#### Phase 11A — Frontend Test Coverage
- [Frontend] 8 Pinia store test suites: data (5 tests), settings (5), session (5), terminal (5), workflow (8), notify (3), portfolio (2), ml (5) — total 38 store tests
- [Frontend] 22 terminal panel smoke tests: Watchlist, QuoteDetail, Candlestick, OrderEntry, Position, News, AIChat, SystemMonitor, BacktestResult, FactorAnalysis, PortfolioSummary, PositionDetail, RiskDashboard, TradeHistory, Schedule, Notify, BrokerConfig, Settings, ModelRegistry, PredictionDashboard, AlphaMiningWorkspace, RLMonitor
- [Frontend] 8 workflow + terminal shell component tests: NodePalette, PropertyPanel, ExecutionLog, CustomNode, WorkflowCanvas, CommandBar, StatusBar, PushPinBar
- [Frontend] 4 DockView component tests: DockView, DockContainer, DockSplitter, DockTab
- [Frontend] Total: 76 tests across 42 test files, all passing

#### Phase 11C — Go Deep Test Coverage
- [Go] 13 market adapter test files: Yahoo, EastMoney, Sina, Tencent, Binance, OKX, CoinGecko, TuShare, AKShare, Mootdx, Baidu, Polygon — each testing Name/Markets/RequiresAuth + helpers
- [Go] 3 AI capability test files: factor, quote, skills — verifying registration with nil bridge
- [Go] storage package: expanded from 1→5 tests (create, reopen, migrations, WAL mode)
- [Go] config package: expanded from 2→5 tests (defaults, file override, missing file)
- [Go] schedule package: expanded from 2→5 tests (add job, remove job, list jobs)
- [Go] notify package: expanded from 2→5 tests (add, list limit/offset, mark read)
- [Go] Total: 251 tests across 20+ packages, all passing (+75 from Phase 10)

#### Phase 11B — Python Test Coverage
- [Python] Volatility factor tests: 5 tests (ATR, vol_20d, vol_60d, bollinger_width)
- [Python] Volume factor tests: 4 tests (volume_ratio, OBV, VWAP deviation)
- [Python] Cross-sectional factor tests: 13 tests (zscore_momentum, rank_momentum, zscore_volatility, zscore_volume, size_factor)
- [Python] LLM provider tests: 16 tests (OpenAI, DeepSeek, Ollama, Anthropic instantiation)
- [Python] Total: 120 test functions across 16 test files, all passing (+38 from Phase 10)

#### Phase 10.2 — Alpha Mining Engine
- [Python] AlphaMiningEngine: genetic programming factor discovery via gplearn (optional, graceful degradation)
- [Python] Alpha mining test suite: skipif gplearn not installed, verifies formula validity and graceful degradation

#### Phase 10.3 — RL Trading Engine
- [Python] TradingEnv: Gymnasium trading environment with discrete/continuous action spaces, OHLCV state representation, portfolio simulation
- [Python] PPOTrainer: Proximal Policy Optimization with actor-critic network, clipped surrogate objective, 4-epoch updates
- [Python] DQNTrainer: Deep Q-Network with epsilon-greedy exploration, experience replay buffer, target network
- [Python] SACTrainer: Soft Actor-Critic stub (full implementation coming in Phase 10.3.1)
- [Python] ReplayBuffer: fixed-capacity deque-based replay buffer for off-policy RL
- [Python] RLTrain: server-streaming gRPC that decodes Arrow OHLCV, creates env + trainer, yields RLTrainUpdate per episode
- [Python] RLPredict: gRPC endpoint returning safe default hold action (full model loading coming later)
- [Python] RL engine test suite: 14 test cases covering env, PPO, DQN, and ReplayBuffer
- [Workflow] RLEnvNode: configures RL trading environment (window size, action type, initial cash)
- [Workflow] RLTrainNode: prepares RL training configuration (algorithm, total episodes, learning rate)
- [Workflow] RLPredictNode: RL inference node (model_id + observation → action + action_value)
- [Frontend] RLMonitorPanel: real-time RL training dashboard with algorithm selector, episode counter, reward/sharpe charts, start/pause/save controls
- [Frontend] mlStore extended: rlTrainingEpisodes, rlTrainingRunning, rlAlgorithm state and actions

#### Phase 10.4 — Risk Modeling
- [Python] GARCHEngine: GARCH/GJR-GARCH/EGARCH volatility models via arch package (optional, graceful degradation)
- [Python] CovarianceEngine: Ledoit-Wolf shrinkage and sample covariance estimation via scikit-learn
- [Python] RiskModel: gRPC endpoint routing to GARCH or covariance engines, returns Arrow-encoded results
- [Python] Risk engine test suite: 10 test cases covering GARCH variants, covariance methods, and missing dependency handling
- [Workflow] RiskModelNode: computes volatility/covariance_matrix/model_metrics from returns data with model type selection
- [Frontend] RiskDashboard extended: GARCH volatility chart with model selector (GARCH/GJR-GARCH/EGARCH), AIC/BIC metrics display
- [Frontend] mlStore extended: riskModelResult state and setRiskModelResult action

#### Phase 10.1 — Revenue Prediction Engine
- [Proto] Extended ml.proto: 7 RPCs (Train/Predict/Evaluate/AlphaMining/RLTrain/RLPredict/RiskModel) + 16 message types
- [Python] TreeEngine: XGBoost/LightGBM training, prediction, evaluation, feature importance
- [Python] DeepEngine: LSTM/Transformer time-series prediction (torch optional, graceful degradation)
- [Python] Model serialization module: joblib + torch.save dual-track
- [Python] MLService gRPC entry point with Train/Predict/Evaluate routing to sub-engines
- [Engine] ML domain types: MLModel, ModelType, ModelCategory, ModelStatus, ModelFilter, TrainingJob
- [Engine] ModelRegistry: CRUD + state machine (training→ready/failed, ready→archived) + dual-track storage
- [Engine] Evaluator: ComputeIC, ComputeIR, ComputeSharpe utility functions
- [Workflow] FeatureEngineerNode: standardize/minmax normalization, NA fill, lag alignment, anti look-ahead bias
- [Workflow] TrainModelNode: gRPC training via Python sidecar, model registration in SQLite
- [Workflow] PredictNode: model inference via Python sidecar
- [Workflow] EvaluateModelNode: MSE/MAE/RMSE/IC computation in Go
- [Storage] Migration 010: ml_models, ml_predictions, ml_evaluations tables with FK cascade
- [Go] MLClient: gRPC client for all 7 ML RPCs with timeout/retry logic
- [Frontend] mlStore: Pinia store for ML state (models, predictions, training jobs)
- [Frontend] ModelRegistry panel: model CRUD, filtering, search, detail view, workflow integration
- [Frontend] PredictionDashboard panel: prediction distribution, IC timeline, scatter, quantile views
- [Python] Updated pyproject.toml with ML dependencies (xgboost, lightgbm, scikit-learn, torch, joblib, gplearn, gymnasium, arch)
- [Docs] Phase 10 design spec and 10.1 implementation plan
- [Tests] 14 Go tests (ml package), 13 Python tests, 13 workflow node tests, 1 integration test

## [2026.6.17] - 2026-06-17

### Added

#### Phase 1 — Engine-First
- [Engine] Go module initialization with module path `quantflow`, config/logging/Makefile
- [Workflow] BaseNode interface, NodeRegistry, 5 built-in nodes, DAG types, TopoSort, Engine
- [Storage] SQLite WAL + embedded migration framework + WorkflowRepo
- [Frontend] qf CLI tool + example workflows + sample data
- [Docs] Phase 1 design doc, implementation plan, benchmarks, integration test

#### Phase 2 — Frontend + Trading Engine
- [Frontend] Wails v3 desktop shell integration with embedded Vue 3 frontend
- [Frontend] Terminal Mode: CommandBar (Ctrl+K), DockView recursive docking system, 8 panels
- [Frontend] Workflow Mode: vue-flow canvas with CustomNode, NodePalette, PropertyPanel, ExecutionLog
- [Frontend] Pinia stores: terminal, workflow, data, session with undo/redo and serialization
- [Frontend] Terminal↔Workflow bidirectional flow: panel→node, node→"Pin to Terminal"
- [Engine] Trading Engine: OMS, PaperEngine, OrderMatcher, RiskPipeline, bar-by-bar pipeline
- [Engine] MarketDataHub: Go channel pub/sub, L0 TTL cache, 3 data adapters (Yahoo/EastMoney/Binance)
- [Storage] Migration 004_trading: orders, trades, positions tables
- [Storage] Migration 005_ohlcv_cache: OHLCV data cache table
- [Docs] Phase 2 design spec and implementation plan

#### Phase 2.5 — Data Source Hardening
- [MarketData] 14 real-world data adapters with HTTP (Yahoo, EastMoney, Binance, Sina, Tencent, OKX, CoinGecko, TuShare, Baidu, Polygon + Mootdx/AKShare stubs)
- [MarketData] AdapterRegistry with FallbackChain: 7-source A-share (mootdx→tushare→...→akshare), 3-source crypto (binance→okx→coingecko)
- [MarketData] RetryWithBudget + TransientError + CheckBudget for resilient data fetching
- [MarketData] MarketForSymbol: automatic market detection from symbol suffix/prefix
- [MarketData] Adapter interface enhanced: IsAvailable, RequiresAuth, HealthCheck

#### Phase 3 — Python gRPC Sidecar + Factor Engine + Backtesting
- [Python] Python gRPC sidecar project: pyproject.toml, proto definitions (factor/health/ml/data), gRPC server with HealthService
- [Python] Factor engine: 25 alpha factors across 5 categories (momentum:5, trend:5, volatility:5, volume:5, cross_sectional:5)
- [Python] Arrow IPC zero-copy DataFrame transfer for factor computation
- [Python] 19 tests: 13 unit + 6 gRPC integration
- [Python] Go PythonBridge: gRPC client with health check, factor compute (single/batch), retry logic
- [Engine] Backtesting engine: bar-by-bar Runner, PerformanceMetrics (CAGR/Sharpe/Sortino/MaxDD/Calmar/WinRate/ProfitFactor)
- [Engine] A-share engine (CNEngine): T+1 settlement, ±10% price limits, 0.05% stamp duty on sell, 100-share lots
- [Engine] US engine (USEngine): simpler rules, fractional shares, PDT tracking
- [Engine] 7 backtest tests: metrics, SMA cross strategy, T+1 enforcement, trading days
- [Workflow] FactorNode: alpha factor computation node (calls Python sidecar)
- [Workflow] StrategyNode: strategy configuration node (sma_cross, rsi_threshold, momentum, custom)
- [Workflow] BacktestNode: historical backtesting node (CN/US/HK/CRYPTO markets)
- [Workflow] 8 new node tests: FactorNode, StrategyNode, BacktestNode, registration
- [Frontend] BacktestResultPanel: equity curve (ECharts), drawdown chart, metrics grid, trade list
- [Frontend] FactorAnalysisPanel: 25-factor catalog with search, category filtering, parameter display
- [Docs] Phase 3 design spec and implementation plan

#### Phase 4 — AI Agent System
- [AI] Go AgentOrchestrator: ReAct loop (think->act->observe), CapabilityRegistry with 10 built-in capabilities
- [AI] EventEmitter for real-time SSE agent step events to frontend
- [AI] AgentProfile manager: 4 YAML-based profiles (general, quant_analyst, trader, research_assistant)
- [Frontend] TradeHistory panel: trades/orders tables with symbol search, status filter, and CSV export
- [AI] AgentNode: workflow-integrated AI node with typed input/output ports
- [AI] Capabilities: quote_lookup, search_symbol, list_factors, compute_factor, search_skills
- [Python] LLM Service: gRPC streaming Chat with 4 providers (OpenAI, Anthropic, DeepSeek, Ollama)
- [Python] PromptTemplate engine with token budget management and skill injection
- [Python] Skill Knowledge Base: 15 Markdown skills across 5 categories with frontmatter loader
- [Frontend] AIChatPanel upgrade: SSE streaming, Markdown rendering with syntax highlighting, tool call visualization, profile/model selectors
- [Docs] Phase 4 design spec and implementation plan
- [Python] 13 tests: LLM service, PromptTemplate, providers, Skill KB
- [Go] 24 tests: CapabilityRegistry, EventEmitter, ProfileManager, AgentNode, LLM client, AgentNode tests

#### Phase 5 — Broker Integration + Portfolio & Risk + Notification + Scheduler
- [Trading] Broker interface with OMS routing (paper/live mode), BinanceBroker with REST API (spot orders, account, positions), FutuBroker stub
- [Trading] 4 new workflow nodes: PlaceOrder, CancelOrder, PositionQuery, OrderQuery
- [Notify] NotificationMgr with multi-channel broadcast, TelegramNotifier (MarkdownV2), InAppNotifier
- [Notify] 2 new workflow nodes: Notify, Alert
- [Schedule] robfig/cron-based scheduler with workflow triggering, timeout/overlap protection
- [Schedule] 2 new workflow nodes: Schedule, Wait
- [Portfolio] PortfolioService: summary, positions, allocation, daily P&L snapshots
- [Portfolio] RiskMetrics computation: VaR(historical), CVaR, MaxDrawdown, Sharpe, Sortino, Calmar
- [Portfolio] 3 new workflow nodes: PortfolioSummary, RiskMetrics, Allocation
- [Storage] Migrations 006-009: broker_config, notifications, schedule_tasks, daily_pnl, position_snapshots

#### Phase 6 — Frontend Panels + SSE + Pinia Stores
- [Frontend] 7 new panels: PortfolioSummary, PositionDetail, RiskDashboard, TradeHistory, SchedulePanel, NotifyPanel, BrokerConfig
- [Frontend] portfolioStore and notifyStore (Pinia) with auto-refresh
- [Frontend] Enhanced OrderEntryPanel with broker selector (paper/binance/futu)
- [Frontend] CSV export utility for trade data
- [Frontend] ECharts integration: equity curve, allocation pie, drawdown chart, price history

#### Phase 7 — Theme System + i18n + Settings
- [Frontend] CSS Variables theme system: dark/light dual themes + 3 density levels (compact/default/comfortable)
- [Frontend] vue-i18n integration with Chinese and English translations (~80 keys each)
- [Frontend] themeStore and settingsStore (Pinia) with localStorage persistence
- [Frontend] SettingsPanel: 9 configuration sections (appearance, language, notifications, data, trading, display, shortcuts, storage, about)
- [Frontend] 5 core panels migrated to CSS Variables for automatic theme switching

#### Phase 8 — Node Expansion (20 → 34)
- [Workflow] 4 indicator nodes: MACD, RSI, BollingerBands, EMA
- [Workflow] 3 data nodes: Merge, Filter, Resample
- [Workflow] 2 signal nodes: ThresholdSignal, SignalCombine
- [Workflow] 2 risk nodes: StopLoss, PositionSizer
- [Workflow] 3 utility nodes: HTTPRequest, MathOperation, JSONParse
- [Workflow] NodeRegistry total: 34 registered node types

#### Phase 9 — Factor Atoms + Signal Engineering (34 → 54)
- [Workflow] 12 factor atom nodes: pct_change, delta, std_dev, rank, scale, cross_over, compare, bool_combine, rolling_maxmin, rolling_zscore, arithmetic, if_else
- [Workflow] 5 signal engineering nodes: rank_select, hold_signal, rebalance, entry_signal, exit_signal
- [Workflow] 3 control/output nodes: if_condition, sub_workflow, chart_data
- [Workflow] NodeRegistry total: 54 registered node types

### Changed
- [Engine] Go module restructured from `app/` to project root (Wails v3 standard layout)
- [Docs] Phase 1 restructured from Proposal's 4-parallel-track to Engine-First serial milestones
- [Frontend] Migrated 5 core panels (PortfolioSummary, PositionDetail, RiskDashboard, TradeHistory, OrderEntryPanel) from hardcoded colors to CSS custom properties for theme support

---

## Template (copy for each release)

```markdown
## [YYYY.M.D] - YYYY-MM-DD

### Added
- [Scope] New feature description and why

### Changed
- [Scope] Description of what changed and why

### Fixed
- [Scope] Bug description and root cause

### Removed
- [Scope] What was removed and why
```

**Scopes**: `[Terminal]` `[Workflow]` `[Engine]` `[Broker]` `[MarketData]` `[AI]` `[Frontend]` `[Storage]` `[Python]` `[Docs]`
