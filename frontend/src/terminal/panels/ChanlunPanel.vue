<script setup lang="ts">
import { ref, computed } from 'vue'
import { useStockName } from '@/lib/composables/useStockName'

defineProps<{
  panelId: string
  params?: Record<string, any>
}>()

interface Fractal {
  symbol: string
  type: '顶分型' | '底分型'
  date: string
  candle3: string
  confirmed: boolean
  merged: boolean
}

interface BiSegment {
  from_date: string
  to_date: string
  direction: 'up' | 'down'
  from_price: number
  to_price: number
  pct: number
  bars: number
}

interface ZSBlock {
  from_date: string
  to_date: string
  high: number
  low: number
  direction: 'up' | 'down'
  bars: number
  zg: number
  zd: number
}

const searchSymbol = ref('')
const selectedSymbol = ref('')
const { name } = useStockName(selectedSymbol)
const loading = ref(false)

const fractals = ref<Fractal[]>([])
const biSegments = ref<BiSegment[]>([])
const zsBlocks = ref<ZSBlock[]>([])

const activeTab = ref<'fractals' | 'bi' | 'zs'>('fractals')

const fractalSummary = computed(() => {
  const top = fractals.value.filter((f: Fractal) => f.type === '顶分型').length
  const bottom = fractals.value.filter((f: Fractal) => f.type === '底分型').length
  const merged = fractals.value.filter((f: Fractal) => f.merged).length
  return { top, bottom, merged, total: fractals.value.length }
})

const biSummary = computed(() => {
  const up = biSegments.value.filter((b: BiSegment) => b.direction === 'up').length
  const down = biSegments.value.filter((b: BiSegment) => b.direction === 'down').length
  const avgPct = biSegments.value.length
    ? biSegments.value.reduce((s: number, b: BiSegment) => s + Math.abs(b.pct), 0) / biSegments.value.length
    : 0
  return { up, down, avgPct, total: biSegments.value.length }
})

const zsSummary = computed(() => {
  const up = zsBlocks.value.filter((z: ZSBlock) => z.direction === 'up').length
  const down = zsBlocks.value.filter((z: ZSBlock) => z.direction === 'down').length
  return { up, down, total: zsBlocks.value.length }
})

async function processQuery() {
  loading.value = true
  selectedSymbol.value = searchSymbol.value

  try {
    const app = (window as any).go?.main?.App
    if (!app) return
    const result = await app.GetChanlun(selectedSymbol.value)
    if (result) {
      fractals.value = result.fractals || []
      biSegments.value = result.bi_list || []
      zsBlocks.value = result.zs_list || []
    }
  } catch (e) {
    console.error('[Chanlun] fetch failed:', e)
  } finally {
    loading.value = false
  }
}

function formatPct(v: number): string {
  return (v * 100).toFixed(2) + '%'
}
</script>

<template>
  <div class="chanlun-panel">
    <div class="panel-header">
      <div class="header-left">
        <h3>Chanlun Analysis</h3>
        <span class="subtitle">缠论分析</span>
        <span v-if="selectedSymbol" class="symbol-info">{{ selectedSymbol }} {{ name }}</span>
      </div>
      <div class="symbol-input">
        <input
          v-model="searchSymbol"
          type="text"
          placeholder="输入股票代码 e.g. 600519"
          @keyup.enter="processQuery"
          class="symbol-field"
        />
        <button @click="processQuery" :disabled="loading || !searchSymbol" class="query-btn">
          {{ loading ? '分析中...' : '分析' }}
        </button>
      </div>
    </div>

    <div v-if="selectedSymbol" class="analysis-content">
      <!-- Tab bar -->
      <div class="tabs">
        <button
          :class="{ active: activeTab === 'fractals' }"
          @click="activeTab = 'fractals'"
        >
          分型 ({{ fractalSummary.total }})
        </button>
        <button
          :class="{ active: activeTab === 'bi' }"
          @click="activeTab = 'bi'"
        >
          笔 ({{ biSummary.total }})
        </button>
        <button
          :class="{ active: activeTab === 'zs' }"
          @click="activeTab = 'zs'"
        >
          中枢 ({{ zsSummary.total }})
        </button>
      </div>

      <!-- Fractals Table -->
      <div v-if="activeTab === 'fractals'" class="tab-content">
        <div class="summary-bar">
          <span class="stat">顶分型: {{ fractalSummary.top }}</span>
          <span class="stat">底分型: {{ fractalSummary.bottom }}</span>
          <span class="stat">已合并: {{ fractalSummary.merged }}</span>
        </div>
        <table v-if="fractals.length" class="data-table">
          <thead>
            <tr>
              <th>日期</th>
              <th>类型</th>
              <th>确认</th>
              <th>合并</th>
              <th>K线组</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="f in fractals" :key="f.date">
              <td>{{ f.date }}</td>
              <td>
                <span :class="f.type === '顶分型' ? 'tag-top' : 'tag-bottom'">
                  {{ f.type }}
                </span>
              </td>
              <td>{{ f.confirmed ? '✓' : '✗' }}</td>
              <td>{{ f.merged ? '已合并' : '-' }}</td>
              <td class="mono">{{ f.candle3 }}</td>
            </tr>
          </tbody>
        </table>
        <p v-else class="empty-hint">暂无分型数据</p>
      </div>

      <!-- Bi Table -->
      <div v-if="activeTab === 'bi'" class="tab-content">
        <div class="summary-bar">
          <span class="stat up">向上笔: {{ biSummary.up }}</span>
          <span class="stat down">向下笔: {{ biSummary.down }}</span>
          <span class="stat">均幅: {{ formatPct(biSummary.avgPct) }}</span>
        </div>
        <table v-if="biSegments.length" class="data-table">
          <thead>
            <tr>
              <th>起点</th>
              <th>终点</th>
              <th>方向</th>
              <th>起始价</th>
              <th>终点价</th>
              <th>幅度</th>
              <th>K线数</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="b in biSegments" :key="b.from_date">
              <td>{{ b.from_date }}</td>
              <td>{{ b.to_date }}</td>
              <td>
                <span :class="b.direction === 'up' ? 'tag-up' : 'tag-down'">
                  {{ b.direction === 'up' ? '↑' : '↓' }}
                </span>
              </td>
              <td class="mono">{{ b.from_price.toFixed(2) }}</td>
              <td class="mono">{{ b.to_price.toFixed(2) }}</td>
              <td :class="b.pct > 0 ? 'positive' : 'negative'">
                {{ formatPct(b.pct) }}
              </td>
              <td>{{ b.bars }}</td>
            </tr>
          </tbody>
        </table>
        <p v-else class="empty-hint">暂无笔数据</p>
      </div>

      <!-- ZS Table -->
      <div v-if="activeTab === 'zs'" class="tab-content">
        <div class="summary-bar">
          <span class="stat up">上涨中枢: {{ zsSummary.up }}</span>
          <span class="stat down">下跌中枢: {{ zsSummary.down }}</span>
        </div>
        <table v-if="zsBlocks.length" class="data-table">
          <thead>
            <tr>
              <th>起点</th>
              <th>终点</th>
              <th>方向</th>
              <th>高点 (ZG)</th>
              <th>低点 (ZD)</th>
              <th>通道高</th>
              <th>通道低</th>
              <th>K线数</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="z in zsBlocks" :key="z.from_date">
              <td>{{ z.from_date }}</td>
              <td>{{ z.to_date }}</td>
              <td>
                <span :class="z.direction === 'up' ? 'tag-up' : 'tag-down'">
                  {{ z.direction === 'up' ? '↑' : '↓' }}
                </span>
              </td>
              <td class="mono">{{ z.zg.toFixed(2) }}</td>
              <td class="mono">{{ z.zd.toFixed(2) }}</td>
              <td class="mono">{{ z.high.toFixed(2) }}</td>
              <td class="mono">{{ z.low.toFixed(2) }}</td>
              <td>{{ z.bars }}</td>
            </tr>
          </tbody>
        </table>
        <p v-else class="empty-hint">暂无中枢数据</p>
      </div>
    </div>

    <div v-else class="empty-state">
      <p>输入股票代码开始缠论分析</p>
      <small>支持 分型 → 笔 → 中枢 逐级分析</small>
    </div>
  </div>
</template>

<style scoped>
.chanlun-panel {
  height: 100%;
  display: flex;
  flex-direction: column;
  padding: 12px;
  gap: 10px;
  overflow-y: auto;
}
.panel-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
}
.header-left h3 {
  margin: 0;
  font-size: 15px;
}
.subtitle {
  font-size: 11px;
  color: var(--color-text-tertiary, #6b7280);
  margin-left: 6px;
}
.symbol-info {
  font-size: 11px;
  color: var(--color-accent, #534ab7);
  margin-left: 8px;
  font-weight: 600;
}
.symbol-input {
  display: flex;
  gap: 4px;
}
.symbol-field {
  width: 140px;
  padding: 4px 8px;
  border: 1px solid var(--color-border, #2a2a3e);
  background: var(--color-bg-panel, #1a1a2e);
  color: var(--color-text, #e5e7eb);
  font-size: 13px;
}
.query-btn {
  padding: 4px 12px;
  background: var(--color-accent, #534ab7);
  color: #fff;
  border: none;
  cursor: pointer;
  font-size: 13px;
}
.query-btn:disabled {
  opacity: 0.5;
  cursor: default;
}
.analysis-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.tabs {
  display: flex;
  gap: 2px;
}
.tabs button {
  padding: 4px 12px;
  border: 1px solid var(--color-border, #2a2a3e);
  background: var(--color-bg-panel, #1a1a2e);
  color: var(--color-text-tertiary, #6b7280);
  cursor: pointer;
  font-size: 12px;
}
.tabs button.active {
  background: var(--color-accent, #534ab7);
  color: #fff;
  border-color: var(--color-accent, #534ab7);
}
.tab-content {
  flex: 1;
  overflow-y: auto;
}
.summary-bar {
  display: flex;
  gap: 16px;
  padding: 6px 0;
  font-size: 12px;
  color: var(--color-text-tertiary, #6b7280);
}
.data-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 12px;
}
.data-table th, .data-table td {
  padding: 4px 8px;
  border-bottom: 1px solid var(--color-border, #2a2a3e);
  text-align: left;
}
.data-table th {
  color: var(--color-text-tertiary, #6b7280);
  font-weight: 600;
}
.mono { font-family: monospace; }
.positive { color: #4ade80; }
.negative { color: #f87171; }
.tag-top { color: #f87171; }
.tag-bottom { color: #4ade80; }
.tag-up { color: #4ade80; }
.tag-down { color: #f87171; }
.stat.up { color: #4ade80; }
.stat.down { color: #f87171; }
.empty-hint {
  color: var(--color-text-tertiary, #6b7280);
  font-style: italic;
  padding: 20px 0;
  text-align: center;
}
.empty-state {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  color: var(--color-text-tertiary, #6b7280);
  gap: 4px;
}
.empty-state p { font-size: 14px; margin: 0; }
.empty-state small { font-size: 11px; }
</style>
