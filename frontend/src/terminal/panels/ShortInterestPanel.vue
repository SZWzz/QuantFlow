<script setup lang="ts">
import { ref, onMounted, computed, watch } from 'vue'
import { useSymbolContext } from '@/stores/symbolContext'
import SkeletonPanel from '@/terminal/components/SkeletonPanel.vue'

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const ctx = useSymbolContext()
const pg = ctx.getOrCreatePanelGroup(props.panelId)

interface ShortInterestRow {
  date: string
  short_interest: number
  avg_daily_vol: number
  days_to_cover: number
  short_pct: number
}

const symbol = ref(props.params?.symbol || ctx.getGroupSymbol(pg.groupId) || 'AAPL')
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

async function fetchData() {
  const app = (window as any).go?.main?.App
  if (!app?.GetShortInterest) return
  loading.value = true
  try {
    const result = await app.GetShortInterest(symbol.value)
    rows.value = (result || []).map((r: any) => ({
      date: r.date || '',
      short_interest: r.short_interest || 0,
      avg_daily_vol: r.avg_daily_vol || 0,
      days_to_cover: r.days_to_cover || 0,
      short_pct: r.short_pct || 0,
    }))
  } catch (e) {
    console.error('[ShortInterest]', e)
    rows.value = []
  } finally {
    loading.value = false
  }
}

function formatSI(v: number): string {
  if (v >= 1e9) return (v / 1e9).toFixed(2) + 'B'
  if (v >= 1e6) return (v / 1e6).toFixed(1) + 'M'
  if (v >= 1e3) return (v / 1e3).toFixed(0) + 'K'
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
    <div class="panel-header">
      <h3>{{ $t('misc.short_interest') }}</h3>
      <input v-model="symbol" class="sym-input" :placeholder="$t('common.symbol')" @change="fetchData" />
      <button class="refresh-btn" @click="fetchData" :disabled="loading">⟳</button>
    </div>

    <SkeletonPanel v-if="loading && rows.length === 0" type="card" :rows="2" />

    <template v-else-if="rows.length > 0">
      <div class="stats-row">
        <div class="stat-card">
          <div class="stat-label">{{ $t('misc.short_interest') }}</div>
          <div class="stat-value">{{ formatSI(latest?.short_interest || 0) }}</div>
        </div>
        <div class="stat-card">
          <div class="stat-label">{{ $t('misc.days_to_cover') }}</div>
          <div class="stat-value">{{ latest?.days_to_cover?.toFixed(1) || '--' }}</div>
        </div>
        <div class="stat-card">
          <div class="stat-label">{{ $t('misc.short_pct') }}</div>
          <div class="stat-value">{{ latest ? formatPct(latest.short_pct) : '--' }}</div>
        </div>
        <div class="stat-card">
          <div class="stat-label">{{ $t('misc.trend') }}</div>
          <div class="stat-value" :class="trend === 'increasing' ? 'up' : trend === 'decreasing' ? 'down' : ''">
            {{ trend === 'increasing' ? '↑' : trend === 'decreasing' ? '↓' : '→' }}
          </div>
        </div>
      </div>

      <div class="table-wrapper">
        <div class="table-header">
          <span class="col-date">{{ $t('common.date') }}</span>
          <span class="col-si">{{ $t('misc.short_interest') }}</span>
          <span class="col-vol">{{ $t('misc.avg_daily_vol') }}</span>
          <span class="col-dtc">{{ $t('misc.days_to_cover') }}</span>
          <span class="col-pct">{{ $t('misc.short_pct_col') }}</span>
        </div>
        <div class="table-body">
          <div v-for="r in rows" :key="r.date" class="table-row">
            <span class="col-date">{{ r.date }}</span>
            <span class="col-si">{{ formatSI(r.short_interest) }}</span>
            <span class="col-vol">{{ formatSI(r.avg_daily_vol) }}</span>
            <span class="col-dtc">{{ r.days_to_cover.toFixed(1) }}</span>
            <span class="col-pct">{{ formatPct(r.short_pct) }}</span>
          </div>
        </div>
      </div>
    </template>

    <div v-else class="empty-state">{{ $t('common.no_data') }}</div>
  </div>
</template>

<style scoped>
.short-interest-panel {
  padding: 12px; height: 100%; display: flex; flex-direction: column;
  color: var(--color-text, #e5e7eb); background: var(--color-bg-panel, #1a1a2e); overflow: hidden;
}
.panel-header { display: flex; align-items: center; gap: 8px; margin-bottom: 8px; flex-shrink: 0; }
.panel-header h3 { margin: 0; font-size: 14px; font-weight: 600; }
.sym-input { padding: 2px 6px; font-size: 11px; border: 1px solid var(--color-border-strong); border-radius: 4px; background: var(--color-bg-elevated); color: var(--color-text-primary); width: 70px; }
.refresh-btn { margin-left: auto; padding: 4px 10px; border: 1px solid var(--color-border-strong); border-radius: 4px; background: var(--color-bg-elevated); color: var(--color-text-primary); cursor: pointer; font-size: 13px; }
.refresh-btn:disabled { opacity: 0.5; cursor: not-allowed; }
.empty-state { flex: 1; display: flex; align-items: center; justify-content: center; color: var(--color-text-tertiary); font-size: 13px; }
.stats-row { display: grid; grid-template-columns: repeat(4, 1fr); gap: 8px; margin-bottom: 12px; }
.stat-card { padding: 10px; border: 1px solid var(--color-border-subtle); border-radius: 8px; text-align: center; }
.stat-label { font-size: 10px; color: var(--color-text-tertiary); margin-bottom: 4px; }
.stat-value { font-size: 14px; font-weight: 700; font-variant-numeric: tabular-nums; }
.up { color: #dc2626; }
.down { color: #16a34a; }
.table-wrapper { flex: 1; overflow: hidden; display: flex; flex-direction: column; }
.table-header { display: flex; padding: 4px 0; border-bottom: 1px solid var(--color-border-strong); font-size: 10px; color: var(--color-text-tertiary); text-transform: uppercase; flex-shrink: 0; }
.table-body { flex: 1; overflow-y: auto; font-size: 12px; }
.table-row { display: flex; padding: 3px 0; align-items: center; border-bottom: 1px solid var(--color-border-subtle); }
.table-row:hover { background: var(--color-bg-elevated); }
.col-date { width: 80px; }
.col-si, .col-vol, .col-dtc, .col-pct { width: 80px; text-align: right; font-variant-numeric: tabular-nums; }
</style>
