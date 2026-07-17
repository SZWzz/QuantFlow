<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { GetTop10Holders, type ShareholderRecord } from '@/lib/wails'
import { useSymbolContext } from '@/stores/symbolContext'

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const ctx = useSymbolContext()

const groupId = computed(() => ctx.getPanelGroupId(props.panelId))
const linkedSymbol = computed(() => ctx.linkGroups[groupId.value]?.activeSymbol)

const symbol = ref(props.params?.symbol || linkedSymbol.value || '')
const holders = ref<ShareholderRecord[]>([])
const loading = ref(false)

watch(linkedSymbol, (s) => { if (s) { symbol.value = s; fetchData() } })
onMounted(() => { if (symbol.value) fetchData() })

async function fetchData() {
  loading.value = true
  try { holders.value = await GetTop10Holders(symbol.value) }
  catch { holders.value = [] }
  finally { loading.value = false }
}
</script>

<template>
  <div class="shareholder-panel">
    <div class="toolbar">
      <input v-model="symbol" placeholder="股票代码" class="sym-input" @keyup.enter="fetchData" />
      <button @click="fetchData" :disabled="loading" class="btn">{{ loading ? '加载中' : '查询' }}</button>
    </div>
    <div v-if="holders.length" class="holders-table">
      <h4>十大流通股东</h4>
      <table>
        <thead><tr><th>股东名称</th><th>类型</th><th>持股(万)</th><th>占比</th><th>变动</th></tr></thead>
        <tbody>
          <tr v-for="h in holders" :key="h.name">
            <td class="holder-name">{{ h.name }}</td>
            <td>{{ h.type }}</td>
            <td class="num">{{ (h.shares/10000).toFixed(0) }}</td>
            <td class="num">{{ h.pct?.toFixed(2) }}%</td>
            <td :class="'num '+(h.change>=0?'up':'down')">{{ h.change>=0?'+':'' }}{{ (h.change/10000).toFixed(0) }}</td>
          </tr>
        </tbody>
      </table>
    </div>
    <div v-else class="empty">输入股票代码查看十大流通股东</div>
  </div>
</template>

<style scoped>
.shareholder-panel { padding: 16px; height: 100%; overflow-y: auto; }
.toolbar { display: flex; gap: 8px; margin-bottom: 16px; }
.sym-input { padding: 6px 12px; border: 1px solid var(--color-border); border-radius: 4px; background: var(--color-bg-panel); color: var(--color-text-primary); font-size: 12px; font-family: 'JetBrains Mono', monospace; width: 140px; }
.btn { padding: 6px 16px; background: var(--color-accent); color: #fff; border: none; border-radius: 4px; cursor: pointer; font-size: 12px; font-weight: 600; }
.btn:disabled { opacity: 0.5; }
h4 { font-size: 13px; margin-bottom: 8px; }
table { width: 100%; border-collapse: collapse; font-size: 11px; }
th, td { padding: 5px 8px; text-align: left; border-bottom: 1px solid var(--color-border); }
th { color: var(--color-text-tertiary); font-weight: 600; }
.holder-name { font-weight: 600; max-width: 200px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.num { font-family: 'JetBrains Mono', monospace; text-align: right; }
.up { color: var(--color-success); }
.down { color: var(--color-danger); }
.empty { text-align: center; padding: 48px; color: var(--color-text-tertiary); }
</style>
