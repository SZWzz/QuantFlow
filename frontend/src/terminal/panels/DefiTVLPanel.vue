<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import SkeletonPanel from '@/terminal/components/SkeletonPanel.vue'
import { usePanelCache } from '@/lib/composables/usePanelCache'

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()

interface Protocol {
  name: string
  chain: string
  tvl: number
  change_1d: number
  change_7d: number
  mcap: number
  category: string
}

const protocols = ref<Protocol[]>([])
const loading = ref(false)
const loadError = ref('')
const { fetchWithCache } = usePanelCache()
const search = ref('')
const sortKey = ref<string>('tvl')
const sortDir = ref<number>(-1)

const filtered = computed(() => {
  const kw = search.value.toLowerCase()
  let arr = kw ? protocols.value.filter(p =>
    p.name.toLowerCase().includes(kw) || p.chain.toLowerCase().includes(kw)
  ) : [...protocols.value]
  arr.sort((a, b) => {
    const aVal = a[sortKey.value as keyof Protocol] ?? 0
    const bVal = b[sortKey.value as keyof Protocol] ?? 0
    return (typeof aVal === 'number' ? aVal - bVal : String(aVal).localeCompare(String(bVal))) * sortDir.value
  })
  return arr
})

function toggleSort(key: string) {
  if (sortKey.value === key) sortDir.value *= -1
  else { sortKey.value = key; sortDir.value = -1 }
}

function sortArrow(key: string): string {
  if (sortKey.value !== key) return ''
  return sortDir.value === -1 ? ' ▼' : ' ▲'
}

async function fetchData() {
  const app = (window as any).go?.main?.App
  if (!app?.GetDeFiTVL) return
  loadError.value = ''
  loading.value = true
  try {
    const { data: result } = await fetchWithCache<any>('defi_tvl', () => app.GetDeFiTVL(), 3 * 60 * 1000)
    const items = result?.data || []
    protocols.value = items.slice(0, 150).map((p: any) => ({
      name: p.name || p.id || '?',
      chain: p.chain || (p.chains?.[0] || 'multi'),
      tvl: p.tvl || p.tvl30d?.at(-1)?.[1] || 0,
      change_1d: p.change_1d || 0,
      change_7d: p.change_7d || 0,
      mcap: p.mcap || 0,
      category: p.category || '',
    }))
  } catch (e: any) {
    loadError.value = e?.message || String(e)
    protocols.value = []
  } finally {
    loading.value = false
  }
}

function fmTVL(n: number): string {
  if (n >= 1e9) return '$' + (n / 1e9).toFixed(2) + 'B'
  if (n >= 1e6) return '$' + (n / 1e6).toFixed(1) + 'M'
  if (n >= 1e3) return '$' + (n / 1e3).toFixed(0) + 'K'
  return '$' + n.toFixed(0)
}

function changeColor(n: number): string {
  if (n > 0) return '#16a34a'
  if (n < 0) return '#dc2626'
  return 'var(--color-text-tertiary)'
}

onMounted(fetchData)
</script>

<template>
  <div class="defi-tvl-panel">
    <div class="panel-header">
      <h3>{{ $t('misc.defi_tvl') }}</h3>
      <input v-model="search" :placeholder="$t('common.search')" class="search-input" />
      <button class="refresh-btn" @click="fetchData" :disabled="loading">⟳</button>
    </div>

    <div v-if="loadError" class="error-state" @click="fetchData">{{ loadError }} ⟳</div>

    <SkeletonPanel v-else-if="loading && protocols.length === 0" type="table" :rows="10" />

    <div v-else-if="protocols.length === 0 && !loading" class="empty-state">{{ $t('common.no_data') }}</div>

    <template v-else>
      <div class="table-wrapper">
        <div class="table-header">
          <span class="col-rank">#</span>
          <span class="col-name sortable" @click="toggleSort('name')">{{ $t('quote.name') }}{{ sortArrow('name') }}</span>
          <span class="col-chain sortable" @click="toggleSort('chain')">{{ $t('misc.chain') }}{{ sortArrow('chain') }}</span>
          <span class="col-tvl sortable" @click="toggleSort('tvl')">{{ $t('misc.tvl') }}{{ sortArrow('tvl') }}</span>
          <span class="col-1d sortable" @click="toggleSort('change_1d')">1d{{ sortArrow('change_1d') }}</span>
          <span class="col-7d sortable" @click="toggleSort('change_7d')">7d{{ sortArrow('change_7d') }}</span>
        </div>
        <div class="table-body">
          <div v-for="(p, i) in filtered" :key="p.name" class="table-row">
            <span class="col-rank">{{ i + 1 }}</span>
            <span class="col-name">{{ p.name }}</span>
            <span class="col-chain">{{ p.chain }}</span>
            <span class="col-tvl">{{ fmTVL(p.tvl) }}</span>
            <span class="col-1d" :style="{ color: changeColor(p.change_1d) }">{{ p.change_1d > 0 ? '+' : '' }}{{ (p.change_1d * 100).toFixed(2) }}%</span>
            <span class="col-7d" :style="{ color: changeColor(p.change_7d) }">{{ p.change_7d > 0 ? '+' : '' }}{{ (p.change_7d * 100).toFixed(2) }}%</span>
          </div>
        </div>
      </div>
    </template>
  </div>
</template>

<style scoped>
.defi-tvl-panel {
  padding: 12px; height: 100%; display: flex; flex-direction: column;
  color: var(--color-text, var(--color-border)); background: var(--color-bg-panel, var(--color-bg-panel)); overflow: hidden;
}
.panel-header { display: flex; align-items: center; gap: 8px; margin-bottom: 8px; flex-shrink: 0; }
.panel-header h3 { margin: 0; font-size: 14px; font-weight: 600; }
.search-input {
  padding: 3px 8px; border: 1px solid var(--color-border-strong); border-radius: var(--radius-sm);
  background: var(--color-bg-elevated); color: var(--color-text-primary); font-size: 12px; width: 120px;
}
.refresh-btn {
  padding: 4px 10px; border: 1px solid var(--color-border-strong); border-radius: var(--radius-sm);
  background: var(--color-bg-elevated); color: var(--color-text-primary); cursor: pointer; font-size: 13px;
  margin-left: auto;
}
.refresh-btn:disabled { opacity: 0.5; cursor: not-allowed; }
.empty-state {
  flex: 1; display: flex; align-items: center; justify-content: center;
  color: var(--color-text-tertiary); font-size: 13px;
}
.error-state {
  display: flex; align-items: center; justify-content: center; padding: 12px;
  color: var(--color-error); font-size: 13px; cursor: pointer;
}
.table-wrapper { flex: 1; overflow: hidden; display: flex; flex-direction: column; }
.table-header {
  display: flex; padding: 4px 0; border-bottom: 1px solid var(--color-border-strong);
  font-size: 10px; color: var(--color-text-tertiary); text-transform: uppercase; flex-shrink: 0;
}
.sortable { cursor: pointer; user-select: none; }
.sortable:hover { color: var(--color-text-primary); }
.table-body { flex: 1; overflow-y: auto; font-size: 11px; }
.table-row {
  display: flex; padding: 3px 0; align-items: center;
  border-bottom: 1px solid var(--color-border-subtle);
}
.table-row:hover { background: var(--color-bg-elevated); }
.col-rank { width: 24px; color: var(--color-text-tertiary); }
.col-name { width: 100px; font-weight: 600; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.col-chain { width: 60px; color: var(--color-text-tertiary); font-size: 10px; }
.col-tvl { width: 80px; text-align: right; font-weight: 500; font-variant-numeric: tabular-nums; }
.col-1d { width: 64px; text-align: right; font-variant-numeric: tabular-nums; }
.col-7d { flex: 1; min-width: 0; text-align: right; font-variant-numeric: tabular-nums; }
</style>
