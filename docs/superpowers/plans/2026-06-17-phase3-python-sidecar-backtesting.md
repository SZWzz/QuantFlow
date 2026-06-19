# Phase 3: Python gRPC Sidecar + Factor Engine + Backtesting — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Establish the Go↔Python gRPC bridge, implement the factor computation engine (25+ factors), build the multi-market backtesting engine (A-shares + US v1), and integrate everything into the workflow + terminal UI.

**Architecture:** Seven serial milestones. M1 scaffolds the Python gRPC sidecar with protobuf contracts. M2 implements the factor engine with 25+ factors. M3 builds the Go PythonBridge gRPC client. M4 implements the backtesting engine with market-specific rules. M5 creates workflow nodes (FactorNode, StrategyNode, BacktestNode). M6 builds frontend panels (BacktestResultPanel, FactorAnalysisPanel). M7 integrates everything end-to-end.

**Tech Stack:** Go 1.22+, Python 3.12+, gRPC (grpcio + protobuf), pandas, pyarrow, Vue 3 + ECharts, SQLite WAL

**Spec:** [docs/superpowers/specs/2026-06-17-phase3-python-sidecar-backtesting.md](../specs/2026-06-17-phase3-python-sidecar-backtesting.md)

---

## Prerequisites

Before starting, verify Phase 2.5 is clean:

```bash
cd /Volumes/shenzy/vibe_coding/quant/QuantFlow
go build ./... && go test ./internal/... -count=1
# All must pass. If not, fix Phase 2.5 first.
```

Verify Python 3.12+ is available:

```bash
python3 --version  # Must be 3.12+
```

---

## Milestone 1: Python gRPC Sidecar Skeleton

### Task 1: Python project setup

**Files:**
- Create: `python/pyproject.toml`
- Create: `python/requirements.txt`
- Create: `python/src/__init__.py`

- [ ] **Step 1: Create pyproject.toml**

```bash
mkdir -p python/src python/tests python/proto
```

Create `python/pyproject.toml`:

```toml
[project]
name = "quantflow-python"
version = "2026.6.17"
description = "QuantFlow Python gRPC Sidecar"
requires-python = ">=3.12"
dependencies = [
    "grpcio>=1.60",
    "grpcio-tools>=1.60",
    "protobuf>=4.25",
    "pandas>=2.1",
    "numpy>=1.26",
    "pyarrow>=14.0",
]

[project.optional-dependencies]
dev = [
    "pytest>=8.0",
    "pytest-asyncio>=0.23",
    "pytest-cov>=4.0",
]

[tool.pytest.ini_options]
testpaths = ["tests"]
asyncio_mode = "auto"
```

- [ ] **Step 2: Create requirements.txt**

```
grpcio>=1.60
grpcio-tools>=1.60
protobuf>=4.25
pandas>=2.1
numpy>=1.26
pyarrow>=14.0
```

- [ ] **Step 3: Install Python dependencies**

```bash
cd python && python3 -m venv venv && source venv/bin/activate && pip install -e ".[dev]"
```

- [ ] **Step 4: Commit**

```bash
git add python/pyproject.toml python/requirements.txt python/src/__init__.py
git commit -m "feat(m1): initialize Python gRPC sidecar project structure"
```

---

### Task 2: Protobuf definitions + code generation

**Files:**
- Create: `python/proto/factor.proto`
- Create: `python/proto/ml.proto`
- Create: `python/proto/health.proto`
- Create: `python/proto/data.proto`
- Create: `internal/python/proto/` (Go generated code, after `protoc`)

- [ ] **Step 1: Write factor.proto**

```protobuf
syntax = "proto3";

package quantflow;

option go_package = "quantflow/internal/python/proto;proto";

service FactorService {
  rpc ComputeFactor(ComputeFactorRequest) returns (ComputeFactorResponse);
  rpc ListFactors(ListFactorsRequest) returns (ListFactorsResponse);
}

message ComputeFactorRequest {
  string factor_name = 1;
  repeated string symbols = 2;
  string start_date = 3;
  string end_date = 4;
  map<string, string> params = 5;
  bytes ohlcv_data = 6;              // Arrow IPC bytes
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
  string error = 4;
}

message FactorMeta {
  string name = 1;
  string category = 2;
  string description = 3;
  map<string, string> default_params = 4;
}

message ListFactorsRequest {}

message ListFactorsResponse {
  repeated FactorMeta factors = 1;
}
```

- [ ] **Step 2: Write health.proto**

```protobuf
syntax = "proto3";

package quantflow;

option go_package = "quantflow/internal/python/proto;proto";

service HealthService {
  rpc Ping(PingRequest) returns (PingResponse);
  rpc GetStatus(GetStatusRequest) returns (StatusResponse);
}

message PingRequest {}
message PingResponse {
  bool healthy = 1;
  string version = 2;
  int64 uptime_seconds = 3;
}

message GetStatusRequest {}
message StatusResponse {
  bool healthy = 1;
  string version = 2;
  int64 uptime_seconds = 3;
  int32 active_requests = 4;
  int64 memory_mb = 5;
}
```

- [ ] **Step 3: Write ml.proto and data.proto (minimal stubs)**

Create `python/proto/ml.proto` and `python/proto/data.proto` with minimal service definitions (one RPC each).

- [ ] **Step 4: Generate Python protobuf code**

```bash
cd python
source venv/bin/activate
python -m grpc_tools.protoc \
  -Iproto \
  --python_out=src/proto \
  --grpc_python_out=src/proto \
  proto/factor.proto proto/ml.proto proto/health.proto proto/data.proto
```

Create `python/src/proto/__init__.py` (empty).

- [ ] **Step 5: Generate Go protobuf code**

```bash
mkdir -p internal/python/proto

# Install protoc plugins if needed
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

protoc \
  -Ipython/proto \
  --go_out=internal/python/proto \
  --go_opt=paths=source_relative \
  --go-grpc_out=internal/python/proto \
  --go-grpc_opt=paths=source_relative \
  python/proto/factor.proto python/proto/health.proto python/proto/ml.proto python/proto/data.proto
```

- [ ] **Step 6: Add Go protobuf dependency**

```bash
go get google.golang.org/protobuf google.golang.org/grpc
go mod tidy
```

- [ ] **Step 7: Verify both sides compile**

```bash
# Python
cd python && source venv/bin/activate && python -c "from src.proto import factor_pb2; print('OK')"

# Go
cd /Volumes/shenzy/vibe_coding/quant/QuantFlow && go build ./internal/python/proto/...
```

- [ ] **Step 8: Commit**

```bash
git add python/proto/ python/src/proto/ internal/python/proto/
git commit -m "feat(m1): add protobuf definitions and generated code for factor/health/ml/data services"
```

---

### Task 3: Python gRPC server

**Files:**
- Create: `python/src/server.py`
- Create: `python/src/factor/__init__.py`
- Create: `python/src/factor/engine.py`
- Create: `python/src/factor/registry.py`
- Create: `python/src/ml/__init__.py`
- Create: `python/src/ml/engine.py`
- Create: `python/src/llm/__init__.py`
- Create: `python/src/llm/engine.py`
- Create: `python/src/data/__init__.py`
- Create: `python/src/data/fetcher.py`

- [ ] **Step 1: Implement server.py**

```python
"""QuantFlow Python gRPC Sidecar — main entry point."""
import asyncio
import logging
from concurrent import futures

import grpc

from src.proto import factor_pb2_grpc, health_pb2_grpc, ml_pb2_grpc, data_pb2_grpc
from src.factor.engine import FactorService
from src.ml.engine import MLService
from src.llm.engine import LLMService
from src.data.fetcher import DataService

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

DEFAULT_PORT = 50051


class HealthService(health_pb2_grpc.HealthServiceServicer):
    def __init__(self):
        self.start_time = asyncio.get_event_loop().time()

    async def Ping(self, request, context):
        return health_pb2.PingResponse(healthy=True, version="2026.6.17")

    async def GetStatus(self, request, context):
        elapsed = asyncio.get_event_loop().time() - self.start_time
        return health_pb2.StatusResponse(
            healthy=True,
            version="2026.6.17",
            uptime_seconds=int(elapsed),
            active_requests=0,
        )


async def serve(port: int = DEFAULT_PORT):
    server = grpc.aio.server(futures.ThreadPoolExecutor(max_workers=10))
    factor_pb2_grpc.add_FactorServiceServicer_to_server(FactorService(), server)
    ml_pb2_grpc.add_MLServiceServicer_to_server(MLService(), server)
    health_pb2_grpc.add_HealthServiceServicer_to_server(HealthService(), server)
    data_pb2_grpc.add_DataServiceServicer_to_server(DataService(), server)

    server.add_insecure_port(f"[::]:{port}")
    logger.info(f"QuantFlow Python sidecar listening on port {port}")
    await server.start()
    await server.wait_for_termination()


if __name__ == "__main__":
    asyncio.run(serve())
```

- [ ] **Step 2: Implement FactorEngine (registry-first)**

Create `python/src/factor/registry.py`:

```python
"""Factor registry — maps factor names to implementations."""
from dataclasses import dataclass, field
from typing import Callable, Dict, Any
import pandas as pd

@dataclass
class FactorMeta:
    name: str
    category: str
    description: str
    default_params: Dict[str, str] = field(default_factory=dict)

# Global registry
_registry: Dict[str, FactorMeta] = {}
_compute_funcs: Dict[str, Callable] = {}

def register(meta: FactorMeta):
    """Decorator to register a factor computation function."""
    def decorator(func: Callable):
        _registry[meta.name] = meta
        _compute_funcs[meta.name] = func
        return func
    return decorator

def list_factors() -> list[FactorMeta]:
    return list(_registry.values())

def compute(factor_name: str, ohlcv: pd.DataFrame, params: Dict[str, Any]) -> pd.Series:
    """Compute a factor. Raises KeyError if factor not found."""
    if factor_name not in _compute_funcs:
        raise KeyError(f"Unknown factor: {factor_name}")
    return _compute_funcs[factor_name](ohlcv, params)
```

Create `python/src/factor/engine.py`:

```python
"""FactorService gRPC implementation."""
import io
import time
import logging

import pandas as pd
import pyarrow as pa
import pyarrow.ipc as ipc

from src.proto import factor_pb2, factor_pb2_grpc
from src.factor.registry import list_factors, compute

logger = logging.getLogger(__name__)


class FactorService(factor_pb2_grpc.FactorServiceServicer):
    async def ComputeFactor(self, request, context):
        t0 = time.time()
        try:
            # Decode Arrow IPC bytes → pandas DataFrame
            if request.ohlcv_data:
                reader = ipc.open_stream(request.ohlcv_data)
                table = reader.read_all()
                df = table.to_pandas()
            else:
                df = pd.DataFrame()

            # Compute factor for each symbol
            results = []
            for symbol in request.symbols:
                # Filter data for this symbol if multi-symbol DataFrame
                symbol_df = df[df["symbol"] == symbol] if "symbol" in df.columns else df
                values = compute(request.factor_name, symbol_df, dict(request.params))

                results.append(factor_pb2.FactorResult(
                    symbol=symbol,
                    dates=values.index.astype(str).tolist() if hasattr(values, 'index') else [],
                    values=values.tolist(),
                ))

            elapsed_ms = int((time.time() - t0) * 1000)
            return factor_pb2.ComputeFactorResponse(
                factor_name=request.factor_name,
                results=results,
                compute_time_ms=elapsed_ms,
            )
        except Exception as e:
            logger.exception(f"ComputeFactor failed: {e}")
            return factor_pb2.ComputeFactorResponse(
                factor_name=request.factor_name,
                error=str(e),
            )

    async def ListFactors(self, request, context):
        factors = list_factors()
        return factor_pb2.ListFactorsResponse(
            factors=[
                factor_pb2.FactorMeta(
                    name=f.name,
                    category=f.category,
                    description=f.description,
                    default_params=f.default_params,
                )
                for f in factors
            ]
        )
```

- [ ] **Step 3: Implement stub services**

Create minimal stub implementations for `MLService`, `LLMService`, `DataService` that return "not implemented" errors.

- [ ] **Step 4: Test server starts**

```bash
cd python && source venv/bin/activate && python -m src.server &
sleep 2
# Test with grpcurl or a quick Python client
python -c "
import grpc
from src.proto import health_pb2, health_pb2_grpc
ch = grpc.insecure_channel('localhost:50051')
stub = health_pb2_grpc.HealthServiceStub(ch)
resp = stub.Ping(health_pb2.PingRequest())
print(f'Server healthy: {resp.healthy}, version: {resp.version}')
"
kill %1
```

- [ ] **Step 5: Commit**

```bash
git add python/src/server.py python/src/factor/ python/src/ml/ python/src/llm/ python/src/data/
git commit -m "feat(m1): implement Python gRPC server with FactorService, HealthService, and stub services"
```

---

## Milestone 2: Factor Engine — 25+ Factors

### Task 4: Momentum factors

**Files:**
- Create: `python/src/factor/momentum.py`
- Create: `python/tests/test_factor_momentum.py`

- [ ] **Step 1: Implement 5 momentum factors**

```python
from src.factor.registry import register, FactorMeta

@register(FactorMeta(
    name="momentum_20d",
    category="momentum",
    description="20-day price momentum (close / close_20d_ago - 1)",
    default_params={"period": "20"},
))
def momentum_20d(ohlcv: pd.DataFrame, params: dict) -> pd.Series:
    period = int(params.get("period", 20))
    return ohlcv["close"].pct_change(period)

@register(FactorMeta(
    name="momentum_60d",
    category="momentum",
    description="60-day price momentum",
    default_params={"period": "60"},
))
def momentum_60d(ohlcv: pd.DataFrame, params: dict) -> pd.Series:
    period = int(params.get("period", 60))
    return ohlcv["close"].pct_change(period)

@register(FactorMeta(
    name="momentum_120d",
    category="momentum",
    description="120-day price momentum",
    default_params={"period": "120"},
))
def momentum_120d(ohlcv: pd.DataFrame, params: dict) -> pd.Series:
    period = int(params.get("period", 120))
    return ohlcv["close"].pct_change(period)

@register(FactorMeta(
    name="momentum_5d_minus_20d",
    category="momentum",
    description="Short-term momentum minus medium-term (reversal signal)",
))
def momentum_5d_minus_20d(ohlcv: pd.DataFrame, params: dict) -> pd.Series:
    return ohlcv["close"].pct_change(5) - ohlcv["close"].pct_change(20)

@register(FactorMeta(
    name="rsi_14",
    category="momentum",
    description="14-day Relative Strength Index",
    default_params={"period": "14"},
))
def rsi_14(ohlcv: pd.DataFrame, params: dict) -> pd.Series:
    period = int(params.get("period", 14))
    delta = ohlcv["close"].diff()
    gain = delta.clip(lower=0).rolling(period).mean()
    loss = (-delta.clip(upper=0)).rolling(period).mean()
    rs = gain / loss
    return 100 - (100 / (1 + rs))
```

- [ ] **Step 2: Write tests**

```python
import pandas as pd
import numpy as np
from src.factor.momentum import momentum_20d, rsi_14

def test_momentum_20d():
    df = pd.DataFrame({"close": [100 + i for i in range(30)]})
    result = momentum_20d(df, {"period": "20"})
    assert result.iloc[-1] == pytest.approx(29/120, 0.01)
    assert pd.isna(result.iloc[0])

def test_rsi_14():
    # RSI for all-up market = 100
    df = pd.DataFrame({"close": [100 + i for i in range(20)]})
    result = rsi_14(df, {"period": "14"})
    assert result.iloc[-1] == pytest.approx(100.0, 0.1)
```

- [ ] **Step 3: Verify tests pass**

```bash
cd python && source venv/bin/activate && python -m pytest tests/test_factor_momentum.py -v
```

- [ ] **Step 4: Commit**

---

### Task 5: Trend, Volatility, Volume factors

**Files:**
- Create: `python/src/factor/trend.py`
- Create: `python/src/factor/volatility.py`
- Create: `python/src/factor/volume.py`
- Create: `python/tests/test_factor_trend.py`
- Create: `python/tests/test_factor_volatility.py`
- Create: `python/tests/test_factor_volume.py`

- [ ] **Step 1: Implement 5 trend factors**

```
ma_5, ma_20, ma_60, ma_5_minus_ma_20, macd_12_26_9
```

Each factor: simple moving average cross, MACD with signal line.

- [ ] **Step 2: Implement 5 volatility factors**

```
atr_14, volatility_20d, volatility_60d, bollinger_width_20, beta_60d
```

Each factor: ATR (average true range), rolling std, Bollinger band width.

- [ ] **Step 3: Implement 5 volume factors**

```
volume_ratio_5d, volume_ratio_20d, obv, vwap_deviation, turnover_20d
```

Each factor: volume ratio vs moving average, OBV, VWAP deviation.

- [ ] **Step 4: Write tests for all factors**

- [ ] **Step 5: Verify all tests pass**

```bash
cd python && source venv/bin/activate && python -m pytest tests/ -v
```

- [ ] **Step 6: Commit**

---

### Task 6: Cross-sectional factors + FactorEngine integration test

**Files:**
- Create: `python/src/factor/cross_sectional.py`
- Create: `python/tests/test_factor_engine.py`

- [ ] **Step 1: Implement 5 cross-sectional factors**

```
zscore_momentum, rank_momentum, sector_neutral_momentum, size_factor, industry_dummy
```

- [ ] **Step 2: Write FactorEngine integration test**

```python
async def test_list_factors():
    factors = list_factors()
    assert len(factors) >= 25
    categories = {f.category for f in factors}
    assert "momentum" in categories
    assert "trend" in categories

async def test_compute_factor_via_service():
    # Start server, call ComputeFactor via gRPC, verify response
    ...
```

- [ ] **Step 3: Verify full test suite**

```bash
cd python && source venv/bin/activate && python -m pytest tests/ -v --cov=src/factor
```

- [ ] **Step 4: Commit**

---

## Milestone 3: Go PythonBridge

### Task 7: PythonBridge — connection + health

**Files:**
- Create: `internal/python/bridge.go`
- Create: `internal/python/bridge_test.go`

- [ ] **Step 1: Implement PythonBridge**

```go
package python

import (
    "context"
    "fmt"
    "time"

    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"

    pb "quantflow/internal/python/proto"
)

type BridgeOptions struct {
    Address        string
    DialTimeout    time.Duration
    RequestTimeout time.Duration
    MaxRetries     int
}

func DefaultOptions() BridgeOptions {
    return BridgeOptions{
        Address:        "localhost:50051",
        DialTimeout:    5 * time.Second,
        RequestTimeout: 30 * time.Second,
        MaxRetries:     3,
    }
}

type PythonBridge struct {
    conn         *grpc.ClientConn
    FactorClient pb.FactorServiceClient
    HealthClient pb.HealthServiceClient
    opts         BridgeOptions
}

func NewPythonBridge(opts BridgeOptions) (*PythonBridge, error) {
    ctx, cancel := context.WithTimeout(context.Background(), opts.DialTimeout)
    defer cancel()

    conn, err := grpc.DialContext(ctx, opts.Address,
        grpc.WithTransportCredentials(insecure.NewCredentials()),
        grpc.WithBlock(),
    )
    if err != nil {
        return nil, fmt.Errorf("python bridge: dial %s: %w", opts.Address, err)
    }

    return &PythonBridge{
        conn:         conn,
        FactorClient: pb.NewFactorServiceClient(conn),
        HealthClient: pb.NewHealthServiceClient(conn),
        opts:         opts,
    }, nil
}

func (b *PythonBridge) Ping(ctx context.Context) error {
    ctx, cancel := context.WithTimeout(ctx, b.opts.RequestTimeout)
    defer cancel()

    resp, err := b.HealthClient.Ping(ctx, &pb.PingRequest{})
    if err != nil {
        return fmt.Errorf("ping: %w", err)
    }
    if !resp.Healthy {
        return fmt.Errorf("python sidecar unhealthy")
    }
    return nil
}

func (b *PythonBridge) IsHealthy(ctx context.Context) bool {
    return b.Ping(ctx) == nil
}

func (b *PythonBridge) Close() error {
    return b.conn.Close()
}
```

- [ ] **Step 2: Write tests (with Python sidecar running)**

```go
func TestPythonBridge_Integration(t *testing.T) {
    // This test requires Python sidecar running on localhost:50051
    // Skip if not available
    bridge, err := NewPythonBridge(DefaultOptions())
    if err != nil {
        t.Skipf("Python sidecar not available: %v", err)
    }
    defer bridge.Close()

    ctx := context.Background()
    if err := bridge.Ping(ctx); err != nil {
        t.Fatalf("Ping failed: %v", err)
    }
}
```

- [ ] **Step 3: Commit**

---

### Task 8: FactorClient — gRPC factor calls with retry

**Files:**
- Create: `internal/python/factor_client.go`
- Create: `internal/python/factor_client_test.go`

- [ ] **Step 1: Implement FactorClient wrapper**

```go
type FactorResult struct {
    Symbol string
    Dates  []string
    Values []float64
}

func (b *PythonBridge) ComputeFactor(ctx context.Context, req *pb.ComputeFactorRequest) ([]FactorResult, error) {
    ctx, cancel := context.WithTimeout(ctx, b.opts.RequestTimeout)
    defer cancel()

    var lastErr error
    for attempt := 0; attempt < b.opts.MaxRetries; attempt++ {
        resp, err := b.FactorClient.ComputeFactor(ctx, req)
        if err != nil {
            lastErr = err
            if isTransient(err) && attempt < b.opts.MaxRetries-1 {
                time.Sleep(time.Duration(attempt+1) * 500 * time.Millisecond)
                continue
            }
            return nil, fmt.Errorf("compute factor: %w", err)
        }
        if resp.Error != "" {
            return nil, fmt.Errorf("python factor error: %s", resp.Error)
        }

        results := make([]FactorResult, len(resp.Results))
        for i, r := range resp.Results {
            results[i] = FactorResult{
                Symbol: r.Symbol,
                Dates:  r.Dates,
                Values: r.Values,
            }
        }
        return results, nil
    }
    return nil, fmt.Errorf("compute factor after %d retries: %w", b.opts.MaxRetries, lastErr)
}

func (b *PythonBridge) ListFactors(ctx context.Context) ([]*pb.FactorMeta, error) {
    ctx, cancel := context.WithTimeout(ctx, b.opts.RequestTimeout)
    defer cancel()

    resp, err := b.FactorClient.ListFactors(ctx, &pb.ListFactorsRequest{})
    if err != nil {
        return nil, fmt.Errorf("list factors: %w", err)
    }
    return resp.Factors, nil
}

func isTransient(err error) bool {
    s := status.Convert(err)
    switch s.Code() {
    case codes.Unavailable, codes.DeadlineExceeded, codes.ResourceExhausted:
        return true
    }
    return false
}
```

- [ ] **Step 2: Write integration test**

Test ListFactors (always works) and ComputeFactor with mock OHLCV data.

- [ ] **Step 3: Commit**

---

## Milestone 4: Backtesting Engine

### Task 9: Backtest runner + core types

**Files:**
- Create: `internal/backtest/config.go`
- Create: `internal/backtest/types.go`
- Create: `internal/backtest/runner.go`
- Create: `internal/backtest/metrics.go`
- Create: `internal/backtest/equity.go`

- [ ] **Step 1: Define config and types**

```go
// config.go
package backtest

import "time"

type Config struct {
    StartDate    time.Time
    EndDate      time.Time
    InitialCash  float64
    Commission   float64   // e.g., 0.0003 for A-shares
    Slippage     float64   // e.g., 0.001 (10 bps)
    Benchmark    string    // e.g., "000300.SH"
}

func DefaultConfig() Config {
    return Config{
        InitialCash: 1_000_000,
        Commission:  0.0003,
        Slippage:    0.001,
    }
}
```

```go
// types.go
package backtest

import "quantflow/internal/trading"

type Strategy struct {
    ID          string
    Name        string
    SignalFunc  func(bar trading.OHLCVBar, portfolio *Portfolio) *trading.Signal
    RiskConfig  trading.RiskConfig
}

type Portfolio struct {
    Cash      float64
    Positions map[string]*trading.Position
}

type EquityPoint struct {
    Date   time.Time
    Equity float64
    Cash   float64
}

type Result struct {
    Config      Config
    EquityCurve []EquityPoint
    Trades      []trading.Trade
    Metrics     Metrics
}

type Metrics struct {
    TotalReturn     float64
    CAGR            float64
    MaxDrawdown     float64
    SharpeRatio     float64
    SortinoRatio    float64
    CalmarRatio     float64
    WinRate         float64
    ProfitFactor    float64
    TotalTrades     int
    AnnualVolatility float64
}
```

- [ ] **Step 2: Implement metrics calculator**

```go
// metrics.go
func ComputeMetrics(equityCurve []EquityPoint, trades []trading.Trade, tradingDays int) Metrics {
    // 1. Total return: (final / initial - 1)
    // 2. CAGR: (final/initial)^(1/years) - 1
    // 3. Max drawdown: max peak-to-trough
    // 4. Sharpe: mean(daily_return) / std(daily_return) * sqrt(252)
    // 5. Sortino: mean(daily_return) / std(negative_daily_return) * sqrt(252)
    // 6. Calmar: CAGR / MaxDrawdown
    // 7. Win rate: winning_trades / total_trades
    // 8. Profit factor: gross_profit / gross_loss
    ...
}
```

- [ ] **Step 3: Implement runner**

```go
// runner.go
type Runner struct {
    config  Config
    oms     *trading.OMS
    matcher *trading.OrderMatcher
    risk    *trading.RiskPipeline
}

func NewRunner(config Config) *Runner {
    return &Runner{
        config:  config,
        oms:     trading.NewOMS(),
        matcher: trading.NewOrderMatcher(),
        risk:    trading.NewRiskPipeline(config.ToRiskConfig()),
    }
}

func (r *Runner) Run(ctx context.Context, strategy Strategy, bars []trading.OHLCVBar) (*Result, error) {
    portfolio := &Portfolio{
        Cash:      r.config.InitialCash,
        Positions: make(map[string]*trading.Position),
    }

    var equityCurve []EquityPoint
    var allTrades []trading.Trade

    // Sort bars by date
    sort.Slice(bars, func(i, j int) bool { return bars[i].Date < bars[j].Date })

    for _, bar := range bars {
        select {
        case <-ctx.Done():
            return nil, ctx.Err()
        default:
        }

        // 1. Update market prices
        r.oms.UpdateMarketPrice(bar.Symbol, bar.Close)

        // 2. Generate signal
        signal := strategy.SignalFunc(bar, portfolio)
        if signal != nil {
            // 3. Risk check
            if err := r.risk.Check(*signal); err != nil {
                continue
            }
            // 4. Place order
            order := signalToOrder(signal, r.config)
            r.oms.PlaceOrder(order)
        }

        // 5. Match pending orders against this bar
        r.matcher.MatchBar(bar, r.oms)

        // 6. Record equity
        equity := portfolio.Cash
        for _, pos := range r.oms.GetAllPositions() {
            equity += pos.Quantity * bar.Close
        }
        equityCurve = append(equityCurve, EquityPoint{
            Date:   bar.Date,
            Equity: equity,
            Cash:   portfolio.Cash,
        })
    }

    allTrades = r.oms.GetAllTrades()
    metrics := ComputeMetrics(equityCurve, allTrades, len(bars))

    return &Result{
        Config:      r.config,
        EquityCurve: equityCurve,
        Trades:      allTrades,
        Metrics:     metrics,
    }, nil
}
```

- [ ] **Step 4: Write tests**

```go
func TestRunner_SMACross(t *testing.T) {
    // Create synthetic OHLCV data with known pattern
    // SMA(5) > SMA(20) → buy, SMA(5) < SMA(20) → sell
    // Verify: number of trades, final equity, metrics
}
```

- [ ] **Step 5: Commit**

---

### Task 10: Market-specific engines (A-shares + US)

**Files:**
- Create: `internal/backtest/engine_cn.go`
- Create: `internal/backtest/engine_us.go`
- Create: `internal/backtest/engine_cn_test.go`

- [ ] **Step 1: Implement A-share engine**

```go
// engine_cn.go
type CNEngine struct {
    Runner
}

func NewCNEngine(config Config) *CNEngine {
    // Set A-share specific defaults
    config.Commission = 0.0003   // 万三佣金
    config.Slippage = 0.001     // 10 bps
    return &CNEngine{Runner: *NewRunner(config)}
}

func (e *CNEngine) Run(ctx context.Context, strategy Strategy, bars []trading.OHLCVBar) (*Result, error) {
    // Override with A-share rules:
    // - T+1: cannot sell shares bought today
    // - Price limits: ±10% (or ±20% for ChiNext/STAR)
    // - Stamp duty: 0.05% on sell only (2024新政)
    // - Min lot: 100 shares, multiples of 100

    // Wrap strategy to enforce T+1 and price limits
    ...
}
```

- [ ] **Step 2: Implement US engine (simpler rules)**

```go
type USEngine struct {
    Runner
}
// T+2 settlement (doesn't affect bar-by-bar simulation)
// PDT rule: check pattern day trader status
// Fractional shares: no lot size restriction
```

- [ ] **Step 3: Write A-share specific tests**

Test T+1 violation rejection, price limit cap, stamp duty calculation.

- [ ] **Step 4: Commit**

---

## Milestone 5: Workflow Nodes

### Task 11: FactorNode + StrategyNode

**Files:**
- Create: `internal/workflow/nodes/factor.go`
- Create: `internal/workflow/nodes/strategy.go`

- [ ] **Step 1: Implement FactorNode**

```go
type FactorNode struct {
    BaseNode
    FactorName string            `json:"factor_name"`
    Symbols    []string          `json:"symbols"`     // or input port
    Params     map[string]string `json:"params"`
}

func (n *FactorNode) NodeType() string { return "factor" }
func (n *FactorNode) Category() string { return "alpha" }

func (n *FactorNode) Execute(ctx context.Context, inputs map[string]any) (map[string]any, error) {
    // 1. Get PythonBridge from context/dependency injection
    // 2. Load OHLCV from inputs or fetch from MarketDataHub
    // 3. Marshal to Arrow IPC
    // 4. Call PythonBridge.ComputeFactor()
    // 5. Return factor values as output
    ...
}
```

- [ ] **Step 2: Implement StrategyNode**

```go
type StrategyNode struct {
    BaseNode
    SignalType  string  `json:"signal_type"`  // "sma_cross", "rsi_threshold", "custom"
    Params      map[string]string `json:"params"`
}

func (n *StrategyNode) NodeType() string { return "strategy" }
func (n *StrategyNode) Category() string { return "strategy" }
```

- [ ] **Step 3: Register new nodes in register.go**

```go
r.RegisterWithCategory("factor", NewFactorNode, "alpha")
r.RegisterWithCategory("strategy", NewStrategyNode, "strategy")
```

- [ ] **Step 4: Commit**

---

### Task 12: BacktestNode + PerformanceNode

**Files:**
- Create: `internal/workflow/nodes/backtest.go`
- Create: `internal/workflow/nodes/performance.go`

- [ ] **Step 1: Implement BacktestNode**

```go
type BacktestNode struct {
    BaseNode
    Market      string  `json:"market"`       // "CN", "US"
    StartDate   string  `json:"start_date"`
    EndDate     string  `json:"end_date"`
    InitialCash float64 `json:"initial_cash"`
}

func (n *BacktestNode) NodeType() string { return "backtest" }
func (n *BacktestNode) Category() string { return "backtest" }

func (n *BacktestNode) Execute(ctx context.Context, inputs map[string]any) (map[string]any, error) {
    // 1. Parse strategy from inputs (from StrategyNode)
    // 2. Load OHLCV data for universe
    // 3. Run backtest via Runner
    // 4. Return Result as JSON
    ...
}
```

- [ ] **Step 2: Implement PerformanceNode**

Takes BacktestResult as input, computes additional metrics, formats for display.

- [ ] **Step 3: Register nodes**

- [ ] **Step 4: Write workflow integration test**

```go
func TestBacktestWorkflow_Integration(t *testing.T) {
    // Build workflow: DataLoader → StrategyNode → BacktestNode
    // Execute
    // Verify BacktestResult has valid metrics
}
```

- [ ] **Step 5: Commit**

---

## Milestone 6: Frontend Panels

### Task 13: BacktestResultPanel

**Files:**
- Create: `frontend/src/terminal/panels/BacktestResultPanel.vue`

- [ ] **Step 1: Layout design**

```
┌──────────────────────────────────────────┐
│ Backtest: SMA Cross Strategy             │
├──────────────────────────────────────────┤
│ ┌──────────┬──────────┬──────────┬─────┐ │
│ │ Total Ret │ Sharpe   │ Max DD   │ Win │ │
│ │ +15.3%   │ 1.42     │ -8.7%    │ 62% │ │
│ └──────────┴──────────┴──────────┴─────┘ │
│ ┌──────────────────────────────────────┐ │
│ │ Equity Curve (ECharts)               │ │
│ │  /\    /\                             │ │
│ │ /  \  /  \  /\                        │ │
│ │/    \/    \/  \                       │ │
│ └──────────────────────────────────────┘ │
│ ┌──────────────────────────────────────┐ │
│ │ Drawdown Chart                        │ │
│ │ ┌──────────────────────────────────┐ │ │
│ │ │ ▁▁▁▁▁▁▁▁▁▃▁▁▁▁▁▁▁▁▁▁▁▁▁▂▁▁▁▁▁▁ │ │ │
│ │ └──────────────────────────────────┘ │ │
│ └──────────────────────────────────────┘ │
│ ┌──────────────────────────────────────┐ │
│ │ Trade List (scrollable table)         │ │
│ │ Date       | Side | Qty | Price | P&L │ │
│ └──────────────────────────────────────┘ │
└──────────────────────────────────────────┘
```

- [ ] **Step 2: Implement using ECharts**

- Equity curve: line chart with benchmark overlay
- Drawdown: area chart below zero
- Metric cards: summary stats
- Trade table: sorted list with P&L

- [ ] **Step 3: Register in panel registry**

- [ ] **Step 4: Commit**

---

### Task 14: FactorAnalysisPanel + Enhanced Panels

**Files:**
- Create: `frontend/src/terminal/panels/FactorAnalysisPanel.vue`
- Create: `frontend/src/terminal/panels/EquityCurvePanel.vue` (enhance existing or new)
- Create: `frontend/src/terminal/panels/PerformanceMetricsPanel.vue`

- [ ] **Step 1: FactorAnalysisPanel**

- Factor catalog: searchable list of available factors
- Factor metadata: category, description, default params
- Factor preview: compute and show latest values for a symbol
- IC analysis placeholder: table for future IC stats

- [ ] **Step 2: Enhanced panels**

- EquityCurvePanel: accepts BacktestResult, shows multi-strategy overlay
- PerformanceMetricsPanel: metric cards grid

- [ ] **Step 3: Register all panels**

- [ ] **Step 4: Commit**

---

## Milestone 7: Integration + End-to-End

### Task 15: Wire everything into App struct

**Files:**
- Modify: `app.go`

- [ ] **Step 1: Add PythonBridge + BacktestRunner to App**

```go
type App struct {
    // ... existing
    pythonBridge *python.PythonBridge
    backtestCN   *backtest.CNEngine
    backtestUS   *backtest.USEngine
}
```

- [ ] **Step 2: Add Wails-bound functions**

```go
func (a *App) ListFactors() ([]FactorMeta, error) { ... }
func (a *App) ComputeFactor(factorName string, symbol string, start, end string) ([]FactorResult, error) { ... }
func (a *App) RunBacktest(strategyJSON, configJSON string) (*BacktestResultJSON, error) { ... }
func (a *App) GetPythonStatus() (PythonStatus, error) { ... }
```

- [ ] **Step 3: Handle Python unavailable gracefully**

When Python is not running, return a clear error message (not a crash):
```
"Python sidecar is not running. Start it with: python -m src.server"
```

- [ ] **Step 4: Commit**

---

### Task 16: End-to-end integration test + smoke test

- [ ] **Step 1: Go integration test**

```go
func TestFullPipeline_FactorToBacktest(t *testing.T) {
    // 1. Start Python sidecar (or skip if not available)
    // 2. Compute factor via PythonBridge
    // 3. Build strategy from factor
    // 4. Run backtest
    // 5. Verify result has metrics > 0
}
```

- [ ] **Step 2: Smoke test checklist**

```
1. Start Python sidecar: cd python && source venv/bin/activate && python -m src.server
2. Build Go: go build .
3. Run Go tests: go test ./... -count=1
4. Run Python tests: python -m pytest tests/ -x -q
5. List factors via CLI: go run . factor list
6. Compute factor: go run . factor compute momentum_20d --symbol=000001.SZ
7. Run backtest: go run . backtest run --strategy=sma_cross --symbols=000001.SZ
```

- [ ] **Step 3: Update CHANGELOG.md**

- [ ] **Step 4: Update version date if needed**

- [ ] **Step 5: Final commit for Phase 3**

```bash
git add -A
git commit -m "feat(phase3): Python gRPC sidecar + factor engine + backtesting engine"
```

---

## Final Verification Checklist

Before declaring Phase 3 complete:

- [ ] `python -m src.server` starts successfully and serves on localhost:50051
- [ ] `go build .` succeeds
- [ ] `go test ./... -count=1` all pass (Go side)
- [ ] `python -m pytest tests/ -x -q` all pass (Python side)
- [ ] `ListFactors()` returns 25+ factors
- [ ] `ComputeFactor("momentum_20d", "000001.SZ", ...)` returns valid values
- [ ] `BacktestRunner.Run()` produces valid metrics for SMA cross strategy
- [ ] A-share engine enforces T+1, price limits, stamp duty
- [ ] FactorNode + StrategyNode + BacktestNode form a valid workflow DAG
- [ ] BacktestResultPanel renders equity curve + drawdown + metrics + trades
- [ ] FactorAnalysisPanel shows factor catalog with search
- [ ] PythonBridge handles Python unavailable gracefully (no Go crash)
- [ ] CHANGELOG.md updated with Phase 3 entries
- [ ] Version date matches today

---

## Estimated Timeline

| Milestone | Content | Estimate |
|-----------|---------|----------|
| M1 | Python gRPC skeleton (project, proto, server) | 0.5 day |
| M2 | Factor engine (25+ factors) | 1 day |
| M3 | Go PythonBridge (gRPC client) | 0.5 day |
| M4 | Backtesting engine (runner + metrics + CN/US engines) | 1.5 days |
| M5 | Workflow nodes (FactorNode, StrategyNode, BacktestNode) | 1 day |
| M6 | Frontend panels (BacktestResult, FactorAnalysis) | 1 day |
| M7 | Integration + E2E testing | 0.5 day |
| **Total** | | **~6 days** |
