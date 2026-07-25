<script setup lang="ts">
import { onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useBrokerStatus } from '@/lib/composables/useBrokerStatus'
import { PanelHeader, EmptyState, ErrorState, LoadingState } from '@/terminal/components/panel'

defineProps<{ panelId: string; params?: Record<string, any> }>()

const { t } = useI18n()
const { brokers, loading, error: loadError, fetchBrokerStatuses } = useBrokerStatus()

function statusDot(c: boolean): string { return 'dot ' + (c ? 'dot-connected' : 'dot-disconnected') }
function statusText(c: boolean): string { return c ? t('common.connected') : t('common.disconnected') }
function badgeClass(c: boolean): string { return 'status-badge ' + (c ? 'connected' : 'disconnected') }

onMounted(fetchBrokerStatuses)
</script>

<template>
  <div class="broker-status">
    <PanelHeader
      :title="t('broker.title')"
      :controls="[{ icon: 'refresh', title: t('broker.refresh'), action: fetchBrokerStatuses, loading }]"
    />
    <LoadingState v-if="loading && !brokers.length" type="card" :rows="2" />
    <ErrorState v-else-if="loadError && !brokers.length" :description="loadError" @retry="fetchBrokerStatuses" />
    <div v-else class="card-grid">
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
      <EmptyState v-if="!loading && !brokers.length" :title="t('broker.no_brokers')" class="grid-empty" />
    </div>
  </div>
</template>

<style scoped>
.broker-status { height: 100%; display: flex; flex-direction: column; overflow: hidden; color: var(--color-text-primary); }

.card-grid { flex: 1; overflow-y: auto; display: grid; grid-template-columns: 1fr 1fr; gap: var(--space-sm); padding: var(--space-sm) var(--panel-padding); align-content: start; }
.grid-empty { grid-column: 1 / -1; }
.broker-card { background: var(--color-bg-subtle); border: 1px solid var(--color-accent-soft); border-radius: var(--radius-md); padding: var(--space-md); display: flex; flex-direction: column; gap: var(--space-sm); }
.broker-card.dimmed { opacity: 0.55; }
.card-header { display: flex; justify-content: space-between; align-items: center; }
.card-name-row { display: flex; align-items: center; gap: var(--space-sm); }
.dot { width: 10px; height: 10px; border-radius: 50%; flex-shrink: 0; }
.dot-connected { background: var(--color-down); }
.dot-disconnected { background: var(--color-text-tertiary); }
.card-name { font-size: var(--font-base); font-weight: 700; }
.status-badge { padding: var(--space-xs) var(--space-sm); border-radius: var(--radius-sm); font-size: var(--font-xs); font-weight: 500; }
.status-badge.connected { background: var(--color-down-soft); color: var(--color-down); }
.status-badge.disconnected { background: var(--color-bg-elevated); color: var(--color-text-tertiary); }
.card-body { display: flex; flex-direction: column; gap: var(--space-xs); }
.info-row { display: flex; justify-content: space-between; align-items: center; }
.info-label { font-size: var(--font-xs); color: var(--color-text-tertiary); text-transform: uppercase; }
.info-value { font-size: var(--font-xs); font-weight: 500; }
.info-value.muted { color: var(--color-text-tertiary); }
</style>
