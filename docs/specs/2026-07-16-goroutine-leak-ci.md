# Goroutine 泄漏检测 (Goroutine Leak Detection in CI)

## Motivation

QuantFlow 大量使用 goroutine：WebSocket Hub、市场数据轮询器、工作流引擎、AI Agent 流式输出、Python gRPC 连接。goroutine 泄漏会导致内存逐渐耗尽、应用卡死。

已有 `docs/specs/2026-07-07-fix-goroutine-leaks-shutdown.md` 修复了 goroutine 泄漏和关停顺序问题，但缺少持续检测机制——新代码可能引入新的泄漏。

## Design

### 检测策略

在测试中集成 `go.uber.org/goleak`，检测测试执行后是否有未退出的 goroutine：

```go
// internal/xxx/xxx_test.go
import (
    "testing"
    "go.uber.org/goleak"
)

func TestMain(m *testing.M) {
    goleak.VerifyTestMain(m)
}
```

### 适用范围

每个核心包在 `TestMain` 中集成 `goleak.VerifyTestMain`：

| 包 | goroutine 风险 | 优先级 |
|----|:-------------:|:------:|
| `internal/ws/` | 高 (Hub, Client, Handler goroutines) | P0 |
| `internal/market/` | 高 (Poller, WS connector, MinuteCache) | P0 |
| `internal/trading/` | 中 (Order matcher, CashLedger) | P1 |
| `internal/workflow/` | 高 (Engine, ExecutionQueue, Cache) | P0 |
| `internal/python/` | 中 (gRPC connection manager) | P1 |
| `internal/ai/` | 中 (Agent ReAct loop, streaming) | P1 |
| `internal/notify/` | 低 (Telegram sender goroutine) | P2 |

### 新增/修改文件

| 文件 | 操作 | 说明 |
|------|------|------|
| `internal/market/ws_test.go` | 新建 | WS Hub goroutine 泄漏测试 |
| `internal/market/poller_test.go` | 修改 | 已有测试 + goleak |
| `internal/trading/engine_test.go` | 修改 | 已有测试 + goleak |
| `internal/workflow/engine_test.go` | 修改 | 已有测试 + goleak |
| `internal/python/bridge_test.go` | 修改 | 已有测试 + goleak |
| `go.mod` | 修改 | 新增 `go.uber.org/goleak` 依赖 |

### 测试写法

```go
// internal/market/ws_test.go
func TestMain(m *testing.M) {
    goleak.VerifyTestMain(m,
        // 允许 Wails 内部 goroutine (非本项目管理)
        goleak.IgnoreAny(),
        // 可加自定义忽略规则
    )
}

func TestHubStartStop(t *testing.T) {
    hub := NewHub()
    if err := hub.Start(); err != nil {
        t.Fatal(err)
    }
    hub.Stop()
    // goleak 在 TestMain 中自动检测，这里无需额外操作
}
```

### CI 集成

```yaml
# .github/workflows/ci.yml — goroutine leak check step
- name: Goroutine Leak Test
  run: |
    go test -run TestMain ./internal/market/... -count=1 -timeout 30s
    go test -run TestMain ./internal/ws/... -count=1 -timeout 30s
    go test -run TestMain ./internal/workflow/... -count=1 -timeout 30s
```

独立于常规测试，避免泄漏检测拖慢全量测试。

或直接修改 `go test ./... -run .` 使其每次运行都检查（推荐——覆盖所有 test）：

所有包的 `TestMain` 集成后，常规 `go test` 自动包含泄漏检测。

### 泄漏分析辅助

`internal/debug/goroutines.go`（可选）：

```go
// debug.GoroutineSnapshot() — 在运行时获取当前 goroutine 堆栈
// 可通过 IPC 调用，在 SystemMonitor 面板中展示
```

## Acceptance Criteria

- [ ] `go.uber.org/goleak` 加入 go.mod
- [ ] `internal/ws/`, `internal/market/`, `internal/workflow/` 的 TestMain 集成 goroutine 泄漏检测
- [ ] `go test ./internal/ws/... -count=1` 通过（无泄漏）
- [ ] `go test ./internal/market/... -count=1` 通过（无泄漏）
- [ ] `go test ./internal/workflow/... -count=1` 通过（无泄漏）
- [ ] CI 中包含 goroutine 泄漏检测步骤
- [ ] 引入新 goroutine 的测试如果未正确清理 → CI 失败
- [ ] `go.uber.org/goleak` 的 `IgnoreAny()` 处理已知的 Wails 内部 goroutine

## Risks / Trade-offs

- **风险**: `goleak` 可能误报（Wails 框架内部的 goroutine 在测试结束时未退出）。→ 使用 `goleak.IgnoreAny()` 忽略已知的背景 goroutine
- **风险**: 泄漏检测增加测试时间（需要等待 goroutine 退出超时）。→ 默认超时 1s，影响很小
- **Trade-off**: 只在测试中检测，不在运行时。运行时 goroutine profiler 可通过 SystemMonitor 手动触发
