# 浮窗面板 (Tear-off Windows) — 设计文档

## Motivation

对标 Bloomberg Terminal 的 ADS `tear_off_to_new_frame` 功能 — 将 DockView 中的面板拖出为独立的 macOS 窗口，支持跨显示器布局、独立的窗口管理（最小化/最大化/关闭）。

## 技术约束

- **Wails v3** (`v3.0.0-alpha2.111`): 支持 `application.Current().Window.NewWithOptions()` 创建多窗口
- **每个窗口是独立 WebView**: 加载同一 SPA，通过 URL hash 路由区分
- **跨窗口通信**: `app.Event.Emit("name", data)` → Go 广播到所有窗口的 `Events.On("name", cb)`
- **窗口识别**: 通过 `WebviewWindowOptions.Name` 设置唯一标识

## 数据流

```
DockTab 工具栏 [↗] 按钮
  ↓
1. 前端: Call.ByName("main.App.TearOffPanel", panelId, instanceId, label, paramsJson)
  ↓
2. Go: application.Current().Window.NewWithOptions(...)
   - Name: "tearoff-{instanceId}"
   - Title: label
   - URL:  "/#/tearoff/{instanceId}"
   - 存储 instanceId → WebviewWindow 映射
  ↓
3. 前端: 新 WebView 加载 SPA → hash 路由 → /tearoff/:instanceId
   - TearOffPanel.vue 调用 GetTearOffPanelInfo(instanceId)
   - Go 返回 {panelId, label, params}
   - TearOffPanel.vue 渲染 <component :is="getPanelComponent(panelId)" />
  ↓
4. 面板正常通过 Go IPC 获取数据（无需特殊同步）
  ↓
5. 窗口关闭: OnWindowEvent(WindowClosing) → 清理映射
```

## 核心 API

### Go Backend (`app_tearoff.go`)

| 方法 | 说明 |
|------|------|
| `TearOffPanel(panelId, instanceId, label, paramsJson string) error` | 创建新窗口 |
| `CloseTearOffWindow(instanceId string) error` | 关闭指定窗口 |
| `GetTearOffPanelInfo(instanceId string) (string, string, error)` | 返回 `(label, paramsJson)` |
| `ListTearOffWindows() []string` | 返回所有 instanceId 列表 |

### Frontend

| 文件 | 操作 | 说明 |
|------|------|------|
| `frontend/src/terminal/TearOffPanel.vue` | 新建 | 面板独立渲染组件 |
| `frontend/src/terminal/DockView/DockTab.vue` | 修改 | 添加 [↗] 撕下按钮 |
| `frontend/src/App.vue` | 修改 | 检测 /tearoff 路由跳过 mode 同步 |
| `frontend/src/main.ts` | 修改 | 注册 `/tearoff/:instanceId` 路由 |
| `frontend/src/lib/wails.ts` | 修改 | 可选: 添加事件监听工具函数 |

## 改动文件清单

### Go (1 个新文件 + 2 个修改)

| 文件 | 操作 | 说明 |
|------|------|------|
| `app_tearoff.go` | 新建 | App 新增 4 个 IPC 方法 + WindowManager 封装 |
| `app.go` | 修改 | App struct 新增 `tearOffWindows` 字段 |
| `main.go` | 修改 | 可选: 调整窗口关闭行为 |

### Vue/TS (3 个新文件 + 4 个修改)

| 文件 | 操作 | 说明 |
|------|------|------|
| `frontend/src/terminal/TearOffPanel.vue` | 新建 | 通过 instanceId 查询面板信息并渲染 |
| `frontend/src/terminal/DockView/DockTab.vue` | 修改 | Tab 工具栏新增 [↗] 按钮 |
| `frontend/src/main.ts` | 修改 | 注册 `/tearoff/:instanceId` 路由 |
| `frontend/src/App.vue` | 修改 | tear-off 模式下跳过 mode sync |
| `frontend/src/types/wails-runtime.d.ts` | 修改 | 添加 TearOffPanel 等方法类型 |

## 实现要点

### 1. Go 窗口管理

```go
type App struct {
    // ... existing fields
    tearOffWindows   map[string]*tearOffEntry  // instanceId → window
    tearOffWindowsMu sync.RWMutex
}

type tearOffEntry struct {
    Win      *application.WebviewWindow
    PanelID  string
    InstanceID string
    Label    string
    Params   string  // JSON
}
```

`TearOffPanel` 实现:
```go
func (a *App) TearOffPanel(panelId, instanceId, label, paramsJson string) error {
    app := application.Current()
    win := app.Window.NewWithOptions(application.WebviewWindowOptions{
        Name:   fmt.Sprintf("tearoff-%s", instanceId),
        Title:  label,
        Width:  800, Height: 600,
        MinWidth: 400, MinHeight: 300,
        URL: fmt.Sprintf("/#/tearoff/%s", instanceId),
    })
    entry := &tearOffEntry{Win: win, PanelID: panelId, InstanceID: instanceId, Label: label, Params: paramsJson}
    a.tearOffWindowsMu.Lock()
    a.tearOffWindows[instanceId] = entry
    a.tearOffWindowsMu.Unlock()
    return nil
}
```

### 2. 前端路由

main.ts:
```typescript
{
    path: '/tearoff/:instanceId',
    name: 'tearoff',
    component: () => import('@/terminal/TearOffPanel.vue'),
}
```

### 3. App.vue tear-off 检测

```typescript
const isTearOff = computed(() => route.path.startsWith('/tearoff'))
```
将 mode sync watchers 包在 `if (!isTearOff.value)` 中。

### 4. TearOffPanel.vue

```vue
<script setup lang="ts">
import { ref, onMounted, shallowRef } from 'vue'
import { useRoute } from 'vue-router'
import { getPanelComponent } from '@/terminal/panels/registry'

const route = useRoute()
const instanceId = route.params.instanceId as string
const panelId = ref('')
const params = ref<Record<string, any>>()
const panelComponent = shallowRef()

onMounted(async () => {
    const info = await (window as any).go.main.App.GetTearOffPanelInfo(instanceId)
    panelId.value = info[0]
    params.value = info[1] ? JSON.parse(info[1]) : undefined
    panelComponent.value = getPanelComponent(panelId.value)
})
</script>
<template>
    <component :is="panelComponent" :panel-id="panelId" :params="params" v-if="panelComponent" />
</template>
```

### 5. DockTab 撕下按钮

在标签栏添加 [↗] 按钮（在关闭按钮旁边或菜单中），点击后:
1. 存根当前面板信息
2. 调用 `TearOffPanel(panelId, instanceId, label, JSON.stringify(params))`
3. 调用 `closeTab(leafId, tabId)` 移除面板

## 状态同步（MVP 不做）

- 主题同步: 每个窗口独立从 localStorage 读取 `quantflow-session`
- 后续通过 Wails Events 实现: `app.Event.Emit("app:theme-changed", theme)`

## Risks / Trade-offs

- **内存**: 每个窗口一个独立 WebView 进程。保守估计每窗口 ~100MB。建议最多同时 5 个。
- **Wails v3 Alpha**: 多窗口 API 可能存在未发现的 bug。窗口创建必须在主 goroutine 进行。
- **AppleScript/Window 管理**: 新窗口不支持 Wails v2 的 `BrowserOpenURL` 替代方案。
- **跨窗口状态**: localStorage 各自独立，首次打开为默认 dark 主题。
