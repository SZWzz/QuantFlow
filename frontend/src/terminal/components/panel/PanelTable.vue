<script setup lang="ts">
import { computed, useSlots } from 'vue'
import LoadingState from './LoadingState.vue'

interface Column {
  key: string
  label: string
  width?: number
  flex?: number
  align?: 'left' | 'right' | 'center'
  format?: 'price' | 'percent' | 'volume' | 'number'
  colorize?: boolean
  formatter?: (val: any) => string
}

const props = withDefaults(defineProps<{
  columns: Column[]
  data: any[]
  loading?: boolean
  striped?: boolean
  clickable?: boolean
  rowKey?: (row: any, idx: number) => string | number
}>(), {
  striped: true,
  loading: false,
  clickable: false,
})

const emit = defineEmits<{
  (e: 'rowClick', row: any): void
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

const slots = useSlots()
const hasAction = computed(() => !!slots.action)
</script>

<template>
  <div class="panel-table-wrapper">
    <div class="table-header-row">
      <span
        v-for="col in columns"
        :key="col.key"
        :class="['th', col.align || 'left']"
        :style="colStyle(col)"
      >
        {{ col.label }}
      </span>
      <span v-if="hasAction" class="th action-th"> </span>
    </div>
    <div class="table-body">
      <div
        v-for="(row, idx) in data"
        :key="getKey(row, idx)"
        :class="[
          'table-row',
          { striped: striped && idx % 2 === 1, clickable: clickable },
        ]"
        @click="emit('rowClick', row)"
      >
        <span
          v-for="col in columns"
          :key="col.key"
          :class="['td', col.align || 'left', { colorize: col.colorize }]"
          :style="[{ color: col.colorize ? colorize(row[col.key]) : undefined }, colStyle(col)]"
        >
          {{ formatCell(row, col) }}
        </span>
        <span v-if="hasAction" class="td action-td">
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
  padding: 6px 0;
  border-bottom: 1.5px solid var(--color-border-strong);
  font-size: var(--table-header-size);
  color: var(--table-header-color);
  font-weight: var(--table-header-weight);
  flex-shrink: 0;
  letter-spacing: 0.01em;
}

.th,
.td {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  padding: 0 6px;
}

.th.left,
.td.left {
  text-align: left;
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
