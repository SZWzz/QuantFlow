# Log Panel: 终端日志集成到前端 UI

## Motivation

当前日志系统通过 `slog.NewTextHandler(os.Stderr)` 输出到 stderr，用户必须单独启动终端窗口才能看到日志。开发时需要同时盯着终端和 GUI 两个窗口，调试体验割裂。用户希望日志直接显示在前端面板里，不再依赖独立终端。

## Design

### 数据流

```
Go slog.Info/Warn/Error/Debug 调用
  → custom slog.Handler (双重写入: 原 TextHandler + RingBuffer)
  → RingBuffer 按时间顺序保留最近 N 条日志
  → App.SubscribeLogs() 通过 Wails Events.Emit('log', entry) 推送到前端
  → 前端 Events.On('log', callback) 接收并更新 LogPanel
  → LogPanel 呈现代码终端风格的可滚动日志视图
                                    ┌──────────────────────┐
                                    │   LogPanel.vue        │
                                    │ ┌──────────────────┐  │
                                    │ │ [INFO] hub init'd │  │
                                    │ │ [WARN] sidecar…   │  │
                                    │ │ [ERROR] quote…    │  │
                                    │ └──────────────────┘  │
                                    │ 筛选: ███ debug/info  │
                                    └──────────────────────┘
```

### RingBuffer 设计

- 固定容量: 5000 条（可配置）
- 每条日志结构: `{time, level, message, attrs map[string]any}`
- 线程安全: `sync.Mutex` 保护
- 实现 `slog.Handler` 接口: `Handle()`, `WithAttrs()`, `WithGroup()`
- `Handle()` 调用时: 写入原 TextHandler（保持 stderr 输出）+ 追加到 RingBuffer
- `Lines(fromID int)` 方法: 返回 fromID 之后的新行（用于前端连上后拉取历史）

### 前端推送策略

- 使用 **Wails v3 Events API** (`Events.Emit / Events.On`)
- 事件名: `"log"`
- 数据类型: `LogEntry` (JSON)
- App 启动时自动开始推送，前端连上后立即收到
- **拉取历史**: App 暴露 `GetLogs(afterID int) ([]LogEntry, error)` 方法，面板挂载时拉取已有日志

### 关键行为

1. **双重写入不变**: stderr 仍然输出日志（用于启动早期崩溃诊断），只追加不替换
2. **启动日志不丢**: RingBuffer 在 `logging.Setup()` 时创建，早于前端加载，启动阶段的日志全部捕获
3. **前端面板不影响后端**: LogPanel 只是消费者，后端日志系统不依赖前端连接
4. **日志级别保留**: 前端可以按 debug/info/warn/error 筛选

### New / Modified Files

| 文件 | 动作 |
|------|------|
| `internal/logging/logging.go` | **MODIFY** — 添加 RingBuffer 类型 + 修改 Setup() 使用双写 handler |
| `internal/logging/log_entry.go` | **CREATE** — LogEntry 结构体 + RingBuffer 实现 |
| `app.go` | **MODIFY** — 添加 `SubscribeLogs()` 启动方法 + `GetLogs()` 导出方法 |
| `frontend/src/lib/wails.ts` | **MODIFY** — 添加 `Events.On` 封装 + `GetLogs()` 类型方法 |
| `frontend/src/terminal/panels/LogPanel.vue` | **CREATE** — 日志面板组件（虚拟滚动 + 级别筛选 + 搜索 + 自动滚动） |
| `frontend/src/lib/composables/useLogger.ts` | **CREATE** — 日志订阅 composable |
| `frontend/src/terminal/panelToNode.ts` | **MODIFY** — 可选，LogPanel 不需要 workflow 映射，可跳过 |

### API Changes

**Go 新增导出方法 (App struct):**
```go
// GetLogs returns log entries after the given ID (0 = all).
func (a *App) GetLogs(afterID int) ([]LogEntry, error)
```

**Go logging 包修改:**
```go
// LogEntry is a single log record sent to the frontend.
type LogEntry struct {
    ID        int64             `json:"id"`
    Time      string            `json:"time"`
    Level     string            `json:"level"`
    Message   string            `json:"message"`
    Attrs     map[string]any    `json:"attrs,omitempty"`
}

// RingBuffer is a thread-safe circular buffer of LogEntry.
type RingBuffer struct {
    mu     sync.Mutex
    buffer []LogEntry
    nextID int64
    head   int
    count  int
    max    int
}

// NewRingBuffer creates a ring buffer with capacity.
func NewRingBuffer(capacity int) *RingBuffer

// Push appends a log entry; oldest entries dropped when full.
func (rb *RingBuffer) Push(entry LogEntry)

// Lines returns entries with ID > afterID (newest-first up to limit).
func (rb *RingBuffer) Lines(afterID int64, limit int) []LogEntry
```

**Go slog.Handler wrapper (双写):**
```go
type dualHandler struct {
    inner slog.Handler  // original TextHandler -> stderr
    rb    *RingBuffer
}
// Handle: writes to inner, then pushes to ring buffer
// WithAttrs / WithGroup: delegates to inner
```

**前端新增:**
```typescript
// lib/wails.ts
import { Events } from '@wailsio/runtime'

export function onLogEntry(callback: (entry: LogEntry) => void): () => void {
  return Events.On('log', (event) => callback(event.data))
}

export async function GetLogs(afterID: number): Promise<LogEntry[]> {
  return wailsCall<LogEntry[]>('GetLogs', afterID)
}
```

**Wails Events 生命周期:**
- Go 端: `application.Emit(context, "log", entry)` — 每次 RingBuffer.Push 后调用
- 前端: `Events.On('log', handler)` — 在 useLogger composable 的 onMounted 注册, onUnmounted 取消

### LogPanel UI 设计

- 类似 `tail -f` 的终端风格（等宽字体, 暗色背景）
- 每行: `[LEVEL] time message key=val key=val`
- 级别着色: DEBUG 灰, INFO 白, WARN 黄, ERROR 红
- 底部自动滚动（新日志到达时滚动到底部, 用户向上滚动时暂停）
- 右上角筛选: 级别 filter 按钮组 + 关键字搜索框
- 右上角清空按钮 (用 `confirmDialog`)
- 容量限制: 前端保留最近 2000 条（防止内存泄漏）

## Acceptance Criteria

- [ ] 启动应用后，打开日志面板能看到从启动开始的完整日志
- [ ] 前端不需要单独开终端窗口就能看到实时日志
- [ ] 按 level 筛选（debug/info/warn/error）正确过滤
- [ ] 关键字搜索能高亮匹配行
- [ ] stderr 仍然有日志输出（启动早期崩溃可追溯）
- [ ] 日志面板不影响系统性能（5000 ring buffer + 2000 前端上限）
- [ ] 面板可以正常打开/关闭/缩放

## Risks / Trade-offs

- **Wails Events API 未经验证**: 当前代码库 `Events.On/Emit` 零使用。Go 端 `application.Emit` 的签名和 frontend `Events.On` 的匹配需要 PoC 验证。**备选方案**: 如果 Events API 有问题，退回到已有的 WebSocket hub，加一个新 topic `system:log`，用 `useWebSocket()` composable 订阅。
- **启动崩溃日志**: 如果应用在 logging.Setup 之前崩溃，日志不会被捕获（和现状一样，看 stderr）。
- **性能**: 高频日志（如 debug 级别每笔 tick）可能压爆 ring buffer。合理做法: 日志写入只在 `Handle()` 调用中同步操作（RingBuffer.Push 是 O(1) 且持有锁很短），不阻塞原 handler。前端 Events.Emit 是异步的，不阻塞 Go 执行。
- **Events.Emit 频率**: 如果一瞬间产生数百条日志，Emit 可能过于频繁。考虑加一个 100ms 的 debounce timer，批量发送。但 debounce 会增加 latency——对于日志面板可接受。初始实现不做 debounce，如果实测有性能问题再加。
