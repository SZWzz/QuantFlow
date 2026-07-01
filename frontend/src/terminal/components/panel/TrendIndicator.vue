<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  direction: 'up' | 'down' | 'flat'
  change?: number
}>()

const arrow = computed(() => {
  const map = { up: '▲', down: '▼', flat: '▬' }
  return map[props.direction]
})

const formattedChange = computed(() => {
  if (props.change == null) return ''
  return (props.change >= 0 ? '+' : '') + props.change.toFixed(2) + '%'
})
</script>

<template>
  <span :class="['trend-indicator', `trend-${direction}`]">
    <span class="trend-arrow">{{ arrow }}</span>
    <span v-if="change != null" class="trend-value">{{ formattedChange }}</span>
  </span>
</template>

<style scoped>
.trend-indicator {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: var(--font-sm);
  font-weight: 500;
}

.trend-up {
  color: var(--color-up);
}

.trend-down {
  color: var(--color-down);
}

.trend-flat {
  color: var(--color-text-tertiary);
}

.trend-arrow {
  font-size: 0.8em;
}

.trend-value {
  font-variant-numeric: tabular-nums;
}
</style>
