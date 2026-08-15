<script setup lang="ts">
import { ref, onMounted, computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useSymbolContext } from '@/stores/symbolContext'
import { useStockName } from '@/lib/composables/useStockName'
import { usePanelCache } from '@/lib/composables/usePanelCache'
import { useWailsApp } from '@/lib/composables/useWailsApp'
import { PanelHeader, PanelTable, StatItem, EmptyState, LoadingState, type Column } from '@/terminal/components/panel'
import { logger } from '@/lib/logger'

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const { t } = useI18n()
const ctx = useSymbolContext()
const pg = ctx.getOrCreatePanelGroup(props.panelId)
const app = useWailsApp()

interface ShortInterestRow {
  date: string
  short_interest: number
  avg_daily_vol: number
  days_to_cover: number
  short_pct: number
}

const symbol = ref(props.params?.symbol || ctx.getGroupSymbol(pg.groupId) || 'AAPL')
const { name } = useStockName(symbol)
const { fetchWithCache } = usePanelCache()
const rows = ref<ShortInterestRow[]>([])
const loading = ref(false)

const latest = computed(() => rows.value[0] || null)

const trend = computed(() => {
  if (rows.value.length < 2) return 'stable'
  const recent = rows.value.slice(0, 3).reduce((s, r) => s + r.short_interest, 0) / 3
  const older = rows.value.slice(-3).reduce((s, r) => s + r.short_interest, 0) / 3
  if (recent > older * 1.1) return 'increasing'
  if (recent < older * 0.9) return 'decreasing'
  return 'stable'
})

const trendDirection = computed<'up' | 'down' | 'flat'>(() =>
  trend.value === 'increasing' ? 'up' : trend.value === 'decreasing' ? 'down' : 'flat',
)

const subtitle = computed(() => [symbol.value, name.value].filter(Boolean).join(' '))

const cols = computed<Column[]>(() => [
  { key: 'date', label: t('common.date') },
  { key: 'short_interest', label: t('misc.short_interest'), align: 'right', mono: true, formatter: formatSI },
  { key: 'avg_daily_vol', label: t('misc.avg_daily_vol'), align: 'right', mono: true, formatter: formatSI },
  { key: 'days_to_cover', label: t('misc.days_to_cover'), align: 'right', mono: true, formatter: (v: number) => v.toFixed(1) },
  { key: 'short_pct', label: t('misc.short_pct_col'), align: 'right', mono: true, formatter: formatPct },
])

async function fetchData() {
  if (!app?.GetShortInterest) return
  loading.value = true
  try {
    const { data } = await fetchWithCache<any>('short_interest:' + symbol.value, () => app.GetShortInterest(symbol.value))
    rows.value = (data || []).map((r: any) => ({
      date: r.date || '',
      short_interest: r.short_interest || 0,
      avg_daily_vol: r.avg_daily_vol || 0,
      days_to_cover: r.days_to_cover || 0,
      short_pct: r.short_pct || 0,
    }))
  } catch (e) {
    logger.error('[ShortInterest]', e)
    rows.value = []
  } finally {
    loading.value = false
  }
}

function formatSI(v: number): string {
  if (v >= 1e8) return (v / 1e8).toFixed(2) + '亿'
  if (v >= 1e4) return (v / 1e4).toFixed(1) + '万'
  return v.toFixed(0)
}

function formatPct(v: number): string {
  return (v * 100).toFixed(2) + '%'
}

watch(() => ctx.linkGroups[pg.groupId].activeSymbol, (newSym) => {
  if (newSym && newSym !== symbol.value) { symbol.value = newSym; fetchData() }
})
onMounted(fetchData)
</script>

<template>
  <div class="short-interest-panel">
    <PanelHeader
      :title="t('misc.short_interest')"
      :subtitle="subtitle"
      :controls="[{ icon: 'refresh', title: t('common.refresh'), action: fetchData, loading }]"
    >
      <template #controls>
        <input v-model="symbol" class="sym-input" :placeholder="t('common.symbol')" @change="fetchData" />
      </template>
    </PanelHeader>

    <LoadingState v-if="loading && rows.length === 0" type="card" :rows="2" />

    <template v-else-if="rows.length > 0">
      <div class="stats-row">
        <StatItem :label="t('misc.short_interest')" :value="formatSI(latest?.short_interest || 0)" />
        <StatItem :label="t('misc.days_to_cover')" :value="latest?.days_to_cover?.toFixed(1) || '--'" />
        <StatItem :label="t('misc.short_pct')" :value="latest ? formatPct(latest.short_pct) : '--'" />
        <div class="stat-item">
          <span class="stat-label">{{ t('misc.trend') }}</span>
          <span class="stat-value" :class="trendDirection">{{ trendDirection === 'up' ? '↑' : trendDirection === 'down' ? '↓' : '→' }}</span>
        </div>
      </div>

      <PanelTable :columns="cols" :data="rows" :loading="loading" sticky-header />
    </template>

    <EmptyState v-else :title="t('common.no_data')" />
  </div>
</template>

<style scoped>
.short-interest-panel { height: 100%; display: flex; flex-direction: column; overflow: hidden; }

.sym-input {
  padding: var(--space-xs) var(--space-sm);
  font-size: var(--font-xs);
  border: 1px solid var(--color-border-strong);
  border-radius: var(--radius-sm);
  background: var(--color-bg-elevated);
  color: var(--color-text-primary);
  width: 70px;
}

.stats-row {
  display: flex;
  gap: var(--space-xl);
  padding: var(--space-sm) var(--panel-padding);
  border-bottom: 1px solid var(--color-border-subtle);
  flex-shrink: 0;
}

/* 趋势项需要按涨跌着色，StatItem 不支持 value 着色，沿用自绘结构 */
.stat-item { display: flex; flex-direction: column; gap: var(--space-xs); min-width: 0; }
.stat-label { font-size: var(--font-xs); color: var(--color-text-tertiary); white-space: nowrap; }
.stat-value { font-family: var(--font-mono); font-size: var(--font-xl); font-weight: 600; color: var(--color-text-primary); }
.stat-value.up { color: var(--color-up); }
.stat-value.down { color: var(--color-down); }
</style>
