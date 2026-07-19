<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import VChart from 'vue-echarts'
import 'echarts'
import { GetDupontAnalysis, GetPeerRadar, type DupontBreakdown, type PeerRadar } from '@/lib/wails'
import { useChartTheme } from '@/lib/composables/useChartTheme'
import { useSymbolContext } from '@/stores/symbolContext'
import { PanelHeader } from '@/terminal/components/panel'

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const ctx = useSymbolContext()
const chartTheme = useChartTheme()

// Register this panel in the link group system so it gets the red dot and symbol sync
const panelGroup = ctx.getOrCreatePanelGroup(props.panelId)
const groupId = computed(() => panelGroup.groupId)
const linkedSymbol = computed(() => ctx.linkGroups[groupId.value]?.activeSymbol)

const symbol = ref(props.params?.symbol || linkedSymbol.value || '600519.SH')
const breakdown = ref<DupontBreakdown | null>(null)
const peers = ref<PeerRadar[]>([])

watch(linkedSymbol, (s) => { if (s) { symbol.value = s; fetchData() } })
onMounted(() => fetchData())

async function fetchData() {
  if (!symbol.value) return
  try {
    const [db, pr] = await Promise.all([
      GetDupontAnalysis(symbol.value).catch(() => null),
      GetPeerRadar(symbol.value).catch(() => []),
    ])
    breakdown.value = db
    peers.value = pr || []
  } catch { /* ignore */ }
}

const radarOption = computed(() => {
  if (peers.value.length === 0) return null
  const indic = peers.value[0] ? Object.keys(peers.value[0].metrics) : []
  return {
    tooltip: {
      backgroundColor: chartTheme.tooltipBg,
      textStyle: { color: chartTheme.tooltipText },
    },
    legend: { data: peers.value.map(p => p.name), bottom: 0, textStyle: { color: chartTheme.axisColor } },
    radar: {
      indicator: indic.map(k => ({ name: k, max: Math.max(...peers.value.map(p => p.metrics[k] || 0)) * 1.2 })),
      center: ['50%', '55%'],
      radius: '65%',
      axisName: { color: chartTheme.axisColor },
      axisLine: { lineStyle: { color: chartTheme.splitColor } },
      splitLine: { lineStyle: { color: chartTheme.splitColor } },
    },
    series: [{
      type: 'radar',
      data: peers.value.map((p, i) => ({
        name: p.name,
        value: indic.map(k => p.metrics[k] || 0),
        lineStyle: { color: i === 0 ? chartTheme.palette[0] : chartTheme.axisColor },
        areaStyle: i === 0 ? { color: chartTheme.palette[0] + '26' } : undefined,
      })),
    }],
  }
})

function pct(v: number | undefined) { return v != null ? v.toFixed(1) + '%' : '-' }
</script>

<template>
  <div class="dupont-panel">
    <PanelHeader title="杜邦分析">
      <template #controls>
        <input v-model="symbol" placeholder="股票代码" @keyup.enter="fetchData" class="sym-input" />
        <button class="btn btn-sm btn-primary" @click="fetchData">分析</button>
      </template>
    </PanelHeader>

    <div class="dupont-body">
      <!-- 杜邦树为自绘结构：分解节点 + 乘号连接，共享组件表达不了，保留但 token 化 -->
      <div v-if="breakdown" class="dupont-tree">
        <div class="tree-root">
          <span class="tree-label">ROE</span>
          <span class="tree-value">{{ breakdown.roe.toFixed(1) }}%</span>
        </div>
        <div class="tree-branches">
          <div class="tree-node">
            <span class="tree-label">净利率</span>
            <span class="tree-value">{{ breakdown.net_margin.toFixed(1) }}%</span>
            <div class="tree-sub">净利润 / 营业收入</div>
          </div>
          <div class="tree-mult">×</div>
          <div class="tree-node">
            <span class="tree-label">周转率</span>
            <span class="tree-value">{{ breakdown.asset_turnover.toFixed(2) }}</span>
            <div class="tree-sub">收入 / 总资产</div>
          </div>
          <div class="tree-mult">×</div>
          <div class="tree-node">
            <span class="tree-label">权益乘数</span>
            <span class="tree-value">{{ breakdown.equity_multiplier.toFixed(1) }}</span>
            <div class="tree-sub">总资产 / 净资产</div>
          </div>
        </div>
      </div>

      <div class="radar-section">
        <h4 class="section-title">同行对比</h4>
        <VChart v-if="radarOption" :option="radarOption" autoresize class="radar-chart" />
        <div v-else class="radar-empty">同行数据暂不可用<br/><small>（需概念板块数据 + 同行财报数据）</small></div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.dupont-panel { height: 100%; display: flex; flex-direction: column; overflow: hidden; }
.dupont-body {
  flex: 1; min-height: 0; overflow-y: auto;
  display: flex; flex-direction: column; gap: var(--space-md);
  padding: var(--panel-padding);
}
.sym-input {
  padding: var(--space-xs) var(--space-sm);
  font-size: var(--font-xs);
  border: 1px solid var(--color-border-strong);
  border-radius: var(--radius-sm);
  background: var(--color-bg-elevated);
  color: var(--color-text-primary);
  font-family: var(--font-mono);
  width: 110px;
}
.dupont-tree {
  display: flex; flex-direction: column; align-items: center; gap: var(--space-md);
  padding: var(--space-xl); background: var(--color-bg-subtle); border-radius: var(--radius-lg);
  border: 1px solid var(--color-border);
}
.tree-root {
  display: flex; flex-direction: column; align-items: center;
  padding: var(--space-md) var(--space-xl); background: var(--color-accent-soft); border: 1px solid var(--color-accent);
  border-radius: var(--radius-md);
}
.tree-label { font-size: var(--font-xs); color: var(--color-text-tertiary); }
.tree-value { font-size: var(--font-xl); font-weight: 700; color: var(--color-accent); font-family: var(--font-mono); }
.tree-branches { display: flex; align-items: center; gap: var(--space-md); }
.tree-node {
  display: flex; flex-direction: column; align-items: center; gap: var(--space-xs);
  padding: var(--space-md); background: var(--color-bg-panel); border: 1px solid var(--color-border);
  border-radius: var(--radius-sm); min-width: 120px;
}
.tree-mult { font-size: var(--font-lg); color: var(--color-text-tertiary); font-weight: 700; }
.tree-sub { font-size: var(--font-xs); color: var(--color-text-tertiary); }
.radar-section { flex: 1; display: flex; flex-direction: column; }
.radar-section h4 { margin-bottom: var(--space-sm); }
.radar-chart { flex: 1; min-height: 300px; }
.radar-empty { text-align: center; padding: var(--space-xl); color: var(--color-text-tertiary); font-size: var(--font-sm); }
.radar-empty small { display: block; margin-top: var(--space-xs); font-size: var(--font-xs); opacity: 0.7; }
</style>
