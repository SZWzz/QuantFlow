<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import { useSymbolContext } from '@/stores/symbolContext'

const props = defineProps<{
  panelId: string
  params?: Record<string, any>
}>()

const ctx = useSymbolContext()
const pg = ctx.getOrCreatePanelGroup(props.panelId)

interface Drawing {
  id: number
  type: 'trendline' | 'horizontal' | 'fibonacci' | 'text'
  points: { x: number; y: number }[]
  color: string
  text?: string
}

const symbol = ref(props.params?.symbol || ctx.getGroupSymbol(pg.groupId) || '600519')
const activeTool = ref('cursor')
const drawings = ref<Drawing[]>([])
const isDrawing = ref(false)
const startPoint = ref<{ x: number; y: number } | null>(null)
const currentPoint = ref<{ x: number; y: number } | null>(null)
const active颜色 = ref('#58a6ff')
const canvasRef = ref<HTMLCanvasElement | null>(null)
const nextId = ref(1)

function storageKey(): string {
  return 'drawing-panel-' + symbol.value
}

function loadDrawings() {
  try {
    const raw = localStorage.getItem(storageKey()) || '[]'
    drawings.value = JSON.parse(raw)
    nextId.value = drawings.value.reduce((max, d) => Math.max(max, d.id), 0) + 1
  } catch {
    drawings.value = []
  }
}

function saveDrawings() {
  localStorage.setItem(storageKey(), JSON.stringify(drawings.value))
}

function selectTool(tool: string) {
  activeTool.value = tool
  isDrawing.value = false
  startPoint.value = null
  currentPoint.value = null
}

function getMousePos(e: MouseEvent): { x: number; y: number } {
  const canvas = canvasRef.value
  if (!canvas) return { x: 0, y: 0 }
  const rect = canvas.getBoundingClientRect()
  return {
    x: e.clientX - rect.left,
    y: e.clientY - rect.top,
  }
}

function onMouseDown(e: MouseEvent) {
  if (activeTool.value === 'cursor') return
  isDrawing.value = true
  startPoint.value = getMousePos(e)
  currentPoint.value = getMousePos(e)
}

function onMouseMove(e: MouseEvent) {
  if (!isDrawing.value) return
  currentPoint.value = getMousePos(e)
  renderCanvas()
}

function onMouseUp(e: MouseEvent) {
  if (!isDrawing.value) return
  const end = getMousePos(e)
  const start = startPoint.value!
  const tool = activeTool.value

  if (tool === 'text') {
    const t = prompt('Enter text:')
    if (!t) {
      isDrawing.value = false
      renderCanvas()
      return
    }
    drawings.value.push({
      id: nextId.value++,
      type: 'text',
      points: [end],
      color: active颜色.value,
      text: t,
    })
  } else if (tool === 'fibonacci') {
    const dx = end.x - start.x
    const dy = end.y - start.y
    const points: { x: number; y: number }[] = []
    const ratios = [0, 0.236, 0.382, 0.5, 0.618, 0.786, 1]
    for (const r of ratios) {
      points.push({ x: start.x + r * dx, y: start.y + r * dy })
    }
    drawings.value.push({
      id: nextId.value++,
      type: 'fibonacci',
      points: [start, end],
      color: active颜色.value,
    })
  } else {
    drawings.value.push({
      id: nextId.value++,
      type: tool as Drawing['type'],
      points: [start, end],
      color: active颜色.value,
    })
  }

  isDrawing.value = false
  startPoint.value = null
  currentPoint.value = null
  saveDrawings()
  renderCanvas()
}

function renderCanvas() {
  const canvas = canvasRef.value
  if (!canvas) return
  const ctx = canvas.getContext('2d')
  if (!ctx) return

  canvas.width = canvas.clientWidth
  canvas.height = canvas.clientHeight

  // Background
  ctx.fillStyle = 'var(--color-bg-elevated)'
  ctx.fillRect(0, 0, canvas.width, canvas.height)

  // Grid
  ctx.strokeStyle = 'var(--color-border-strong)'
  ctx.lineWidth = 0.5
  const gridSize = 40
  for (let x = gridSize; x < canvas.width; x += gridSize) {
    ctx.beginPath()
    ctx.moveTo(x, 0)
    ctx.lineTo(x, canvas.height)
    ctx.stroke()
  }
  for (let y = gridSize; y < canvas.height; y += gridSize) {
    ctx.beginPath()
    ctx.moveTo(0, y)
    ctx.lineTo(canvas.width, y)
    ctx.stroke()
  }

  // Saved drawings
  for (const d of drawings.value) {
    drawShape(ctx, d)
  }

  // In-progress drawing
  if (isDrawing.value && startPoint.value && currentPoint.value) {
    const preview: Drawing = {
      id: -1,
      type: activeTool.value as Drawing['type'],
      points: [startPoint.value, currentPoint.value],
      color: active颜色.value,
    }
    drawShape(ctx, preview)
  }
}

function drawShape(ctx: CanvasRenderingContext2D, d: Drawing) {
  ctx.strokeStyle = d.color
  ctx.fillStyle = d.color
  ctx.lineWidth = 2
  ctx.font = '13px monospace'
  ctx.setLineDash([])

  if (d.type === 'trendline') {
    const [a, b] = d.points
    if (!b) return
    ctx.beginPath()
    ctx.moveTo(a.x, a.y)
    ctx.lineTo(b.x, b.y)
    ctx.stroke()
  } else if (d.type === 'horizontal') {
    const [a, b] = d.points
    if (!b) return
    ctx.beginPath()
    ctx.moveTo(0, b.y)
    ctx.lineTo(ctx.canvas.width, b.y)
    ctx.stroke()
    // Label
    ctx.fill文字(b.y.toFixed(0), 6, b.y - 4)
  } else if (d.type === 'fibonacci') {
    const [a, b] = d.points
    if (!b) return
    const dx = b.x - a.x
    const dy = b.y - a.y
    const ratios = [0, 0.236, 0.382, 0.5, 0.618, 0.786, 1]
    const colors = ['#f87171', '#fb923c', '#fbbf24', '#4ade80', '#22d3ee', '#818cf8', '#e879f9']
    for (let i = 0; i < ratios.length; i++) {
      const y = a.y + ratios[i] * dy
      ctx.strokeStyle = colors[i]
      ctx.lineWidth = 1
      ctx.setLineDash([4, 4])
      ctx.beginPath()
      ctx.moveTo(0, y)
      ctx.lineTo(ctx.canvas.width, y)
      ctx.stroke()
      ctx.fill文字((ratios[i] * 100).toFixed(1) + '%', 6, y - 4)
    }
    ctx.setLineDash([])
  } else if (d.type === 'text') {
    const [p] = d.points
    if (!p) return
    ctx.fill文字(d.text || '', p.x, p.y)
  }
}

function clearAll() {
  drawings.value = []
  saveDrawings()
  renderCanvas()
}

watch(symbol, () => {
  loadDrawings()
  // Defer render until canvas is ready
  setTimeout(() => renderCanvas(), 0)
})

watch(() => ctx.linkGroups[pg.groupId].activeSymbol, (newSym) => {
  if (pg.linked && newSym && newSym !== symbol.value) {
    symbol.value = newSym
  }
})

onMounted(() => {
  loadDrawings()
  setTimeout(() => renderCanvas(), 100)
})
</script>

<template>
  <div class="drawing-panel">
    <div class="panel-header">
      <h3>{{ $t('workflow.drawing_tools') }}</h3>
      <span class="symbol-badge">{{ symbol }}</span>
    </div>
    <div class="panel-body">
      <div class="toolbar">
        <button
          :class="['tool-btn', { active: activeTool === 'cursor' }]"
          title="光标"
          @click="selectTool('cursor')"
        >&#10037;</button>
        <button
          :class="['tool-btn', { active: activeTool === 'trendline' }]"
          title="趋势线"
          @click="selectTool('trendline')"
        >&#9585;</button>
        <button
          :class="['tool-btn', { active: activeTool === 'horizontal' }]"
          title="水平线"
          @click="selectTool('horizontal')"
        >&#9473;</button>
        <button
          :class="['tool-btn', { active: activeTool === 'fibonacci' }]"
          title="斐波那契回撤"
          @click="selectTool('fibonacci')"
        >F</button>
        <button
          :class="['tool-btn', { active: activeTool === 'text' }]"
          title="文字"
          @click="selectTool('text')"
        >T</button>
        <div class="toolbar-divider"></div>
        <input
          type="color"
          v-model="active颜色"
          class="color-picker"
          title="颜色"
        />
        <div class="toolbar-divider"></div>
        <button class="tool-btn clear-btn" title="全部清除" @click="clearAll">&#10005;</button>
      </div>
      <div class="canvas-container">
        <canvas
          ref="canvasRef"
          class="drawing-canvas"
          @mousedown="onMouseDown"
          @mousemove="onMouseMove"
          @mouseup="onMouseUp"
        ></canvas>
      </div>
    </div>
  </div>
</template>

<style scoped>
.drawing-panel {
  display: flex;
  flex-direction: column;
  height: 100%;
  background: var(--color-bg-panel);
  color: #e5e7eb;
}

.panel-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 12px;
  border-bottom: 1px solid var(--color-border-strong);
}

.panel-header h3 {
  margin: 0;
  font-size: 14px;
  font-weight: 600;
}

.symbol-badge {
  font-size: 11px;
  color: #58a6ff;
  background: rgba(88, 166, 255, 0.1);
  padding: 2px 8px;
  border-radius: 3px;
  font-variant-numeric: tabular-nums;
}

.panel-body {
  flex: 1;
  display: flex;
  overflow: hidden;
}

.toolbar {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
  padding: 8px 4px;
  border-right: 1px solid var(--color-border-strong);
  width: 60px;
  flex-shrink: 0;
}

.tool-btn {
  width: 36px;
  height: 36px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--color-bg-elevated);
  border: 1px solid var(--color-border-strong);
  color: #e5e7eb;
  border-radius: 4px;
  cursor: pointer;
  font-size: 14px;
  transition: background 0.15s, border-color 0.15s;
}

.tool-btn:hover {
  background: var(--color-border-strong);
  border-color: #4b5563;
}

.tool-btn.active {
  border-color: #58a6ff;
  background: rgba(88, 166, 255, 0.15);
  color: #58a6ff;
}

.tool-btn.clear-btn:hover {
  border-color: #f87171;
  color: #f87171;
  background: rgba(248, 113, 113, 0.1);
}

.toolbar-divider {
  width: 28px;
  height: 1px;
  background: var(--color-border-strong);
  margin: 4px 0;
}

.color-picker {
  width: 28px;
  height: 28px;
  border: 1px solid var(--color-border-strong);
  border-radius: 4px;
  background: var(--color-bg-elevated);
  cursor: pointer;
  padding: 2px;
}

.color-picker::-webkit-color-swatch-wrapper {
  padding: 0;
}

.color-picker::-webkit-color-swatch {
  border: none;
  border-radius: 2px;
}

.canvas-container {
  flex: 1;
  overflow: hidden;
  position: relative;
}

.drawing-canvas {
  width: 100%;
  height: 100%;
  display: block;
  cursor: crosshair;
}
</style>
