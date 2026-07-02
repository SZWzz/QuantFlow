<script setup lang="ts">
import { computed, watch, ref } from 'vue'

const props = withDefaults(defineProps<{
  title: string
  value?: number
  change?: number
  format?: 'price' | 'percent' | 'volume' | 'number'
  sparkline?: number[]
  ohlcv?: { open: number; high: number; low: number; close: number }[]
  clickable?: boolean
}>(), {
  format: 'price',
})

const emit = defineEmits<{
  (e: 'click'): void
}>()

const valueChanged = ref(false)

watch(() => props.value, (newVal, oldVal) => {
  if (oldVal !== undefined && newVal !== oldVal) {
    valueChanged.value = true
    setTimeout(() => { valueChanged.value = false }, 600)
  }
})

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

interface CandleShape {
  x: number; barW: number; top: number; bot: number
  yHigh: number; yLow: number; isUp: boolean
}

const candles = computed(() => {
  const d = props.ohlcv
  if (!d?.length) return []
  const data = d.slice(-30)
  let minLow = Infinity, maxHigh = -Infinity
  for (const c of data) {
    if (c.low < minLow) minLow = c.low
    if (c.high > maxHigh) maxHigh = c.high
  }
  const range = maxHigh - minLow || 1
  const sw = 100 / data.length
  const pad = sw * 0.15
  const barW = Math.max(sw - pad * 2, 1)
  const scaleH = 28
  return data.map((c): CandleShape => {
    const x = data.indexOf(c) * sw + pad
    const yHigh = 30 - ((c.high - minLow) / range) * scaleH - 1
    const yLow = 30 - ((c.low - minLow) / range) * scaleH - 1
    const yOpen = 30 - ((c.open - minLow) / range) * scaleH - 1
    const yClose = 30 - ((c.close - minLow) / range) * scaleH - 1
    return {
      x, barW,
      top: Math.min(yOpen, yClose),
      bot: Math.max(yOpen, yClose),
      yHigh, yLow,
      isUp: c.close >= c.open,
    }
  })
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
    <div :class="['card-value', { 'number-changed': valueChanged }]">{{ formattedValue }}</div>
    <svg
      v-if="ohlcv?.length"
      class="sparkline"
      viewBox="0 0 100 30"
      preserveAspectRatio="none"
    >
      <template v-for="(c, i) in candles" :key="i">
        <line
          :x1="c.x + c.barW / 2" :y1="c.yHigh"
          :x2="c.x + c.barW / 2" :y2="c.yLow"
          :stroke="c.isUp ? 'var(--color-up)' : 'var(--color-down)'"
          stroke-width="0.6"
        />
        <rect
          :x="c.x" :y="c.top"
          :width="c.barW" :height="Math.max(c.bot - c.top, 1)"
          :fill="c.isUp ? 'var(--color-up)' : 'var(--color-down)'"
          :rx="0.5"
        />
      </template>
    </svg>
    <svg
      v-else-if="sparkline?.length"
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
.number-changed {
  animation: number-flash 0.6s ease-out;
}

@keyframes number-flash {
  0% {
    color: var(--color-accent);
    text-shadow: 0 0 8px var(--color-accent-glow);
  }
  100% {
    color: inherit;
    text-shadow: none;
  }
}
</style>
