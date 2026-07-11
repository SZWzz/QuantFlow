# 前端面板懒加载与 WS 订阅生命周期管理

## Motivation

当前 QuantFlow 有 93 个终端面板，所有面板在 `TerminalMode.vue` 中同时挂载，即使不可见也保持活跃：

```vue
<!-- 当前实现：所有面板同时渲染 -->
<WatchlistPanel :symbols="watchlist" />
<CandlestickPanel :symbol="symbol" />
<MarketOverviewPanel />
<!-- ... 90+ 个面板并列 ... -->
```

这导致三个问题：

1. **内存浪费**：每个面板持有响应式数据（quotes、ohlcv、trades 等），不可见面板的数据从不展示但占内存
2. **WebSocket 订阅浪费**：每个面板独立调用 `useWebSocket`/`useRealtimeData`，即使不可见也维持订阅。后端 `QuotePoller` 因此为不可见面板轮询
3. **首次加载慢**：所有面板的初始数据请求（Wails IPC + HTTP fetch）在挂载时同时触发，造成启动毛刺

## Design

### 1. DockView 活性追踪

`DockView` 已有 tab 切换机制，但面板卸载由各面板的 `v-if` 控制。改为统一活性追踪：

```typescript
// frontend/src/stores/terminal.ts (Pinia)
export const useTerminalStore = defineStore('terminal', () => {
  const activePanels = ref<Set<string>>(new Set())

  // 面板激活/停用由 DockView 驱动
  function activatePanel(id: string) { activePanels.value.add(id) }
  function deactivatePanel(id: string) { activePanels.value.delete(id) }

  // 面板是否可见（供面板组件使用）
  function isPanelActive(id: string) { return activePanels.value.has(id) }
})
```

### 2. 面板懒加载包装器

创建 `LazyPanel` 组件替代直接的 `<component>` 渲染：

```vue
<!-- frontend/src/terminal/components/LazyPanel.vue -->
<script setup lang="ts">
const props = defineProps<{
  panelId: string
  component: Component
  props?: Record<string, any>
}>()

const terminalStore = useTerminalStore()
const visible = ref(false)
const mounted = ref(false)

// 面板靠近视口时渲染
const observer = new IntersectionObserver(([entry]) => {
  if (entry.isIntersecting && !mounted.value) {
    mounted.value = true
    terminalStore.activatePanel(props.panelId)
  }
  visible.value = entry.isIntersecting
})
</script>

<template>
  <div ref="el" v-show="visible.value">
    <component :is="component" v-if="mounted" v-bind="props" />
  </div>
</template>
```

### 3. WS 订阅与活性联动

修改 `useRealtimeData` composable，使其在面板不可见时自动退订：

```typescript
// frontend/src/lib/composables/useRealtimeData.ts
export function useRealtimeData<T = any>(
  panelId: string,
  topics: string[] | Ref<string[]> | (() => string[]),
  handler: (topic: string, data: T) => void,
) {
  const ws = useWebSocket()
  const terminalStore = useTerminalStore()
  const wsUrl = `...`

  // 仅在面板活跃时连接
  watch(() => terminalStore.isPanelActive(panelId), (active) => {
    if (active) {
      ws.connect(wsUrl, resolveTopics())
    } else {
      ws.disconnect()
    }
  }, { immediate: true })

  onUnmounted(() => ws.disconnect())
}
```

### 4. 面板注册表

所有面板需要在注册表中声明其 WS topic 依赖，以便统一管理和去重：

```typescript
// frontend/src/terminal/panels/registry.ts
export interface PanelRegistration {
  id: string
  component: Component
  title: string
  category: PanelCategory
  topics?: string[] | ((props: any) => string[])  // WS topics
}

export const panelRegistry: PanelRegistration[] = [
  {
    id: 'watchlist',
    component: WatchlistPanel,
    title: '自选股',
    category: 'market',
    topics: (props) => props.symbols?.map(s => `market:quote:${s}`) ?? [],
  },
  // ... 所有面板
]
```

### 5. 实现步骤

**Phase 1 — LazyPanel + 活性追踪（2 天）**

| 文件 | 改动 |
|------|------|
| `frontend/src/stores/terminal.ts` | 新增 `activePanels`、`activatePanel`、`deactivatePanel`、`isPanelActive` |
| `frontend/src/terminal/components/LazyPanel.vue` | 新建 — IntersectionObserver 懒加载包装器 |
| `frontend/src/terminal/DockView/DockView.vue` | 面板切换时调用 `activatePanel/deactivatePanel` |
| `frontend/src/terminal/TerminalMode.vue` | 面板渲染改为 `<LazyPanel>` 包装 |

**Phase 2 — WS 订阅联动（1 天）**

| 文件 | 改动 |
|------|------|
| `frontend/src/lib/composables/useRealtimeData.ts` | 新增 `panelId` 参数，watch `isPanelActive` 连接/断开 |
| `frontend/src/terminal/panels/registry.ts` | 新建 — 面板注册表含 WS topic 声明 |

**Phase 3 — 面板注册表迁移（2 天）**

逐面板迁移到注册表模式，同时清理重复的 WS 订阅代码。

### 6. 预期效果

| 指标 | 当前 | 优化后 |
|------|------|--------|
| 同时挂载的面板 | 93 | 10-15（可见的） |
| 活跃 WS 订阅数 | 数百 | <30 |
| 首次加载时间 | ~3s（所有面板同时请求） | <1s（仅可见面板） |
| 内存占用 | ~200MB+ | 预估 <80MB |

## Acceptance Criteria

- [ ] 首次加载时只渲染可见面板，不可见面板不触发任何数据请求
- [ ] 切换到新 tab 时对应面板在 200ms 内完成挂载
- [ ] 面板切出后自动断开 WS 订阅
- [ ] 面板切回后自动重新订阅 WS，通过 IPC 获取初始数据
- [ ] QuotePoller 的轮询 symbol 数不超过可见面板所需
- [ ] IntersectionObserver 在面板完全不可见时正确触发 deactivate
- [ ] 所有面板的功能不受懒加载影响
- [ ] 后端构建通过，所有已有前端测试通过
- [ ] CHANGELOG 更新

## Risks / Trade-offs

| 风险 | 缓解 |
|------|------|
| 面板切换后有数据空白（WS 重连延迟） | 切回时通过一次 Wails IPC 获取初始数据，WS 作为增量更新 |
| IntersectionObserver 兼容性 | Vue 3 支持的浏览器都支持 IO，无需 polyfill |
| 面板注册表迁移工作量大 | Phase 1 不需要注册表，只加 LazyPanel 就见效。注册表是 Phase 3 可选优化 |
| WS 断开/重连可能导致消息丢失 | `useRealtimeData` 已有指数退避重连，切回时 IPC 获取全量，WS 做增量 |
