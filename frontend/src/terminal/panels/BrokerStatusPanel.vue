<script setup lang="ts">
defineProps<{ panelId: string; params?: Record<string, any> }>()

interface BrokerInfo {
  name: string
  market: string
  status: 'available' | 'unavailable'
  detail: string
}

const brokers: BrokerInfo[] = [
  { name: 'Paper Trading', market: 'Simulated', status: 'available', detail: '内置模拟交易' },
  { name: 'Alpaca', market: 'US', status: 'unavailable', detail: '需配置 API Key' },
  { name: 'Binance', market: 'Crypto', status: 'unavailable', detail: '需配置 API Key' },
  { name: 'Futu', market: 'HK/CN', status: 'unavailable', detail: 'FutuOpenD 未实现（stub）' },
  { name: 'IBKR', market: 'US/HK', status: 'unavailable', detail: '未实现' },
  { name: 'OKX', market: 'Crypto', status: 'unavailable', detail: '未实现' },
]

function statusDotClass(s: string): string { return 'dot dot-' + s }
function statusLabel(s: string): string { return s === 'available' ? '可用' : '不可用' }
</script>

<template>
  <div class="broker-status">
    <div class="card-grid">
      <div v-for="b in brokers" :key="b.name" :class="['broker-card', { dimmed: b.status === 'unavailable' }]">
        <div class="card-header">
          <div class="card-name-row">
            <span :class="statusDotClass(b.status)"></span>
            <span class="card-name">{{ b.name }}</span>
          </div>
          <span :class="['status-badge', b.status]">{{ statusLabel(b.status) }}</span>
        </div>
        <div class="card-body">
          <div class="info-row"><span class="info-label">市场</span><span class="info-value">{{ b.market }}</span></div>
          <div class="info-row"><span class="info-value muted">{{ b.detail }}</span></div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.broker-status { padding: 10px; background: var(--bg); height: 100%; overflow-y: auto; color: var(--text); }
.card-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 10px; }
.broker-card { background: var(--card); border: 1px solid var(--border); border-radius: 6px; padding: 12px; display: flex; flex-direction: column; gap: 8px; }
.broker-card.dimmed { opacity: 0.55; }
.card-header { display: flex; justify-content: space-between; align-items: center; }
.card-name-row { display: flex; align-items: center; gap: 8px; }
.dot { width: 10px; height: 10px; border-radius: 50%; flex-shrink: 0; }
.dot-available { background: #22c55e; }
.dot-unavailable { background: #6b7280; }
.card-name { font-size: 14px; font-weight: 700; }
.status-badge { padding: 2px 8px; border-radius: 3px; font-size: 11px; font-weight: 500; }
.status-badge.available { background: #064e3b; color: #22c55e; }
.status-badge.unavailable { background: #1f2937; color: #6b7280; }
.card-body { display: flex; flex-direction: column; gap: 2px; }
.info-row { display: flex; justify-content: space-between; align-items: center; }
.info-label { font-size: 11px; color: var(--muted); text-transform: uppercase; }
.info-value { font-size: 12px; font-weight: 500; }
.info-value.muted { color: var(--muted); font-size: 11px; }
</style>
