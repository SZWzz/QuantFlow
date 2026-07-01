<script setup lang="ts">
withDefaults(defineProps<{
  type?: 'table' | 'card' | 'chart' | 'inline'
  rows?: number
  cols?: number
  inlineWidth?: string
}>(), {
  type: 'table',
  rows: 5,
  cols: 4,
  inlineWidth: '120px',
})
</script>

<template>
  <div :class="['loading-state', `type-${type}`]">
    <template v-if="type === 'table'">
      <div v-for="i in rows" :key="i" class="skeleton-row">
        <div v-for="j in cols" :key="j" class="skeleton-cell" />
      </div>
    </template>
    <template v-else-if="type === 'card'">
      <div v-for="i in rows" :key="i" class="skeleton-card">
        <div class="skeleton-line w-60" />
        <div class="skeleton-line w-40" />
        <div class="skeleton-line w-80" />
      </div>
    </template>
    <template v-else-if="type === 'chart'">
      <div class="skeleton-chart" />
    </template>
    <template v-else>
      <div class="skeleton-inline" :style="{ width: inlineWidth }" />
    </template>
  </div>
</template>

<style scoped>
.loading-state {
  width: 100%;
}

@keyframes shimmer {
  0% { background-position: -200% 0; }
  100% { background-position: 200% 0; }
}

.skeleton-bg {
  background: linear-gradient(90deg, var(--color-bg-elevated) 25%, var(--color-bg-hover) 50%, var(--color-bg-elevated) 75%);
  background-size: 200% 100%;
  animation: shimmer 1.5s ease-in-out infinite;
  border-radius: var(--radius-sm);
}

.skeleton-row {
  display: flex;
  gap: var(--space-sm);
  padding: var(--space-sm) 0;
}

.skeleton-cell {
  flex: 1;
  height: 16px;
  background: linear-gradient(90deg, var(--color-bg-elevated) 25%, var(--color-bg-hover) 50%, var(--color-bg-elevated) 75%);
  background-size: 200% 100%;
  animation: shimmer 1.5s ease-in-out infinite;
  border-radius: var(--radius-sm);
}

.skeleton-card {
  display: flex;
  flex-direction: column;
  gap: var(--space-sm);
  padding: var(--space-md);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  margin-bottom: var(--space-sm);
  background: var(--color-bg-panel);
}

.skeleton-line {
  height: 12px;
  background: linear-gradient(90deg, var(--color-bg-elevated) 25%, var(--color-bg-hover) 50%, var(--color-bg-elevated) 75%);
  background-size: 200% 100%;
  animation: shimmer 1.5s ease-in-out infinite;
  border-radius: var(--radius-sm);
}

.w-60 { width: 60%; }
.w-40 { width: 40%; }
.w-80 { width: 80%; }

.skeleton-chart {
  height: 200px;
  background: linear-gradient(90deg, var(--color-bg-elevated) 25%, var(--color-bg-hover) 50%, var(--color-bg-elevated) 75%);
  background-size: 200% 100%;
  animation: shimmer 1.5s ease-in-out infinite;
  border-radius: var(--radius-lg);
}

.skeleton-inline {
  height: 16px;
  background: linear-gradient(90deg, var(--color-bg-elevated) 25%, var(--color-bg-hover) 50%, var(--color-bg-elevated) 75%);
  background-size: 200% 100%;
  animation: shimmer 1.5s ease-in-out infinite;
  border-radius: var(--radius-sm);
}
</style>
