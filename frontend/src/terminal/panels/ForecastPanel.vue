<script setup lang="ts">
import { ref, onMounted, watch, computed } from 'vue'
import { useSymbolContext } from '@/stores/symbolContext'
import { useStockName } from '@/lib/composables/useStockName'
import SkeletonPanel from '@/terminal/components/SkeletonPanel.vue'

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const ctx = useSymbolContext(); const pg = ctx.getOrCreatePanelGroup(props.panelId)
const symbol = ref(props.params?.symbol || ctx.getGroupSymbol(pg.groupId) || '600519')
const { name } = useStockName(symbol); const loading = ref(false); const error = ref(''); const result = ref<any>(null)

const forecastTable = computed(() => result.value?.forecast_table || [])
const segments = computed(() => result.value?.segments || [])
const latestRev = computed(() => result.value?.latest_rev || 0)

async function loadData() {
  loading.value = true; error.value = ''
  try {
    const app = (window as any).go?.main?.App
    if (!app?.GetForecast) return
    const resp = await app.GetForecast(symbol.value)
    result.value = resp?.data ? JSON.parse(resp.data) : resp
  } catch (e: any) { error.value = e.message || '加载失败' }
  finally { loading.value = false }
}
watch(symbol, loadData)
watch(() => ctx.linkGroups[pg.groupId].activeSymbol, (n) => { if (n && n !== symbol.value) { symbol.value = n; loadData() } })
onMounted(loadData)
</script>

<template>
  <div class="fp"><div class="h"><h3>财务预测</h3><div class="hr"><span class="s">{{ symbol }} {{ name }}</span><button class="r" @click="loadData" :disabled="loading">⟳</button></div></div>
    <SkeletonPanel v-if="loading && !result" type="card" :rows="2" />
    <div v-else-if="error" class="st err">{{ error }}</div>
    <div v-else-if="!forecastTable.length" class="st">{{ result?.error || '暂无预测数据' }}</div>
    <template v-else>
      <div class="prev"><span class="pl">最近营收</span><span class="pv">{{ (latestRev/1e8).toFixed(2) }}<small> 亿</small></span></div>
      <div class="tb"><div class="th"><span class="tc">情景</span><span class="tc">增速</span><span class="tc">Y1营收(亿)</span><span class="tc">Y2营收(亿)</span><span class="tc">Y1利润(亿)</span><span class="tc">Y2利润(亿)</span></div>
        <div v-for="(r,i) in forecastTable" :key="i" class="tr" :class="'sc-'+r.scenario"><span class="tc">{{ r.scenario }}</span><span class="tc">{{ r.growth }}</span><span class="tc">{{ (r.y1_rev/1e8).toFixed(2) }}</span><span class="tc">{{ (r.y2_rev/1e8).toFixed(2) }}</span><span class="tc">{{ (r.y1_profit/1e8).toFixed(2) }}</span><span class="tc">{{ (r.y2_profit/1e8).toFixed(2) }}</span></div>
      </div>
    </template>
  </div>
</template>

<style scoped>
.fp { padding: 12px; height: 100%; display: flex; flex-direction: column; color: var(--color-text, #e5e7eb); background: var(--color-bg-panel, #1a1a2e); overflow: auto; }
.h { display: flex; justify-content: space-between; align-items: center; margin-bottom: 10px; }
.h h3 { margin: 0; font-size: 14px; font-weight: 600; }
.hr { display: flex; align-items: center; gap: 8px; }
.s { font-size: 11px; padding: 2px 8px; border-radius: 4px; background: rgba(34,197,94,0.15); color: #4ade80; font-family: 'JetBrains Mono', monospace; }
.r { padding: 4px 10px; border: 1px solid var(--color-border-strong); border-radius: 4px; background: var(--color-bg-elevated); color: var(--color-text-primary); cursor: pointer; font-size: 13px; }
.st { display: flex; align-items: center; justify-content: center; flex: 1; color: var(--color-text-tertiary); font-size: 13px; }
.err { color: var(--color-error); }
.prev { display: flex; gap: 8px; align-items: baseline; margin-bottom: 10px; }
.pl { font-size: 10px; color: var(--color-text-tertiary); text-transform: uppercase; }
.pv { font-size: 20px; font-weight: 700; }
.pv small { font-size: 11px; font-weight: 400; color: var(--color-text-tertiary); }
.tb { width: 100%; }
.th { display: flex; border-bottom: 2px solid var(--color-border-strong); font-size: 10px; color: var(--color-text-tertiary); text-transform: uppercase; }
.tc { flex: 1; padding: 4px 2px; text-align: right; }
.tr { display: flex; border-bottom: 1px solid var(--color-border-subtle); font-size: 12px; font-variant-numeric: tabular-nums; }
.tr:hover { background: var(--color-bg-elevated); }
.sc-保守 { background: rgba(239,68,68,0.04); } .sc-乐观 { background: rgba(34,197,94,0.04); }
</style>
