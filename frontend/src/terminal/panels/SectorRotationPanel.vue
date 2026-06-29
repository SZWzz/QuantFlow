<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useSessionStore } from '@/stores/session'
import { usePanelCache } from '@/lib/composables/usePanelCache'
import SkeletonPanel from '@/terminal/components/SkeletonPanel.vue'

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const session = useSessionStore()

const { fetchWithCache } = usePanelCache()
const market = ref<'CN' | 'HK' | 'US'>(props.params?.market || session.ui.activeMarket || 'CN')
const lookback = ref(20)
const loading = ref(false)

interface SectorStrength {
  name: string
  change_pct: number
  rs_ratio: number
  rs_momentum: number
}

const sectors = ref<SectorStrength[]>([])
let chartInstance: any = null

function calculateRRG(changePct: number, allPcts: number[]): { rs_ratio: number; rs_momentum: number } {
  const mean = allPcts.reduce((a, b) => a + b, 0) / (allPcts.length || 1)
  const std = Math.sqrt(allPcts.reduce((a, b) => a + (b - mean) ** 2, 0) / (allPcts.length || 1))
  const rs_ratio = std > 0 ? ((changePct - mean) / std) * 10 + 100 : 100
  const rs_momentum = rs_ratio - 100 + (changePct > 0 ? 2 : -2)
  return {
    rs_ratio: Math.round(rs_ratio * 10) / 10,
    rs_momentum: Math.round(rs_momentum * 10) / 10,
  }
}

async function fetchData() {
  loading.value = true
  try {
    const app = (window as any).go?.main?.App
    if (!app?.GetIndustryRanks) return
    const { data: ranks } = await fetchWithCache<any>('industry_ranks', () => app.GetIndustryRanks(20))
    if (ranks && ranks.length > 0) {
      const pcts = ranks.map((r: any) => r.changePct || 0)
      sectors.value = ranks.map((r: any) => {
        const rrg = calculateRRG(r.changePct || 0, pcts)
        return {
          name: r.name,
          change_pct: r.changePct || 0,
          rs_ratio: rrg.rs_ratio,
          rs_momentum: rrg.rs_momentum,
        }
      })
    }
  } catch (e) {
    console.error('[SectorRotation]', e)
  } finally {
    loading.value = false
  }
  setTimeout(renderChart, 300)
}

function renderChart() {
  if (typeof window === 'undefined' || !(window as any).echarts) return
  const echarts = (window as any).echarts
  const el = document.getElementById('rrg-chart')
  if (!el) return
  if (!chartInstance) chartInstance = echarts.init(el)

  const option = {
    tooltip: {
      trigger: 'item',
      formatter: (params: any) => {
        const s = sectors.value[params.dataIndex]
        if (!s) return ''
        return `<b>${s.name}</b><br/>涨幅: ${s.change_pct >= 0 ? '+' : ''}${s.change_pct.toFixed(2)}%<br/>RS-Ratio: ${s.rs_ratio}<br/>RS-Momentum: ${s.rs_momentum}`
      },
    },
    grid: { left: 50, right: 50, top: 40, bottom: 40 },
    xAxis: { min: 85, max: 115, splitLine: { show: true, lineStyle: { color: '#2a2a3e', type: 'dashed' } }, axisLabel: { color: '#6b7280', fontSize: 10 } },
    yAxis: { min: 85, max: 115, splitLine: { show: true, lineStyle: { color: '#2a2a3e', type: 'dashed' } }, axisLabel: { color: '#6b7280', fontSize: 10 } },
    series: [{
      type: 'scatter',
      data: sectors.value.map(s => [s.rs_ratio, s.rs_momentum, s.name]),
      symbolSize: 14,
      itemStyle: {
        color: (params: any) => {
          const x = params.data[0]
          const y = params.data[1]
          if (x >= 100 && y >= 100) return '#22c55e'
          if (x < 100 && y >= 100) return '#f59e0b'
          if (x < 100 && y < 100) return '#ef4444'
          return '#3b82f6'
        },
      },
      label: { show: true, formatter: (params: any) => params.data[2], fontSize: 9, color: '#9ca3af', position: 'right' },
    }],
  }
  chartInstance.setOption(option, true)
}

function switchMarket(mkt: 'CN' | 'HK' | 'US') {
  market.value = mkt
  fetchData()
}

onMounted(fetchData)
</script>

<template>
  <div class="sector-rotation-panel">
    <div class="panel-header">
      <h3>{{ $t('misc.sector_rotation') }}</h3>
      <div class="market-tabs">
        <button :class="['mkt-tab', { active: market === 'CN' }]" @click="switchMarket('CN')">CN</button>
        <button :class="['mkt-tab', { active: market === 'HK' }]" @click="switchMarket('HK')">HK</button>
        <button :class="['mkt-tab', { active: market === 'US' }]" @click="switchMarket('US')">US</button>
      </div>
      <div class="lookback-tabs">
        <button :class="['lb-tab', { active: lookback === 5 }]" @click="lookback = 5">5d</button>
        <button :class="['lb-tab', { active: lookback === 20 }]" @click="lookback = 20">20d</button>
        <button :class="['lb-tab', { active: lookback === 60 }]" @click="lookback = 60">60d</button>
      </div>
      <button class="refresh-btn" @click="fetchData" :disabled="loading">⟳</button>
    </div>

    <SkeletonPanel v-if="loading && sectors.length === 0" type="chart" />

    <template v-else>
      <div class="rrg-hint">
        <span class="quadrant-label leading">{{ $t('misc.rrg_leading') }}</span>
        <span class="quadrant-label weakening">{{ $t('misc.rrg_weakening') }}</span>
        <span class="quadrant-label lagging">{{ $t('misc.rrg_lagging') }}</span>
        <span class="quadrant-label improving">{{ $t('misc.rrg_improving') }}</span>
      </div>
      <div id="rrg-chart" class="rrg-chart"></div>

      <div class="table-wrapper">
        <div class="table-header">
          <span class="col-sector">{{ $t('misc.sector') }}</span>
          <span class="col-pct">{{ $t('quote.change_pct') }}</span>
          <span class="col-ratio">RS-Ratio</span>
          <span class="col-momentum">RS-Momentum</span>
          <span class="col-signal">{{ $t('misc.signal') }}</span>
        </div>
        <div class="table-body">
          <div v-for="s in sectors" :key="s.name" class="table-row">
            <span class="col-sector">{{ s.name }}</span>
            <span class="col-pct" :class="s.change_pct >= 0 ? 'up' : 'down'">
              {{ s.change_pct >= 0 ? '+' : '' }}{{ s.change_pct.toFixed(2) }}%
            </span>
            <span class="col-ratio">{{ s.rs_ratio.toFixed(1) }}</span>
            <span class="col-momentum">{{ s.rs_momentum.toFixed(1) }}</span>
            <span class="col-signal" :class="s.rs_ratio >= 100 && s.rs_momentum >= 100 ? 'leading' : s.rs_ratio < 100 && s.rs_momentum < 100 ? 'lagging' : 'neutral'">
              {{ s.rs_ratio >= 100 && s.rs_momentum >= 100 ? $t('misc.leading') : s.rs_ratio < 100 && s.rs_momentum < 100 ? $t('misc.lagging') : s.rs_ratio >= 100 ? $t('misc.improving') : $t('misc.weakening') }}
            </span>
          </div>
        </div>
      </div>
    </template>
  </div>
</template>

<style scoped>
.sector-rotation-panel {
  padding: 12px;
  height: 100%;
  display: flex;
  flex-direction: column;
  color: var(--color-text, #e5e7eb);
  background: var(--color-bg-panel, #1a1a2e);
  overflow: hidden;
}
.panel-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
  flex-shrink: 0;
}
.panel-header h3 { margin: 0; font-size: 14px; font-weight: 600; }
.market-tabs, .lookback-tabs { display: flex; gap: 4px; }
.mkt-tab, .lb-tab {
  padding: 2px 10px; border: 1px solid var(--color-border-strong); border-radius: 4px;
  background: transparent; color: var(--color-text-tertiary); cursor: pointer; font-size: 11px;
}
.mkt-tab.active, .lb-tab.active { color: #60a5fa; border-color: #3b82f6; background: rgba(59,130,246,0.1); }
.refresh-btn {
  padding: 4px 10px; border: 1px solid var(--color-border-strong); border-radius: 4px;
  background: var(--color-bg-elevated); color: var(--color-text-primary); cursor: pointer; font-size: 13px;
  margin-left: auto;
}
.refresh-btn:disabled { opacity: 0.5; cursor: not-allowed; }

.rrg-hint {
  display: flex; justify-content: space-around; margin-bottom: 4px; font-size: 10px;
}
.quadrant-label { padding: 2px 8px; border-radius: 4px; }
.quadrant-label.leading { color: #22c55e; }
.quadrant-label.weakening { color: #f59e0b; }
.quadrant-label.lagging { color: #ef4444; }
.quadrant-label.improving { color: #3b82f6; }

.rrg-chart { height: 200px; margin-bottom: 8px; flex-shrink: 0; }

.table-wrapper { flex: 1; overflow: hidden; display: flex; flex-direction: column; }
.table-header {
  display: flex; padding: 4px 0; border-bottom: 1px solid var(--color-border-strong);
  font-size: 10px; color: var(--color-text-tertiary); text-transform: uppercase; flex-shrink: 0;
}
.table-body { flex: 1; overflow-y: auto; font-size: 12px; }
.table-row {
  display: flex; padding: 3px 0; align-items: center;
  border-bottom: 1px solid var(--color-border-subtle);
}
.table-row:hover { background: var(--color-bg-elevated); }
.col-sector { width: 80px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.col-pct { width: 64px; text-align: right; font-weight: 500; }
.col-ratio, .col-momentum { width: 64px; text-align: right; color: var(--color-text-secondary); }
.col-signal { width: 60px; text-align: center; font-size: 11px; font-weight: 500; }
.up { color: #dc2626; }
.down { color: #16a34a; }
.leading { color: #22c55e; }
.lagging { color: #ef4444; }
.neutral { color: #f59e0b; }
</style>
