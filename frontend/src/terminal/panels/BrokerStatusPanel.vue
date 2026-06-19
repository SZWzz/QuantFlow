<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'

defineProps<{ panelId: string; params?: Record<string, any> }>()

// -- Broker definitions --
interface BrokerInfo {
  name: string
  market: string
  status: 'connected' | 'degraded' | 'disconnected'
  latency: number
  accountBalance: number
  currency: string
  todayTrades: number
  lastHeartbeat: string
  configured: boolean
}

const brokers = ref<BrokerInfo[]>([
  {
    name: 'Paper Trading', market: 'Simulated',
    status: 'connected', latency: 2, accountBalance: 100000, currency: 'USD',
    todayTrades: 24, lastHeartbeat: new Date().toISOString(), configured: true,
  },
  {
    name: 'Futu', market: 'US/HK/CN',
    status: 'connected', latency: 45, accountBalance: 528000, currency: 'HKD',
    todayTrades: 8, lastHeartbeat: new Date().toISOString(), configured: true,
  },
  {
    name: 'Binance', market: 'Crypto',
    status: 'degraded', latency: 120, accountBalance: 15500, currency: 'USDT',
    todayTrades: 3, lastHeartbeat: new Date(Date.now() - 120000).toISOString(), configured: true,
  },
  {
    name: 'Alpaca', market: 'US',
    status: 'disconnected', latency: 0, accountBalance: 0, currency: 'USD',
    todayTrades: 0, lastHeartbeat: '', configured: false,
  },
  {
    name: 'IBKR', market: 'US/HK',
    status: 'disconnected', latency: 0, accountBalance: 0, currency: 'USD',
    todayTrades: 0, lastHeartbeat: '', configured: false,
  },
  {
    name: 'OKX', market: 'Crypto',
    status: 'disconnected', latency: 0, accountBalance: 0, currency: 'USDT',
    todayTrades: 0, lastHeartbeat: '', configured: false,
  },
])

// -- Test connection --
const testingBroker = ref<string | null>(null)

async function testConnection(broker: BrokerInfo) {
  testingBroker.value = broker.name
  // Simulate 1s connection check
  await new Promise(resolve => setTimeout(resolve, 1000))
  const idx = brokers.value.findIndex(b => b.name === broker.name)
  if (idx >= 0) {
    if (broker.configured) {
      brokers.value[idx].status = 'connected'
      brokers.value[idx].latency = Math.floor(Math.random() * 80 + 2)
      brokers.value[idx].lastHeartbeat = new Date().toISOString()
    }
  }
  testingBroker.value = null
}

// -- Auto-refresh every 30s --
let timer: ReturnType<typeof setInterval> | null = null

function refreshHeartbeats() {
  for (const b of brokers.value) {
    if (b.status !== 'disconnected') {
      b.lastHeartbeat = new Date().toISOString()
      b.latency = b.latency + Math.floor(Math.random() * 6 - 3)
      if (b.latency < 1) b.latency = 1
    }
  }
}

onMounted(() => {
  timer = setInterval(refreshHeartbeats, 30000)
})

onUnmounted(() => {
  if (timer) clearInterval(timer)
})

// -- Helpers --
function statusDotClass(s: string): string {
  return 'dot dot-' + s
}

function statusLabel(s: string): string {
  return s.charAt(0).toUpperCase() + s.slice(1)
}

function formatTime(iso: string): string {
  if (!iso) return '—'
  return new Date(iso).toLocaleTimeString('zh-CN', { hour12: false })
}

function fmtMoney(n: number, currency: string): string {
  if (n >= 1e6) return currency + ' ' + (n / 1e6).toFixed(2) + 'M'
  if (n >= 1e3) return currency + ' ' + (n / 1e3).toFixed(1) + 'K'
  return currency + ' ' + n.toFixed(2)
}
</script>

<template>
  <div class="broker-status">
    <div class="card-grid">
      <div
        v-for="broker in brokers"
        :key="broker.name"
        :class="['broker-card', { dimmed: !broker.configured }]"
      >
        <!-- Header -->
        <div class="card-header">
          <div class="card-name-row">
            <span :class="statusDotClass(broker.status)"></span>
            <span class="card-name">{{ broker.name }}</span>
          </div>
          <span v-if="!broker.configured" class="not-configured-badge">Not Configured</span>
        </div>

        <!-- Status info -->
        <div class="card-body">
          <div class="info-row">
            <span class="info-label">Status</span>
            <span :class="['info-value', 'status-' + broker.status]">
              {{ statusLabel(broker.status) }}
            </span>
          </div>
          <div class="info-row">
            <span class="info-label">Latency</span>
            <span class="info-value">{{ broker.latency > 0 ? broker.latency + ' ms' : '—' }}</span>
          </div>
          <div class="info-row">
            <span class="info-label">Heartbeat</span>
            <span class="info-value muted">{{ formatTime(broker.lastHeartbeat) }}</span>
          </div>
          <div class="info-row">
            <span class="info-label">Balance</span>
            <span class="info-value">{{ fmtMoney(broker.accountBalance, broker.currency) }}</span>
          </div>
          <div class="info-row">
            <span class="info-label">Today's Trades</span>
            <span class="info-value">{{ broker.todayTrades }}</span>
          </div>
        </div>

        <!-- Footer -->
        <div class="card-footer">
          <button
            class="test-btn"
            :disabled="testingBroker === broker.name"
            @click="testConnection(broker)"
          >
            {{ testingBroker === broker.name ? 'Testing...' : 'Test Connection' }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.broker-status {
  padding: 10px;
  background: var(--bg);
  height: 100%;
  overflow-y: auto;
  color: var(--text);
  font-variant-numeric: tabular-nums;
}

/* -- Card grid -- */
.card-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 10px;
}

.broker-card {
  background: var(--card);
  border: 1px solid var(--border);
  border-radius: 6px;
  padding: 12px;
  display: flex;
  flex-direction: column;
  gap: 10px;
  transition: opacity 0.2s;
}

.broker-card.dimmed {
  opacity: 0.50;
}

/* -- Card header -- */
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.card-name-row {
  display: flex;
  align-items: center;
  gap: 8px;
}

.dot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  flex-shrink: 0;
}
.dot-connected    { background: #22c55e; }
.dot-degraded     { background: #f59e0b; }
.dot-disconnected { background: #ef4444; }

.card-name {
  font-size: 14px;
  font-weight: 700;
  color: var(--text);
}

.not-configured-badge {
  padding: 2px 8px;
  background: var(--bg);
  border: 1px solid var(--border);
  border-radius: 3px;
  font-size: var(--font-xs);
  color: var(--muted);
  text-transform: uppercase;
  letter-spacing: 0.3px;
}

/* -- Card body -- */
.card-body {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.info-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.info-label {
  font-size: var(--font-xs);
  color: var(--muted);
  text-transform: uppercase;
  letter-spacing: 0.3px;
}

.info-value {
  font-size: 12px;
  font-weight: 500;
  color: var(--text);
}

.info-value.muted { color: var(--muted); }

.status-connected    { color: #22c55e; }
.status-degraded     { color: #f59e0b; }
.status-disconnected { color: var(--muted); }

/* -- Card footer -- */
.card-footer {
  padding-top: 6px;
  border-top: 1px solid var(--input);
}

.test-btn {
  width: 100%;
  padding: 5px;
  background: var(--input);
  border: 1px solid var(--border);
  border-radius: 4px;
  color: var(--accent);
  font-size: 11px;
  font-weight: 500;
  cursor: pointer;
  transition: background 0.15s;
}
.test-btn:hover:not(:disabled) { background: var(--card); }
.test-btn:disabled { opacity: 0.5; cursor: not-allowed; }
</style>
