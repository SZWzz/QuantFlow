<!-- frontend/src/terminal/panels/GeopoliticsPanel.vue -->
<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import VChart from 'vue-echarts'
import 'echarts'

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()

interface TopicRisk {
  id: string
  title: string
  title_cn: string
  risk_level: string   // high / medium / low
  tone: number         // current average tone (-10 to +10)
  tone_change: number  // 7-day tone change
  vol_change: number   // 7-day volume change (%)
  associated: string   // assets affected
  updated_at: number
}

interface VolumePoint {
  Date: string
  Value: number
  Query: string
}

interface TonePoint {
  Date: string
  Tone: number
  Query: string
}

const riskLevels = ['all', 'high', 'medium', 'low'] as const
const activeLevel = ref('all')
const risks = ref<TopicRisk[]>([])
const loading = ref(true)
const selectedTopic = ref<TopicRisk | null>(null)
const detailVolumes = ref<VolumePoint[]>([])
const detailTones = ref<TonePoint[]>([])
const detailLoading = ref(false)

const riskLevelLabels: Record<string, string> = {
  all: '全部', high: '高风险', medium: '中风险', low: '低风险'
}

const riskBadgeMap: Record<string, string> = {
  high: '高', medium: '中', low: '低'
}

async function loadRisks() {
  loading.value = true
  try {
    const app = (window as any).go?.main?.App
    if (app?.GetGeopoliticsRisks) {
      const result = await app.GetGeopoliticsRisks()
      risks.value = result.risks || []
    }
  } catch(e) {
    console.error('[Geopolitics] loadRisks:', e)
  }
  loading.value = false
}

async function loadDetail(topic: TopicRisk) {
  selectedTopic.value = topic
  detailLoading.value = true
  try {
    const app = (window as any).go?.main?.App
    if (app?.GetGeopoliticsDetail) {
      const result = await app.GetGeopoliticsDetail(topic.id, '7d')
      if (result.volumes?.length > 0) {
        detailVolumes.value = result.volumes
      }
      if (result.tones?.length > 0) {
        detailTones.value = result.tones
      }
    }
  } catch(e) {
    console.error('[Geopolitics] loadDetail:', e)
  }
}

onMounted(() => {
  loadRisks()
})

// ── Filtered risks by risk level tab ────────────────────────────────
const filteredRisks = computed(() => {
  if (activeLevel.value === 'all') return risks.value
  return risks.value.filter(r => r.risk_level === activeLevel.value)
})

// ── ECharts tone trend option ───────────────────────────────────────
const toneChartOption = computed(() => {
  if (!selectedTopic.value || detailTones.value.length === 0) return {}
  const dates = detailTones.value.map(p => p.Date)
  const tones = detailTones.value.map(p => +p.Tone.toFixed(2))
  const volumes = detailVolumes.value.length > 0
    ? detailVolumes.value.map(p => +p.Value.toFixed(1))
    : []

  const series: any[] = [
    {
      name: 'Tone',
      type: 'line',
      data: tones,
      smooth: true,
      yAxisIndex: 0,
      areaStyle: { color: 'rgba(239, 68, 68, 0.08)' },
      lineStyle: { color: '#ef4444', width: 2 },
      itemStyle: { color: '#ef4444' },
      showSymbol: false,
    },
  ]

  if (volumes.length > 0) {
    series.push({
      name: 'Volume',
      type: 'bar',
      data: volumes,
      yAxisIndex: 1,
      itemStyle: { color: 'rgba(59, 130, 246, 0.3)' },
    })
  }

  return {
    tooltip: {
      trigger: 'axis' as const,
      formatter: (params: any) => {
        let html = `${params[0].axisValue}<br/>`
        for (const p of params) {
          html += `${p.marker} ${p.seriesName}: ${p.value}<br/>`
        }
        return html
      },
    },
    legend: {
      data: ['Tone', ...(volumes.length > 0 ? ['Volume'] : [])],
      top: 0,
      textStyle: { fontSize: 10, color: 'var(--color-text-secondary)' },
    },
    grid: { left: 10, right: 15, top: 30, bottom: 10 },
    xAxis: {
      type: 'category' as const,
      data: dates,
      axisLabel: { fontSize: 9, rotate: 30 },
    },
    yAxis: [
      {
        type: 'value' as const,
        name: 'Tone',
        min: -10,
        max: 10,
        axisLabel: { fontSize: 9 },
        splitLine: { lineStyle: { color: 'var(--color-border-subtle)' } },
      },
      {
        type: 'value' as const,
        name: 'Volume',
        axisLabel: { fontSize: 9 },
        splitLine: { show: false },
      },
    ],
    series,
  }
})

// ── Tone display helpers ──────────────────────────────────────────
function toneColor(t: number): string {
  if (t <= -3) return '#dc2626'
  if (t < 0) return '#f59e0b'
  return '#16a34a'
}

function toneLabel(t: number): string {
  if (t <= -3) return '负面'
  if (t < 0) return '偏负'
  if (t === 0) return '中性'
  if (t <= 3) return '偏正'
  return '正面'
}

function riskBadgeClass(level: string): string {
  return `badge-${level}`
}

function riskIcon(level: string): string {
  if (level === 'high') return '🔴'
  if (level === 'medium') return '🟡'
  return '🟢'
}

function formatVolChange(v: number): string {
  const sign = v >= 0 ? '+' : ''
  return `${sign}${v.toFixed(1)}%`
}

function formatToneChange(c: number): string {
  const sign = c >= 0 ? '+' : ''
  return `${sign}${c.toFixed(2)}`
}

function formatTime(ts: number): string {
  if (!ts) return ''
  return new Date(ts).toLocaleString('zh-CN', { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' })
}

</script>

<template>
  <div class="geopolitics-panel" :data-panel-id="panelId">
    <!-- Header -->
    <div class="panel-header">
      <h3>{{ $t('geo.title') }}</h3>
      <div class="header-actions">
        <button class="btn-sm" @click="loadRisks()">&#128260; {{ $t('common.refresh') }}</button>
      </div>
    </div>

    <!-- Risk level filter tabs -->
    <div class="filter-tabs">
      <button
        v-for="level in riskLevels" :key="level"
        :class="['tab', { active: activeLevel === level }]"
        @click="activeLevel = level"
      >
        {{ riskLevelLabels[level] }}
      </button>
    </div>

    <!-- Main content: card grid + detail panel -->
    <div class="content-area">
      <!-- Card grid -->
      <div class="card-grid" :class="{ 'with-detail': selectedTopic }">
        <div v-if="loading" class="empty-state">{{ $t('common.loading') }}</div>
        <div v-else-if="filteredRisks.length === 0" class="empty-state">{{ $t('common.no_data') }}</div>
        <div
          v-for="risk in filteredRisks" :key="risk.id"
          :class="['topic-card', { selected: selectedTopic?.id === risk.id }]"
          @click="loadDetail(risk)"
        >
          <div class="card-header">
            <span class="card-title">{{ risk.title_cn }}</span>
            <span :class="['risk-badge', riskBadgeClass(risk.risk_level)]">
              {{ riskIcon(risk.risk_level) }} {{ riskBadgeMap[risk.risk_level] }}
            </span>
          </div>
          <div class="card-body">
            <div class="card-metric">
              <span class="metric-label">{{ $t('geo.sentiment_score') }}</span>
              <span class="metric-value" :style="{ color: toneColor(risk.tone) }">
                {{ risk.tone.toFixed(1) }}
              </span>
              <span class="metric-sub" :style="{ color: toneColor(risk.tone) }">
                {{ toneLabel(risk.tone) }}
              </span>
            </div>
            <div class="card-metric">
              <span class="metric-label">{{ $t('geo.sentiment_change') }}</span>
              <span :class="['metric-value', risk.tone_change >= 0 ? 'text-green' : 'text-red']">
                {{ formatToneChange(risk.tone_change) }}
              </span>
            </div>
            <div class="card-metric">
              <span class="metric-label">{{ $t('geo.discussion_change') }}</span>
              <span :class="['metric-value', risk.vol_change >= 0 ? 'text-red' : 'text-green']">
                {{ formatVolChange(risk.vol_change) }}
              </span>
            </div>
            <div class="card-footer">
              <span class="asset-tag">{{ risk.associated }}</span>
              <span class="update-time">{{ formatTime(risk.updated_at) }}</span>
            </div>
          </div>
        </div>
      </div>

      <!-- Detail panel -->
      <div v-if="selectedTopic" class="detail-panel">
        <div class="detail-header">
          <div>
            <h4>{{ selectedTopic.title_cn }}</h4>
            <span class="detail-subtitle">{{ selectedTopic.title }}</span>
          </div>
          <button class="btn-close" @click="selectedTopic = null">&times;</button>
        </div>

        <!-- Risk summary -->
        <div class="detail-summary">
          <div class="summary-card">
            <span class="summary-label">{{ $t('geo.risk_level') }}</span>
            <span :class="['risk-badge', riskBadgeClass(selectedTopic.risk_level)]">
              {{ riskIcon(selectedTopic.risk_level) }} {{ riskBadgeMap[selectedTopic.risk_level] }}
            </span>
          </div>
          <div class="summary-card">
            <span class="summary-label">{{ $t('geo.sentiment_score') }}</span>
            <span class="summary-value" :style="{ color: toneColor(selectedTopic.tone) }">
              {{ selectedTopic.tone.toFixed(1) }} ({{ toneLabel(selectedTopic.tone) }})
            </span>
          </div>
          <div class="summary-card">
            <span class="summary-label">{{ $t('geo.discussion_change') }}</span>
            <span :class="selectedTopic.vol_change >= 0 ? 'text-red' : 'text-green'">
              {{ formatVolChange(selectedTopic.vol_change) }}
            </span>
          </div>
        </div>

        <!-- Tone trend chart -->
        <div class="chart-container">
          <div v-if="detailLoading" class="empty-state small">{{ $t('common.loading') }}</div>
          <VChart v-else-if="detailTones.length > 0" :option="toneChartOption" style="height: 220px" autoresize />
          <div v-else class="empty-state small">{{ $t('geo.no_sentiment') }}</div>
        </div>

        <!-- Associated assets -->
        <div class="associated-section">
          <span class="section-label">{{ $t('geo.linked_assets') }}</span>
          <span class="asset-tag big">{{ selectedTopic.associated }}</span>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.geopolitics-panel {
  display: flex;
  flex-direction: column;
  height: 100%;
  background: var(--color-bg-panel);
  color: var(--color-text-primary);
  font-size: 13px;
}

.panel-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 12px;
  border-bottom: 1px solid var(--color-border);
}
.panel-header h3 { margin: 0; font-size: 14px; }

.header-actions { display: flex; gap: 8px; align-items: center; }

.btn-sm {
  padding: 2px 8px;
  font-size: 11px;
  border: 1px solid var(--color-border);
  border-radius: 4px;
  background: transparent;
  color: var(--color-text-secondary);
  cursor: pointer;
}
.btn-sm:hover { background: var(--color-bg-hover); }

/* ── Filter tabs ─────────────────────────────────────────────── */
.filter-tabs {
  display: flex;
  gap: 2px;
  padding: 6px 12px;
  border-bottom: 1px solid var(--color-border);
  overflow-x: auto;
}
.tab {
  padding: 3px 10px;
  font-size: 11px;
  border: none;
  border-radius: 4px;
  background: transparent;
  color: var(--color-text-secondary);
  cursor: pointer;
  white-space: nowrap;
}
.tab.active { background: var(--color-accent); color: #fff; }
.tab:hover:not(.active) { background: var(--color-bg-hover); }

/* ── Content area ────────────────────────────────────────────── */
.content-area { display: flex; flex: 1; overflow: hidden; }

/* ── Card grid ───────────────────────────────────────────────── */
.card-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  grid-auto-rows: min-content;
  gap: 8px;
  padding: 8px;
  overflow-y: auto;
  flex: 1;
}
.card-grid.with-detail {
  flex: 0 0 60%;
}

.topic-card {
  border: 1px solid var(--color-border-subtle);
  border-radius: 6px;
  padding: 10px;
  cursor: pointer;
  transition: background 0.15s;
}
.topic-card:hover { background: var(--color-bg-hover); }
.topic-card.selected {
  background: var(--color-bg-selected);
  border-color: var(--color-accent);
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}
.card-title {
  font-weight: 600;
  font-size: 13px;
}

.risk-badge {
  display: inline-flex;
  align-items: center;
  gap: 2px;
  padding: 1px 6px;
  border-radius: 4px;
  font-size: 10px;
  font-weight: 600;
}
.badge-high { background: rgba(220, 38, 38, 0.12); color: #dc2626; }
.badge-medium { background: rgba(245, 158, 11, 0.12); color: #d97706; }
.badge-low { background: rgba(22, 163, 74, 0.12); color: #16a34a; }

.card-body { display: flex; flex-direction: column; gap: 4px; }

.card-metric {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.metric-label { font-size: 11px; color: var(--color-text-tertiary); }
.metric-value { font-weight: 600; font-size: 14px; font-variant-numeric: tabular-nums; }
.metric-sub { font-size: 10px; margin-left: 4px; }

.text-green { color: #16a34a; }
.text-red { color: #dc2626; }

.card-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-top: 6px;
  padding-top: 6px;
  border-top: 1px solid var(--color-border-subtle);
}
.asset-tag {
  font-size: 10px;
  padding: 1px 5px;
  border-radius: 3px;
  background: var(--color-bg-subtle);
  color: var(--color-text-secondary);
}
.asset-tag.big { font-size: 12px; padding: 3px 8px; }
.update-time { font-size: 10px; color: var(--color-text-tertiary); }

/* ── Detail panel ────────────────────────────────────────────── */
.detail-panel {
  flex: 1;
  border-left: 1px solid var(--color-border);
  padding: 12px;
  overflow-y: auto;
  min-width: 280px;
}
.detail-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 12px;
}
.detail-header h4 { margin: 0; font-size: 15px; }
.detail-subtitle { font-size: 11px; color: var(--color-text-tertiary); }
.btn-close {
  background: none; border: none; font-size: 18px;
  color: var(--color-text-secondary); cursor: pointer;
}

.detail-summary {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 8px;
  margin-bottom: 12px;
}
.summary-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 8px;
  border-radius: 6px;
  background: var(--color-bg-subtle);
}
.summary-label { font-size: 11px; color: var(--color-text-tertiary); margin-bottom: 4px; }
.summary-value { font-size: 14px; font-weight: 600; }

.chart-container { margin-bottom: 12px; }

.associated-section {
  display: flex;
  align-items: center;
  gap: 8px;
}
.section-label { font-size: 11px; color: var(--color-text-tertiary); }

.empty-state {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 40px;
  color: var(--color-text-tertiary);
  grid-column: 1 / -1;
}
.empty-state.small { padding: 20px; font-size: 12px; }
</style>
