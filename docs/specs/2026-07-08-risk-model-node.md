# RiskModel 工作流节点实现

## Motivation

RiskModelNode 当前是桩实现，返回 "not yet implemented" 错误。实际上 gRPC proto、Python handler、Go MLClient 包装已全部就位，只需补 NodeContext 注入和 Execute 逻辑即可端到端打通。

### 现有资产

| 组件 | 文件 | 状态 |
|------|------|------|
| gRPC Proto | `internal/python/proto/ml.pb.go:960` | ✅ `RiskModelRequest`/`RiskModelResponse` |
| gRPC Client Stub | `internal/python/proto/ml_grpc.pb.go:123` | ✅ `mLServiceClient.RiskModel()` |
| Go MLClient | `internal/python/ml_client.go:129` | ✅ `MLClient.RiskModel()` with timeout |
| Python Handler | `python/src/ml/engine.py:295` | ✅ GARCH/GJR-GARCH/EGARCH + Covariance |
| NodeContext | `internal/workflow/context.go:6` | ❌ 缺少 MLClient 字段 |
| RiskModelNode | `internal/workflow/nodes/risk_model.go:61` | ❌ Execute 是桩 |

## Design

### 数据流

```
RiskModelNode.Execute()
    │
    ├─ 1. 从 inputs["returns_data"] 提取 OHLCV bars → 计算日收益率序列
    │      returns = (close[t] - close[t-1]) / close[t-1]
    │
    ├─ 2. 将收益率编码为 Arrow IPC 格式 (与 Python sidecar 协议一致)
    │
    ├─ 3. 构造 RiskModelRequest{ModelType, ReturnsData, Params}
    │
    ├─ 4. 调用 nctx.MLClient.RiskModel(ctx, req) → gRPC → Python sidecar
    │
    └─ 5. 解析 RiskModelResponse → 输出 volatility, covariance_matrix, model_metrics
```

### 修改文件

| 文件 | 操作 | 说明 |
|------|------|------|
| `internal/workflow/context.go` | 修改 | 添加 `RiskModelService` 接口字段 |
| `internal/workflow/nodes/risk_model.go` | 修改 | 实现 Execute：计算 returns + 调用 gRPC |
| `internal/workflow/nodes/risk_model_test.go` | 新建 | 单元测试（参数验证 + mock） |
| `CHANGELOG.md` | 修改 | 记录变更 |

### NodeContext 变更

```go
// RiskModelService wraps the Python sidecar risk modeling capability.
RiskModelService RiskModelServiceInterface

type RiskModelServiceInterface interface {
    RiskModel(ctx context.Context, req *pb.RiskModelRequest) (*pb.RiskModelResponse, error)
}
```

### Execute 逻辑

```go
func (n *RiskModelNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any, nctx *workflow.NodeContext) (map[string]any, error) {
    modelType := getStringParam(params, "model_type", "garch")
    returnsData := inputs["returns_data"]

    // 1. 从 OHLCV bars 计算日收益率
    returns := computeReturns(returnsData)

    // 2. 编码为 Arrow
    arrowBytes, err := encodeReturnsToArrow(returns)
    if err != nil { ... }

    // 3. 调用 Python sidecar
    if nctx == nil || nctx.RiskModelService == nil {
        // 降级：返回简单波动率（std dev）
        return fallbackVolatility(returns), nil
    }

    req := &pb.RiskModelRequest{
        ModelType:   modelType,
        ReturnsData: arrowBytes,
        Params: map[string]string{
            "p": getStringParam(params, "p", "1"),
            "q": getStringParam(params, "q", "1"),
            "method": getStringParam(params, "method", "ledoit_wolf"),
        },
    }
    resp, err := nctx.RiskModelService.RiskModel(ctx, req)
    ...
}
```

## Acceptance Criteria

- [ ] NodeContext 添加 `RiskModelService` 接口字段
- [ ] RiskModelNode.Execute() 从 OHLCV 输入计算日收益率并调用 gRPC
- [ ] Python sidecar 不可用时，降级返回简单波动率（历史标准差）
- [ ] 支持 garch / gjr_garch / egarch / covariance 四种模型类型
- [ ] `go vet` + `go test ./internal/workflow/...` 通过
- [ ] 向后兼容：已有 workflow 不受影响

## Risks / Trade-offs

- **Arrow 编码依赖**：Go 端需要将 float64 收益率序列编码为 Arrow IPC 格式才能传给 Python。当前 `app_ml.go:AssessRisk` 发送的是空 JSON `[]`（也是 broken），本次一并修复。
- **Python 不可用降级**：RiskModelNode 在无 sidecar 时不报错，降级返回 `std(returns)`。这允许纯 Go 环境下 workflow 继续运行（虽然结果精度较低）。
- **不修改 AssessRisk**：`app_ml.go:AssessRisk` 发送空数据的 bug 留待后续修复，本次聚焦 workflow node。
