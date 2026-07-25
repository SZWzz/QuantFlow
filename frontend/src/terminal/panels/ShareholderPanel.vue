<script setup lang="ts">
import PanelShell from '@/terminal/components/panel/PanelShell.vue'
import { ref, computed, onMounted, watch } from 'vue'
import { GetTop10Holders, type ShareholderRecord } from '@/lib/wails'
import { useSymbolContext } from '@/stores/symbolContext'
import { PanelHeader, PanelTable, EmptyState, type Column } from '@/terminal/components/panel'

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const ctx = useSymbolContext()

const panelGroup = ctx.getOrCreatePanelGroup(props.panelId)
const groupId = computed(() => panelGroup.groupId)
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

const cols: Column[] = [
  { key: 'name', label: '股东名称', flex: 2, cellClass: () => 'holder-name' },
  { key: 'type', label: '类型' },
  { key: 'shares', label: '持股(万)', align: 'right', mono: true, formatter: (v: number) => (v / 10000).toFixed(0) },
  { key: 'pct', label: '占比', align: 'right', mono: true, formatter: (v: number) => v?.toFixed(2) + '%' },
  { key: 'change', label: '变动', align: 'right', mono: true, colorize: true, formatter: (v: number) => (v >= 0 ? '+' : '') + (v / 10000).toFixed(0) },
]
</script>

<template>
  <PanelShell state="loaded">
    <template #loaded>
        <div class="shareholder-panel">
    <PanelHeader title="十大流通股东">
      <template #controls>
        <input v-model="symbol" placeholder="股票代码" class="sym-input" @keyup.enter="fetchData" />
        <button class="btn btn-sm btn-primary" :disabled="loading" @click="fetchData">{{ loading ? '加载中' : '查询' }}</button>
      </template>
    </PanelHeader>

    <PanelTable
      v-if="holders.length"
      :columns="cols"
      :data="holders"
      :loading="loading"
      sticky-header
    />
    <EmptyState v-else title="输入股票代码查看十大流通股东" />
  </div>
</template>

<style scoped>
.shareholder-panel { height: 100%; display: flex; flex-direction: column; overflow: hidden; }
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
:deep(.td.holder-name) { font-weight: 600; }
</style>
