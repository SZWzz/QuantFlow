<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import SkeletonPanel from '@/terminal/components/SkeletonPanel.vue'
import { useSymbolContext } from '@/stores/symbolContext'
import { useStockName } from '@/lib/composables/useStockName'
import { usePanelCache } from '@/lib/composables/usePanelCache'

interface SECFiling {
  symbol: string
  form: string
  date: string
  description: string
  url: string
  filer: string
}

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const { t } = useI18n()
const ctx = useSymbolContext()
const pg = ctx.getOrCreatePanelGroup(props.panelId)
const symbol = ref(props.params?.symbol || ctx.getGroupSymbol(pg.groupId) || 'AAPL')
const { name } = useStockName(symbol)
const loading = ref(false)
const error = ref('')
const filings = ref<SECFiling[]>([])
const selectedFormType = ref('All')

const formTypes = ['All', '13F', '4', '10-Q', '8-K', '10-K']
const app = (window as any).go?.main?.App
const { fetchWithCache } = usePanelCache()

const filteredFilings = computed(() => {
  if (selectedFormType.value === 'All') return filings.value
  return filings.value.filter(f => f.form.toUpperCase() === selectedFormType.value)
})

async function loadFilings() {
  loading.value = true
  error.value = ''
  try {
    const { data } = await fetchWithCache('sec_filings:' + symbol.value, async () => {
      const resp = await app.GetSECFilings(symbol.value)
      return resp?.data ? JSON.parse(resp.data) : resp
    })
    filings.value = data || []
  } catch (e: any) {
    error.value = e.message || '加载失败'
  } finally {
    loading.value = false
  }
}

function formBadgeClass(form: string): string {
  const f = form.toUpperCase()
  if (f === '13F' || f === '13G') return 'badge-blue'
  if (f === '4') return 'badge-green'
  if (f.startsWith('10')) return 'badge-yellow'
  if (f === '8-K') return 'badge-orange'
  return 'badge-gray'
}

function setFormType(type: string) {
  selectedFormType.value = type
}

function openUrl(url: string) {
  if (!url) return
  const app = (window as any).go?.main?.App
  if (app?.OpenURL) app.OpenURL(url)
}

function handleSymbolSubmit(e: Event) {
  const input = e.target as HTMLInputElement
  symbol.value = input.value.trim().toUpperCase()
  input.blur()
}

watch(symbol, loadFilings)
watch(() => ctx.linkGroups[pg.groupId].activeSymbol, (newSym) => {
  if (newSym && newSym !== symbol.value) { symbol.value = newSym; loadFilings() }
})
onMounted(loadFilings)
</script>

<template>
  <div class="darkpool-panel">
    <div class="panel-header">
      <h3>{{ $t('institutional_trades') }}</h3>
      <span class="symbol-badge">{{ symbol }} {{ name }}</span>
      <div class="header-controls">
        <input
          class="symbol-input"
          :value="symbol"
          placeholder="AAPL"
          @keyup.enter="handleSymbolSubmit"
        />
        <button class="refresh-btn" @click="loadFilings" :disabled="loading">
          {{ loading ? '...' : '⟳' }}
        </button>
      </div>
    </div>

    <div class="panel-subtitle">SEC 文件中的机构/内部人交易活动</div>

    <div class="filter-bar">
      <div class="filter-buttons">
        <button
          v-for="ft in formTypes"
          :key="ft"
          :class="['filter-btn', { active: selectedFormType === ft }]"
          @click="setFormType(ft)"
        >{{ ft === 'All' ? $t('common.all') : ft }}</button>
      </div>
    </div>

    <SkeletonPanel v-if="loading" type="table" :rows="5" />

    <div v-else-if="error" class="error-state">
      <span class="error-text">{{ error }}</span>
      <button class="retry-btn" @click="loadFilings">{{ $t('common.retry') }}</button>
    </div>

    <div v-else-if="filteredFilings.length === 0" class="empty-state">暂无机构交易数据</div>

    <div v-else class="table-wrapper">
      <table class="filings-table">
        <thead>
          <tr>
            <th>{{ $t('common.date') }}</th>
            <th>{{ $t('sec_form') }}</th>
            <th>{{ $t('filer') }}</th>
            <th>描述</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="(f, i) in filteredFilings" :key="f.url || i">
            <td class="date-cell">{{ f.date }}</td>
            <td>
              <span
                :class="['form-badge', formBadgeClass(f.form)]"
                @click="openUrl(f.url)"
              >{{ f.form }}</span>
            </td>
            <td class="filer-cell">{{ f.filer }}</td>
            <td class="desc-cell">{{ f.description }}</td>
          </tr>
        </tbody>
      </table>
    </div>

    <div class="panel-footer">{{ $t('data_from_finnhub') }}</div>
  </div>
</template>

<style scoped>
.darkpool-panel {
  padding: 12px;
  height: 100%;
  display: flex;
  flex-direction: column;
  color: var(--color-text, var(--color-border));
  background: var(--color-bg-panel, var(--color-bg-panel));
  overflow: hidden;
}

.header-controls {
  display: flex;
  gap: 6px;
  align-items: center;
}
.symbol-input {
  width: 100px;
  padding: 4px 8px;
  border: 1px solid var(--color-border-strong);
  border-radius: var(--radius-sm);
  background: var(--color-bg-elevated);
  color: var(--color-text-primary);
  font-size: 13px;
}
.refresh-btn {
  padding: 4px 10px;
  border: 1px solid var(--color-border-strong);
  border-radius: var(--radius-sm);
  background: var(--color-bg-elevated);
  color: var(--color-text-primary);
  cursor: pointer;
  font-size: 13px;
}
.refresh-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
.panel-subtitle {
  font-size: 11px;
  color: var(--color-text-tertiary);
  margin-bottom: 8px;
  flex-shrink: 0;
}
.filter-bar {
  display: flex;
  margin-bottom: 8px;
  flex-shrink: 0;
}
.filter-buttons {
  display: flex;
  gap: 2px;
}
.filter-btn {
  padding: 3px 10px;
  border: 1px solid var(--color-border-strong);
  border-radius: var(--radius-sm);
  background: var(--color-bg-elevated);
  color: var(--color-text-secondary);
  cursor: pointer;
  font-size: 11px;
}
.filter-btn.active {
  background: var(--color-accent);
  color: var(--color-text-primary);
  border-color: var(--color-accent);
}
.error-state {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
}
.error-text {
  color: var(--color-up);
  font-size: 12px;
}
.retry-btn {
  padding: 4px 14px;
  border: 1px solid var(--color-up);
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--color-up);
  cursor: pointer;
  font-size: 11px;
}
.retry-btn:hover {
  background: rgba(248, 113, 113, 0.1);
}

.table-wrapper {
  flex: 1;
  overflow: auto;
}
.filings-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 12px;
}
.filings-table th {
  text-align: left;
  padding: 6px 8px;
  color: var(--color-text-secondary);
  border-bottom: 1px solid var(--color-border-strong);
  font-weight: 500;
  white-space: nowrap;
}
.filings-table td {
  padding: 6px 8px;
  border-bottom: 1px solid var(--color-bg-elevated);
  color: var(--color-text-primary);
}
.date-cell {
  white-space: nowrap;
  color: var(--color-text-secondary);
}
.form-badge {
  display: inline-block;
  padding: 2px 10px;
  border-radius: var(--radius-lg);
  font-size: 11px;
  font-weight: 600;
  cursor: pointer;
}
.badge-blue {
  background: rgba(59, 130, 246, 0.15);
  color: var(--color-accent);
}
.badge-green {
  background: rgba(34, 197, 94, 0.15);
  color: var(--color-down);
}
.badge-yellow {
  background: rgba(234, 179, 8, 0.15);
  color: var(--color-accent);
}
.badge-orange {
  background: rgba(249, 115, 22, 0.15);
  color: var(--color-accent);
}
.badge-gray {
  background: var(--color-bg-elevated);
  color: var(--color-text-secondary);
}
.filer-cell {
  white-space: nowrap;
  font-weight: 500;
}
.desc-cell {
  max-width: 200px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--color-text-secondary);
}
.symbol-badge { font-size: 11px; padding: 2px 8px; border-radius: var(--radius-sm); background: rgba(59,130,246,0.15); color: var(--color-accent); font-family: monospace; margin-right: 8px; }
.panel-footer {
  flex-shrink: 0;
  padding-top: 8px;
  font-size: 10px;
  color: var(--color-text-tertiary);
  border-top: 1px solid var(--color-border-subtle);
  margin-top: auto;
}
</style>
