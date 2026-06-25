<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import { useResearchStore } from '@/stores/research'
import { useSymbolContext } from '@/stores/symbolContext'

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const store = useResearchStore()
const ctx = useSymbolContext()
const pg = ctx.getOrCreatePanelGroup(props.panelId)

const symbol = ref(props.params?.symbol || ctx.getGroupSymbol(pg.groupId) || 'AAPL')

const scoreColor = computed(() => {
  const s = store.sentiment?.score ?? 0
  if (s > 0.15) return '#22c55e'
  if (s < -0.15) return '#ef4444'
  return 'var(--color-text-tertiary)'
})

watch(symbol, (newVal) => {
  if (newVal) store.fetchSentiment(newVal)
}, { immediate: true })

watch(() => ctx.linkGroups[pg.groupId].activeSymbol, (newSym) => {
  if (pg.linked && newSym && newSym !== symbol.value) {
    symbol.value = newSym
  }
})

function refresh() {
  store.fetchSentiment(symbol.value)
}

function handleSymbolSubmit(e: Event) {
  const input = e.target as HTMLInputElement
  symbol.value = input.value.trim().toUpperCase()
  input.blur()
}
</script>

<template>
  <div class="sentiment-panel">
    <div class="panel-header">
      <h3>{{ $t('research.sentiment') }}</h3>
      <div class="header-controls">
        <input
          class="symbol-input"
          :value="symbol"
          :placeholder="$t('research.hint_enter_symbol')"
          @keyup.enter="handleSymbolSubmit"
        />
        <button class="refresh-btn" @click="refresh" :disabled="store.loading">
          {{ store.loading ? '...' : '⟳' }}
        </button>
      </div>
    </div>

    <div v-if="store.sentiment" class="sentiment-content">
      <div class="score-gauge">
        <svg viewBox="0 0 200 120" class="gauge-svg">
          <path d="M20 100 A80 80 0 0 1 180 100" fill="none" stroke="var(--color-border-strong)" stroke-width="14" />
          <path
            d="M20 100 A80 80 0 0 1 180 100"
            fill="none"
            :stroke="scoreColor"
            stroke-width="14"
            stroke-dasharray="251"
            :stroke-dashoffset="251 - (251 * ((store.sentiment.score + 1) / 2))"
            stroke-linecap="round"
          />
        </svg>
        <div class="score-text">
          <span class="score-label" :style="{ color: scoreColor }">
            {{ store.sentiment.label.toUpperCase() }}
          </span>
          <span class="score-value" :style="{ color: scoreColor }">
            {{ store.sentiment.score > 0 ? '+' : '' }}{{ (store.sentiment.score * 100).toFixed(1) }}
          </span>
          <span class="score-confidence">
            置信度: {{ (store.sentiment.confidence * 100).toFixed(0) }}%
          </span>
        </div>
      </div>

      <div class="keywords-section">
        <h4>{{ $t('research.keywords') }}</h4>
        <div class="keyword-tags">
          <span v-for="kw in store.sentiment.keywords" :key="kw" class="keyword-tag">{{ kw }}</span>
          <span v-if="store.sentiment.keywords.length === 0" class="no-data">{{ $t('research.no_keywords') }}</span>
        </div>
      </div>

      <div class="info-row">
        <span>Source: {{ store.sentiment.source || 'auto' }}</span>
        <span v-if="store.sentiment.compute_time_ms > 0">Compute: {{ store.sentiment.compute_time_ms }}ms</span>
      </div>
    </div>

    <div v-else class="empty-state">
      <p>输入代码后按 ↵ 分析情绪</p>
    </div>
  </div>
</template>

<style scoped>
.sentiment-panel {
  padding: 16px;
  height: 100%;
  display: flex;
  flex-direction: column;
  color: var(--color-text, #e5e7eb);
  background: var(--color-bg, var(--color-bg-panel));
}
.panel-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}
.panel-header h3 { margin: 0; font-size: 14px; font-weight: 600; }
.header-controls { display: flex; gap: 8px; align-items: center; }
.symbol-input {
  width: 100px; padding: 4px 8px; border: 1px solid var(--color-border-strong);
  border-radius: 4px; background: var(--color-bg-elevated); color: #e5e7eb; font-size: 13px;
}
.refresh-btn {
  padding: 4px 10px; border: 1px solid var(--color-border-strong); border-radius: 4px;
  background: var(--color-bg-elevated); color: #e5e7eb; cursor: pointer; font-size: 13px;
}
.refresh-btn:disabled { opacity: 0.5; cursor: not-allowed; }
.mock-banner {
  padding: 6px 10px; margin-bottom: 12px; border-radius: 4px;
  background: #78350f; color: #fbbf24; font-size: 12px; text-align: center;
}
.sentiment-content { flex: 1; display: flex; flex-direction: column; gap: 16px; }
.score-gauge { position: relative; text-align: center; }
.gauge-svg { width: 200px; height: 120px; }
.score-text { margin-top: -20px; display: flex; flex-direction: column; align-items: center; gap: 2px; }
.score-label { font-size: 14px; font-weight: 600; text-transform: uppercase; }
.score-value { font-size: 28px; font-weight: 700; }
.score-confidence { font-size: 12px; color: var(--color-text-secondary); }
.keywords-section h4 { margin: 0 0 8px 0; font-size: 13px; color: var(--color-text-secondary); }
.keyword-tags { display: flex; flex-wrap: wrap; gap: 6px; }
.keyword-tag {
  padding: 2px 10px; border-radius: 12px; font-size: 12px;
  background: var(--color-bg-elevated); color: #e5e7eb; border: 1px solid var(--color-border-strong);
}
.no-data { color: var(--color-text-tertiary); font-size: 12px; }
.info-row {
  display: flex; justify-content: space-between; font-size: 11px; color: var(--color-text-tertiary);
}
.empty-state { flex: 1; display: flex; align-items: center; justify-content: center; color: var(--color-text-tertiary); }
</style>
