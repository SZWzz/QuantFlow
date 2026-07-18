<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  label: string
  value: string | number
  /** 涨跌百分数，如 1.23 表示 +1.23%；不传则不显示 */
  delta?: number
}>()

const deltaText = computed(() => {
  if (props.delta == null) return ''
  return (props.delta >= 0 ? '+' : '') + props.delta.toFixed(2) + '%'
})
const deltaClass = computed(() => (props.delta != null && props.delta >= 0 ? 'badge-up' : 'badge-down'))
</script>

<template>
  <div class="stat-item">
    <span class="stat-label">{{ label }}</span>
    <span class="stat-value-row">
      <span class="stat-value">{{ value }}</span>
      <span v-if="delta != null" :class="['stat-delta', 'badge', deltaClass]">{{ deltaText }}</span>
    </span>
  </div>
</template>

<style scoped>
.stat-item {
  display: flex;
  flex-direction: column;
  gap: var(--space-xs);
  min-width: 0;
}

.stat-label {
  font-size: var(--font-xs);
  color: var(--color-text-tertiary);
  white-space: nowrap;
}

.stat-value-row {
  display: flex;
  align-items: baseline;
  gap: var(--space-sm);
}

.stat-value {
  font-family: var(--font-mono);
  font-size: var(--font-xl);
  font-weight: 600;
  font-variant-numeric: tabular-nums;
  color: var(--color-text-primary);
  white-space: nowrap;
}
</style>
