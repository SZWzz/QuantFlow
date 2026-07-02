# 画线工具初始化修复 — 实施计划

## 前置

- Spec: `docs/specs/2026-07-02-drawing-tool-init-fix.md`
- 涉及文件：仅 `frontend/src/terminal/panels/CandlestickPanel.vue`（~50 行改动）

## 任务拆解

### Task 1: 替换 initChartControllers 函数

**文件**: `frontend/src/terminal/panels/CandlestickPanel.vue`

**改动**:
1. 删除旧 `watch(option, () => nextTick(() => dc?.render()))`
2. 添加模块级变量 `echartsMoveHandler` / `echartsOutHandler`
3. 重写 `initChartControllers()`:
   - 保护条件 `if (dc && crosshair) return`
   - 保留 ref/null/data 守卫
   - echarts 不可用时单次 `setTimeout(100ms)` 重试（非循环）
   - 清理旧 echarts 事件处理器再注册新的

**代码**:

```typescript
let echartsMoveHandler: ((params: any) => void) | null = null
let echartsOutHandler: (() => void) | null = null

function initChartControllers() {
  if (dc && crosshair) return

  const chart = klineChartRef.value
  const dCanvas = drawingCanvasRef.value
  const cCanvas = crosshairCanvasRef.value
  if (!chart || !dCanvas || !cCanvas) return
  if (!ohlcvData.value.length) return
  const echart = chart.getEchartsInstance?.()
  if (!echart) {
    setTimeout(() => initChartControllers(), 100)
    return
  }

  if (dc) {
    dc.saveDrawings()
    dc.destroy()
  }
  crosshair?.destroy()

  dc = new DrawingController()
  dc.mount(echart, dCanvas, symbol.value)

  crosshair = new Crosshair()
  crosshair.mount(echart, cCanvas)

  if (echartsMoveHandler) echart.off('mousemove', echartsMoveHandler)
  if (echartsOutHandler) echart.off('mouseout', echartsOutHandler)
  echartsMoveHandler = (params: any) => {
    if (!drawingMode.value && params?.event) {
      crosshair?.show(params.event.offsetX, params.event.offsetY)
    }
  }
  echartsOutHandler = () => {
    crosshair?.hide()
  }
  echart.on('mousemove', echartsMoveHandler)
  echart.on('mouseout', echartsOutHandler)
}
```

**验证**: TypeScript 编译通过 (`npx vue-tsc --noEmit`)

---

### Task 2: 替换 watch(klineChartRef) + 新增 watch(symbol)

**文件**: `frontend/src/terminal/panels/CandlestickPanel.vue`

**位置**: 紧接 `initChartControllers()` 之后

**代码**:

```typescript
watch(klineChartRef, (chart) => {
  if (!chart || (dc && crosshair)) return
  nextTick(() => initChartControllers())
})

watch(symbol, () => {
  dc?.saveDrawings()
  dc?.destroy()
  dc = null
  crosshair?.destroy()
  crosshair = null
  if (klineChartRef.value) {
    nextTick(() => initChartControllers())
  }
})
```

**说明**:
- `watch(klineChartRef)` — 首次加载 + tab 切换回 K 线时初始化；`(dc && crosshair)` guard 防止重复 init
- `watch(symbol)` — symbol 变更时：保存旧画线 → 销毁旧 dc/crosshair → 置 null → 触发新 init；`klineChartRef.value` 在 symbol 切换时不变（KlineChart 组件不销毁）

**验证**: TypeScript 编译通过

---

### Task 3: 更新 toggleDrawingMode 添加兜底重试

**文件**: `frontend/src/terminal/panels/CandlestickPanel.vue`

**代码**:

```typescript
function toggleDrawingMode() {
  drawingMode.value = !drawingMode.value
  if (drawingMode.value && !dc) {
    initChartControllers()
    if (!dc) {
      setTimeout(() => {
        initChartControllers()
        dc?.setMode('trendline')
      }, 120)
    }
  }
  dc?.setMode(drawingMode.value ? 'trendline' : 'cursor')
}
```

**说明**: `setTimeout(120ms)` 兜底 `initChartControllers` 内的 `setTimeout(100ms)`——如果 echarts 未就绪，等 VChart 挂载完成后自动重试并设置画线模式。

---

### Task 4: 更新 CHANGELOG

**文件**: `CHANGELOG.md`

在 `## [2026.7.2]` 的 `### Fixed` 下更新条目：

```markdown
### Fixed
- [Frontend] **画线工具无法使用** — 三个问题：...
```

---

### Task 5: 编译验证

```bash
cd frontend && npx vue-tsc --noEmit
cd /Volumes/shenzy/vibe_coding/QuantFlow && wails3 build
```

---

## 执行顺序

1. Task 1 → Task 2 → Task 3（这三个修改在同一文件，可一次编辑完成）
2. Task 4（CHANGELOG）
3. Task 5（验证）

每次 Task 后 `git add -p` 只添加相关改动，然后 commit。
