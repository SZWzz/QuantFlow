<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { usePanelCache } from '@/lib/composables/usePanelCache'
import { useWailsApp } from '@/lib/composables/useWailsApp'
import { PanelHeader, EmptyState, ErrorState, LoadingState } from '@/terminal/components/panel'
import { logger } from '@/lib/logger'

defineProps<{ panelId: string; params?: Record<string, any> }>()
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
  const app = useWailsApp()
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
    logger.error('[GasFee]', e)
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

/** gas 水平分档着色：>100 危险 / >30 偏高 / 其余正常（自绘卡片用，token 化） */
function gasColor(gwei: string): string {
  const n = Number(gwei)
  if (isNaN(n)) return 'var(--color-text-tertiary)'
  if (n > 100) return 'var(--color-danger)'
  if (n > 30) return 'var(--color-warn)'
  return 'var(--color-success)'
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
    <PanelHeader
      :title="$t('misc.gas_tracker')"
      :controls="[{ icon: 'refresh', title: $t('common.refresh'), action: fetchData, loading }]"
    />

    <ErrorState v-if="loadError" :description="loadError" @retry="fetchData" />
    <LoadingState v-else-if="loading && !gas" type="card" :rows="1" />

    <EmptyState
      v-else-if="!gas"
      :title="$t('misc.gas_empty')"
      :description="$t('misc.gas_hint')"
    />

    <div v-else class="gas-body">
      <!-- 自绘卡片：PanelTable/StatItem 表达不了动态边框+分档着色，保留但 token 化 -->
      <div class="gas-card-grid">
        <div class="gas-card" :style="{ borderColor: gasColor(gas.SafeGasPrice) }">
          <div class="gas-label">{{ $t('misc.gas_safe') }}</div>
          <div class="gas-value" :style="{ color: gasColor(gas.SafeGasPrice) }">
            {{ gweiPrice(gas.SafeGasPrice) }}
          </div>
          <div class="gas-est">{{ usdEstimate(gas.SafeGasPrice, '~30m') }}</div>
        </div>
        <div class="gas-card" :style="{ borderColor: gasColor(gas.ProposeGasPrice) }">
          <div class="gas-label">{{ $t('misc.gas_average') }}</div>
          <div class="gas-value" :style="{ color: gasColor(gas.ProposeGasPrice) }">
            {{ gweiPrice(gas.ProposeGasPrice) }}
          </div>
          <div class="gas-est">{{ usdEstimate(gas.ProposeGasPrice, '~3m') }}</div>
        </div>
        <div class="gas-card" :style="{ borderColor: gasColor(gas.FastGasPrice) }">
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
    </div>
  </div>
</template>

<style scoped>
.gas-fee-panel {
  height: 100%;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.gas-body {
  flex: 1;
  overflow-y: auto;
  padding: var(--panel-padding);
}

.gas-card-grid { display: flex; gap: var(--space-sm); margin-bottom: var(--space-md); }
.gas-card {
  flex: 1; border: 1px solid; border-radius: var(--radius-lg); padding: var(--space-md) var(--space-sm); text-align: center;
  background: var(--color-bg-elevated);
}
.gas-label { font-size: var(--font-xs); text-transform: uppercase; color: var(--color-text-tertiary); margin-bottom: var(--space-xs); }
.gas-value { font-size: var(--font-lg); font-weight: 700; font-variant-numeric: tabular-nums; }
.gas-est { font-size: var(--font-xs); color: var(--color-text-tertiary); margin-top: var(--space-xs); }
.gas-detail { display: flex; flex-direction: column; gap: var(--space-xs); }
.detail-row { display: flex; justify-content: space-between; font-size: var(--font-xs); padding: var(--space-xs) 0; }
.detail-label { color: var(--color-text-tertiary); }
.detail-value { font-weight: 500; font-variant-numeric: tabular-nums; }
</style>
