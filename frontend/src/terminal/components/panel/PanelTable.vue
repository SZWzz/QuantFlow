<script setup lang="ts">
import { computed, useSlots } from 'vue'
import LoadingState from './LoadingState.vue'
import type { Column } from './types'

const props = withDefaults(defineProps<{
  columns: Column[]
  data: any[]
  loading?: boolean
  striped?: boolean
  clickable?: boolean
  rowKey?: (row: any, idx: number) => string | number
  rowTestId?: string
  /** 行级附加 class（如涨跌闪烁 .flash-up/.flash-down），返回字符串拼到行根元素上 */
  rowClass?: (row: any) => string
  /** 隐藏表头行（分组表格中仅首组显示表头） */
  hideHeader?: boolean
  /** 表头粘性定位（长表滚动时保持可见） */
  stickyHeader?: boolean
  /** 受控排序状态：当前排序列与方向（null 表示未排序） */
  sortKey?: string
  sortDir?: 'asc' | 'desc' | null
}>(), {
  striped: true,
  loading: false,
  clickable: false,
  hideHeader: false,
  stickyHeader: false,
  sortKey: '',
  sortDir: null,
})

const emit = defineEmits<{
  (e: 'rowClick', row: any): void
  (e: 'rowContextmenu', row: any, event: MouseEvent): void
  /** 点击 sortable 表头；dir 为下一状态（新列 asc → desc → null 清除），父组件受控应用 */
  (e: 'sortChange', key: string, dir: 'asc' | 'desc' | null): void
}>()

function getKey(row: any, idx: number): string | number {
  if (props.rowKey) return props.rowKey(row, idx)
  return row.id ?? row.symbol ?? row.code ?? row.key ?? idx
}

function colStyle(col: Column) {
  const s: Record<string, string> = {}
  if (col.width) s.width = col.width + 'px'
  else if (col.flex) s.flex = String(col.flex)
  else s.flex = '1'
  return s
}

function formatCell(row: any, col: Column) {
  const v = row[col.key]
  if (v == null || v === '') return '--'
  if (col.formatter) return col.formatter(v)
  switch (col.format) {
    case 'price':
      return typeof v === 'number' ? v.toFixed(2) : v
    case 'percent':
      return typeof v === 'number' ? (v >= 0 ? '+' : '') + v.toFixed(2) + '%' : v
    case 'volume':
      if (typeof v !== 'number') return v
      if (v >= 1e8) return (v / 1e8).toFixed(2) + '亿'
      if (v >= 1e4) return (v / 1e4).toFixed(1) + '万'
      return String(v)
    case 'number':
      return typeof v === 'number' ? v.toFixed(4) : v
    default:
      return v
  }
}

function colorize(v: number): string {
  return v >= 0 ? 'var(--color-up)' : 'var(--color-down)'
}

/** 仅有限数值才上色；占位（undefined/null/字符串 '--' 等）不着色 */
function colorizeColor(row: any, col: Column): string | undefined {
  if (!col.colorize) return undefined
  const v = row[col.key]
  if (typeof v !== 'number' || !Number.isFinite(v)) return undefined
  return colorize(v)
}

const NUMERIC_FORMATS = new Set(['price', 'percent', 'volume', 'number'])
function isMono(col: Column): boolean {
  return col.mono ?? (col.format ? NUMERIC_FORMATS.has(col.format) : false)
}

/** 单元格原生 tooltip：仅当列定义了 title hook 时输出 */
function cellTitle(row: any, col: Column): string | undefined {
  return col.title ? col.title(row) : undefined
}

/** 单元格附加 class：空串归一为 undefined，避免输出空 class */
function cellClass(row: any, col: Column): string | undefined {
  if (!col.cellClass) return undefined
  return col.cellClass(row) || undefined
}

function onSort(col: Column) {
  let dir: 'asc' | 'desc' | null
  if (props.sortKey === col.key) {
    dir = props.sortDir === 'asc' ? 'desc' : props.sortDir === 'desc' ? null : 'asc'
  } else {
    dir = 'asc'
  }
  emit('sortChange', col.key, dir)
}

/** 当前排序列的 aria-sort 值；仅应用于排序中的列（ARIA 规范不要求每列都标 none） */
function ariaSort(col: Column): 'ascending' | 'descending' | undefined {
  if (!col.sortable || props.sortKey !== col.key || !props.sortDir) return undefined
  return props.sortDir === 'asc' ? 'ascending' : 'descending'
}

/** 行键盘激活：仅当事件源自行本身（避免行内按钮等交互元素的 keydown 冒泡误触发行点击） */
function onRowKeydown(row: any, e: KeyboardEvent) {
  if (!props.clickable || e.target !== e.currentTarget) return
  e.preventDefault()
  emit('rowClick', row)
}

/** 行间方向键导航：↑↓ 在相邻行间移动焦点 */
function focusSiblingRow(e: KeyboardEvent, dir: 1 | -1) {
  if (!props.clickable || e.target !== e.currentTarget) return
  e.preventDefault()
  const row = e.currentTarget as HTMLElement
  const sibling = (dir === 1 ? row.nextElementSibling : row.previousElementSibling) as HTMLElement | null
  if (sibling?.classList.contains('table-row')) sibling.focus()
}

const slots = useSlots()
const hasAction = computed(() => !!slots.action)
</script>

<template>
  <div class="panel-table-wrapper" role="table">
    <div v-if="!hideHeader" class="table-header-row" :class="{ sticky: stickyHeader }" role="row">
      <span
        v-for="col in columns"
        :key="col.key"
        :class="['th', col.align || 'left', { sortable: col.sortable }]"
        :style="colStyle(col)"
        role="columnheader"
        :aria-sort="ariaSort(col)"
        @click="col.sortable && onSort(col)"
      >
        <button v-if="col.sortable" type="button" class="sort-trigger">
          {{ col.label }}
          <span v-if="sortKey === col.key && sortDir" class="sort-arrow" aria-hidden="true">{{ sortDir === 'asc' ? '↑' : '↓' }}</span>
        </button>
        <template v-else>{{ col.label }}</template>
      </span>
      <span v-if="hasAction" class="th action-th" role="columnheader"> </span>
    </div>
    <div class="table-body" role="rowgroup">
      <div
        v-for="(row, idx) in data"
        :key="getKey(row, idx)"
        :class="[
          'table-row',
          { striped: striped && idx % 2 === 1, clickable: clickable },
          props.rowClass ? props.rowClass(row) : '',
        ]"
        :data-testid="rowTestId"
        role="row"
        :tabindex="clickable ? 0 : undefined"
        @click="emit('rowClick', row)"
        @keydown.enter="onRowKeydown(row, $event)"
        @keydown.space="onRowKeydown(row, $event)"
        @keydown.down="focusSiblingRow($event, 1)"
        @keydown.up="focusSiblingRow($event, -1)"
        @contextmenu="emit('rowContextmenu', row, $event)"
      >
        <span
          v-for="col in columns"
          :key="col.key"
          :class="['td', col.align || 'left', { colorize: col.colorize, mono: isMono(col) }, cellClass(row, col)]"
          :style="[{ color: colorizeColor(row, col) }, colStyle(col)]"
          :title="cellTitle(row, col)"
          role="cell"
        >
          {{ formatCell(row, col) }}
        </span>
        <span v-if="hasAction" class="td action-td" role="cell">
          <slot name="action" :row="row" />
        </span>
      </div>
      <LoadingState v-if="loading && data.length === 0" type="table" :rows="3" :cols="columns.length" />
    </div>
  </div>
</template>

<style scoped>
.panel-table-wrapper {
  flex: 1;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.table-header-row {
  display: flex;
  padding: var(--space-xs) 0;
  border-bottom: 1px solid var(--color-border);
  font-size: var(--table-header-size);
  color: var(--table-header-color);
  font-weight: var(--table-header-weight);
  flex-shrink: 0;
  letter-spacing: 0.01em;
}

.table-header-row.sticky {
  position: sticky;
  top: 0;
  z-index: 1;
  background: var(--color-bg-panel);
}

.th,
.td {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  padding: 0 var(--space-xs);
}

.th.left,
.td.left {
  text-align: left;
}

.th.sortable {
  cursor: pointer;
  user-select: none;
}

.th.sortable:hover {
  color: var(--color-text-primary);
}

/* 排序触发器：内嵌 button 提供原生键盘可达，外观与表头文字一致（click 冒泡到 th 处理） */
.sort-trigger {
  display: inline-flex;
  align-items: center;
  padding: 0;
  border: 0;
  background: none;
  font: inherit;
  letter-spacing: inherit;
  color: inherit;
  cursor: pointer;
}

.sort-arrow {
  margin-left: var(--space-xs);
}

.th.right,
.td.right {
  text-align: right;
}

.th.center,
.td.center {
  text-align: center;
}

.table-body {
  flex: 1;
  overflow-y: auto;
  font-size: var(--table-row-size);
}

.table-row {
  display: flex;
  align-items: center;
  min-height: var(--table-row-height);
  border-bottom: 1px solid var(--table-border);
  transition: background var(--transition-fast);
  cursor: default;
}

.table-row:hover {
  background: var(--table-row-hover);
}

.table-row.clickable {
  cursor: pointer;
}

.table-row.striped {
  background: var(--table-row-odd);
}

.table-row.striped:hover {
  background: var(--table-row-hover);
}

.td {
  font-variant-numeric: tabular-nums;
}

.td.mono {
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
}

.td.colorize {
  font-weight: 500;
}

.action-th,
.action-td {
  width: 40px;
  flex-shrink: 0;
  text-align: center;
}
</style>
