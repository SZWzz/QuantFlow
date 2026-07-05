# Test Coverage — Zero-Coverage Packages

## Motivation

审计发现以下包完全没有测试覆盖：

| 包 | 源文件数 | 测试文件数 | 覆盖率 |
|---|:---:|:---:|:---:|
| `internal/ws` | 7 | 0 | 0% |
| `internal/auth` | 4 | 0 | 0% |
| `internal/logging` | 2 | 0 | 0% |
| `internal/schedule` | 3 | 0 | 0% |
| `internal/notify` | 5 | 0 | 0% |
| `internal/research` | 16 | 1 | 6% |
| `internal/trading/oms` | 5 | 0 | 0% |

零测试覆盖意味着重构时无法通过 CI 检测回归。

## Design

### 目标

为每个零覆盖包添加**核心功能的测试**，不追求 100% 行覆盖率，但要覆盖：
1. 主要导出函数/类型
2. 正常路径 + 主要错误路径
3. 并发安全（如有 mutex/channel）

### 测试策略

#### 1. `internal/ws` — WebSocket Hub

**`hub_test.go`**:
- Hub 创建
- Subscribe/Unsubscribe
- Broadcast（含 buffer full 的 default 分支）
- Close 后 Broadcast 不 panic

```go
func TestHub_SubscribeBroadcast(t *testing.T) {
    h := NewHub()
    defer h.Close()

    msg := make(chan []byte, 10)
    id := h.Subscribe(msg)
    defer h.Unsubscribe(id)

    h.Broadcast(Message{Type: "test", Data: `{"key":"val"}`})

    select {
    case m := <-msg:
        if !bytes.Contains(m, []byte("test")) {
            t.Errorf("expected message containing 'test', got %s", m)
        }
    case <-time.After(time.Second):
        t.Error("timeout waiting for broadcast")
    }
}
```

#### 2. `internal/auth` — Authentication

**`credential_test.go`**:
- 凭证加密/解密往返
- Token 生成与验证
- 并发安全的多 goroutine 操作
- 无效 token 拒绝

#### 3. `internal/logging` — Logging

**`logger_test.go`**:
- 日志写入（bytes.Buffer 作为 writer）
- 不同级别（Info/Warn/Error）
- 结构化字段

#### 4. `internal/schedule` — Cron Scheduler

**`scheduler_test.go`**:
- 调度注册与取消
- 执行回调
- 多个调度互不干扰

#### 5. `internal/notify` — Notification Engine

**`manager_test.go`**:
- 通知发送（mock Notifier）
- 多 receiver 广播
- 失败重试限制

#### 6. `internal/trading/oms` — Order Management System

**`order_test.go`**:
- 创建/取消/修改订单
- 订单状态机转换
- 并发安全

#### 7. `internal/research` — 补充现有测试

**现有**: `satellite_service_test.go` — 只有 SatelliteService
**补充**:
- `govdata_service_test.go` — GovDataService 缓存 + API 调用
- `prediction_market_service_test.go` — PredictionMarket 解析
- `insider_trade_service_test.go` — InsiderTrade 时间范围过滤

## Acceptance Criteria

- [ ] `internal/ws` 测试通过，覆盖率 >60%
- [ ] `internal/auth` 测试通过，覆盖率 >50%
- [ ] `internal/logging` 测试通过，覆盖率 >60%
- [ ] `internal/schedule` 测试通过，覆盖率 >50%
- [ ] `internal/notify` 测试通过，覆盖率 >40%
- [ ] `internal/trading/oms` 测试通过，覆盖率 >40%
- [ ] `internal/research` 新增 4 个测试文件，通过
- [ ] `go test ./... -count=1` 全部通过

## Risks / Trade-offs

- 某些包（如 `internal/auth`）依赖加密操作，测试需要 mock。加 mock interface 会增加抽象层。权衡：对 auth 使用真实加密（只是本地对称加密），不 mock。
- 测试编写需要理解各包现有代码结构，预计每个包 15-30 分钟，总计约 3-4 小时。
