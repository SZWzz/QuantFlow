<script setup lang="ts">
import { ref, onMounted, watch, computed } from 'vue'
import { useSymbolContext } from '@/stores/symbolContext'
import { useStockName } from '@/lib/composables/useStockName'
import { usePanelCache } from '@/lib/composables/usePanelCache'
import SkeletonPanel from '@/terminal/components/SkeletonPanel.vue'

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const ctx = useSymbolContext()
const pg = ctx.getOrCreatePanelGroup(props.panelId)
const symbol = ref(props.params?.symbol || ctx.getGroupSymbol(pg.groupId) || '600519')
const { name } = useStockName(symbol)
const { fetchWithCache } = usePanelCache()
const loading = ref(false)
const error = ref('')
const result = ref<any>(null)

const scenarios = computed(() => result.value?.scenarios || {})
const buySell = computed(() => result.value?.buy_sell || {})
const fcf = computed(() => result.value?.fcf || 0)
const maxPrice = computed(() => {
  const vals = Object.values(scenarios.value).map((s: any) => s.value_per_share || 0)
  return Math.max(...vals, buySell.value?.current_price || 0) * 1.15
})

const bsColor = computed(() => {
  const s = buySell.value?.suggestion
  return s === '买入' ? '#22c55e' : s === '增持' ? '#3b82f6' : s === '减持' ? '#ef4444' : '#f59e0b'
})

async function loadData() {
  loading.value = true; error.value = ''
  try {
    const app = (window as any).go?.main?.App
    if (!app?.GetValuation) return
    const { data } = await fetchWithCache(
      `valuation:${symbol.value}`,
      async () => {
        const resp = await app.GetValuation(symbol.value)
        return resp?.data ? JSON.parse(resp.data) : resp
      },
    )
    result.value = data
  } catch (e: any) { error.value = e.message || '加载失败' }
  finally { loading.value = false }
}

watch(symbol, loadData)
watch(() => ctx.linkGroups[pg.groupId].activeSymbol, (newSym) => { if (newSym && newSym !== symbol.value) { symbol.value = newSym; loadData() } })
onMounted(loadData)
</script>

<template>
  <div class="val-panel">
    <div class="panel-header"><h3>DCF 估值</h3><div class="header-right"><span class="s">{{ symbol }} {{ name }}</span><button class="r" @click="loadData" :disabled="loading">⟳</button></div></div>
    <SkeletonPanel v-if="loading && !result" type="card" :rows="2" />
    <div v-else-if="error" class="st err">{{ error }}</div>
    <div v-else-if="!result?.scenarios || !Object.keys(result.scenarios).length" class="st">{{ result?.error || '暂无估值数据' }}</div>
    <template v-else>
      <div class="fcf"><span class="lbl">自由现金流</span><span class="val">{{ fcf.toLocaleString() }}</span></div>
      <div v-for="s in ['保守','基准','乐观']" :key="s" class="row"><span class="sn">{{ s }}</span><div class="trk"><div class="fl" :style="{ width: ((scenarios[s]?.value_per_share||0)/maxPrice*100)+'%', background: s==='保守'?'#ef4444':s==='乐观'?'#22c55e':'#f59e0b' }"/><span class="p">{{ scenarios[s]?.value_per_share?.toFixed(2) || '--' }}</span></div></div>
      <div v-if="buySell.current_price" class="row"><span class="sn cp">当前</span><div class="trk"><div class="fl" :style="{ width: (buySell.current_price/maxPrice*100)+'%', background: 'var(--color-accent)' }"/><span class="p cpv">{{ buySell.current_price?.toFixed(2) }}</span></div></div>
      <div v-if="buySell.suggestion" class="bs"><span class="bsv" :style="{ color: bsColor }">{{ buySell.suggestion }}</span><span class="bsp">空间 {{ buySell.upside_pct>0?'+':'' }}{{ buySell.upside_pct }}%</span><span class="bsf">公允价值 {{ buySell.fair_value?.toFixed(2) }}</span></div>
    </template>
  </div>
</template>

<style scoped>
.val-panel { padding: 12px; height: 100%; display: flex; flex-direction: column; color: var(--color-text, var(--color-border)); background: var(--color-bg-panel, var(--color-bg-panel)); overflow: hidden; }
.panel-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 10px; }
.panel-header h3 { margin: 0; font-size: 14px; font-weight: 600; }
.header-right { display: flex; align-items: center; gap: 8px; }
.s { font-size: 11px; padding: 2px 8px; border-radius: 4px; background: rgba(99,102,241,0.15); color: var(--color-accent); font-family: 'JetBrains Mono', monospace; }
.r { padding: 4px 10px; border: 1px solid var(--color-border-strong); border-radius: 4px; background: var(--color-bg-elevated); color: var(--color-text-primary); cursor: pointer; font-size: 13px; }
.st { display: flex; align-items: center; justify-content: center; flex: 1; color: var(--color-text-tertiary); font-size: 13px; }
.err { color: var(--color-error); }
.fcf { display: flex; gap: 8px; align-items: baseline; margin-bottom: 10px; }
.lbl { font-size: 10px; color: var(--color-text-tertiary); text-transform: uppercase; }
.val { font-size: 20px; font-weight: 700; font-variant-numeric: tabular-nums; }
.row { display: flex; align-items: center; gap: 8px; margin-bottom: 6px; }
.sn { width: 32px; font-size: 11px; color: var(--color-text-secondary); }
.cp { color: var(--color-accent); font-weight: 600; }
.trk { flex: 1; height: 22px; background: var(--color-bg-subtle); border-radius: 4px; position: relative; overflow: hidden; }
.fl { height: 100%; border-radius: 4px; opacity: 0.5; }
.p { position: absolute; right: 8px; top: 50%; transform: translateY(-50%); font-size: 11px; font-weight: 600; }
.cpv { color: var(--color-accent); }
.bs { display: flex; align-items: center; gap: 12px; margin-top: 12px; padding: 10px 14px; border: 1px solid var(--color-border); border-radius: 8px; background: var(--color-bg-elevated); }
.bsv { font-size: 16px; font-weight: 700; }
.bsp { font-size: 12px; color: var(--color-text-secondary); }
.bsf { font-size: 12px; color: var(--color-text-tertiary); margin-left: auto; }
</style>
