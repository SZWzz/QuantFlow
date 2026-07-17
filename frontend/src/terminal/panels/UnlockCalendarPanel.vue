<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { GetUnlockCalendar, type UnlockEvent } from '@/lib/wails'

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
  <div class="unlock-panel">
    <div class="toolbar">
      <h4>限售解禁日历</h4>
      <select v-model.number="days" @change="fetchData" class="day-select">
        <option :value="7">7天</option><option :value="30">30天</option><option :value="90">90天</option>
      </select>
    </div>
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
    <div v-else class="empty">未来 {{ days }} 天暂无解禁</div>
  </div>
</template>

<style scoped>
.unlock-panel { padding: 16px; height: 100%; overflow-y: auto; }
.toolbar { display: flex; justify-content: space-between; align-items: center; margin-bottom: 12px; }
.toolbar h4 { font-size: 13px; margin: 0; }
.day-select { padding: 4px 8px; border: 1px solid var(--color-border); border-radius: 4px; background: var(--color-bg-panel); color: var(--color-text-primary); font-size: 11px; }
.event-list { display: flex; flex-direction: column; gap: 8px; }
.event-item { padding: 10px 12px; border: 1px solid var(--color-border); border-radius: var(--radius-md); background: var(--color-bg-subtle); }
.event-item.warn { border-color: var(--color-danger); background: var(--color-danger-soft); }
.event-header { display: flex; gap: 10px; align-items: center; margin-bottom: 4px; }
.event-symbol { font-weight: 700; font-size: 12px; font-family: 'JetBrains Mono', monospace; }
.event-name { font-size: 12px; }
.event-date { font-size: 11px; color: var(--color-text-tertiary); margin-left: auto; }
.event-detail { display: flex; gap: 12px; font-size: 11px; color: var(--color-text-secondary); }
.warn-tag { color: var(--color-danger); font-weight: 700; }
.empty { text-align: center; padding: 48px; color: var(--color-text-tertiary); }
</style>
