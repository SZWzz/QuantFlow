<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(defineProps<{
  signal: 'bullish' | 'bearish' | 'neutral'
  showLabel?: boolean
  size?: 'sm' | 'md' | 'lg'
}>(), {
  showLabel: true,
  size: 'md',
})

const label = computed(() => {
  const map = { bullish: '偏多', bearish: '偏空', neutral: '中性' }
  return map[props.signal]
})
</script>

<template>
  <span :class="['signal-badge', `signal-${signal}`, `size-${size}`]">
    <span class="signal-dot" />
    <span v-if="showLabel" class="signal-label">{{ label }}</span>
  </span>
</template>

<style scoped>
.signal-badge {
  display: inline-flex;
  align-items: center;
  gap: 4px;
}

.signal-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  flex-shrink: 0;
}

.signal-bullish .signal-dot {
  background: var(--color-up);
}

.signal-bearish .signal-dot {
  background: var(--color-down);
}

.signal-neutral .signal-dot {
  background: var(--color-text-tertiary);
}

.signal-bullish {
  color: var(--color-up);
}

.signal-bearish {
  color: var(--color-down);
}

.signal-neutral {
  color: var(--color-text-tertiary);
}

.signal-label {
  font-weight: 600;
}

.size-sm .signal-label {
  font-size: var(--font-xs);
}

.size-md .signal-label {
  font-size: var(--font-sm);
}

.size-lg .signal-label {
  font-size: var(--font-base);
}

.size-lg .signal-dot {
  width: 8px;
  height: 8px;
}
</style>
