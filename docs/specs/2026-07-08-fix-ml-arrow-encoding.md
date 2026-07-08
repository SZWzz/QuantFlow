# Fix ML Pipeline — Arrow IPC Encoding + AssessRisk

## Motivation

Go→Python ML gRPC 管道存在两个问题：

1. **Arrow 编码不匹配**：Go 侧全部 ML 调用发送 JSON，但 Python 侧 `_decode_arrow()` 只接受 PyArrow IPC 格式。当前 Python 捕获 Arrow 解码异常后返回空结果，所有 ML 功能（Train/Predict/Evaluate/AlphaMining/RiskModel/RLPredict）实际上不工作。

2. **AssessRisk 发送空数据**：`app_ml.go:AssessRisk()` 发送空 JSON `[]`，从未返回有效风险指标。

### 根因

Go 项目未引入 Apache Arrow 库，无法生成 Arrow IPC 字节。Python 侧严格期望 Arrow 格式。

## Design

### 方案：Python 侧添加 JSON fallback

不引入 Arrow Go 依赖。在 Python `_decode_arrow()` 中添加 JSON fallback：尝试 Arrow IPC → 失败则 JSON 解析。Go 侧继续发送 JSON，Python 正确解析。

### 数据流（修复后）

```
Go: json.Marshal(returns) → RiskModelRequest{ReturnsData: jsonBytes}
    ↓ gRPC
Python: _decode_arrow(data)
    ├─ try pyarrow.ipc.open_stream(data) → Table  ✅ (future Arrow)
    └─ except: json.loads(data) → pd.DataFrame → Table  ✅ (current JSON)
    ↓
GARCH engine / Covariance engine 正常计算
```

### 修改文件

| 文件 | 操作 | 说明 |
|------|------|------|
| `python/src/ml/engine.py` | 修改 | `_decode_arrow()` 添加 JSON fallback |
| `app_ml.go` | 修改 | `AssessRisk` 从 OHLCV 数据计算 returns（不再发空数组） |
| `internal/workflow/nodes/risk_model.go` | 修改 | 清理占位逻辑，更新 Arrow TODO 注释 |
| `CHANGELOG.md` | 修改 | 记录变更 |

### Python 侧变更

```python
def _decode_arrow(self, data: bytes) -> pa.Table:
    # Try Arrow IPC format first (future Go Arrow support).
    try:
        reader = pa.ipc.open_stream(data)
        return reader.read_all()
    except Exception:
        pass
    
    # Fallback: JSON-encoded data from Go.
    try:
        import json
        parsed = json.loads(data)
        if isinstance(parsed, list):
            if len(parsed) == 0:
                return pa.table({})
            # Single list of floats → one-column table
            if isinstance(parsed[0], (int, float)):
                return pa.table({"values": pa.array(parsed, type=pa.float64())})
            # List of lists → multi-column
            if isinstance(parsed[0], list):
                cols = {}
                for i, col in enumerate(parsed):
                    cols[f"col_{i}"] = pa.array(col, type=pa.float64())
                return pa.table(cols)
            # List of dicts → columns by key
            if isinstance(parsed[0], dict):
                return pa.Table.from_pandas(pd.DataFrame(parsed))
        if isinstance(parsed, dict):
            return pa.table({k: pa.array([v] if not isinstance(v, list) else v, type=pa.float64())
                             for k, v in parsed.items()})
    except Exception:
        pass
    
    return pa.table({})
```

### Go AssessRisk 修复

```go
func (a *App) AssessRisk(symbols []string, modelType string) (map[string]interface{}, error) {
    if a.bridge == nil {
        return nil, nil
    }
    // Fetch OHLCV data and compute returns
    returns := a.computeReturnsForSymbols(symbols)
    
    returnsJSON, err := json.Marshal(returns)
    if err != nil {
        return nil, fmt.Errorf("assess_risk: marshal returns: %w", err)
    }
    client := python.NewMLClient(a.bridge)
    req := &pb.RiskModelRequest{
        ModelType:   modelType,
        ReturnsData: returnsJSON,
        Params:      map[string]string{"symbols": strings.Join(symbols, ",")},
    }
    ...
}
```

## Acceptance Criteria

- [ ] Python `_decode_arrow()` 支持 JSON float list → single-column Arrow Table
- [ ] Python `_decode_arrow()` 支持 JSON list-of-lists → multi-column Arrow Table
- [ ] 现有 Arrow IPC 路径不受影响（向后兼容）
- [ ] `AssessRisk` 发送实际收益率数据而非空数组
- [ ] RiskModelNode 通过 JSON 路径能正确调用 Python GARCH/协方差（当 sidecar 可用时）
- [ ] `go test ./...` 通过（无 Go 侧回归）
- [ ] `python -m pytest tests/ -x -q` 通过（无 Python 侧回归）

## Risks / Trade-offs

- **JSON fallback 性能**：JSON 比 Arrow IPC 慢，但 ML 调用频率低（用户手动触发），不影响实时性能
- **不引入 Arrow Go 依赖**：保持依赖轻量，未来如需高性能可后续添加 Arrow Go 库
- **空数据兼容**：Python 返回空 Table 时不报错，Go 侧已有的降级逻辑继续工作
