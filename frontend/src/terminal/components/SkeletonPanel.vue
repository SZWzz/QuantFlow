<script setup lang="ts">
withDefaults(defineProps<{
  rows?: number
  type?: 'table' | 'card' | 'chart'
}>(), {
  rows: 5,
  type: 'table',
})
</script>

<template>
  <div class="skeleton-panel" :class="`type-${type}`">
    <div v-if="type === 'table'" class="skeleton-table">
      <div class="skeleton-header-row">
        <span v-for="i in 6" :key="i" class="skeleton-cell skeleton-pulse" :style="{ width: `${8 + Math.random() * 8}%` }" />
      </div>
      <div v-for="r in rows" :key="r" class="skeleton-data-row" :style="{ animationDelay: `${r * 0.1}s` }">
        <span v-for="i in 6" :key="i" class="skeleton-cell skeleton-pulse" :style="{ width: `${10 + Math.random() * 12}%` }" />
      </div>
    </div>

    <div v-else-if="type === 'card'" class="skeleton-card-grid">
      <div v-for="r in rows" :key="r" class="skeleton-card skeleton-pulse" :style="{ animationDelay: `${r * 0.15}s` }">
        <div class="skeleton-card-title" />
        <div class="skeleton-card-value" />
        <div class="skeleton-card-sub" />
      </div>
    </div>

    <div v-else class="skeleton-chart skeleton-pulse">
      <div class="skeleton-chart-area" />
    </div>
  </div>
</template>

<style scoped>
.skeleton-panel {
  padding: 16px;
  height: 100%;
  display: flex;
  flex-direction: column;
  background: var(--color-bg-panel, #1a1a2e);
}

.skeleton-pulse {
  background: linear-gradient(
    90deg,
    var(--color-bg-elevated, #2a2a3e) 25%,
    var(--color-bg-hover, #3a3a4e) 50%,
    var(--color-bg-elevated, #2a2a3e) 75%
  );
  background-size: 200% 100%;
  animation: shimmer 1.5s ease-in-out infinite;
  border-radius: var(--radius-sm);
}

@keyframes shimmer {
  0% { background-position: 200% 0; }
  100% { background-position: -200% 0; }
}

/* Table type */
.skeleton-table {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.skeleton-header-row,
.skeleton-data-row {
  display: flex;
  gap: 12px;
  padding: 8px 0;
  border-bottom: 1px solid var(--color-border-subtle, #2a2a3e);
}
.skeleton-header-row .skeleton-cell {
  height: 10px;
  opacity: 0.5;
}
.skeleton-data-row .skeleton-cell {
  height: 14px;
}

/* Card type */
.skeleton-card-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(160px, 1fr));
  gap: 12px;
}
.skeleton-card {
  padding: 16px;
  display: flex;
  flex-direction: column;
  gap: 8px;
  min-height: 80px;
}
.skeleton-card-title {
  height: 10px;
  width: 40%;
}
.skeleton-card-value {
  height: 20px;
  width: 60%;
}
.skeleton-card-sub {
  height: 10px;
  width: 30%;
}

/* Chart type */
.skeleton-chart {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
}
.skeleton-chart-area {
  width: 80%;
  height: 60%;
  border-radius: var(--radius-lg);
}
</style>
