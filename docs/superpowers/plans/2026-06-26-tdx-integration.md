# 通达信能力集成 — 实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended).

**Goal:** 移植某 TDX 协议项目（MIT 协议）的核心能力到 QuantFlow：34 个指标、缠论、策略库、选股扫描、回测增强、MAC 协议。

**Architecture:** Python sidecar 新增 `indicators/` + `chanlun/` + `strategies/` 三个模块，gRPC 暴露；Go 端新增对应 workflow nodes 和数据适配器。

**Tech Stack:** Python 3.12, gRPC, pandas, Go 1.22+, Vue 3 + vue-echarts

## Global Constraints

- 所有移植代码遵循 MIT 协议，保留原作者署名
- 指标计算统一走 Python sidecar gRPC，Go 不重复实现
- Workflow node 命名规范：`indicator_xxx` (如 `indicator_kdj`)
- 策略文件放在 `python/src/strategies/` 下
- npm run build + go build 每一步后通过

---

### Task 1: 移植 34 个技术指标到 Python sidecar

**Files:**
- Create: `python/src/indicators/__init__.py` — 指标注册表
- Create: `python/src/indicators/macd.py, kdj.py, dmi.py, atr.py, wr.py, cci.py, bias.py, obv.py, mfi.py, sar.py, vwap.py, boll.py, rsi.py, ema.py, sma.py, ma.py, zhuoyao.py, aroon.py, asi.py, brar.py, cr.py, dpo.py, emv.py, ktn.py, mass.py, mtm.py, psy.py, roc.py, trix.py, xsii.py, bbi.py, dfma.py, expma.py, taq.py`
- Create: `python/proto/indicator.proto` — gRPC 定义
- Modify: `python/src/server.py` — 注册 ComputeIndicators 服务

**Interfaces:**
- Produces: `compute_indicators(df: pd.DataFrame, indicator_names: List[str], params: Dict) -> Dict[str, np.ndarray]`

- [ ] **Step 1: 从某外部项目复制指标源码**

```bash
cp -r /path/to/external-tdx-project/indicators/*.py /Volumes/etx/coding/rebuild/quantflow/python/src/indicators/
```

创建 `__init__.py`：
```python
"""Technical indicator registry — ported from 某 TDX 项目 (MIT)."""
from . import macd, kdj, dmi, atr, wr, cci, bias, obv, mfi, sar, vwap, boll, rsi, ema, sma, ma
from . import zhuoyao, aroon, asi, brar, cr, dpo, emv, ktn, mass, mtm, psy, roc, trix, xsii
from . import bbi, dfma, expma, taq

def compute_indicators(df, names, params={}):
    """Compute multiple indicators from OHLCV DataFrame."""
    results = {}
    for name in names:
        if hasattr(eval(name), 'compute'):
            results[name.upper()] = eval(name).compute(df, **params.get(name, {}))
    return results
```

- [ ] **Step 2: 定义 gRPC proto**

```protobuf
// python/proto/indicator.proto
service IndicatorService {
  rpc ComputeIndicators(IndicatorRequest) returns (IndicatorResponse);
}

message IndicatorRequest {
  bytes ohlcv_data = 1;  // Arrow IPC bytes
  repeated string indicator_names = 2;
  map<string, string> params = 3;
}

message IndicatorResponse {
  map<string, bytes> results = 1;  // indicator_name -> Arrow IPC bytes
}
```

- [ ] **Step 3: 生成 proto 代码 + 注册服务**

```bash
cd /Volumes/etx/coding/rebuild/quantflow/python
python -m grpc_tools.protoc -Iproto --python_out=src/proto --grpc_python_out=src/proto proto/indicator.proto
```

在 `server.py` 中注册。

- [ ] **Step 4: 构建验证**

```bash
cd /Volumes/etx/coding/rebuild/quantflow
go build -o /dev/null .
```

---

### Task 2: Go 端新增 19 个 Indicator Workflow Nodes

**Files:**
- Create: `internal/workflow/nodes/indicator_kdj.go` 等 19 个文件

- [ ] **Step 1: 为每个指标创建 workflow node**

每个 node 模板：
```go
// indicator_kdj.go
func (n *IndicatorKDJNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any, nctx *workflow.NodeContext) (map[string]any, error) {
    ohlcv := inputs["ohlcv_data"]
    if nctx == nil || nctx.Bridge == nil { return nil, fmt.Errorf("python bridge required") }
    bridge := nctx.Bridge.(*python.PythonBridge)
    result, err := bridge.ComputeIndicator(ctx, ohlcv, "KDJ", params)
    return map[string]any{"kdj_k": result["K"], "kdj_d": result["D"], "kdj_j": result["J"]}, err
}
```

重复 19 次（KDJ, DMI, ATR, WR, CCI, BIAS, BIAS_SIGNAL, OBV, MFI, SAR, VWAP, ZHUOYAO, AROON, ASI, BRAR, DPO, EMV, MASS, PSY, ROC, TRIX, BBI）。

- [ ] **Step 2: 注册 nodes**

在 `internal/workflow/nodes/registry.go` 中注册所有新节点。

- [ ] **Step 3: 构建**

```bash
go build -o /dev/null .
```

---

### Task 3: K 线复权

**Files:**
- Modify: `internal/market/adapters/mootdx.go`

- [ ] **Step 1: 加复权参数**

```go
func (a *MootdxAdapter) FetchOHLCV(ctx context.Context, symbol, interval string, start, end int64, fqfactor string) ([]market.OHLCVBar, error) {
    // fqfactor: "" (不复权), "qfq" (前复权), "hfq" (后复权)
    req := &pb.DataRequest{
        Symbol: symbol, Interval: interval, Start: start, End: end,
        Fqfactor: fqfactor,
    }
    // ...
}
```

- [ ] **Step 2: 更新 app.go fetchOHLCV 调用**

添加可选复权参数，默认 `"qfq"`。

---

### Task 4: 缠论分析模块

**Files:**
- Create: `python/src/chanlun/` — 从某外部项目移植全部文件
- Create: `python/proto/chanlun.proto`
- Create: Go `internal/workflow/nodes/chanlun.go`

- [ ] **Step 1: 移植缠论代码**

```bash
cp -r /path/to/external-tdx-project/chanlun/* /Volumes/etx/coding/rebuild/quantflow/python/src/chanlun/
```

- [ ] **Step 2: 定义 gRPC proto + Go node**

`ChanlunNode` 输出：分型列表、笔列表、中枢列表、买卖点列表、背驰信号。

---

### Task 5: 内置策略库 + 选股扫描

**Files:**
- Create: `python/src/strategies/` — 16 个策略文件
- Create: Go `internal/workflow/nodes/strategy_scan.go`
- Create: Go `internal/workflow/nodes/batch_backtest.go`

- [ ] **Step 1: 移植策略**

```bash
cp -r /path/to/external-tdx-project/strategies/* /Volumes/etx/coding/rebuild/quantflow/python/src/strategies/
```

- [ ] **Step 2: StrategyScanNode**

全市场并发扫描，ProcessPoolExecutor 4 进程，按夏普排名返回 top N。

---

### Task 6: 回测增强

**Files:**
- Modify: `internal/backtest/engine_cn.go` — 滑点模型 + 执行仿真
- Create: `internal/backtest/portfolio_runner.go` — 组合回测

- [ ] **Step 1: 滑点模型**

```go
type SlippageModel interface {
    Apply(order Order, bar OHLCVBar) float64
}
type SquareRootSlippage struct{ Base, VolRatio float64 }
```

- [ ] **Step 2: 执行仿真**

`TWAPExecution`, `VWAPExecution` 拆单逻辑。

---

### Task 7: 因子分析增强

**Files:**
- Modify: `python/src/factor/engine.py` — IC/分层/衰减/预处理

- [ ] **Step 1: 移植某外部项目 factor_analysis 模块**

IC 分析、quantile 分层回测、IC 衰减、去极值/中性化/标准化。

---

### Task 8: MAC 协议适配器

**Files:**
- Create: `internal/market/adapters/mac.go` — 原生 TCP 连接 TDX MAC 协议

- [ ] **Step 1: 实现 MAC 客户端**

直接 TCP 连接端口 7709，解析二进制协议，提供：板块排名、资金流向、集合竞价、异动监控、多日分时。

---

### Task 9: 离线数据同步

**Files:**
- Create: `internal/market/offline_sync.go`

- [ ] **Step 1: 日线同步 + .day 文件读写**

全市场日线下载到 `data/tdx/` 目录，读写通达信 `.day` 格式。

---

### Task 10: 前端面板（缠论 + 选股 + 指标）

**Files:**
- Create: `frontend/src/terminal/panels/ChanlunPanel.vue`
- Create: `frontend/src/terminal/panels/StockScannerPanel.vue`
- Create: `frontend/src/terminal/panels/IndicatorPanel.vue`

---

### Task 11: 全栈打包验证

```bash
cd frontend && npm run build -q
cd .. && go build -o build/quantflow .
rsync -a --delete python/ build/python/
```
