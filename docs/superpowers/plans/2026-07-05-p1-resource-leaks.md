# 实施计划：P1 Resource Leaks + Data Integrity

参考：`docs/specs/2026-07-05-p1-resource-leaks.md`

## Task 1: HTTP Response Body Close（15+ adapter 文件）

对每个 adapter 添加 `defer resp.Body.Close()`，确保所有路径（包括 error 路径）都能关闭 body。

**需要改动的文件**（逐个修改）：

1. `internal/market/adapters/binance.go` — 搜索 `resp, err := client.Do` 改为 `defer resp.Body.Close()`
2. `internal/market/adapters/sina.go` — 同上
3. `internal/market/adapters/tencent.go` — 同上
4. `internal/market/adapters/gateio.go` — 同上
5. `internal/market/adapters/akshare.go` — 同上
6. `internal/market/adapters/coingecko.go` — 同上
7. `internal/market/adapters/okx.go` — 同上
8. `internal/market/adapters/yahoo.go` — 同上
9. `internal/market/adapters/eastmoney.go` — 同上
10. `internal/market/adapters/eastmoney_signals.go` — 同上
11. `internal/research/satellite.go` — ↑
12. `internal/research/gdelt.go` — ↑
13. `internal/trading/adapters/alpaca.go` — ↑
14. `internal/trading/adapters/binance.go` — ↑

每个文件的标准改动模式：
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

// 继续原有逻辑...
```

---

## Task 2: JSON Error 处理（25+ 处）

**优先处理关键路径**（改变调用行为的）：

- `internal/auth/credential.go:124` — marshal 失败应 return error
- `internal/notify/manager.go:82` — marshal 失败应 warn
- `app.go:375-385` — workflow JSON 序列化失败应 return error

**次优先**（加 slog.Warn 日志）：
- `app_market.go:440` — last_minute_ticks 反序列化
- `internal/ws/hub.go:83` — WS 广播 marshal
- 其他 20+ 处

改动模式：
```go
// before
data, _ := json.Marshal(obj)

// after
data, err := json.Marshal(obj)
if err != nil {
    slog.Warn("marshal JSON failed", "type", fmt.Sprintf("%T", obj), "error", err)
    return ...
}
```

---

## Task 3: PeakEquity 数据竞争

**`internal/trading/risk_pipeline.go:39-51`**：

```go
type RiskPipeline struct {
    mu     sync.Mutex
    config RiskConfig
}

func (r *RiskPipeline) CheckDrawdown(currentEquity float64) (float64, error) {
    r.mu.Lock()
    peak := r.config.PeakEquity
    if currentEquity > peak {
        r.config.PeakEquity = currentEquity
        peak = currentEquity
    }
    r.mu.Unlock()

    drawdown := (peak - currentEquity) / peak * 100
    // return drawdown, nil
}
```

---

## Task 4: WS Hub Broadcast 竞态

**`internal/ws/hub.go:72-93`**：

将两次 `RLock` 合并为一次 `Lock`：
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
            slog.Warn("broadcast buffer full", "client", client.ID())
        }
    }
}
```

---

## Task 5: RebalancePanel alert

**`frontend/src/terminal/panels/RebalancePanel.vue:166`**：

```typescript
// before
window.alert('已触发再平衡，请检查执行结果')

// after
import { alertDialog } from '@/lib/wails'
await alertDialog('已触发再平衡，请检查执行结果')
```

---

## Task 6: 验证

```bash
go vet ./...
go test ./... -race -count=1
cd frontend && npx vue-tsc --noEmit && npx vitest run
go build ./...
```
