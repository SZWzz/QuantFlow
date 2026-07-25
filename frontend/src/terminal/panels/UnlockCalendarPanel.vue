<script setup lang="ts">
import PanelShell from '@/terminal/components/panel/PanelShell.vue'
import { ref, onMounted } from 'vue'
import { GetUnlockCalendar, type UnlockEvent } from '@/lib/wails'
import { PanelHeader, EmptyState } from '@/terminal/components/panel'

const days = ref(30)
const events = ref<UnlockEvent[]>([])

onMounted(() => fetchData())

async function fetchData() {
  try { events.value = await GetUnlockCalendar(days.value) }
  catch { events.value = [] }
}

function isWarning(e: UnlockEvent) { return e.unlock_pct > 5 }
</script>

<template>
  <PanelShell state="loaded">
    <template #loaded>
        <div class="unlock-panel">
    <PanelHeader title="限售解禁日历">
      <template #controls>
        <select v-model.number="days" class="day-select" @change="fetchData">
          <option :value="7">7天</option><option :value="30">30天</option><option :value="90">90天</option>
        </select>
      </template>
    </PanelHeader>

    <div v-if="events.length" class="event-list">
      <div v-for="e in events" :key="e.symbol+e.unlock_date" class="event-item" :class="{warn:isWarning(e)}">
        <div class="event-header">
          <span class="event-symbol">{{ e.symbol }}</span>
          <span class="event-name">{{ e.name }}</span>
          <span class="event-date">{{ e.unlock_date }}</span>
        </div>
        <div class="event-detail">
          <span>解禁 {{ (e.unlock_shares/10000).toFixed(0) }}万股</span>
          <span>占总股本 {{ e.unlock_pct?.toFixed(2) }}%</span>
          <span>市值 {{ (e.market_value/1e8).toFixed(1) }}亿</span>
          <span v-if="isWarning(e)" class="warn-tag">⚠ 高冲击</span>
        </div>
      </div>
    </div>
    <EmptyState v-else :title="`未来 ${days} 天暂无解禁`" />
  </div>
</template>

<style scoped>
.unlock-panel { height: 100%; display: flex; flex-direction: column; overflow: hidden; }
.day-select {
  padding: var(--space-xs) var(--space-sm);
  font-size: var(--font-xs);
  border: 1px solid var(--color-border-strong);
  border-radius: var(--radius-sm);
  background: var(--color-bg-elevated);
  color: var(--color-text-primary);
}
.event-list {
  flex: 1; min-height: 0; overflow-y: auto;
  display: flex; flex-direction: column; gap: var(--space-sm);
  padding: var(--panel-padding);
}
.event-item { padding: var(--space-sm) var(--space-md); border: 1px solid var(--color-border); border-radius: var(--radius-md); background: var(--color-bg-subtle); }
.event-item.warn { border-color: var(--color-danger); background: var(--color-danger-soft); }
.event-header { display: flex; gap: var(--space-sm); align-items: center; margin-bottom: var(--space-xs); }
.event-symbol { font-weight: 700; font-size: var(--font-xs); font-family: var(--font-mono); }
.event-name { font-size: var(--font-xs); }
.event-date { font-size: var(--font-xs); color: var(--color-text-tertiary); margin-left: auto; }
.event-detail { display: flex; gap: var(--space-md); font-size: var(--font-xs); color: var(--color-text-secondary); }
.warn-tag { color: var(--color-danger); font-weight: 700; }
</style>
