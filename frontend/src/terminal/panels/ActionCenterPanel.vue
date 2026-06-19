<script setup lang="ts">
import { ref, computed } from 'vue'

defineProps<{ panelId: string; params?: Record<string, any> }>()

// -- Event types --
type EventType = 'stop-loss' | 'take-profit' | 'dividend' | 'split' | 'large-order'

interface ActionEvent {
  id: number
  type: EventType
  message: string
  timestamp: string
  needsApproval: boolean
}

// -- Mock events (12 items, realistic symbols and prices) --
const events = ref<ActionEvent[]>([
  {
    id: 1, type: 'stop-loss',
    message: 'TSLA stop-loss at $185.00 (-8.2%)',
    timestamp: new Date(Date.now() - 2 * 60000).toISOString(),
    needsApproval: false,
  },
  {
    id: 2, type: 'take-profit',
    message: 'AAPL take-profit at $210.50 (+15.3%)',
    timestamp: new Date(Date.now() - 8 * 60000).toISOString(),
    needsApproval: false,
  },
  {
    id: 3, type: 'dividend',
    message: '600519 分红 ¥21.91/股 除权日 2026-06-25',
    timestamp: new Date(Date.now() - 30 * 60000).toISOString(),
    needsApproval: false,
  },
  {
    id: 4, type: 'split',
    message: 'NVDA 10:1 split effective 2026-06-20',
    timestamp: new Date(Date.now() - 120 * 60000).toISOString(),
    needsApproval: false,
  },
  {
    id: 5, type: 'large-order',
    message: 'Buy 5000 NVDA @ $148.32 (~$741,600) needs approval',
    timestamp: new Date(Date.now() - 180 * 60000).toISOString(),
    needsApproval: true,
  },
  {
    id: 6, type: 'stop-loss',
    message: 'GOOGL stop-loss at $155.00 (-5.1%)',
    timestamp: new Date(Date.now() - 240 * 60000).toISOString(),
    needsApproval: false,
  },
  {
    id: 7, type: 'dividend',
    message: '000858 分红 ¥3.20/股 除权日 2026-07-02',
    timestamp: new Date(Date.now() - 360 * 60000).toISOString(),
    needsApproval: false,
  },
  {
    id: 8, type: 'take-profit',
    message: 'META take-profit at $580.00 (+22.7%)',
    timestamp: new Date(Date.now() - 480 * 60000).toISOString(),
    needsApproval: false,
  },
  {
    id: 9, type: 'large-order',
    message: 'Sell 2000 300750 @ ¥210.00 (~¥420,000) needs approval',
    timestamp: new Date(Date.now() - 600 * 60000).toISOString(),
    needsApproval: true,
  },
  {
    id: 10, type: 'split',
    message: 'AMZN 20:1 split effective 2026-06-25',
    timestamp: new Date(Date.now() - 720 * 60000).toISOString(),
    needsApproval: false,
  },
  {
    id: 11, type: 'stop-loss',
    message: 'MSFT stop-loss at $390.00 (-4.7%)',
    timestamp: new Date(Date.now() - 840 * 60000).toISOString(),
    needsApproval: false,
  },
  {
    id: 12, type: 'dividend',
    message: 'AAPL dividend $0.25/share ex-date 2026-08-07',
    timestamp: new Date(Date.now() - 960 * 60000).toISOString(),
    needsApproval: false,
  },
])

// -- Sorted newest first --
const sortedEvents = computed(() => {
  return [...events.value].sort((a, b) =>
    new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime()
  )
})

function dismissEvent(id: number) {
  events.value = events.value.filter(e => e.id !== id)
}

function approveEvent(id: number) {
  const ev = events.value.find(e => e.id === id)
  if (ev) {
    ev.needsApproval = false
    ev.message = ev.message.replace('needs approval', 'APPROVED')
  }
}

// -- Helpers --
function eventIcon(type: EventType): string {
  switch (type) {
    case 'stop-loss':    return '🔴' // red circle
    case 'take-profit':  return '🟡' // yellow circle
    case 'dividend':     return '🔵' // blue circle
    case 'split':        return '⚪'       // white circle
    case 'large-order':  return '🟠' // orange circle
  }
}

function eventLabel(type: EventType): string {
  switch (type) {
    case 'stop-loss':    return 'Stop-Loss Triggered'
    case 'take-profit':  return 'Take-Profit Triggered'
    case 'dividend':     return 'Dividend Announcement'
    case 'split':        return 'Stock Split'
    case 'large-order':  return 'Large Order Pending'
  }
}

function borderClass(type: EventType): string {
  return 'border-' + type
}

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
      <span class="ac-title">Action Center</span>
      <span class="ac-count">{{ events.length }} pending</span>
    </div>

    <!-- Event feed -->
    <div v-if="events.length > 0" class="event-feed">
      <div
        v-for="ev in sortedEvents"
        :key="ev.id"
        :class="['event-card', borderClass(ev.type)]"
      >
        <div class="event-left">
          <span class="event-icon">{{ eventIcon(ev.type) }}</span>
        </div>
        <div class="event-body">
          <div class="event-header">
            <span class="event-type-label">{{ eventLabel(ev.type) }}</span>
            <span class="event-time">{{ formatTime(ev.timestamp) }}</span>
          </div>
          <p class="event-message">{{ ev.message }}</p>
          <div class="event-actions">
            <button
              v-if="ev.needsApproval"
              class="evt-btn approve-btn"
              @click="approveEvent(ev.id)"
            >
              Approve
            </button>
            <button
              v-else-if="ev.type === 'large-order'"
              class="evt-btn confirmed-btn"
              disabled
            >
              Confirmed
            </button>
            <button class="evt-btn dismiss-btn" @click="dismissEvent(ev.id)">
              Dismiss
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- Empty state -->
    <div v-else class="empty-state">
      <p class="empty-icon">&#10003;</p>
      <p class="empty-text">No pending actions</p>
      <p class="empty-sub">All clear</p>
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

.border-stop-loss   { border-left-color: #ef4444; }
.border-take-profit { border-left-color: #f59e0b; }
.border-dividend    { border-left-color: #3b82f6; }
.border-split       { border-left-color: #6b7280; }
.border-large-order { border-left-color: #f97316; }

.event-left {
  flex-shrink: 0;
  padding-top: 2px;
}

.event-icon {
  font-size: 14px;
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

.approve-btn {
  background: #0a3d1a;
  color: var(--up);
  border: 1px solid var(--up);
}

.confirmed-btn {
  background: #0a3d1a;
  color: var(--up);
  border: 1px solid var(--up);
  opacity: 0.5;
  cursor: default;
}

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
