<!-- frontend/src/terminal/panels/GeopoliticsPanel.vue -->
<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import VChart from 'vue-echarts'
import 'echarts'
import { useChartTheme } from '@/lib/composables/useChartTheme'
import { usePanelCache } from '@/lib/composables/usePanelCache'
import { getIcon } from '@/lib/icons'
import { useAddToWorkflow } from '@/terminal/composables/useAddToWorkflow'
import { PanelHeader, LoadingState, EmptyState } from '@/terminal/components/panel'

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()

interface TopicRisk { id: string; title: string; title_cn: string; risk_level: string; tone: number; tone_change: number; vol_change: number; associated: string; updated_at: number }
interface VolumePoint { Date: string; Value: number; Query: string }
interface TonePoint { Date: string; Tone: number; Query: string }

const riskLevels = ['all', 'high', 'medium', 'low'] as const
const activeLevel = ref('all')
const risks = ref<TopicRisk[]>([])
const loading = ref(true)
const selectedTopic = ref<TopicRisk | null>(null)
const detailVolumes = ref<VolumePoint[]>([])
const detailTones = ref<TonePoint[]>([])
const detailLoading = ref(false)

const riskLevelLabels: Record<string, string> = { all: '全部', high: '高风险', medium: '中风险', low: '低风险' }
const riskBadgeMap: Record<string, string> = { high: '高', medium: '中', low: '低' }

const { fetchWithCache } = usePanelCache()
const { control: addToWfControl, addToWorkflow } = useAddToWorkflow(props.panelId)
const chartTheme = useChartTheme()

async function loadRisks() { loading.value = true; try { const app = (window as any).go?.main?.App; if (app?.GetGeopoliticsRisks) { const { data: result } = await fetchWithCache<any>('geopolitics_risks', () => app.GetGeopoliticsRisks(), 30 * 60 * 1000); risks.value = result?.risks || [] } } catch(e) { console.error('[Geopolitics] loadRisks:', e) } loading.value = false }

async function loadDetail(topic: TopicRisk) { selectedTopic.value = topic; detailLoading.value = true; try { const app = (window as any).go?.main?.App; if (app?.GetGeopoliticsDetail) { const { data: result } = await fetchWithCache<any>(`geopolitics_detail:${topic.id}`, () => app.GetGeopoliticsDetail(topic.id, '7d'), 30 * 60 * 1000); if (result.volumes?.length > 0) detailVolumes.value = result.volumes; if (result.tones?.length > 0) detailTones.value = result.tones } } catch(e) { console.error('[Geopolitics] loadDetail:', e) } }

onMounted(() => loadRisks())

const filteredRisks = computed(() => activeLevel.value === 'all' ? risks.value : risks.value.filter(r => r.risk_level === activeLevel.value))

const toneChartOption = computed(() => {
  if (!selectedTopic.value || detailTones.value.length === 0) return {}
  const dates = detailTones.value.map(p => p.Date); const tones = detailTones.value.map(p => +p.Tone.toFixed(2))
  const volumes = detailVolumes.value.length > 0 ? detailVolumes.value.map(p => +p.Value.toFixed(1)) : []
  const pal = chartTheme.palette
  const series: any[] = [{ name: 'Tone', type: 'line', data: tones, smooth: true, yAxisIndex: 0, areaStyle: { color: pal[3] + '14' }, lineStyle: { color: pal[3], width: 2 }, itemStyle: { color: pal[3] }, showSymbol: false }]
  if (volumes.length > 0) { series.push({ name: 'Volume', type: 'bar', data: volumes, yAxisIndex: 1, itemStyle: { color: pal[0] + '4D' } }) }
  return { tooltip: { trigger: 'axis' as const, formatter: (params: any) => { let html = `${params[0].axisValue}<br/>`; for (const p of params) html += `${p.marker} ${p.seriesName}: ${p.value}<br/>`; return html } }, legend: { data: ['Tone', ...(volumes.length > 0 ? ['Volume'] : [])], top: 0, textStyle: { fontSize: 10, color: chartTheme.axisColor } }, grid: { left: 10, right: 15, top: 30, bottom: 10 }, xAxis: { type: 'category' as const, data: dates, axisLabel: { fontSize: 9, rotate: 30 } }, yAxis: [{ type: 'value' as const, name: 'Tone', min: -10, max: 10, axisLabel: { fontSize: 9 }, splitLine: { lineStyle: { color: 'var(--color-border-subtle)' } } }, { type: 'value' as const, name: 'Volume', axisLabel: { fontSize: 9 }, splitLine: { show: false } }], series }
})

function toneColor(t: number): string { const pal = chartTheme.palette; if (t <= -3) return pal[3]; if (t < 0) return pal[2]; return pal[1] }
function toneLabel(t: number): string { if (t <= -3) return '负面'; if (t < 0) return '偏负'; if (t === 0) return '中性'; if (t <= 3) return '偏正'; return '正面' }
function riskBadgeClass(level: string): string { return `badge-${level}` }
function riskIcon(level: string): string { if (level === 'high') return '🔴'; if (level === 'medium') return '🟡'; return '🟢' }
function formatVolChange(v: number): string { const sign = v >= 0 ? '+' : ''; return `${sign}${v.toFixed(1)}%` }
function formatToneChange(c: number): string { const sign = c >= 0 ? '+' : ''; return `${sign}${c.toFixed(2)}` }
function formatTime(ts: number): string { if (!ts) return ''; return new Date(ts).toLocaleString('zh-CN', { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' }) }
</script>

<template>
  <div class="geopolitics-panel" :data-panel-id="panelId">
    <PanelHeader title="地缘风险">
      <template #controls>
        <button v-if="addToWfControl" class="btn btn-sm" @click="addToWorkflow()" :title="$t('workflow.add_to_workflow')" v-html="getIcon('plus')" />
        <button class="btn btn-sm" @click="loadRisks()">&#128260; {{ $t('common.refresh') }}</button>
      </template>
    </PanelHeader>

    <div class="filter-tabs">
      <button v-for="level in riskLevels" :key="level" :class="['btn btn-sm', { 'btn-primary': activeLevel === level }]" @click="activeLevel = level">{{ riskLevelLabels[level] }}</button>
    </div>

    <div class="content-area">
      <div class="card-grid" :class="{ 'with-detail': selectedTopic }">
        <LoadingState v-if="loading" type="card" :rows="4" />
        <EmptyState v-else-if="filteredRisks.length === 0" title="暂无数据" />
        <div v-for="risk in filteredRisks" :key="risk.id" :class="['topic-card', { selected: selectedTopic?.id === risk.id }]" @click="loadDetail(risk)">
          <div class="card-header"><span class="card-title">{{ risk.title_cn }}</span><span :class="['risk-badge', riskBadgeClass(risk.risk_level)]">{{ riskIcon(risk.risk_level) }} {{ riskBadgeMap[risk.risk_level] }}</span></div>
          <div class="card-body">
            <div class="card-metric"><span class="metric-label">{{ $t('geo.sentiment_score') }}</span><span class="metric-value" :style="{ color: toneColor(risk.tone) }">{{ risk.tone.toFixed(1) }}</span><span class="metric-sub" :style="{ color: toneColor(risk.tone) }">{{ toneLabel(risk.tone) }}</span></div>
            <div class="card-metric"><span class="metric-label">{{ $t('geo.sentiment_change') }}</span><span :class="['metric-value', risk.tone_change >= 0 ? 'text-green' : 'text-red']">{{ formatToneChange(risk.tone_change) }}</span></div>
            <div class="card-metric"><span class="metric-label">{{ $t('geo.discussion_change') }}</span><span :class="['metric-value', risk.vol_change >= 0 ? 'text-red' : 'text-green']">{{ formatVolChange(risk.vol_change) }}</span></div>
            <div class="card-footer"><span class="asset-tag">{{ risk.associated }}</span><span class="update-time">{{ formatTime(risk.updated_at) }}</span></div>
          </div>
        </div>
      </div>

      <div v-if="selectedTopic" class="detail-panel">
        <div class="detail-header"><div><h4>{{ selectedTopic.title_cn }}</h4><span class="detail-subtitle">{{ selectedTopic.title }}</span></div><button class="btn-close" @click="selectedTopic = null">&times;</button></div>
        <div class="detail-summary">
          <div class="summary-card"><span class="summary-label">{{ $t('geo.risk_level') }}</span><span :class="['risk-badge', riskBadgeClass(selectedTopic.risk_level)]">{{ riskIcon(selectedTopic.risk_level) }} {{ riskBadgeMap[selectedTopic.risk_level] }}</span></div>
          <div class="summary-card"><span class="summary-label">{{ $t('geo.sentiment_score') }}</span><span class="summary-value" :style="{ color: toneColor(selectedTopic.tone) }">{{ selectedTopic.tone.toFixed(1) }} ({{ toneLabel(selectedTopic.tone) }})</span></div>
          <div class="summary-card"><span class="summary-label">{{ $t('geo.discussion_change') }}</span><span :class="selectedTopic.vol_change >= 0 ? 'text-red' : 'text-green'">{{ formatVolChange(selectedTopic.vol_change) }}</span></div>
        </div>
        <div class="chart-container">
          <LoadingState v-if="detailLoading" type="chart" />
          <VChart v-else-if="detailTones.length > 0" :option="toneChartOption" style="height: 220px" autoresize />
          <EmptyState v-else title="暂无舆情数据" />
        </div>
        <div class="associated-section"><span class="section-label">{{ $t('geo.linked_assets') }}</span><span class="asset-tag big">{{ selectedTopic.associated }}</span></div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.geopolitics-panel { height: 100%; display: flex; flex-direction: column; overflow: hidden; }
.filter-tabs { display: flex; gap: var(--space-xs); padding: var(--space-sm) var(--panel-padding); border-bottom: 1px solid var(--color-border-subtle); overflow-x: auto; }
.content-area { display: flex; flex: 1; overflow: hidden; }
.card-grid { display: grid; grid-template-columns: repeat(2, 1fr); grid-auto-rows: min-content; gap: var(--space-sm); padding: var(--space-sm); overflow-y: auto; flex: 1; }
.card-grid.with-detail { flex: 0 0 60%; }
.topic-card { border: 1px solid var(--color-border-subtle); border-radius: var(--radius-md); padding: var(--space-md); cursor: pointer; transition: background 0.15s; }
.topic-card:hover { background: var(--color-bg-hover); }
.topic-card.selected { background: var(--color-bg-selected); border-color: var(--color-accent); }
.card-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: var(--space-sm); }
.card-title { font-weight: 600; font-size: var(--font-sm); }
.risk-badge { display: inline-flex; align-items: center; gap: var(--space-xs); padding: 0 var(--space-sm); border-radius: var(--radius-sm); font-size: var(--font-xs); font-weight: 600; }
.badge-high { background: var(--color-up-soft); color: var(--color-up); }
.badge-medium { background: var(--color-accent-soft); color: var(--color-accent); }
.badge-low { background: var(--color-down-soft); color: var(--color-down); }
.card-body { display: flex; flex-direction: column; gap: var(--space-xs); }
.card-metric { display: flex; justify-content: space-between; align-items: center; }
.metric-label { font-size: var(--font-xs); color: var(--color-text-tertiary); }
.metric-value { font-weight: 600; font-size: var(--font-sm); font-variant-numeric: tabular-nums; }
.metric-sub { font-size: var(--font-xs); margin-left: var(--space-xs); }
.text-green { color: var(--color-down); }
.text-red { color: var(--color-up); }
.card-footer { display: flex; justify-content: space-between; align-items: center; margin-top: var(--space-sm); padding-top: var(--space-sm); border-top: 1px solid var(--color-border-subtle); }
.asset-tag { font-size: var(--font-xs); padding: 0 var(--space-xs); border-radius: var(--radius-sm); background: var(--color-bg-subtle); color: var(--color-text-secondary); }
.asset-tag.big { font-size: var(--font-xs); padding: var(--space-xs) var(--space-sm); }
.update-time { font-size: var(--font-xs); color: var(--color-text-tertiary); }
.detail-panel { flex: 1; border-left: 1px solid var(--color-border); padding: var(--space-md); overflow-y: auto; min-width: 280px; }
.detail-header { display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: var(--space-md); }
.detail-header h4 { margin: 0; font-size: var(--font-lg); }
.detail-subtitle { font-size: var(--font-xs); color: var(--color-text-tertiary); }
.btn-close { background: none; border: none; font-size: var(--font-lg); color: var(--color-text-secondary); cursor: pointer; }
.detail-summary { display: grid; grid-template-columns: repeat(3, 1fr); gap: var(--space-sm); margin-bottom: var(--space-md); }
.summary-card { display: flex; flex-direction: column; align-items: center; padding: var(--space-sm); border-radius: var(--radius-md); background: var(--color-bg-subtle); }
.summary-label { font-size: var(--font-xs); color: var(--color-text-tertiary); margin-bottom: var(--space-xs); }
.summary-value { font-size: var(--font-sm); font-weight: 600; }
.chart-container { margin-bottom: var(--space-md); }
.associated-section { display: flex; align-items: center; gap: var(--space-sm); }
.section-label { font-size: var(--font-xs); color: var(--color-text-tertiary); }
</style>
