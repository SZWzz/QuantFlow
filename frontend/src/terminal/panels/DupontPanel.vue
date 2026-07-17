<script setup lang="ts">
import { ref, computed, onMounted, watch, nextTick } from 'vue'
import { GetDupontAnalysis, GetPeerRadar, type DupontBreakdown, type PeerRadar } from '@/lib/wails'
import { useSymbolContext } from '@/stores/symbolContext'

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const ctx = useSymbolContext()

const groupId = computed(() => ctx.getPanelGroupId(props.panelId))
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
    await nextTick()
    renderRadar()
  } catch { /* ignore */ }
}

function renderRadar() {
  if (typeof window === 'undefined' || !(window as any).echarts || peers.value.length === 0) return
  const el = document.getElementById('dupont-radar')
  if (!el) return
  const echarts = (window as any).echarts
  const chart = echarts.init(el)

  const indic = peers.value[0] ? Object.keys(peers.value[0].metrics) : []
  chart.setOption({
    tooltip: {},
    legend: { data: peers.value.map(p => p.name), bottom: 0, textStyle: { color: '#9ca3af', fontSize: 10 } },
    radar: {
      indicator: indic.map(k => ({ name: k, max: Math.max(...peers.value.map(p => p.metrics[k] || 0)) * 1.2 })),
      center: ['50%', '55%'],
      radius: '65%',
    },
    series: [{
      type: 'radar',
      data: peers.value.map((p, i) => ({
        name: p.name,
        value: indic.map(k => p.metrics[k] || 0),
        lineStyle: { color: i === 0 ? '#3b82f6' : '#6b7280' },
        areaStyle: i === 0 ? { color: 'rgba(59,130,246,0.15)' } : undefined,
      })),
    }],
  }, true)
}

function pct(v: number | undefined) { return v != null ? v.toFixed(1) + '%' : '-' }
</script>

<template>
  <div class="dupont-panel">
    <div class="toolbar">
      <input v-model="symbol" placeholder="股票代码" @keyup.enter="fetchData" class="sym-input" />
      <button @click="fetchData" class="btn">分析</button>
    </div>

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

    <div v-if="peers.length > 0" class="radar-section">
      <h4>同行对比</h4>
      <div id="dupont-radar" class="radar-chart" />
    </div>
  </div>
</template>

<style scoped>
.dupont-panel { padding: 16px; height: 100%; overflow-y: auto; display: flex; flex-direction: column; gap: 16px; }
.toolbar { display: flex; gap: 8px; }
.sym-input {
  padding: 6px 12px; border: 1px solid var(--color-border); border-radius: var(--radius-sm);
  background: var(--color-bg-panel); color: var(--color-text-primary); font-size: 13px; font-family: 'JetBrains Mono', monospace; width: 160px;
}
.btn {
  padding: 6px 16px; background: var(--color-accent); color: #fff; border: none;
  border-radius: var(--radius-sm); cursor: pointer; font-size: 12px; font-weight: 600;
}
.dupont-tree {
  display: flex; flex-direction: column; align-items: center; gap: 12px;
  padding: 24px; background: var(--color-bg-subtle); border-radius: var(--radius-lg);
  border: 1px solid var(--color-border);
}
.tree-root {
  display: flex; flex-direction: column; align-items: center;
  padding: 12px 24px; background: var(--color-accent-soft); border: 1px solid var(--color-accent);
  border-radius: var(--radius-md);
}
.tree-label { font-size: 11px; color: var(--color-text-tertiary); }
.tree-value { font-size: 24px; font-weight: 700; color: var(--color-accent); font-family: 'JetBrains Mono', monospace; }
.tree-branches { display: flex; align-items: center; gap: 16px; }
.tree-node {
  display: flex; flex-direction: column; align-items: center; gap: 4px;
  padding: 12px 16px; background: var(--color-bg-panel); border: 1px solid var(--color-border);
  border-radius: var(--radius-sm); min-width: 120px;
}
.tree-mult { font-size: 18px; color: var(--color-text-tertiary); font-weight: 700; }
.tree-sub { font-size: 10px; color: var(--color-text-tertiary); }
.radar-section { flex: 1; display: flex; flex-direction: column; }
.radar-section h4 { font-size: 13px; margin-bottom: 8px; }
.radar-chart { flex: 1; min-height: 300px; }
</style>
