<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import SkeletonPanel from '@/terminal/components/SkeletonPanel.vue'
import { usePanelCache } from '@/lib/composables/usePanelCache'

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const { fetchWithCache } = usePanelCache()

interface GasData {
  SafeGasPrice: string
  ProposeGasPrice: string
  FastGasPrice: string
  suggestBaseFee: string
  gasUsedRatio: string
}

const gas = ref<GasData | null>(null)
const loading = ref(false)
const loadError = ref('')
let timer: ReturnType<typeof setInterval> | null = null

async function fetchData() {
  const app = (window as any).go?.main?.App
  if (!app?.GetGasFees) return
  loading.value = true
  loadError.value = ''
  try {
    const { data: raw } = await fetchWithCache<any>('gas_fees', () => app.GetGasFees(), 60 * 1000)
    if (raw?.success === false) {
      gas.value = null
      return
    }
    gas.value = raw?.data as GasData || null
  } catch (e: any) {
    console.error('[GasFee]', e)
    loadError.value = e?.message || String(e)
    gas.value = null
  } finally {
    loading.value = false
  }
}

function gweiPrice(price: string): string {
  const n = Number(price)
  if (isNaN(n)) return '--'
  return n.toFixed(0) + ' Gwei'
}

function usdEstimate(gwei: string, secs?: string): string {
  const n = Number(gwei)
  if (isNaN(n)) return ''
  const base = n * 21000 * 1e-9 * 3500
  if (secs) return `~${(base * 1.5).toFixed(2)} USD (${secs})`
  return `~${base.toFixed(2)} USD`
}

function gasColor(gwei: string): string {
  const n = Number(gwei)
  if (isNaN(n)) return 'var(--color-text-tertiary)'
  if (n > 100) return '#dc2626'
  if (n > 30) return '#eab308'
  return '#16a34a'
}

function blockTime(ratio: string): string {
  const n = parseFloat(ratio)
  if (isNaN(n)) return '--'
  return (n * 100).toFixed(1) + '%'
}

onMounted(() => {
  fetchData()
  timer = setInterval(fetchData, 12000)
})

onUnmounted(() => {
  if (timer) clearInterval(timer)
})
</script>

<template>
  <div class="gas-fee-panel">
    <div class="panel-header">
      <h3>{{ $t('misc.gas_tracker') }}</h3>
      <button class="refresh-btn" @click="fetchData" :disabled="loading">⟳</button>
    </div>

    <div v-if="loadError" class="panel-error">{{ loadError }}</div>
    <SkeletonPanel v-if="loading && !gas" type="table" :rows="3" />

    <div v-else-if="!gas" class="empty-state">
      <div>{{ $t('misc.gas_empty') }}</div>
      <div class="hint">{{ $t('misc.gas_hint') }}</div>
    </div>

    <template v-else>
      <div class="gas-card-grid">
        <div class="gas-card safe" :style="{ borderColor: gasColor(gas.SafeGasPrice) }">
          <div class="gas-label">{{ $t('misc.gas_safe') }}</div>
          <div class="gas-value" :style="{ color: gasColor(gas.SafeGasPrice) }">
            {{ gweiPrice(gas.SafeGasPrice) }}
          </div>
          <div class="gas-est">{{ usdEstimate(gas.SafeGasPrice, '~30m') }}</div>
        </div>
        <div class="gas-card average" :style="{ borderColor: gasColor(gas.ProposeGasPrice) }">
          <div class="gas-label">{{ $t('misc.gas_average') }}</div>
          <div class="gas-value" :style="{ color: gasColor(gas.ProposeGasPrice) }">
            {{ gweiPrice(gas.ProposeGasPrice) }}
          </div>
          <div class="gas-est">{{ usdEstimate(gas.ProposeGasPrice, '~3m') }}</div>
        </div>
        <div class="gas-card fast" :style="{ borderColor: gasColor(gas.FastGasPrice) }">
          <div class="gas-label">{{ $t('misc.gas_fast') }}</div>
          <div class="gas-value" :style="{ color: gasColor(gas.FastGasPrice) }">
            {{ gweiPrice(gas.FastGasPrice) }}
          </div>
          <div class="gas-est">{{ usdEstimate(gas.FastGasPrice, '<30s') }}</div>
        </div>
      </div>

      <div class="gas-detail">
        <div class="detail-row">
          <span class="detail-label">{{ $t('misc.gas_base_fee') }}</span>
          <span class="detail-value">{{ Number(gas.suggestBaseFee || 0).toFixed(2) }} Gwei</span>
        </div>
        <div class="detail-row">
          <span class="detail-label">{{ $t('misc.gas_utilization') }}</span>
          <span class="detail-value">{{ blockTime(gas.gasUsedRatio) }}</span>
        </div>
        <div class="detail-row">
          <span class="detail-label">{{ $t('misc.gas_tx_cost') }}</span>
          <span class="detail-value">{{ usdEstimate(gas.ProposeGasPrice) }}</span>
        </div>
      </div>
    </template>
  </div>
</template>

<style scoped>
.gas-fee-panel {
  padding: 12px; height: 100%; display: flex; flex-direction: column;
  color: var(--color-text, var(--color-border)); background: var(--color-bg-panel, var(--color-bg-panel)); overflow: hidden;
}
.panel-header { display: flex; align-items: center; gap: 8px; margin-bottom: 8px; flex-shrink: 0; }
.panel-header h3 { margin: 0; font-size: 14px; font-weight: 600; }
.refresh-btn {
  padding: 4px 10px; border: 1px solid var(--color-border-strong); border-radius: var(--radius-sm);
  background: var(--color-bg-elevated); color: var(--color-text-primary); cursor: pointer; font-size: 13px;
  margin-left: auto;
}
.refresh-btn:disabled { opacity: 0.5; cursor: not-allowed; }
.panel-error { padding: 8px 12px; margin-bottom: 8px; border-radius: var(--radius-sm); background: rgba(239,68,68,0.1); color: #ef4444; font-size: 12px; }
.empty-state {
  flex: 1; display: flex; flex-direction: column; align-items: center; justify-content: center;
  color: var(--color-text-tertiary); font-size: 13px; gap: 4px;
}
.hint { font-size: 11px; opacity: 0.6; }
.gas-card-grid { display: flex; gap: 8px; margin-bottom: 12px; flex-shrink: 0; }
.gas-card {
  flex: 1; border: 1px solid; border-radius: var(--radius-lg); padding: 12px 8px; text-align: center;
  background: var(--color-bg-elevated);
}
.gas-label { font-size: 10px; text-transform: uppercase; color: var(--color-text-tertiary); margin-bottom: 4px; }
.gas-value { font-size: 18px; font-weight: 700; font-variant-numeric: tabular-nums; }
.gas-est { font-size: 10px; color: var(--color-text-tertiary); margin-top: 4px; }
.gas-detail { display: flex; flex-direction: column; gap: 4px; }
.detail-row { display: flex; justify-content: space-between; font-size: 11px; padding: 2px 0; }
.detail-label { color: var(--color-text-tertiary); }
.detail-value { font-weight: 500; font-variant-numeric: tabular-nums; }
</style>
