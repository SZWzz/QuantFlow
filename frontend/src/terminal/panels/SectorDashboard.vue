<script setup lang="ts">
import { ref, onMounted, watch, nextTick } from 'vue'
import { useTerminalStore } from '@/stores/terminal'
import { GetSectorHeatmap, GetSectorValuation } from '@/lib/wails'
import { useChartTheme } from '@/lib/composables/useChartTheme'

const terminal = useTerminalStore()
const chartTheme = useChartTheme()
const market = ref('CN')
const sectors = ref<any[]>([])
const valuations = ref<any[]>([])
const loading = ref(false)
const sortKey = ref('changePct')

onMounted(() => fetchData())

async function fetchData() {
  loading.value = true
  try {
    const [s, v] = await Promise.all([
      GetSectorHeatmap(market.value).catch(() => []),
      GetSectorValuation(market.value).catch(() => []),
    ])
    sectors.value = s || []
    valuations.value = v || []
  } catch { /* empty */ }
  finally { loading.value = false }
  await nextTick()
  renderHeatmap()
}

function switchMarket(m: string) { market.value = m; fetchData() }

function drillDown(name: string) {
  terminal.openPanel('stock-research', { industry: name, market: market.value })
}

function sortBy(key: string) {
  sortKey.value = key
  if (key === 'changePct') sectors.value.sort((a, b) => b.change_pct - a.change_pct)
  else if (key === 'pe') valuations.value.sort((a, b) => (a.pe || 0) - (b.pe || 0))
}

function renderHeatmap() {
  if (typeof window === 'undefined' || !(window as any).echarts) return
  const el = document.getElementById('sector-heatmap')
  if (!el || sectors.value.length === 0) return
  const echarts = (window as any).echarts
  const chart = echarts.init(el)
  chart.setOption({
    tooltip: { formatter: (p: any) => `${p.name}<br/>涨跌: ${p.value[1]}%` },
    series: [{
      type: 'treemap', roam: false,
      data: sectors.value.map(s => ({ name: s.name, value: Math.abs(s.change_pct) * 100 })),
      label: { show: true, formatter: (p: any) => `${p.name}\n${sectors.value.find(s=>s.name===p.name)?.change_pct?.toFixed(1)}%` },
      itemStyle: { borderColor: chartTheme.bgColor },
      levels: [{ colorMappingBy: 'value', color: [chartTheme.downColor, chartTheme.upColor] }],
    }],
  }, true)
}

const marketLabel = (m: string) => ({ CN: 'A股', HK: '港股', US: '美股' }[m] || m)
</script>

<template>
  <div class="sector-dashboard">
    <div class="toolbar">
      <div class="market-tabs">
        <button v-for="m in ['CN','HK','US']" :key="m"
          :class="{active:market===m}" @click="switchMarket(m)">{{ marketLabel(m) }}</button>
      </div>
    </div>

    <div v-if="loading" class="loading">加载中...</div>

    <div v-else class="dashboard-grid">
      <div class="heatmap-panel">
        <h4>行业热力图</h4>
        <div id="sector-heatmap" class="chart" />
      </div>

      <div class="valuation-panel">
        <h4>估值水位 <span class="sort-link" @click="sortBy('pe')">PE↑</span></h4>
        <table>
          <thead><tr><th>行业</th><th>PE</th><th>分位</th><th>PB</th><th>分位</th><th>ROE</th></tr></thead>
          <tbody>
            <tr v-for="v in valuations" :key="v.name" @click="drillDown(v.name)" class="val-row">
              <td class="sector-name">{{ v.name }}</td>
              <td>{{ v.pe || '-' }}</td>
              <td>
                <span class="pct-bar" :style="{width:(v.pe_pct||50)+'%'}"
                  :class="{high:(v.pe_pct||0)>70,low:(v.pe_pct||0)<30}" />
                {{ v.pe_pct || '-' }}%
              </td>
              <td>{{ v.pb || '-' }}</td>
              <td>{{ v.pb_pct || '-' }}%</td>
              <td>{{ v.roe || '-' }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>

<style scoped>
.sector-dashboard { padding: 16px; height: 100%; overflow-y: auto; display: flex; flex-direction: column; gap: 12px; }
.toolbar { display: flex; justify-content: space-between; align-items: center; }
.market-tabs { display: flex; gap: 4px; }
.market-tabs button {
  padding: 4px 16px; border: 1px solid var(--color-border); background: var(--color-bg-subtle);
  color: var(--color-text-secondary); border-radius: var(--radius-sm); cursor: pointer; font-size: 12px; font-weight: 600;
}
.market-tabs button.active { background: var(--color-accent); color: #fff; border-color: var(--color-accent); }
.loading { text-align: center; padding: 48px; color: var(--color-text-tertiary); }
.dashboard-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 16px; flex: 1; min-height: 0; }
.heatmap-panel, .valuation-panel { overflow: hidden; display: flex; flex-direction: column; }
.heatmap-panel h4, .valuation-panel h4 { font-size: 13px; margin-bottom: 8px; }
.chart { flex: 1; min-height: 300px; }
.sort-link { font-size: 11px; color: var(--color-accent); cursor: pointer; margin-left: 8px; }
table { width: 100%; border-collapse: collapse; font-size: 11px; }
th, td { padding: 4px 8px; text-align: left; border-bottom: 1px solid var(--color-border); }
th { color: var(--color-text-tertiary); font-weight: 600; }
.val-row { cursor: pointer; }
.val-row:hover { background: var(--color-bg-hover); }
.sector-name { font-weight: 600; }
.pct-bar { display: inline-block; height: 6px; background: var(--color-accent); border-radius: 3px; margin-right: 4px; vertical-align: middle; }
.pct-bar.high { background: var(--color-danger); }
.pct-bar.low { background: var(--color-success); }
</style>
