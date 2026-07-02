# 画线工具初始化修复：生命周期、时机与资源泄漏

## Motivation

画线工具（DrawingController + Crosshair）自集成到 K 线图以来一直存在不稳定问题：用户点击画线按钮后无法绘制，或在切换股票/面板后画线失效。之前的一次修复（改用 `watch(option)` + `watch(klineChartRef)` 驱动初始化）解决了 `onMounted` 时机过早的问题，但引入了更严重的副作用——每次 option 变更（包括实时数据推送）都销毁并重建 `DrawingController`/`Crosshair`，导致 echarts 事件处理器泄漏累积，以及画线状态下被意外中断。

当前代码在 `CandlestickPanel.vue:684-722` 已经有一版修复。本 spec 回溯已实施的修复，明确其设计与边界，为后续审查和测试提供依据。

## Design

### 问题分析

| # | 问题 | 根因 | 表现 |
|---|------|------|------|
| 1 | `watch(option)` 每次 option 变更都销毁重建 dc/crosshair | `watch(option, () => nextTick(initChartControllers))` 在实时数据推送、切换指标、切换周期时无条件执行 destroy → mount 循环 | echarts mousemove/mouseout 处理器泄漏累积（只注册不清理）；画线工具状态频繁归零，用户画线过程中被中断 |
| 2 | 切股后画线失效 | `<VChart :key="kc-${symbol}" />` 导致 VChart 重建，旧 echarts 实例销毁；但 `klineChartRef`（KlineChart 组件 ref）不变，`watch(klineChartRef)` 不会触发 | dc/crosshair 持有的 echarts 引用悬空，`convertToPixel`/`convertFromPixel` 失败，画线无法绘制 |
| 3 | 初始化自旋死循环 | 当 `getEchartsInstance()` 返回 null 时使用 `nextTick(() => initChartControllers())` 重试，但 nextTick 是微任务——如果 echarts 持续不可用则无限循环，CPU 100% 冻结 UI | 页面卡死，无法操作 |
| 4 | N/A（已修复） | `onMounted` → `nextTick` 初始化时异步数据未加载，canvas ref 为 null，dc/crosshair 永远为 null | 画线工具完全不可用 |
| 5 | N/A（已修复） | `watch(klineChartRef)` 之前有 `(dc && crosshair)` guard 导致只初始化一次，tab 切换回来时 ref 不变 | Tab 切回 K 线后画线工具失效 |

### 数据流（修复后）

```
CandlestickPanel.vue 画线初始化数据流
═══════════════════════════════════════

场景 1: 首次加载 (initial load)
──────────────────────────────────────
loadOHLCV() → ohlcvData[] → KlineChart 创建
    ↓                                ↓
v-if 为 true, 渲染 canvas        klineChartRef = KlineChart
    ↓                                ↓
                              watch(klineChartRef) fires
                              → nextTick(initChartControllers)
                                    ↓
                              dc = new DrawingController()
                              crosshair = new Crosshair()
                              echarts.on('mousemove', ...)
                              echarts.on('mouseout', ...)


场景 2: 切换 tab 回到 K 线 (tab switch)
──────────────────────────────────────
activeTab = 'kline'
    ↓
KlineChart 重新创建 (v-if 从 false 变 true)
klineChartRef 从 null → KlineChart
    ↓
watch(klineChartRef) fires
→ nextTick(initChartControllers)
    → dc/crosshair 创建 (同场景 1)


场景 3: 切换股票 (symbol change)
──────────────────────────────────────
symbol = '000001'
    ↓
watch(symbol) fires:
  → dc.saveDrawings() (保存旧股票画线)
  → dc.destroy(), dc = null
  → crosshair.destroy(), crosshair = null
    ↓
VChart key 变化: kc-600519 → kc-000001
旧 VChart 销毁 → 新 VChart 创建 → echarts 新实例
    ↓
if (klineChartRef.value) // true, 组件没变
  → nextTick(initChartControllers)
    → dc/crosshair 创建 (关联新 echarts 实例)


场景 4: 用户点击画线按钮 (manual toggle)
──────────────────────────────────────
toggleDrawingMode() 被调用
    ↓
drawingMode = true
if (!dc) → initChartControllers() (兜底)
if (!dc) → setTimeout(120ms) → initChartControllers() again + setMode('trendline')
    ↓
dc?.setMode('trendline')
```

### 模块边界

**初始化入口（唯一）**：
- `watch(klineChartRef)` — 处理首次加载 + tab 切换回 K 线。
  - 保护条件 `if (!chart || (dc && crosshair)) return` 防止重复初始化。

**Symbol 变更处理**：
- `watch(symbol)` — 在 symbol 变更时保存画线、销毁旧 dc/crosshair、置 null、触发重新 init。

**Option 变更不再触发生命周期**：
- `watch(option)` 已移除。option 变更（数据更新、指标切换）时 dc/crosshair 保持不变，避免破坏画线状态和事件处理器泄漏。

**事件处理器管理**：
- 使用模块级变量 `echartsMoveHandler` / `echartsOutHandler` 存储处理器引用。
- 注册前调用 `echarts.off()` 清理旧处理器，确保每个 init 周期只有一套处理器。
- 旧方案：每次 init 生成新闭包 → `echarts.on()` 追加 → 累积 → 泄漏。
- 新方案：存储引用 → `echarts.off()` 清理 → `echarts.on()` 注册。

**安全兜底**：
- `initChartControllers` 内 `if (!echart) { setTimeout(() => initChartControllers(), 100); return }` — 单次 setTimeout，非循环。
- `toggleDrawingMode` 内 `if (!dc) { setTimeout(() => { initChartControllers(); dc?.setMode('trendline') }, 120) }` — 用户按钮点击时额外兜底。

### 修改文件

| 文件 | 变更 |
|------|------|
| `frontend/src/terminal/panels/CandlestickPanel.vue` | 替换 `watch(option)` + `watch(klineChartRef)` + `initChartControllers` + `toggleDrawingMode` 块 |
| `CHANGELOG.md` | 更新 Fixed 条目 |

DrawingController.ts / Crosshair.ts / KlineChart.vue — 无需修改。

### 不变的设计

- **画线持久化**：`DrawingController.saveDrawings()` / `loadDrawings()` 按 symbol 读写 localStorage，symbol 切换时保存旧 symbol → 加载新 symbol。
- **画线坐标系**：使用数据空间 `{dataIndex, price}` 保存，缩放平移后 `echarts.convertToPixel` 重新计算像素位置。
- **十字光标**：与画线共享 Canvas Overlay 层（画线 z-index: 10，十字光标 z-index: 11，pointer-events: none 确保事件穿透）。

## Acceptance Criteria

- [ ] 首次打开面板 → 数据加载完成 → 点击画线按钮 → 可在 K 线上画趋势线/水平线/斐波那契回撤/文字
- [ ] 画线状态下切换到其他 tab（分时图）→ 切回 K 线 tab → 画线工具仍可使用
- [ ] 画线状态下切换股票 symbol → 画线工具自动重新初始化 → 可在新 K 线上画线
- [ ] 切换股票后原股票画线已保存 → 切回原股票 → 画线正确恢复
- [ ] 实时数据推送（option 变更）期间画线状态不被中断 → 已画的线保留，继续可画
- [ ] 切换指标/叠加/副图期间画线状态不被中断
- [ ] 打开 Chrome DevTools Memory 面板 → 反复切换指标/周期 20 次 → echarts listener 数量不增长（无泄漏）
- [ ] 初始化期间 echarts 不可用（罕见边界）→ setTimeout 100ms 后自动恢复，不卡死 UI
- [ ] 极端场景：连续快速切换股票 10 次 → 每次画线工具重建，最终可正常使用，无泄漏
- [ ] 退出面板（onUnmounted）→ 画线保存到 localStorage → 重新打开面板 → 画线恢复

## Risks / Trade-offs

1. **`watch(klineChartRef)` 在 Vue 3 中的触发时机**：ref 在 patch 阶段赋值，watcher 在 post-flush 触发（`watch` 默认是 pre-flush，但 ref 变更发生在 render 阶段，所以按 Vue 3 队列机制推到 post-flush 执行）。`nextTick(initChartControllers)` 将初始化推到所有 DOM 变更之后，这是正确的。但如果未来 Wails 升级或 Vue 变更 ref 触发机制，需要验证此时序。

2. **`watch(symbol)` 与 `watch(klineChartRef)` 的竞态**：当 symbol 变更时，VChart key 变化触发 vue-echarts 内部重建。`watch(symbol)` 的 `nextTick(initChartControllers)` 在 VChart 新实例挂载后执行。如果 VChart 挂载是异步的（vue-echarts 内部逻辑），100ms setTimeout 兜底提供容错。

3. **`setTimeout` 不是可靠的初始化保障**：如果 echarts 实例在 100ms 后仍不可用，setTimeout 回调内 initChartControllers 会静默失败（dc 仍为 null）。用户需要再次点击画线按钮触发重试。考虑更复杂的场景（如网络延迟极高、WebSocket 重连中），可以增加 retry flag 做第二次重试，但当前场景不必要——echarts 不可用只有首次 mount 的极小窗口。

4. **`if (dc && crosshair) return` guard**：阻止 `watch(klineChartRef)` 重复初始化。但如果 dc 被外部代码意外破坏而 crosshair 仍然存在（或反之），guard 会跳过导致状态不一致。当前代码中没有地方单独破坏 dc 或 crosshair（除了 watch(symbol) 同时置空两者），所以不会发生。

5. **`DrawingController.destroy()` 不清除 canvas 内容**：destroy 重置内部引用但不 clear canvas。如果 initChartControllers 在 destroy 后 mount 前返回（因为 echarts 为 null），canvas 会保留上一帧的绘制内容。但由于 destroy 后马上重新 mount 是正常路径，只有 setTimeout 兜底路径中 canvas 会短暂残留旧画面（最多 100ms），视觉上可接受。
