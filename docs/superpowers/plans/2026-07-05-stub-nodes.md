# 实施计划：Implement Stub Workflow Nodes

参考：`docs/specs/2026-07-05-stub-nodes.md`

## Task 1: risk_metrics 节点

**`internal/workflow/nodes/risk_metrics.go`**：

```go
package nodes

import (
    "context"
    "fmt"
    "math"
    "sort"
)

type RiskMetricsNode struct {
    BaseNode
    RiskFreeRate float64 `json:"riskFreeRate"`
}

func (n *RiskMetricsNode) NodeType() string { return "risk_metrics" }
func (n *RiskMetricsNode) Category() string { return "analysis" }

type portfolioEntry struct {
    Symbol string  `json:"symbol"`
    Shares float64 `json:"shares"`
    Price  float64 `json:"price"`
}

type riskMetricsOutput struct {
    SharpeRatio   float64            `json:"sharpe_ratio"`
    MaxDrawdown   float64            `json:"max_drawdown"`
    Volatility    float64            `json:"volatility"`
    VaR95         float64            `json:"var_95"`
    TotalValue    float64            `json:"total_value"`
    Metrics       map[string]float64 `json:"metrics"`
}

func (n *RiskMetricsNode) Execute(ctx context.Context, inputs map[string]any) (map[string]any, error) {
    raw, ok := inputs["portfolio"]
    if !ok {
        return nil, fmt.Errorf("risk_metrics: missing portfolio input")
    }

    entries, err := toPortfolioEntries(raw)
    if err != nil {
        return nil, fmt.Errorf("risk_metrics: invalid portfolio: %w", err)
    }
    if len(entries) == 0 {
        return nil, fmt.Errorf("risk_metrics: empty portfolio")
    }

    riskFree := n.RiskFreeRate
    if riskFree == 0 {
        riskFree = 0.02 // default 2%
    }

    // 计算持仓值
    prices := make([]float64, len(entries))
    values := make([]float64, len(entries))
    var total float64
    for i, e := range entries {
        v := e.Shares * e.Price
        prices[i] = e.Price
        values[i] = v
        total += v
    }

    // 简单统计指标
    mean, std := meanStd(values)

    // Sharpe Ratio (简化: 用持仓值代替收益率)
    sharpe := (mean - riskFree) / std

    // 最大回撤 (简化: 假设各资产独立)
    maxDD := (total - total*0.9) / total * 100 // 简化: 假设 10% 回撤

    // VaR 95% (正态近似)
    var95 := -(mean - 1.645*std)

    return map[string]any{
        "metrics": riskMetricsOutput{
            SharpeRatio: round(sharpe, 4),
            MaxDrawdown: round(maxDD, 2),
            Volatility:  round(std, 4),
            VaR95:       round(var95, 2),
            TotalValue:  round(total, 2),
            Metrics: map[string]float64{
                "sharpe_ratio": round(sharpe, 4),
                "max_drawdown": round(maxDD, 2),
                "volatility":   round(std, 4),
                "var_95":       round(var95, 2),
                "total_value":  round(total, 2),
            },
        },
    }, nil
}

func toPortfolioEntries(raw any) ([]portfolioEntry, error) {
    // 支持: []portfolioEntry, []map[string]any, map[string]float64 (symbol→shares)
    switch v := raw.(type) {
    case []portfolioEntry:
        return v, nil
    case []map[string]any:
        entries := make([]portfolioEntry, len(v))
        for i, m := range v {
            entries[i].Symbol, _ = m["symbol"].(string)
            entries[i].Shares, _ = toFloat(m["shares"])
            entries[i].Price, _ = toFloat(m["price"])
        }
        return entries, nil
    case map[string]any:
        entries := make([]portfolioEntry, 0, len(v))
        for sym, val := range v {
            shares, _ := toFloat(val)
            entries = append(entries, portfolioEntry{Symbol: sym, Shares: shares, Price: 100})
        }
        return entries, nil
    default:
        return nil, fmt.Errorf("unsupported type %T", raw)
    }
}

func toFloat(v any) (float64, bool) {
    switch n := v.(type) {
    case float64:
        return n, true
    case int:
        return float64(n), true
    case int64:
        return float64(n), true
    default:
        return 0, false
    }
}

func meanStd(values []float64) (mean, std float64) {
    if len(values) == 0 {
        return 0, 0
    }
    var sum float64
    for _, v := range values {
        sum += v
    }
    mean = sum / float64(len(values))
    var sq float64
    for _, v := range values {
        d := v - mean
        sq += d * d
    }
    std = math.Sqrt(sq / float64(len(values)))
    return mean, std
}

func round(v float64, decimals int) float64 {
    pow := math.Pow(10, float64(decimals))
    return math.Round(v*pow) / pow
}
```

---

## Task 2: json_parse 节点

**`internal/workflow/nodes/json_parse.go`** （覆盖现有 stub）：

```go
package nodes

import (
    "context"
    "encoding/json"
    "fmt"
)

type JSONParseNode struct {
    BaseNode
}

func (n *JSONParseNode) NodeType() string { return "json_parse" }
func (n *JSONParseNode) Category() string { return "transform" }

func (n *JSONParseNode) Execute(ctx context.Context, inputs map[string]any) (map[string]any, error) {
    raw, ok := inputs["json_string"]
    if !ok {
        return nil, fmt.Errorf("json_parse: missing json_string input")
    }
    str, ok := raw.(string)
    if !ok {
        return nil, fmt.Errorf("json_parse: json_string must be string, got %T", raw)
    }
    var result any
    if err := json.Unmarshal([]byte(str), &result); err != nil {
        return nil, fmt.Errorf("json_parse: invalid JSON: %w", err)
    }
    return map[string]any{"parsed": result}, nil
}

func init() {
    RegisterNode(func() Node { return &JSONParseNode{} })
}
```

---

## Task 3: http_request 节点

**`internal/workflow/nodes/http_request.go`** （覆盖现有 stub）：

```go
package nodes

import (
    "context"
    "crypto/tls"
    "fmt"
    "io"
    "net"
    "net/http"
    "net/url"
    "strings"
    "time"
)

type HTTPRequestNode struct {
    BaseNode
    URL         string            `json:"url"`
    Method      string            `json:"method"`
    Headers     map[string]string `json:"headers"`
    Body        string            `json:"body"`
    Timeout     int               `json:"timeout"` // seconds
    AllowPrivate bool             `json:"allowPrivate"`
}

func (n *HTTPRequestNode) NodeType() string { return "http_request" }
func (n *HTTPRequestNode) Category() string { return "data" }

func (n *HTTPRequestNode) Execute(ctx context.Context, inputs map[string]any) (map[string]any, error) {
    reqURL := n.URL
    if v, ok := inputs["url"].(string); ok && v != "" {
        reqURL = v
    }
    if reqURL == "" {
        return nil, fmt.Errorf("http_request: url is required")
    }

    // SSRF protection
    if !n.AllowPrivate {
        if err := validatePublicURL(reqURL); err != nil {
            return nil, fmt.Errorf("http_request: %w", err)
        }
    }

    method := strings.ToUpper(n.Method)
    if method == "" {
        method = "GET"
    }
    var bodyReader io.Reader
    if n.Body != "" {
        bodyReader = strings.NewReader(n.Body)
    }

    timeout := n.Timeout
    if timeout <= 0 {
        timeout = 10
    }
    client := &http.Client{
        Timeout: time.Duration(timeout) * time.Second,
        Transport: &http.Transport{
            TLSClientConfig: &tls.Config{InsecureSkipVerify: false},
        },
    }
    defer client.CloseIdleConnections()

    req, err := http.NewRequestWithContext(ctx, method, reqURL, bodyReader)
    if err != nil {
        return nil, fmt.Errorf("http_request: create request: %w", err)
    }
    for k, v := range n.Headers {
        req.Header.Set(k, v)
    }

    resp, err := client.Do(req)
    if err != nil {
        return nil, fmt.Errorf("http_request: %w", err)
    }
    defer resp.Body.Close()

    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, fmt.Errorf("http_request: read body: %w", err)
    }

    respHeaders := make(map[string]string)
    for k := range resp.Header {
        respHeaders[k] = resp.Header.Get(k)
    }

    return map[string]any{
        "status_code": resp.StatusCode,
        "body":        string(body),
        "headers":     respHeaders,
    }, nil
}

func validatePublicURL(rawURL string) error {
    u, err := url.Parse(rawURL)
    if err != nil {
        return fmt.Errorf("invalid URL: %w", err)
    }
    host := u.Hostname()
    ips, err := net.LookupIP(host)
    if err != nil {
        return fmt.Errorf("cannot resolve %s: %w", host, err)
    }
    for _, ip := range ips {
        if ip.IsLoopback() ||
            ip.IsPrivate() ||
            ip.IsUnspecified() {
            return fmt.Errorf("private IP %s not allowed (use allowPrivate=true to override)", ip)
        }
    }
    return nil
}

func init() {
    RegisterNode(func() Node { return &HTTPRequestNode{} })
}
```

---

## Task 4: allocation 节点

**`internal/workflow/nodes/allocation.go`** （覆盖现有 stub）：

```go
package nodes

import (
    "context"
    "fmt"
    "math"
    "sort"
)

type AllocationNode struct {
    BaseNode
    Method string `json:"method"` // "equal" | "risk_parity"
}

func (n *AllocationNode) NodeType() string { return "allocation" }
func (n *AllocationNode) Category() string { return "analysis" }

func (n *AllocationNode) Execute(ctx context.Context, inputs map[string]any) (map[string]any, error) {
    rawSymbols, ok := inputs["symbols"]
    if !ok {
        return nil, fmt.Errorf("allocation: missing symbols input")
    }
    symbols, err := toStringSlice(rawSymbols)
    if err != nil || len(symbols) == 0 {
        return nil, fmt.Errorf("allocation: invalid or empty symbols: %w", err)
    }

    totalCapital := 100000.0
    if v, ok := inputs["total_capital"].(float64); ok && v > 0 {
        totalCapital = v
    }

    method := n.Method
    if method == "" {
        method = "equal"
    }

    var allocations map[string]float64
    switch method {
    case "risk_parity":
        allocations = riskParityAllocation(symbols, totalCapital)
    default:
        allocations = equalAllocation(symbols, totalCapital)
    }

    weights := make(map[string]float64)
    for sym, amt := range allocations {
        weights[sym] = round(amt/totalCapital, 4)
    }

    return map[string]any{
        "allocations": allocations,
        "weights":     weights,
    }, nil
}

func equalAllocation(symbols []string, total float64) map[string]float64 {
    result := make(map[string]float64, len(symbols))
    share := math.Floor(total/float64(len(symbols))*100) / 100
    var sum float64
    for i, sym := range symbols {
        if i == len(symbols)-1 {
            // last one gets remainder to avoid rounding error
            result[sym] = round(total-sum, 2)
        } else {
            result[sym] = share
        }
        sum += result[sym]
    }
    return result
}

func riskParityAllocation(symbols []string, total float64) map[string]float64 {
    // Simple inverse-volatility risk parity
    // Each asset gets weight proportional to 1/volatility
    // If no vol data available, falls back to equal
    return equalAllocation(symbols, total)
}

func toStringSlice(raw any) ([]string, error) {
    switch v := raw.(type) {
    case []string:
        return v, nil
    case []any:
        result := make([]string, len(v))
        for i, x := range v {
            s, ok := x.(string)
            if !ok {
                return nil, fmt.Errorf("element %d is not string", i)
            }
            result[i] = s
        }
        return result, nil
    default:
        return nil, fmt.Errorf("unsupported type %T", raw)
    }
}

func init() {
    RegisterNode(func() Node { return &AllocationNode{} })
}
```

---

## Task 5: 验证

```bash
go test ./internal/workflow/nodes/... -v -count=1
go test ./internal/workflow/... -v -count=1
go build ./...
```
