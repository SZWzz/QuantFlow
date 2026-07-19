<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import VChart from 'vue-echarts'
import 'echarts'
import { useTerminalStore } from '@/stores/terminal'
import { GetSectorHeatmap, GetSectorValuation } from '@/lib/wails'
import { useChartTheme } from '@/lib/composables/useChartTheme'
import { PanelHeader, EmptyState, LoadingState } from '@/terminal/components/panel'

const terminal = useTerminalStore()
const chartTheme = useChartTheme()
const market = ref('CN')
const sectors = ref<any[]>([])
const valuations = ref<any[]>([])
const loading = ref(false)
const sortKey = ref('changePct')

const marketTabs = [
  { key: 'CN', label: 'A股' },
  { key: 'HK', label: '港股' },
  { key: 'US', label: '美股' },
]

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

const heatmapOption = computed(() => ({
  tooltip: {
    formatter: (p: any) => {
      const s = sectors.value.find(x => x.name === p.name)
      return `${p.name}<br/>涨跌: ${s?.change_pct?.toFixed(1) ?? '-'}%`
    },
  },
  series: [{
    type: 'treemap',
    roam: false,
    data: sectors.value.map(s => ({ name: s.name, value: Math.abs(s.change_pct) * 100 })),
    label: {
      show: true,
      formatter: (p: any) => `${p.name}\n${sectors.value.find(s => s.name === p.name)?.change_pct?.toFixed(1)}%`,
    },
    itemStyle: { borderColor: chartTheme.bgColor },
    levels: [{ colorMappingBy: 'value', color: [chartTheme.downColor, chartTheme.upColor] }],
  }],
}))
</script>

<template>
  <div class="sector-dashboard">
    <PanelHeader
      title="行业仪表盘"
      :tabs="marketTabs"
      :active-tab="market"
      @tab-change="switchMarket"
    />

    <LoadingState v-if="loading" type="chart" />

    <div v-else class="dashboard-grid">
      <div class="heatmap-panel">
        <h4 class="section-title">行业热力图</h4>
        <VChart v-if="sectors.length" :option="heatmapOption" autoresize class="chart" />
        <EmptyState v-else title="暂无行业数据" />
      </div>

      <div class="valuation-panel">
        <h4 class="section-title">估值水位 <span class="sort-link" @click="sortBy('pe')">PE↑</span></h4>
        <div class="table-wrap">
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
  </div>
</template>

<style scoped>
.sector-dashboard { height: 100%; display: flex; flex-direction: column; overflow: hidden; }
.dashboard-grid {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: var(--space-lg);
  padding: var(--space-md) var(--panel-padding);
}
.heatmap-panel, .valuation-panel { overflow: hidden; display: flex; flex-direction: column; }
.section-title { margin-bottom: var(--space-sm); }
.chart { flex: 1; min-height: 300px; }
.sort-link { font-size: var(--font-xs); color: var(--color-accent); cursor: pointer; margin-left: var(--space-sm); }
.table-wrap { flex: 1; overflow: auto; }
table { width: 100%; border-collapse: collapse; font-size: var(--font-xs); }
th, td { padding: var(--space-xs) var(--space-sm); text-align: left; border-bottom: 1px solid var(--color-border); }
th { color: var(--color-text-tertiary); font-weight: 600; }
.val-row { cursor: pointer; }
.val-row:hover { background: var(--color-bg-hover); }
.sector-name { font-weight: 600; }
.pct-bar { display: inline-block; height: 6px; background: var(--color-accent); border-radius: var(--radius-sm); margin-right: var(--space-xs); vertical-align: middle; }
.pct-bar.high { background: var(--color-danger); }
.pct-bar.low { background: var(--color-success); }
</style>
