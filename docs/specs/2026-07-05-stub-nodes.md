# Implement Stub Workflow Nodes — 4 Remaining Stubs

## Motivation

Workflow 引擎注册了 16 类节点，其中 4 个是 stub（返回硬编码错误或空数据）：

| 节点 | 文件 | 当前行为 |
|------|------|---------|
| `risk_metrics` | `internal/workflow/nodes/risk_metrics.go` | `return nil, errors.New("not implemented")` |
| `json_parse` | `internal/workflow/nodes/json_parse.go` | 返回空 map |
| `http_request` | `internal/workflow/nodes/http_request.go` | `return nil, errors.New("not implemented")` |
| `allocation` | `internal/workflow/nodes/allocation.go` | 返回空 map |

用户拖拽这些节点到画布后无法使用，体验差。

## Design

### 1. risk_metrics — 风险指标计算

接收一个持仓/组合配置输入端口，输出风险指标。

**输入**：
- `portfolio` — `map[string]any` 格式的组合持仓，如 `{"AAPL": 100, "MSFT": 200}`

**输出**：
- `metrics` — 包含夏普比、最大回撤、波动率、VaR 95% 的 map

**实现**：从 `internal/trading/risk_pipeline.go` 复用 `CheckDrawdown` 逻辑，计算静态指标。Python sidecar 不可用时也正常工作（纯 Go）。

**Node definition**:
```go
type RiskMetricsNode struct {
    BaseNode
    RiskFreeRate float64 `json:"riskFreeRate"`
}

func (n *RiskMetricsNode) NodeType() string { return "risk_metrics" }
func (n *RiskMetricsNode) Category() string { return "analysis" }

func (n *RiskMetricsNode) Execute(ctx context.Context, inputs map[string]any) (map[string]any, error) {
    pf, ok := inputs["portfolio"]
    if !ok {
        return nil, fmt.Errorf("risk_metrics: missing portfolio input")
    }
    // 计算基本指标
    metrics := calculateBasicStats(pf)
    return map[string]any{"metrics": metrics}, nil
}
```

### 2. json_parse — JSON 解析

接收字符串输入，解析为结构化数据。

**输入**：
- `json_string` — 字符串

**输出**：
- `parsed` — `map[string]any`

```go
type JSONParseNode struct {
    BaseNode
}

func (n *JSONParseNode) Execute(ctx context.Context, inputs map[string]any) (map[string]any, error) {
    raw, ok := inputs["json_string"]
    if !ok {
        return nil, fmt.Errorf("json_parse: missing json_string input")
    }
    str, ok := raw.(string)
    if !ok {
        return nil, fmt.Errorf("json_parse: json_string must be string, got %T", raw)
    }
    var result map[string]any
    if err := json.Unmarshal([]byte(str), &result); err != nil {
        return nil, fmt.Errorf("json_parse: unmarshal error: %w", err)
    }
    return map[string]any{"parsed": result}, nil
}
```

### 3. http_request — HTTP 请求

发送 HTTP 请求并返回响应。

**输入**：
- `url` — 字符串
- `method` — 字符串（GET/POST/PUT/DELETE）
- `headers` — 可选，map[string]string
- `body` — 可选，字符串

**输出**：
- `status_code` — int
- `body` — string
- `headers` — map[string]string

**安全约束**：
- 默认超时 10 秒
- 禁止内网地址（127.0.0.1, 10.x, 172.16-31.x, 192.168.x）避免 SSRF
- 仅允许 HTTP/HTTPS 协议

```go
type HTTPRequestNode struct {
    BaseNode
    Timeout int `json:"timeout"` // seconds, 0 = default 10
}
```

### 4. allocation — 资金分配

根据风险预算或等权分配资金到多个标的。

**输入**：
- `symbols` — `[]string`
- `total_capital` — float64
- `method` — `"equal"` | `"risk_parity"`

**输出**：
- `allocations` — `map[string]float64`（symbol → 金额）
- `weights` — `map[string]float64`（symbol → 权重）

**实现**：
- `equal`: 均分
- `risk_parity`: 简单波动率倒数加权（纯 Go 实现，不依赖 Python）

## Acceptance Criteria

- [ ] risk_metrics 节点返回夏普比、最大回撤、波动率、VaR
- [ ] json_parse 节点解析合法 JSON，对非法 JSON 返回 error
- [ ] http_request 节点支持 GET/POST，有 SSRF 防护，超时正常
- [ ] allocation 节点等权分配正确（1/n），风险平价四舍五入到分
- [ ] 四节点都通过 `go test ./internal/workflow/... -count=1`
- [ ] 节点在 vue-flow 画布中显示正确（但前端无需改动 — 已有节点注册）

## Risks / Trade-offs

- http_request 的 SSRF 防护需要解析 url 为 net/url，检查 IP 范围。内网用户如果需要访问内网 API 会受限。权衡：加一个 `allow_private: bool` 参数默认 false。
- risk_metrics 的 Python 因子（如 Qlib）不可用，纯 Go 实现只做基本统计。用户如果需要高级计算，可以串联 Python 节点。
