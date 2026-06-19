# Phase 3: Python gRPC Sidecar + Factor Engine + Backtesting

> **Status**: Draft  
> **Date**: 2026-06-17  
> **Depends on**: Phase 1 (Workflow Engine), Phase 2 (Frontend + Trading Engine), Phase 2.5 (Data Source Hardening)

---

## Motivation

Phase 2 delivered a working desktop terminal with real market data and a trading engine. However, QuantFlow's core value proposition — quantitative research — requires Python's ecosystem (pandas, numpy, statsmodels, qlib). The proposal's Phase 3 calls for a Python gRPC sidecar that bridges Go's orchestration with Python's computation.

### Why This Phase Is Critical

1. **Factor computation needs pandas/numpy** — 450+ alpha factors in the Python ecosystem cannot be rewritten in Go. The gRPC bridge is the foundation for all ML/quant features.
2. **Backtesting is the killer feature** — Without backtesting, QuantFlow is just a data viewer + paper trader. Users need: strategy → backtest → metrics → iterate.
3. **Go↔Python boundary design is architecturally load-bearing** — Get this wrong and every future feature (ML, Qlib, LLM) pays the cost. Get it right once.

### Current State vs Target

| Capability | Current (Phase 2.5) | Target (Phase 3) |
|---|---|---|
| Python integration | None | gRPC sidecar with protobuf contracts |
| Factor computation | None | 50+ core factors, extensible architecture |
| Backtesting | PaperEngine only (live bar-by-bar) | Full historical backtesting with metrics |
| Strategy definition | Not formalized | StrategyNode with signal→risk→execute pipeline |
| Backtest visualization | None | EquityCurvePanel + PerformanceMetricsPanel |
| Data passing to Python | N/A | Arrow Flight for zero-copy DataFrame transfer |

---

## Design

### 1. System Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Go Backend (QuantFlow)                     │
│                                                               │
│  ┌──────────────────┐  ┌──────────────────────────────────┐  │
│  │ Workflow Engine   │  │ Backtesting Engine                │  │
│  │ (Kahn + goroutine)│  │ · HistoricalBarFeed               │  │
│  │                    │  │ · BacktestRunner                  │  │
│  │ [FactorNode]──────┼──│ · PerformanceCalculator            │  │
│  │ [StrategyNode]────┼──│ · MultiMarketEngine (CN/US first)  │  │
│  │ [BacktestNode]────┼──│                                    │  │
│  └────────┬───────────┘  └────────────┬─────────────────────┘  │
│           │                             │                        │
│  ┌────────┴─────────────────────────────┴─────────────────────┐ │
│  │              PythonBridge (gRPC Client)                     │ │
│  │  · FactorClient   · MLClient   · LLMClient (stub)          │ │
│  │  · Connection pool   · Health check   · Retry/Timeout       │ │
│  └──────────────────────────┬─────────────────────────────────┘ │
│                             │ gRPC (localhost:50051)             │
│                             │ Arrow Flight for DataFrames       │
└─────────────────────────────┼───────────────────────────────────┘
                              │
┌─────────────────────────────┼───────────────────────────────────┐
│              Python gRPC Sidecar                                │
│                                                                  │
│  ┌──────────────────────────┴──────────────────────────────┐   │
│  │              gRPC Server (grpcio + asyncio)              │   │
│  │  · FactorService     · MLService     · LLMService (stub) │   │
│  │  · HealthService     · DataService                       │   │
│  └──────────┬──────────────┬──────────────┬─────────────────┘   │
│             │               │               │                    │
│  ┌──────────┴──────┐ ┌─────┴──────┐ ┌──────┴──────────────┐   │
│  │ Factor Engine    │ │ ML Engine  │ │ Data Fetch (future)  │   │
│  │ · Alpha factors  │ │ · Qlib     │ │ · AKShare            │   │
│  │ · Tech indicators│ │ · PyTorch  │ │ · TuShare            │   │
│  │ · Cross-sectional│ │ · RL       │ │                      │   │
│  └──────────────────┘ └────────────┘ └──────────────────────┘   │
└──────────────────────────────────────────────────────────────────┘
```

### 2. Protobuf Contracts

#### 2.1 Factor Service

```protobuf
service FactorService {
  // Compute a single factor for a universe of symbols
  rpc ComputeFactor(ComputeFactorRequest) returns (ComputeFactorResponse);
  
  // Compute multiple factors in batch
  rpc ComputeFactorBatch(ComputeFactorBatchRequest) returns (ComputeFactorBatchResponse);
  
  // List available factors with metadata
  rpc ListFactors(ListFactorsRequest) returns (ListFactorsResponse);
  
  // Stream factor values for real-time updates
  rpc StreamFactors(StreamFactorsRequest) returns (stream FactorUpdate);
}

message ComputeFactorRequest {
  string factor_name = 1;           // e.g., "momentum_20d", "rsi_14"
  repeated string symbols = 2;      // e.g., ["000001.SZ", "600519.SH"]
  string start_date = 3;            // "2024-01-01"
  string end_date = 4;              // "2024-12-31"
  map<string, string> params = 5;   // factor-specific parameters
  bytes ohlcv_data = 6;             // Arrow IPC bytes (zero-copy)
}

message FactorResult {
  string symbol = 1;
  repeated string dates = 2;
  repeated double values = 3;
}

message ComputeFactorResponse {
  string factor_name = 1;
  repeated FactorResult results = 2;
  int64 compute_time_ms = 3;
}
```

#### 2.2 ML Service (v1 Stub)

```protobuf
service MLService {
  rpc TrainModel(TrainModelRequest) returns (TrainModelResponse);
  rpc Predict(PredictRequest) returns (PredictResponse);
  rpc ListModels(ListModelsRequest) returns (ListModelsResponse);
}

message TrainModelRequest {
  string model_type = 1;            // "linear", "xgboost", "lstm"
  bytes feature_data = 2;           // Arrow IPC
  bytes target_data = 3;            // Arrow IPC
  map<string, string> hyperparams = 4;
}

message TrainModelResponse {
  string model_id = 1;
  bytes model_bytes = 2;            // Serialized model (pickle/onnx)
  map<string, double> metrics = 3;  // train metrics
  int64 train_time_ms = 4;
}
```

#### 2.3 Health & Data Services

```protobuf
service HealthService {
  rpc Ping(PingRequest) returns (PingResponse);
  rpc GetStatus(GetStatusRequest) returns (StatusResponse);
}

service DataService {
  // Fetch data from Python-side sources (AKShare, TuShare, etc.)
  // Used when Go-side adapters need Python-specific data
  rpc FetchData(FetchDataRequest) returns (FetchDataResponse);
}
```

### 3. Go↔Python Data Flow

```
Data Flow for Factor Computation:
───────────────────────────────────

1. Go Side:
   [Workflow: DataLoader] → fetches OHLCV via MarketDataHub
   [Workflow: FactorNode] → PythonBridge.ComputeFactor()
        ↓
        ├── Marshal OHLCV bars to Arrow RecordBatch (using Go Arrow lib)
        ├── Build ComputeFactorRequest protobuf
        ├── gRPC call → Python sidecar
        ├── Receive ComputeFactorResponse
        ├── Unmarshal factor values
        └── Return typed outputs to downstream workflow nodes

2. Python Side:
   FactorService.ComputeFactor()
        ├── Receive Arrow IPC bytes
        ├── pyarrow.ipc.open_stream() → pandas DataFrame
        ├── FactorEngine.compute(factor_name, df, params)
        ├── Return FactorResult as protobuf
        └── (No disk I/O, no database access)

Data Flow for Backtesting:
───────────────────────────

1. Go Side:
   [BacktestNode] receives strategy config + OHLCV data
        ↓
   BacktestRunner.Run(start, end, strategy, universe)
        ├── For each bar in date range:
        │   ├── Compute signals (Go or Python factors)
        │   ├── RiskPipeline.Check()
        │   ├── OrderMatcher.Match()
        │   └── Update positions + equity curve
        ├── PerformanceCalculator.Compute()
        │   ├── Total return, CAGR, MaxDD, Sharpe, Sortino
        │   ├── Win rate, profit factor, avg trade
        │   └── Monthly returns heatmap
        └── Return BacktestResult

2. Python Side (optional, for factor computation):
   Same as factor flow above — Python computes factors on demand
   during the backtest run.
```

### 4. Backtesting Engine Design

#### 4.1 Core Types

```go
type BacktestConfig struct {
    StartDate   time.Time
    EndDate     time.Time
    Commission  float64          // e.g., 0.0003 for A-shares
    Slippage    float64          // e.g., 0.001 (10 bps)
    InitialCash float64
    Benchmark   string           // e.g., "000300.SH" for CSI 300
}

type Strategy struct {
    ID          string
    Name        string
    SignalRules []SignalRule     // When to buy/sell
    RiskRules   RiskConfig       // Stop loss, take profit, position sizing
    RebalanceFreq string         // "daily" | "weekly" | "monthly"
}

type BacktestResult struct {
    Config      BacktestConfig
    EquityCurve []EquityPoint
    Trades      []Trade
    Metrics     PerformanceMetrics
}

type PerformanceMetrics struct {
    TotalReturn   float64
    CAGR          float64
    MaxDrawdown   float64
    SharpeRatio   float64
    SortinoRatio  float64
    CalmarRatio   float64
    WinRate       float64
    ProfitFactor  float64
    AvgTrade      float64
    TotalTrades   int
    AnnualVolatility float64
    BenchmarkReturn  float64
    Alpha            float64
    Beta             float64
    InformationRatio float64
}
```

#### 4.2 Market-Specific Engines

First two markets (v1), rest follow the same pattern:

| Market | Engine | Key Rules |
|--------|--------|-----------|
| A-shares | `CNEngine` | T+1 settlement, ±10%/±20% price limits, 0.05% stamp duty (sell only), min 100 shares |
| US | `USEngine` | T+2 settlement, no price limits, PDT rule check, fractional shares |

#### 4.3 Execution Flow

```
BacktestRunner.Run(config, strategy, universe)
    │
    ├── 1. Load historical OHLCV for all symbols (from SQLite cache or adapters)
    ├── 2. Sort bars chronologically
    ├── 3. For each trading day:
    │   ├── For each symbol:
    │   │   ├── Compute factors (Go native or Python gRPC)
    │   │   ├── Evaluate signal rules
    │   │   ├── If signal fires:
    │   │   │   ├── RiskPipeline.Check()
    │   │   │   ├── PositionSizer.Calculate()
    │   │   │   ├── Apply commission + slippage
    │   │   │   └── OMS.PlaceOrder()
    │   │   └── If holding:
    │   │       ├── Check stop loss / take profit
    │   │       └── Update market value
    │   ├── Record equity point (total portfolio value)
    │   └── Log daily summary
    ├── 4. Compute performance metrics
    ├── 5. Generate trade list
    └── 6. Return BacktestResult
```

### 5. Python Sidecar Structure

```
python/
├── pyproject.toml
├── requirements.txt
├── src/
│   ├── __init__.py
│   ├── server.py                    # gRPC server entry point (asyncio)
│   ├── factor/
│   │   ├── __init__.py
│   │   ├── engine.py                # FactorEngine: registry + dispatch
│   │   ├── registry.py              # Factor registry with metadata
│   │   ├── momentum.py              # Momentum factors (20+)
│   │   ├── trend.py                 # Trend/moving average factors
│   │   ├── volatility.py            # Volatility factors
│   │   ├── volume.py                # Volume/flow factors
│   │   ├── fundamental.py           # Fundamental factors (requires financials)
│   │   └── cross_sectional.py       # Cross-sectional factors (rank, z-score)
│   ├── ml/
│   │   ├── __init__.py
│   │   ├── engine.py                # MLEngine: train/predict
│   │   └── models/
│   │       └── linear.py            # Linear regression baseline
│   ├── llm/
│   │   ├── __init__.py
│   │   └── engine.py                # LLMEngine: stub for future
│   ├── data/
│   │   ├── __init__.py
│   │   └── fetcher.py               # Python-side data fetching (future)
│   └── proto/                        # Generated protobuf code
│       ├── factor_pb2.py
│       ├── factor_pb2_grpc.py
│       ├── ml_pb2.py
│       ├── ml_pb2_grpc.py
│       └── ...
├── tests/
│   ├── __init__.py
│   ├── test_factor_engine.py
│   ├── test_factor_momentum.py
│   └── test_server.py
└── proto/                            # Protobuf definitions (source)
    ├── factor.proto
    ├── ml.proto
    ├── health.proto
    └── data.proto
```

### 6. Go PythonBridge Design

```
internal/python/
├── bridge.go              # Connection manager, health check
├── factor_client.go       # FactorService gRPC client wrapper
├── ml_client.go           # MLService gRPC client wrapper (stub)
└── proto/                  # Generated Go protobuf code
    ├── factor.pb.go
    ├── factor_grpc.pb.go
    └── ...
```

```go
// bridge.go
type PythonBridge struct {
    conn          *grpc.ClientConn
    FactorClient  pb.FactorServiceClient
    MLClient      pb.MLServiceClient
    HealthClient  pb.HealthServiceClient
    opts          BridgeOptions
}

type BridgeOptions struct {
    Address        string        // default: "localhost:50051"
    DialTimeout    time.Duration // default: 5s
    RequestTimeout time.Duration // default: 30s
    MaxRetries     int           // default: 3
}

func NewPythonBridge(opts BridgeOptions) (*PythonBridge, error)
func (b *PythonBridge) Ping(ctx context.Context) error
func (b *PythonBridge) IsHealthy(ctx context.Context) bool
func (b *PythonBridge) Close() error
```

### 7. Workflow Nodes (New)

| Node | Category | Description | Python Required? |
|------|----------|-------------|:---:|
| `FactorNode` | alpha | Compute alpha factor for symbols | Yes |
| `StrategyNode` | strategy | Define signal→risk→execute pipeline | No |
| `BacktestNode` | backtest | Run historical backtest | No |
| `PerformanceNode` | backtest | Compute performance metrics | No |
| `FactorList` | alpha | List available factors + metadata | Yes |

### 8. Frontend Panels (New/Enhanced)

| Panel | Type | Description |
|-------|------|-------------|
| `BacktestResultPanel` | New | Equity curve, drawdown chart, metrics table, trade list |
| `FactorAnalysisPanel` | New | Factor IC analysis, factor distribution, correlation matrix |
| `PerformanceMetricsPanel` | New | Metric cards: CAGR, Sharpe, MaxDD, Win Rate |
| `EquityCurvePanel` | Enhanced | Multi-strategy overlay, benchmark comparison |
| `StrategyBuilderPanel` | New | Visual strategy config with signal rules editor |

### 9. File Change Map

```
NEW FILES:
├── python/                                    # Entire Python sidecar (new)
│   ├── pyproject.toml
│   ├── requirements.txt
│   ├── src/
│   │   ├── server.py
│   │   ├── factor/{engine,registry,momentum,trend,volatility,volume,cross_sectional}.py
│   │   ├── ml/{engine,models/linear}.py
│   │   ├── llm/engine.py
│   │   └── data/fetcher.py
│   ├── tests/{test_factor_engine,test_momentum,test_server}.py
│   └── proto/{factor,ml,health,data}.proto
│
├── internal/python/                           # Go PythonBridge (new)
│   ├── bridge.go
│   ├── factor_client.go
│   ├── ml_client.go
│   └── proto/*.pb.go                          # Generated
│
├── internal/backtest/                         # Backtesting engine (new)
│   ├── runner.go
│   ├── config.go
│   ├── metrics.go
│   ├── equity.go
│   ├── engine_cn.go                           # A-share rules
│   ├── engine_us.go                           # US market rules
│   ├── runner_test.go
│   └── metrics_test.go
│
├── internal/workflow/nodes/
│   ├── factor.go                              # FactorNode
│   ├── strategy.go                            # StrategyNode
│   ├── backtest.go                            # BacktestNode
│   └── performance.go                         # PerformanceNode

MODIFIED FILES:
├── internal/workflow/nodes/register.go        # Register new nodes
├── app.go                                     # Add PythonBridge + BacktestRunner to App
├── frontend/src/stores/data.ts                # Add backtest results cache
└── frontend/src/terminal/panels/registry.ts   # Register new panels
```

---

## Acceptance Criteria

### Python Sidecar
- [ ] `python -m src.server` starts gRPC server on localhost:50051
- [ ] `HealthService.Ping()` returns healthy status
- [ ] `FactorService.ComputeFactor("momentum_20d", ["000001.SZ"], ...)` returns valid factor values
- [ ] Python sidecar starts/stops cleanly (no dangling processes)
- [ ] Python sidecar handles Go process crash gracefully (no zombie)

### Factor Engine
- [ ] 25+ factors implemented: momentum (5), trend (5), volatility (5), volume (5), cross-sectional (5)
- [ ] Each factor has metadata: name, category, description, default params
- [ ] Factor computation uses Arrow IPC for data transfer (not JSON)
- [ ] `ListFactors()` returns complete factor catalog
- [ ] Factors handle NaN/Inf gracefully (forward fill, then 0)

### Go PythonBridge
- [ ] `PythonBridge.Ping()` works within 100ms
- [ ] `PythonBridge.ComputeFactor()` with retry on transient gRPC errors
- [ ] `PythonBridge.IsHealthy()` returns false when Python is down (no crash)
- [ ] Graceful degradation: workflow nodes work with a warning when Python is unavailable

### Backtesting Engine
- [ ] `BacktestRunner.Run()` for A-shares with T+1, price limits, stamp duty
- [ ] `BacktestRunner.Run()` for US stocks
- [ ] Simple SMA cross strategy backtests correctly against known results
- [ ] Performance metrics match manual calculation (within 1e-6)
- [ ] `EquityCurve` is point-for-point correct for a known test case
- [ ] Backtest with 100 symbols × 250 days completes in <5 seconds

### Workflow Integration
- [ ] `FactorNode` appears in NodePalette under "Alpha" category
- [ ] `StrategyNode` + `BacktestNode` can be connected: Factor → Strategy → Backtest
- [ ] Running a backtest workflow produces `BacktestResult` in ExecutionLog
- [ ] ExecutionLog shows per-node timing (factor: 120ms, backtest: 2.3s)

### Frontend
- [ ] `BacktestResultPanel` shows: equity curve (ECharts), drawdown chart, metrics table, trade list
- [ ] `FactorAnalysisPanel` shows: factor catalog with search, factor metadata
- [ ] New panels accessible from CommandBar (`Ctrl+K` "backtest" / "factor")

### Cross-Cutting
- [ ] `go test ./...` all pass (Go side)
- [ ] `python -m pytest tests/ -x -q` all pass (Python side)
- [ ] `go build .` succeeds
- [ ] CHANGELOG.md updated with Phase 3 entries

---

## Risks / Trade-offs

### Risk 1: gRPC complexity for a desktop app
- **Mitigation**: gRPC over localhost is simple — no TLS, no auth, no service discovery. Just a TCP connection to localhost:50051. If gRPC proves too heavy, we can fall back to stdin/stdout JSON pipe.
- **Why gRPC anyway**: Protobuf contracts are self-documenting, type-safe on both sides, and Arrow Flight integrates naturally.

### Risk 2: Python dependency management for end users
- **Mitigation**: Python sidecar is **optional**. Core features (data viewing, trading, workflow execution) work without it. Only factor computation and ML require Python.
- **Distribution**: Ship Python sidecar as a separate `pip install quantflow-python` or bundle via PyInstaller for advanced users.

### Risk 3: Arrow IPC overhead
- **Mitigation**: For small data (<1000 bars), JSON is simpler and fast enough. Arrow is only used for bulk transfers (>1000 bars per request). We can start with JSON and add Arrow later.

### Risk 4: Backtesting engine accuracy
- **Mitigation**: Validate against known backtest frameworks (zipline, backtrader) with the same strategy + data → results must match within 0.1%.

### Risk 5: Factor computation performance
- **Mitigation**: Factor computation is vectorized (pandas). 100 factors on 500 stocks × 250 days = ~500ms. If slower, add caching and incremental computation.

### Trade-off: Start with 25 factors, not 450
- **Why**: Getting the pipeline right (protobuf → Arrow → pandas → result) is the hard part. Adding more factors is mechanical once the pipeline works. 25 well-tested factors > 450 untested ones.

### Trade-off: A-shares + US first, other markets later
- **Why**: A-shares have the most complex rules (T+1, price limits, stamp duty on sell only). Getting this right validates the engine design. US is simpler and serves as a sanity check. HK/CRYPTO/Futures follow the same pattern.
