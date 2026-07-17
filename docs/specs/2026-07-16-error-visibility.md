# 全局错误可见性 (Error Visibility System)

## Motivation

Wails v3 webview 禁用了 `window.alert()` 和 `window.confirm()`（confirm 静默返回 false），导致所有错误静默失败。当数据源不可用、券商断连、回测异常时，用户看不到任何反馈——面板空白、数据不变、操作无响应，用户不知道发生了什么。

当前 `internal/logging/` 有 ring buffer + `internal/ws/` 有推送能力，但前端没有消费它们的 UI。

## Design

### 三层错误可见性

```
Layer 1: Toast 通知 (临时, 自动消失)
  ┌────────────────────────────────────────────┐
  │ ⚠️ Tencent 行情源超时, 已回退到 Sina 适配器  │  ← 5s 自动 dismiss
  └────────────────────────────────────────────┘

Layer 2: 状态栏 (持久, 全局可见)
  ┌─────────────────────────────────────┐
  │ 📡 A股 ◉ 实时  | 港股 ◉ 实时 | 加密 ◉ 延迟│  ← 底部固定
  │ 💼 Alpaca ◉ 已连接 | Binance ◉ 已连接   │
  │ 🐍 Python ◉ 运行中                     │
  └─────────────────────────────────────┘

Layer 3: LogPanel (历史, 可回查)
  → 现有 LogPanel 升级，接收 ws 推送的 log 条目
```

### 新增/修改文件

| 文件 | 操作 | 说明 |
|------|------|------|
| `frontend/src/lib/useToast.ts` | 新建 | Toast 系统 composable |
| `frontend/src/terminal/components/ToastContainer.vue` | 新建 | 浮动 Toast 容器 (fixed top-right) |
| `frontend/src/terminal/components/StatusBar.vue` | 新建 | 底部状态栏组件 |
| `frontend/src/stores/terminal.ts` | 修改 | 新增 `toasts` state + `addToast/removeToast` actions |
| `frontend/src/stores/terminal.ts` | 修改 | 新增 `connectionStatus` state |
| `frontend/src/terminal/panels/LogPanel.vue` | 修改 | 接入 ws log topic |
| `app_system.go` | 追加 | `GetConnectionStatus() ConnectionStatus` IPC |
| `internal/logging/ring_buffer.go` | 修改 | 通过 WS Hub 广播新日志条目 |

### Toast 系统

```typescript
// useToast.ts
interface Toast {
  id: string
  type: 'info' | 'success' | 'warning' | 'error'
  title: string
  message: string
  duration: number      // ms, 0 = 不自动消失
  action?: { label: string; onClick: () => void }
}

function useToast() {
  const toasts = ref<Toast[]>([])
  const addToast = (t: Omit<Toast, 'id'>) => { ... }
  const removeToast = (id: string) => { ... }
  const success = (msg: string) => addToast({ type: 'success', message: msg, duration: 3000 })
  const error = (msg: string) => addToast({ type: 'error', message: msg, duration: 0 })
  // ...
}
```

### Toast 触发源

| 触发条件 | 类型 | 持续时间 | 消失条件 |
|---------|------|---------|---------|
| 数据源切换 (容灾) | info | 5s | 自动 |
| 数据源全部不可用 | error | 永久 | 手动关闭 |
| 券商连接成功/断开 | warning | 5s | 自动 |
| 回测完成 | success | 3s | 自动 |
| Python sidecar 断连 | error | 永久 | 恢复后自动消失 |
| API Key 验证失败 | warning | 8s | 自动 |

### 状态栏连接状态

```
┌──────────────────────────────────────────────────┐
│ 📊 QuantFlow v2026.7.16                          │
│ 📡 A股: Tencent(实时) 港股: Yahoo(5min ago)        │
│ 💼 Alpaca(已连接) Binance(已连接) IBKR(未配置)     │
│ 🐍 Python(运行中 0.2.3)                          │
│                                                  │
│ 点击任意状态 → 弹出详情对话框                       │
└──────────────────────────────────────────────────┘
```

### 数据流

```
Go side (数据源切换/失败)
  → ws hub publish "system:notification"
    → frontend ws handler → terminalStore.addToast()

Go side (连接状态变化)
  → ws hub publish "system:connection-status"
    → frontend ws handler → terminalStore.updateConnectionStatus()

前端操作失败 (IPC error)
  → composable catch → addToast({ type: 'error', message: error.message })
```

## Acceptance Criteria

- [ ] Toast 组件支持 4 种类型 (info/success/warning/error)，不同颜色图标
- [ ] Toast 自动消失(duration 参数) + 手动关闭按钮
- [ ] 错误类型 Toast 不自动消失，需要用户手动 dismiss
- [ ] 底部状态栏展示数据源、券商、Python 三组连接状态
- [ ] 数据源容灾切换时弹出 info toast（如 "Tencent 超时→Sina 回退"）
- [ ] 数据源全部不可用时弹出 error toast（不自动消失）
- [ ] 状态栏点击弹出详情对话框（含切换时间线、错误日志）
- [ ] LogPanel 收到所有 ws 推送的日志条目
- [ ] Toast 系统全量测试覆盖
- [ ] 状态栏定期刷新（每 10s），通过 `GetConnectionStatus()` IPC

## Risks / Trade-offs

- **风险**: Toast 过多会干扰用户。→ 同类错误 30s 内合并展示，不重复弹出
- **风险**: 状态栏轮询增加 IPC 开销。→ 只在状态变化时通过 ws 推送，不轮询
- **Trade-off**: 不引入 sentry/telemetry，完全本地。错误日志没有远程上报能力
