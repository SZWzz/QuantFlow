<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(defineProps<{
  title: string
  value?: number
  change?: number
  format?: 'price' | 'percent' | 'volume' | 'number'
  sparkline?: number[]
  clickable?: boolean
}>(), {
  format: 'price',
})

const emit = defineEmits<{
  (e: 'click'): void
}>()

const formattedValue = computed(() => {
  const v = props.value
  if (v == null) return '--'
  switch (props.format) {
    case 'price':
      return v.toFixed(2)
    case 'percent':
      return (v >= 0 ? '+' : '') + (v * 100).toFixed(2) + '%'
    case 'volume':
      if (v >= 1e8) return (v / 1e8).toFixed(2) + '亿'
      if (v >= 1e4) return (v / 1e4).toFixed(1) + '万'
      return String(v)
    case 'number':
      return v.toFixed(4)
    default:
      return String(v)
  }
})

const sparkPoints = computed(() => {
  const d = props.sparkline
  if (!d?.length) return ''
  const min = Math.min(...d)
  const max = Math.max(...d)
  const range = max - min || 1
  return d.map((v, i) => `${(i / (d.length - 1)) * 100},${30 - ((v - min) / range) * 30}`).join(' ')
})
</script>

<template>
  <div
    :class="['panel-card', { clickable: clickable }]"
    @click="clickable ? $emit('click') : undefined"
  >
    <div class="card-header">
      <span class="card-title">{{ title }}</span>
      <span
        v-if="change != null"
        :class="['badge', change >= 0 ? 'badge-up' : 'badge-down']"
      >
        {{ change >= 0 ? '+' : '' }}{{ (change * 100).toFixed(2) }}%
      </span>
    </div>
    <div class="card-value">{{ formattedValue }}</div>
    <svg
      v-if="sparkline?.length"
      class="sparkline"
      viewBox="0 0 100 30"
      preserveAspectRatio="none"
    >
      <polyline
        :points="sparkPoints"
        fill="none"
        :stroke="change != null && change >= 0 ? 'var(--color-up)' : 'var(--color-down)'"
        stroke-width="1.5"
      />
    </svg>
  </div>
</template>

<style scoped>
.panel-card {
  display: flex;
  flex-direction: column;
  gap: var(--space-sm);
  padding: var(--card-padding);
  background: var(--gradient-card);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  transition: all var(--transition-normal);
  min-width: var(--card-min-width);
  position: relative;
  overflow: hidden;
}

.panel-card:hover {
  border-color: var(--color-accent);
  box-shadow: var(--shadow-md);
  transform: translateY(-1px);
}

.panel-card.clickable {
  cursor: pointer;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: var(--space-sm);
}

.card-title {
  font-size: var(--font-sm);
  color: var(--color-text-secondary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.card-value {
  font-size: var(--font-kpi);
  font-weight: 600;
  color: var(--color-text-primary);
  font-variant-numeric: tabular-nums;
}

.sparkline {
  width: 100%;
  height: 24px;
  opacity: 0.6;
}
</style>
