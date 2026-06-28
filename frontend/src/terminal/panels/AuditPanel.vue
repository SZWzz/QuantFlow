<script setup lang="ts">
import { ref, onMounted, watch, computed } from 'vue'
import { useSymbolContext } from '@/stores/symbolContext'
import { useStockName } from '@/lib/composables/useStockName'
import SkeletonPanel from '@/terminal/components/SkeletonPanel.vue'

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const ctx = useSymbolContext(); const pg = ctx.getOrCreatePanelGroup(props.panelId)
const symbol = ref(props.params?.symbol || ctx.getGroupSymbol(pg.groupId) || '600519')
const { name } = useStockName(symbol); const loading = ref(false); const error = ref(''); const result = ref<any>(null)

const findings = computed(() => result.value?.findings || [])
const riskScore = computed(() => result.value?.risk_score ?? 0)
const riskGrade = computed(() => result.value?.risk_grade ?? '--')
const gradeColor = computed(() => riskGrade.value === '高风险' ? '#ef4444' : riskGrade.value === '中等风险' ? '#f59e0b' : '#22c55e')
const levelIcon = (l: string) => l === 'high' ? '🔴' : l === 'medium' ? '🟡' : '🟢'

async function loadData() {
  loading.value = true; error.value = ''
  try {
    const app = (window as any).go?.main?.App
    if (!app?.GetAuditFindings) return
    const resp = await app.GetAuditFindings(symbol.value)
    result.value = resp?.data ? JSON.parse(resp.data) : resp
  } catch (e: any) { error.value = e.message || '加载失败' }
  finally { loading.value = false }
}
watch(symbol, loadData)
watch(() => ctx.linkGroups[pg.groupId].activeSymbol, (n) => { if (n && n !== symbol.value) { symbol.value = n; loadData() } })
onMounted(loadData)
</script>

<template>
  <div class="ap"><div class="h"><h3>财务审计</h3><div class="hr"><span class="s">{{ symbol }} {{ name }}</span><button class="r" @click="loadData" :disabled="loading">⟳</button></div></div>
    <SkeletonPanel v-if="loading && !result" type="card" :rows="2" />
    <div v-else-if="error" class="st err">{{ error }}</div>
    <div v-else-if="!findings.length" class="st">暂无审计发现</div>
    <template v-else>
      <div class="sc"><span class="scl">风险等级</span><span class="scv" :style="{ color: gradeColor }">{{ riskGrade }}</span><span class="scs">评分 {{ riskScore }}</span></div>
      <div v-for="(f,i) in findings" :key="i" class="fr" :class="f.level"><span class="fi">{{ levelIcon(f.level) }}</span><div class="fb"><span class="fm">{{ f.metric }}</span><span class="fd">{{ f.detail }}</span></div><span class="fv">{{ f.value }}</span></div>
    </template>
  </div>
</template>

<style scoped>
.ap { padding: 12px; height: 100%; display: flex; flex-direction: column; color: var(--color-text, #e5e7eb); background: var(--color-bg-panel, #1a1a2e); overflow: auto; }
.h { display: flex; justify-content: space-between; align-items: center; margin-bottom: 10px; }
.h h3 { margin: 0; font-size: 14px; font-weight: 600; }
.hr { display: flex; align-items: center; gap: 8px; }
.s { font-size: 11px; padding: 2px 8px; border-radius: 4px; background: rgba(239,68,68,0.15); color: #f87171; font-family: 'JetBrains Mono', monospace; }
.r { padding: 4px 10px; border: 1px solid var(--color-border-strong); border-radius: 4px; background: var(--color-bg-elevated); color: var(--color-text-primary); cursor: pointer; font-size: 13px; }
.st { display: flex; align-items: center; justify-content: center; flex: 1; color: var(--color-text-tertiary); font-size: 13px; }
.err { color: var(--color-error); }
.sc { display: flex; align-items: baseline; gap: 8px; margin-bottom: 8px; }
.scl { font-size: 10px; color: var(--color-text-tertiary); text-transform: uppercase; }
.scv { font-size: 20px; font-weight: 700; }
.scs { font-size: 12px; color: var(--color-text-tertiary); }
.fr { display: flex; align-items: flex-start; gap: 8px; padding: 8px; border-radius: 6px; margin-bottom: 4px; }
.fr.high { background: rgba(239,68,68,0.08); } .fr.medium { background: rgba(245,158,11,0.06); } .fr.low { background: rgba(34,197,94,0.04); }
.fi { font-size: 12px; flex-shrink: 0; margin-top: 1px; }
.fb { flex: 1; min-width: 0; }
.fm { font-size: 12px; font-weight: 600; display: block; }
.fd { font-size: 10px; color: var(--color-text-tertiary); margin-top: 2px; }
.fv { font-size: 11px; font-weight: 600; color: var(--color-text-secondary); white-space: nowrap; }
</style>
