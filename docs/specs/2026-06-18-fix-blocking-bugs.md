# Fix 4 Blocking Bugs — Workflow Params / OHLCVBar / ML Bridge / Interval Case

> 审查报告 ref: 问题.md #1, #2, #3, #5

## Motivation

4 条阻断性 bug 使核心功能完全不可用：

1. **工作流参数永远 nil** — 任何有上游边的节点，用户设置的参数（如 `sma period=2`）被忽略，实际跑的是硬编码默认值
2. **三套互不兼容的 OHLCVBar** — `data_loader → backtest` 主回测流水线永远返回 "no OHLCV data"
3. **ML bridge 没接线** — `train_model`/`predict`/`alpha_mining` 三个 ML 节点永远报 "PythonBridge not set"
4. **interval 大小写不归一** — 前端发 `1d`，Tencent/Baidu 适配器只认 `1D`，A 股 K 线从 UI 拉不到

这 4 条 bug 都是**小改动高回报**——每条的修复范围在 1-5 个文件内，但修完后核心工作流（data_loader→指标→backtest）、ML 训练预测、A 股 K 线 UI 全部复活。

## Design

### #1: 工作流参数传递修复

**根因**：[engine.go:154](internal/workflow/engine.go#L154) 写死 `node.Execute(ctx, inputs, nil)`，且 [engine.go:138-142](internal/workflow/engine.go#L138) 只在**无上游边**时才将 params 倒进 inputs。

**修复**：
- `engine.go:154`: 将 `nil` 改为 `nodeInstance.Params`
- `engine.go:138-142`: 删除这段"params → inputs 兜底"逻辑（各节点 Execute 已经有 `getStringParam(params, ...)` / `getFloatParam(params, ...)` 从 params 取值；保留会导致上游同名 port 输出来不及覆盖 params 的同名 key）
- 事实上当前逻辑是 `if len(inputs) == 0` 才把 params 倒进 inputs——但有上游边的节点 inputs 非空，params 就根本不参与。改为直接传 params 给 Execute，由节点自己处理 params vs inputs 的优先级（节点已用 `getXxxParam(params, ...)` 处理）

**影响文件**：
- `internal/workflow/engine.go` — 1 行改动 + 删除 5 行

### #2: OHLCVBar 类型统一

**根因**：项目存在 3 个互不兼容的 `OHLCVBar` struct：

| 位置 | 字段差异 |
|------|---------|
| `internal/workflow/nodes/data_loader.go:15` | **无 Symbol 字段** |
| `internal/market/types.go:23` | 有 Symbol 字段 |
| `internal/trading/types.go:83` | 有 Symbol 字段 |

`data_loader` 输出 `[]nodes.OHLCVBar`（无 Symbol），`backtest` 断言 `[]trading.OHLCVBar`（有 Symbol）→ 断言失败 → bars 永远空。

**修复**：
1. 删除 `nodes.OHLCVBar`，`data_loader` 直接输出 `[]market.OHLCVBar`（market 包是所有适配器统一产出的类型）
2. `backtest.Execute` 接收 `[]market.OHLCVBar` 后转换为 `[]trading.OHLCVBar`（trading 包是引擎内部类型，保留但通过显式转换）
3. 或者更简洁：改 `backtest.Execute` 接受 `[]market.OHLCVBar`，引擎内部转换
4. SMA/MACD/RSI 等指标节点接收 `[]float64`（close prices），从 inputs 提取时不再依赖具体 OHLCVBar 类型

**影响文件**：
- `internal/workflow/nodes/data_loader.go` — 删除 `OHLCVBar` struct，导入 market，返回 `[]market.OHLCVBar`
- `internal/workflow/nodes/backtest.go` — 改用 `[]market.OHLCVBar` 或加转换函数
- `internal/backtest/engine_cn.go` 等 — 各引擎接受 `[]trading.OHLCVBar`，入口处转换
- `internal/workflow/nodes/sma.go`, `macd.go`, `rsi.go` 等 — 确认从 inputs 取 close prices 的逻辑正确（如果它们取的是 `[]float64`，则不受影响）

### #3: ML Bridge 接线

**根因**：`app.go:startup()` 创建了 `PythonBridge`（line 66-72），但**从不调用**：
- `nodes.SetPythonBridge(a.bridge)` — 定义在 [train_model.go:18](internal/workflow/nodes/train_model.go#L18)
- `nodes.SetModelRegistry(r)` — 定义在 [train_model.go:26](internal/workflow/nodes/train_model.go#L26)

导致 `train_model`、`predict`、`alpha_mining` 三个节点的 Execute 方法在 bridge == nil 检查时直接返回错误。

**修复**：
- 在 `app.go:startup()` 中，bridge 初始化成功后，添加：
  ```go
  nodes.SetPythonBridge(a.bridge)
  ```
- 初始化 `ml.ModelRegistry`（如果不存在则创建内存版），调用：
  ```go
  nodes.SetModelRegistry(ml.NewModelRegistry())
  ```

**影响文件**：
- `app.go` — 添加 2-3 行
- 可能需要创建 `internal/ml/registry.go`（如果 ModelRegistry 还没有 New 函数）

### #5: Interval 大小写归一化

**根因**：前端 `CandlestickPanel.vue:135` 用小写 `['5m', '1h', '1d', '1w']`，而 Go 端各适配器的 interval map 不一致：

| 适配器 | 接受的 interval |
|--------|----------------|
| Tencent (`tencent.go:67`) | `"1D"`, `"1W"`, `"1M"` (大写) |
| Baidu (`baidu.go:80`) | 只检查 `!= "1D"` (大写) |
| EastMoney (`eastmoney.go:98`) | `"1w"`, `"1W"` (混用) |
| Yahoo (`yahoo.go:78`) | 直接传递给 URL（小写 OK） |
| Mootdx (`mootdx.go:99`) | 原样传给 Python sidecar |

**修复**：在最外层（`market.AdapterRegistry.FetchOHLCVWithFallback` 或统一的 normalize 函数）将 interval 标准化后再传入各适配器。同时各适配器内部也做一次 `strings.ToUpper` 兜底。

标准化映射：
```
5m → 5m / 1h → 1h / 1d → 1D / 1w → 1W / 1M → 1M
```

具体方案：在 `market` 包中添加 `NormalizeInterval(interval string) string` 函数，在 `AdapterRegistry.FetchOHLCVWithFallback` 调用各适配器前先 normalize。各适配器内部也用 `strings.ToUpper` 做二次防御。

**影响文件**：
- `internal/market/registry.go` — 调用 normalize
- `internal/market/interval.go` (新建) — `NormalizeInterval` 函数
- `internal/market/adapters/tencent.go` — 加 `strings.ToUpper(interval)` 防御
- `internal/market/adapters/baidu.go` — 加 `strings.ToUpper(interval)` 防御
- `internal/market/adapters/eastmoney.go` — 统一用大写比较
- `frontend/src/terminal/panels/CandlestickPanel.vue` — 可选：发请求时也可以用大写（前端防御）

## Acceptance Criteria

- [ ] **#1**: 创建 `data_loader → sma(period=5) → backtest` 工作流，验证 sma 的 period 参数（5）被正确传递到 Execute 的 params 参数
- [ ] **#1**: 工作流引擎层有单元测试覆盖 params 传递
- [ ] **#2**: `data_loader → backtest` 流水线由 CSV 文件加载 bar 数据后，backtest 能正确接收到 bars 并产出结果
- [ ] **#2**: `nodes.OHLCVBar` 类型已删除，所有引用点编译通过
- [ ] **#2**: `go test ./internal/workflow/...` 全绿
- [ ] **#3**: 启动应用后 `train_model`/`predict`/`alpha_mining` 节点不再报 "PythonBridge not set"
- [ ] **#3**: 若 sidecar 未连，节点报清晰错误而非 nil pointer
- [ ] **#5**: 前端 `CandlestickPanel` 选择 `1d` 后，Go 端能正常返回 A 股日 K 数据
- [ ] **#5**: 所有适配器的 interval 处理有测试覆盖（`"1d"`, `"1D"`, `"1w"`, `"1W"` 都正确映射）
- [ ] **#5**: 回归：minute-level intervals (`5m`, `1h`) 对 mootdx/Yahoo/Binance 正常工作
- [ ] `go build ./...` 编译通过
- [ ] `go test ./...` 全绿
- [ ] `CHANGELOG.md` 更新

## Risks / Trade-offs

- **#1 风险低**：params 一直被传 nil，从未在生产环境用过；改为传实际 params 后各节点的 `getXxxParam` 已有默认值兜底，不会因多传了 params 而崩溃
- **#2 权衡**：统一用 `market.OHLCVBar` 使 market 包成为"黄金源"。`trading.OHLCVBar` 保留作为引擎内部类型，通过显式转换隔离 engine ↔ adapter。这是最小改动方案（不改 engine 内部逻辑）
- **#3 风险低**：`SetPythonBridge` 是幂等函数调用，bridge==nil 时节点已有 nil check 兜底
- **#5 权衡**：在 registry 层 normalize 是最干净的方案（一处修改惠及所有适配器），但各适配器内部也加 `ToUpper` 作为防御层——万一有代码绕过 registry 直接调适配器也不炸
