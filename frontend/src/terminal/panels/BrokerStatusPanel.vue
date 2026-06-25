<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'

defineProps<{ panelId: string; params?: Record<string, any> }>()

interface BrokerStatus { name: string; label: string; market: string; connected: boolean; detail: string }

const { t } = useI18n()
const brokers = ref<BrokerStatus[]>([])
const loading = ref(false)

async function loadStatus() {
  loading.value = true
  try {
    const result = await (window as any).go.main.App.GetBrokerStatuses()
    brokers.value = Array.isArray(result) ? result : []
  } catch { brokers.value = [] }
  finally { loading.value = false }
}

function statusDot(c: boolean): string { return 'dot ' + (c ? 'dot-connected' : 'dot-disconnected') }
function statusText(c: boolean): string { return c ? t('common.connected') : t('common.disconnected') }
function badgeClass(c: boolean): string { return 'status-badge ' + (c ? 'connected' : 'disconnected') }

onMounted(loadStatus)
</script>

<template>
  <div class="broker-status">
    <div class="panel-header">
      <span class="header-title">{{ $t('broker.title') }}</span>
      <button class="refresh-btn" @click="loadStatus" :disabled="loading">{{ loading ? $t('broker.refreshing') : t('broker.refresh') }}</button>
    </div>
    <div class="card-grid">
      <div v-for="b in brokers" :key="b.name" :class="['broker-card', { dimmed: !b.connected }]">
        <div class="card-header">
          <div class="card-name-row">
            <span :class="statusDot(b.connected)"></span>
            <span class="card-name">{{ b.label }}</span>
          </div>
          <span :class="badgeClass(b.connected)">{{ statusText(b.connected) }}</span>
        </div>
        <div class="card-body">
          <div class="info-row"><span class="info-label">{{ $t('broker.market_label') }}</span><span class="info-value">{{ b.market }}</span></div>
          <div class="info-row"><span class="info-value muted">{{ b.detail }}</span></div>
        </div>
      </div>
      <div v-if="!loading && !brokers.length" class="empty-state">{{ $t('broker.no_brokers') }}</div>
    </div>
  </div>
</template>

<style scoped>
.broker-status { padding: 10px; background: var(--color-bg-panel); height: 100%; overflow-y: auto; color: var(--color-text-primary); }
.panel-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 10px; }
.header-title { font-size: 13px; font-weight: 600; }
.refresh-btn { padding: 3px 10px; background: var(--color-bg-subtle); border: 1px solid var(--color-accent-soft); border-radius: 3px; color: #58a6ff; font-size: 11px; cursor: pointer; }
.refresh-btn:disabled { opacity: 0.4; cursor: default; }
.card-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 10px; }
.broker-card { background: var(--color-bg-subtle); border: 1px solid var(--color-accent-soft); border-radius: 6px; padding: 12px; display: flex; flex-direction: column; gap: 8px; }
.broker-card.dimmed { opacity: 0.55; }
.card-header { display: flex; justify-content: space-between; align-items: center; }
.card-name-row { display: flex; align-items: center; gap: 8px; }
.dot { width: 10px; height: 10px; border-radius: 50%; flex-shrink: 0; }
.dot-connected { background: #22c55e; }
.dot-disconnected { background: var(--color-text-tertiary); }
.card-name { font-size: 14px; font-weight: 700; }
.status-badge { padding: 2px 8px; border-radius: 3px; font-size: 11px; font-weight: 500; }
.status-badge.connected { background: rgba(34,197,94,0.12); color: #22c55e; }
.status-badge.disconnected { background: var(--color-bg-elevated); color: var(--color-text-tertiary); }
.card-body { display: flex; flex-direction: column; gap: 2px; }
.info-row { display: flex; justify-content: space-between; align-items: center; }
.info-label { font-size: 11px; color: var(--color-text-tertiary); text-transform: uppercase; }
.info-value { font-size: 12px; font-weight: 500; }
.info-value.muted { color: var(--color-text-tertiary); font-size: 11px; }
.empty-state { padding: 40px; text-align: center; color: var(--color-text-tertiary); font-size: 13px; grid-column: 1 / -1; }
</style>
