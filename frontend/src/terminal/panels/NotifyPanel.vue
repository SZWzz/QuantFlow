<script setup lang="ts">
import { ref, computed } from 'vue'

defineProps<{ panelId: string; params?: Record<string, any> }>()

interface Notif { id: number; level: string; title: string; body: string; is_read: boolean; created_at: string }

const notifications = ref<Notif[]>([
  { id: 1, level: 'trade', title: 'AAPL Filled', body: 'BUY 100@185.30 filled', is_read: false, created_at: '10:35' },
  { id: 2, level: 'warn', title: 'Risk Alert', body: 'NVDA drawdown -5.2%', is_read: false, created_at: '09:52' },
  { id: 3, level: 'info', title: 'Scan Complete', body: 'Morning scan found 3 candidates', is_read: true, created_at: '09:26' },
])

const filter = ref('all')

const filtered = computed(() => {
  if (filter.value === 'all') return notifications.value
  return notifications.value.filter(n => n.level === filter.value)
})

const unreadCount = computed(() => notifications.value.filter(n => !n.is_read).length)

function markRead(id: number) { const n = notifications.value.find(x => x.id === id); if (n) n.is_read = true }
function markAllRead() { notifications.value.forEach(n => n.is_read = true) }

const levelIcon: Record<string, string> = { trade: '💹', warn: '⚠️', error: '❌', info: 'ℹ️' }
</script>

<template>
  <div class="notify-panel">
    <div class="filter-bar">
      <button v-for="lvl in ['all','trade','warn','error','info']" :key="lvl" :class="['filter-btn', { active: filter === lvl }]" @click="filter = lvl">{{ lvl === 'all' ? 'All' : lvl }}</button>
      <span class="unread-badge" v-if="unreadCount > 0">{{ unreadCount }} new</span>
    </div>
    <div class="notify-list">
      <div v-for="n in filtered" :key="n.id" :class="['notify-row', { unread: !n.is_read }]" @click="markRead(n.id)">
        <span class="notify-icon">{{ levelIcon[n.level] || '📌' }}</span>
        <div class="notify-body"><span class="notify-title">{{ n.title }}</span><span class="notify-text">{{ n.body }}</span></div>
        <span class="notify-time">{{ n.created_at }}</span>
      </div>
    </div>
    <div class="footer"><button class="read-all-btn" @click="markAllRead">Mark All Read</button></div>
  </div>
</template>

<style scoped>
.notify-panel { padding: 10px; background: #1a1a2e; height: 100%; display: flex; flex-direction: column; }
.filter-bar { display: flex; gap: 4px; margin-bottom: 8px; align-items: center; }
.filter-btn { padding: 3px 10px; background: #0f2137; border: 1px solid #1a3a5c; border-radius: 3px; color: #5a6380; font-size: 10px; text-transform: capitalize; cursor: pointer; }
.filter-btn.active { background: #1a3a5c; color: #58a6ff; border-color: #58a6ff; }
.unread-badge { margin-left: auto; padding: 2px 8px; background: #0a3d1a; border-radius: 10px; color: #3fb950; font-size: 10px; font-weight: 600; }
.notify-list { flex: 1; overflow-y: auto; }
.notify-row { display: flex; gap: 8px; padding: 8px; border-bottom: 1px solid #0f2137; cursor: pointer; transition: background 0.15s; }
.notify-row:hover { background: #16213e; }
.notify-row.unread { background: #132240; }
.notify-icon { font-size: 16px; flex-shrink: 0; margin-top: 2px; }
.notify-body { flex: 1; min-width: 0; }
.notify-title { display: block; font-size: 12px; font-weight: 600; color: #e0e0e0; }
.notify-row.unread .notify-title { color: #fff; }
.notify-text { display: block; font-size: 11px; color: #5a6380; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.notify-time { font-size: 10px; color: #5a6380; flex-shrink: 0; }
.footer { padding-top: 8px; border-top: 1px solid #0f2137; }
.read-all-btn { width: 100%; padding: 6px; background: #16213e; border: 1px solid #1a3a5c; border-radius: 4px; color: #58a6ff; font-size: 12px; cursor: pointer; }
</style>
