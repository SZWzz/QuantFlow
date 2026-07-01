<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { usePortfolioStore } from '@/stores/portfolio'

defineProps<{ panelId: string; params?: Record<string, any> }>()

const store = usePortfolioStore()

// -- Go Trade type --
interface GoTrade {
  ID: string; Symbol: string; Side: string; Quantity: number
  Price: number; Timestamp: string; PnL: number
}

const DISPLAY_LIMIT = 12

interface TradeEvent {
  id: string
  title: string
  status: 'info'
  time: string
  detail: string
}

const 个事件 = ref<TradeEvent[]>([])

onMounted(async () => {
  try {
    await store.fetchTrades()
    const trades = (store.trades as unknown) as GoTrade[] | null
    if (trades && trades.length > 0) {
      个事件.value = trades.slice(0, DISPLAY_LIMIT).map((t, i) => ({
        id: t.ID || `evt-${i}`,
        title: `${t.Side || '--'} ${t.Symbol || '--'}`,
        status: 'info' as const,
        time: t.Timestamp || new Date().toISOString(),
        detail: `${t.Quantity ?? 0}股 @ ${t.Price ?? 0}`,
      }))
    }
  } catch (e) {
    console.warn('[ActionCenter] fetch trades failed:', e)
  }
})

// -- Sorted newest first --
const sortedEvents = computed(() => {
  return [...个事件.value].sort((a, b) =>
    new Date(b.time).getTime() - new Date(a.time).getTime()
  )
})

function dismissEvent(id: string) {
  个事件.value = 个事件.value.filter(e => e.id !== id)
}

// -- Helpers --
function formatTime(iso: string): string {
  const d = new Date(iso)
  const now = Date.now()
  const diffMin = Math.floor((now - d.getTime()) / 60000)
  if (diffMin < 1) return 'Just now'
  if (diffMin < 60) return diffMin + 'm ago'
  const diffHrs = Math.floor(diffMin / 60)
  if (diffHrs < 24) return diffHrs + 'h ago'
  const diffDays = Math.floor(diffHrs / 24)
  return diffDays + 'd ago'
}
</script>

<template>
  <div class="action-center">
    <!-- Header -->
    <div class="ac-header">
      <span class="ac-title">{{ $t('workflow.action_center') }}</span>
      <span class="ac-count">{{ 个事件.length }} 个事件</span>
    </div>

    <!-- Event feed -->
    <div v-if="个事件.length > 0" class="event-feed">
      <div
        v-for="ev in sortedEvents"
        :key="ev.id"
        class="event-card border-info"
      >
        <div class="event-left">
          <span class="event-icon">&#9679;</span>
        </div>
        <div class="event-body">
          <div class="event-header">
            <span class="event-type-label">{{ ev.title }}</span>
            <span class="event-time">{{ formatTime(ev.time) }}</span>
          </div>
          <p class="event-message">{{ ev.detail }}</p>
          <div class="event-actions">
            <button class="evt-btn dismiss-btn" @click="dismissEvent(ev.id)">
              忽略
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- Empty state -->
    <div v-else class="empty-state">
      <p class="empty-icon">&#10003;</p>
      <p class="empty-text">{{ $t('workflow.no_recent_trades') }}</p>
      <p class="empty-sub">{{ $t('workflow.no_trades') }}</p>
    </div>
  </div>
</template>

<style scoped>
.action-center {
  padding: 10px;
  background: var(--bg);
  height: 100%;
  display: flex;
  flex-direction: column;
  color: var(--text);
}

/* -- Header -- */
.ac-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 10px;
  padding-bottom: 8px;
  border-bottom: 1px solid var(--input);
}

.ac-title {
  font-size: 14px;
  font-weight: 700;
  color: var(--text);
}

.ac-count {
  font-size: var(--font-xs);
  color: var(--muted);
  background: var(--card);
  padding: 2px 8px;
  border-radius: 4px;
}

/* -- Event feed -- */
.event-feed {
  flex: 1;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

/* -- Event card -- */
.event-card {
  display: flex;
  gap: 10px;
  padding: 10px;
  background: var(--card);
  border-radius: 4px;
  border-left: 3px solid;
}

.border-info { border-left-color: var(--color-accent); }

.event-left {
  flex-shrink: 0;
  padding-top: 2px;
}

.event-icon {
  font-size: 14px;
  color: var(--color-accent);
}

.event-body {
  flex: 1;
  min-width: 0;
}

.event-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 4px;
}

.event-type-label {
  font-size: var(--font-xs);
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.3px;
  color: var(--muted);
}

.event-time {
  font-size: var(--font-xs);
  color: var(--muted);
}

.event-message {
  font-size: 12px;
  color: var(--text);
  margin: 0 0 6px 0;
  line-height: 1.4;
}

.event-actions {
  display: flex;
  gap: 6px;
}

.evt-btn {
  padding: 3px 10px;
  border-radius: 3px;
  font-size: 10px;
  font-weight: 600;
  cursor: pointer;
  transition: opacity 0.15s;
  border: none;
}
.evt-btn:hover { opacity: 0.8; }

.dismiss-btn {
  background: transparent;
  color: var(--muted);
  border: 1px solid var(--border);
}

/* -- Empty state -- */
.empty-state {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 4px;
}

.empty-icon {
  font-size: 32px;
  color: var(--muted);
  margin: 0;
}

.empty-text {
  font-size: 14px;
  color: var(--muted);
  margin: 0;
}

.empty-sub {
  font-size: var(--font-xs);
  color: var(--muted);
  margin: 0;
}
</style>
