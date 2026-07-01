<script setup lang="ts">
import { ref, watch, onMounted, onUnmounted } from 'vue'
import { useSymbolContext } from '@/stores/symbolContext'
import { usePanelCache } from '@/lib/composables/usePanelCache'

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const ctx = useSymbolContext()
const pg = ctx.getOrCreatePanelGroup(props.panelId)

const { fetchWithCache } = usePanelCache()
const loading = ref(false)
const error = ref('')
const constituents = ref<any[]>([])
const indexCode = ref(props.params?.symbol || '000300')

const SOURCE = 'akshare'
const DATA_TYPE = 'index'

async function loadData() {
  loading.value = true; error.value = ''
  try {
    const w = (window as any)
    if (w?.go?.main?.App?.FetchData) {
      const { data: result } = await fetchWithCache('index:' + indexCode.value, async () => {
        return await w.go.main.App.FetchData(SOURCE, DATA_TYPE, [indexCode.value], '', '', {})
      })
      if (result?.data) {
        const parsed = typeof result.data === 'string' ? JSON.parse(result.data) : result.data
        if (parsed?.success === false) {
          error.value = parsed.error || '数据获取失败'
        } else {
          constituents.value = Array.isArray(parsed) ? parsed : (parsed?.data || [])
        }
      } else if (result?.error) error.value = result.error
    }
  } catch (e: any) { error.value = e.message || '加载失败' }
  finally { loading.value = false }
}

const indexLabels: Record<string, string> = {
  '000300': '沪深 300',
  '000001': '上证指数',
  '000016': '上证 50',
  '000905': '中证 500',
  '000688': '科创 50',
  '399001': '深证成指',
  '399006': '创业板指',
  '399005': '中小板指',
}

function handleCodeSubmit(e: Event) {
  const input = e.target as HTMLInputElement
  indexCode.value = input.value.trim()
  ctx.setGroupSymbol(pg.groupId, indexCode.value)
}

let unsubSymbol: (() => void) | null = null

onMounted(() => {
  // React when another panel in the group changes the symbol
  unsubSymbol = watch(
    () => ctx.linkGroups[pg.groupId]?.activeSymbol,
    (sym) => {
      if (sym && sym !== indexCode.value) {
        indexCode.value = sym
        loadData()
      }
    },
    { immediate: true }
  )
})

onUnmounted(() => {
  if (unsubSymbol) unsubSymbol()
})
</script>

<template>
  <div class="panel-container">
    <div class="panel-header">
      <div class="header-left">
        <input class="code-input" :value="indexCode" placeholder="指数代码" @keyup.enter="handleCodeSubmit" />
        <span class="index-name">{{ indexLabels[indexCode] || indexCode }}</span>
        <span class="badge">指数成分</span>
      </div>
      <button class="btn-sm" @click="loadData">⟳ 刷新</button>
    </div>
    <div class="panel-body">
      <div v-if="loading" class="state">加载中...</div>
      <div v-else-if="error" class="state error">{{ error }}</div>
      <div v-else-if="constituents.length === 0" class="state">输入指数代码查看成分股</div>

      <template v-else>
        <div class="info-row">共 {{ constituents.length }} 只成分股</div>
        <div class="table-wrap">
          <table class="idx-table">
            <thead>
              <tr>
                <th>代码</th>
                <th>名称</th>
                <th>纳入日期</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="row in constituents" :key="row['品种代码'] || row['code']">
                <td class="td-code">{{ row['品种代码'] || row['code'] || '-' }}</td>
                <td>{{ row['品种名称'] || row['name'] || '-' }}</td>
                <td>{{ row['纳入日期'] || row['date'] || '-' }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </template>
    </div>
  </div>
</template>

<style scoped>
.panel-container { display: flex; flex-direction: column; height: 100%; background: var(--color-bg-panel); color: var(--color-text-primary); font-size: 13px; }
.panel-header { display: flex; justify-content: space-between; align-items: center; padding: 8px 12px; border-bottom: 1px solid var(--color-border); }
.header-left { display: flex; align-items: center; gap: 8px; }
.code-input { width: 80px; padding: 2px 6px; border: 1px solid var(--color-border-subtle); border-radius: 4px; background: var(--color-bg-elevated); color: var(--color-text-primary); font-size: 13px; font-family: monospace; text-align: center; }
.index-name { font-weight: 500; font-size: 14px; }
.badge { font-size: 11px; background: var(--color-primary); color: var(--color-text-primary); padding: 2px 8px; border-radius: 10px; }
.btn-sm { padding: 2px 8px; font-size: 11px; border: 1px solid var(--color-border); border-radius: 4px; background: transparent; color: var(--color-text-secondary); cursor: pointer; }
.btn-sm:hover { background: var(--color-bg-hover); }
.panel-body { flex: 1; overflow: auto; padding: 12px; }
.state { display: flex; align-items: center; justify-content: center; height: 100%; color: var(--color-text-tertiary); font-size: 13px; }
.state.error { color: var(--color-error); }
.info-row { font-size: 12px; color: var(--color-text-tertiary); margin-bottom: 8px; }
.table-wrap { overflow-x: auto; }
.idx-table { width: 100%; border-collapse: collapse; font-size: 12px; }
.idx-table th { text-align: right; padding: 4px 8px; color: var(--color-text-tertiary); font-weight: 500; border-bottom: 1px solid var(--color-border-subtle); white-space: nowrap; }
.idx-table th:first-child { text-align: left; }
.idx-table td { text-align: right; padding: 4px 8px; border-bottom: 1px solid var(--color-border-subtle); }
.idx-table tr:hover td { background: var(--color-bg-hover); }
.td-code { text-align: left !important; color: var(--color-text-secondary); font-family: monospace; }
</style>
