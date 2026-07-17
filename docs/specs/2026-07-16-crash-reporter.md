# Crash Reporter

## Motivation

桌面应用一定会崩溃——goroutine panic、CGO 段错误、内存耗尽。当前 QuantFlow 没有任何崩溃信息收集机制。用户崩溃后只能截图报 bug，开发者完全不知道发生了什么。

需要：崩溃时捕获堆栈 → 本地保存 → 可选上传 → 重启应用。

注意：这是本地优先的隐私设计，不上传任何个人信息。上传是用户 opt-in。

## Design

### 崩溃处理链

```
goroutine panic / 系统信号 (SIGSEGV, SIGABRT)
  ↓
recover() 或 signal.Notify 捕获
  ↓
生成 CrashReport:
  - 时间戳
  - 版本号
  - Go 版本 / OS / Arch
  - panic 值 + stack trace (所有 goroutine)
  - 最近 100 条日志 (logging ring buffer)
  - 应用状态 JSON (mode, active brokers, 面板数)
  ↓
写入本地文件: ~/Library/Logs/QuantFlow/crashes/2026-07-16T10:30:00.json
  ↓
弹窗:
  ┌──────────────────────────────────┐
  │  💥 QuantFlow 崩溃了               │
  │                                   │
  │  已保存崩溃报告:                   │
  │  ~/Library/Logs/QuantFlow/...     │
  │                                   │
  │  ☐ 发送匿名崩溃报告帮助改进         │
  │                                   │
  │  [忽略]  [重启应用]                │
  └──────────────────────────────────┘
  ↓
用户选"发送" → HTTP POST → GitHub Issues / 自建服务
用户选"重启" → exec 新进程 → 当前进程退出
用户选"忽略" → 进程退出
```

### 新增/修改文件

| 文件 | 操作 | 说明 |
|------|------|------|
| `internal/crash/reporter.go` | 新建 | 崩溃报告引擎 |
| `internal/crash/report.go` | 新建 | CrashReport 结构定义 |
| `internal/crash/store.go` | 新建 | 本地存储 + 上传 |
| `internal/crash/darwin.go` | 新建 | macOS 信号处理 |
| `internal/crash/linux.go` | 新建 | Linux 信号处理 |
| `internal/crash/windows.go` | 新建 | Windows 信号处理 |
| `main.go` | 修改 | 启动时注册 crash handler |
| `frontend/src/lib/wails.ts` | 修改 | 新增 `onCrashReport` 事件处理 |

### CrashReport 结构

```go
type CrashReport struct {
    ID          string    `json:"id"`
    Timestamp   time.Time `json:"timestamp"`
    Version     string    `json:"version"`
    GoVersion   string    `json:"go_version"`
    OS          string    `json:"os"`
    Arch        string    `json:"arch"`
    BuildMode   string    `json:"build_mode"` // dev / prod

    Panic       string   `json:"panic"`
    Stack       string   `json:"stack"`       // all goroutines
    Logs        []string `json:"logs"`        // ring buffer last 100

    AppState    AppState `json:"app_state"`    // non-PII
}

type AppState struct {
    TradingMode    string   `json:"trading_mode"`
    ActiveBrokers  []string `json:"active_brokers"`
    PanelCount     int      `json:"panel_count"`
    WorkflowCount  int      `json:"workflow_count"`
    UptimeSeconds  int64    `json:"uptime_seconds"`
    // 不包含: API keys, 持仓明细, 个人身份信息
}
```

### 信号处理

```go
// main.go
func main() {
    if !isTestEnv() {
        crash.StartHandler()
    }
    // ... 正常启动
}

// internal/crash/reporter.go
func StartHandler() {
    c := make(chan os.Signal, 1)
    signal.Notify(c, syscall.SIGABRT, syscall.SIGSEGV, syscall.SIGILL, syscall.SIGBUS)

    go func() {
        for sig := range c {
            report := collectReport(fmt.Sprintf("signal: %v", sig))
            saveReport(report)
            showCrashDialog(report)
            os.Exit(1)
        }
    }()
}
```

注意：无法在 CGO 段错误后可靠执行，但 Go-level panic + 未捕获的 goroutine panic 可被 `recover()` 在顶层捕获。

### 日志绑定

`internal/logging/ring_buffer.go` 需要导出最后 N 条日志：

```go
func (rb *RingBuffer) LastN(n int) []string {
    rb.mu.RLock()
    defer rb.mu.RUnlock()
    // 返回最近的 n 条
}
```

### 崩溃列表面板

新增简易 `CrashHistoryPanel`（或内嵌到 SystemMonitor）：

```
┌──────────────────────────────────┐
│  💥 崩溃历史 (3次)                │
├──────────────────────────────────┤
│  2026-07-15 14:30  SIGSEGV      │  [查看] [分享]
│  2026-07-14 09:15  panic: ...   │  [查看] [分享]
│  2026-07-10 22:00  OOM          │  [查看] [分享]
└──────────────────────────────────┘
```

## Acceptance Criteria

- [ ] `signal.Notify` 捕获 SIGSEGV/SIGABRT/SIGILL/SIGBUS
- [ ] Go-level panic 通过顶层 `recover()` 捕获
- [ ] CrashReport 包含版本号、系统信息、panic 堆栈、最近 100 条日志
- [ ] 报告写入本地 `~/Library/Logs/QuantFlow/crashes/`
- [ ] 崩溃后弹出恢复对话框（含重启按钮）
- [ ] 用户 opt-in 上传崩溃报告（HTTP POST 到配置的 endpoint）
- [ ] 报告 JSON 不含任何 API key 或持仓明细
- [ ] SystemMonitor 面板展示历史崩溃列表
- [ ] 自动清理 30 天前的崩溃报告
- [ ] Go 测试覆盖 report 生成 + 序列化

## Risks / Trade-offs

- **风险**: CGO 段错误（如 SQLite CGO）可能完全无法捕获。→ 无法完美解决，但 Go-level panic 覆盖绝大多数场景
- **风险**: 崩溃后对话框可能不显示（webview 已崩溃）。→ fallback: 写入文件 + 下次启动时检测并显示
- **风险**: `signal.Notify` 与 Wails 的信号处理冲突。→ 测试验证兼容性，必要时用 Wails 的生命周期 hook
- **Trade-off**: 不自建收集服务器，用 GitHub Issues API 或用户自配 endpoint
