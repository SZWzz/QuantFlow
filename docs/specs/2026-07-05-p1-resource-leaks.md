# P1 Resource Leaks + Data Integrity — HTTP Body, JSON Errors, Race Conditions, Alert Polyfill

## Motivation

审计发现 5 类中危 bug，导致资源泄漏、数据静默丢失或 UI 交互失效：

1. **HTTP Response Body 未关闭**（15+ adapter）— 错误路径不 close body，连接池耗尽
2. **JSON marshal/unmarshal 错误静默忽略**（25+ 处）— 序列化失败数据静默丢失
3. **PeakEquity 数据竞争** — 无锁修改导致回撤计算错误
4. **WebSocket Hub Broadcast 竞态** — TOCTOU 窗口导致向已关闭 channel 发送而 panic
5. **RebalancePanel 用 `window.alert()`** — Wails webview 中 alert 是 no-op，用户看不到提示

## Design

### 1. HTTP Response Body 关闭

所有 adapter 文件的 HTTP 请求模式改为：

```go
resp, err := client.Do(req)
if err != nil {
    return nil, fmt.Errorf("request failed: %w", err)
}
defer resp.Body.Close()

if resp.StatusCode != http.StatusOK {
    body, _ := io.ReadAll(resp.Body)
    return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
}
```

需要改动的 adapter 文件（~15 个）：
- `binance.go`, `sina.go`, `tencent.go`, `gateio.go`, `akshare.go`
- `coingecko.go`, `okx.go`, `yahoo.go`, `eastmoney.go`
- `satellite.go`, `gdelt.go`
- `alpaca.go`, `binance.go`（trading/brokers）

统一加 `defer resp.Body.Close()` 并在错误路径也确保 body 被关闭。

### 2. JSON 错误处理

对 `json.Marshal` 和 `json.Unmarshal` 被静默忽略的 25+ 处：

```go
// 之前
data, _ := json.Marshal(obj)

// 之后
data, err := json.Marshal(obj)
if err != nil {
    slog.Warn("marshal JSON failed", "type", fmt.Sprintf("%T", obj), "error", err)
    // 根据上下文决定：return error / 用空值继续
}
```

关键文件：
- `app.go:375-385` — workflow JSON 序列化失败应返回 error
- `app_market.go:440` — last_minute_ticks 反序列化失败应 warn
- `internal/ws/hub.go:83` — WS 广播数据 marshal 失败应 warn
- `internal/auth/credential.go:124` — 凭证加密 marshal 失败应 return error
- `internal/notify/manager.go:82` — 通知 marshal 失败应 warn

### 3. PeakEquity 数据竞争

**`internal/trading/risk_pipeline.go:39-51`**：

```go
type RiskPipeline struct {
    mu     sync.Mutex
    config RiskConfig
}

func (r *RiskPipeline) CheckDrawdown(currentEquity float64) error {
    r.mu.Lock()
    if currentEquity > r.config.PeakEquity {
        r.config.PeakEquity = currentEquity
    }
    maxDD := r.config.PeakEquity
    r.mu.Unlock()
    // 用 maxDD 计算回撤...
}
```

### 4. WebSocket Hub Broadcast 竞态

**`internal/ws/hub.go:72-93`**：

当前代码在两次 `RLock` 之间有窗口。改为一次 `Lock`：

```go
func (h *Hub) Broadcast(msg Message) {
    rawMsg, err := json.Marshal(msg)
    if err != nil {
        slog.Warn("broadcast marshal", "error", err)
        return
    }
    h.mu.RLock()
    defer h.mu.RUnlock()
    for _, client := range h.subs {
        select {
        case client.send <- rawMsg:
        default:
            slog.Warn("broadcast: client send buffer full, dropping message", "client", client.ID())
        }
    }
}
```

### 5. RebalancePanel `window.alert()`

**`frontend/src/terminal/panels/RebalancePanel.vue:166`**：

```typescript
// 之前
window.alert('...')

// 之后
import { alertDialog } from '@/lib/wails'
await alertDialog('...')
```

## Acceptance Criteria

- [ ] 所有 15+ adapter 使用 `defer resp.Body.Close()` 模式
- [ ] 所有 25+ JSON 错误不再静默忽略
- [ ] `CheckDrawdown` 使用 mutex 保护 `PeakEquity`
- [ ] `Hub.Broadcast` 使用单个 `Lock` 而非两次 `RLock`
- [ ] RebalancePanel 使用 `alertDialog` 而非 `window.alert()`
- [ ] `go vet ./...` 通过
- [ ] `go test ./... -race -count=1` 通过（竞态检测器无报错）

## Risks / Trade-offs

- `resp.Body.Close()` 改为 `defer` 后，如果 handler 里做了很多处理才读完 body，连接会保持更久才复用。但 HTTP adapter 都是短请求 (<1s)，可接受。
- JSON marshal 错误加 `slog.Warn` 会增加少量日志。对于关键路径（凭证、通知），改为返回 error 会改变调用者行为。
