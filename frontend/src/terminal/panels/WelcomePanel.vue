<script setup lang="ts">
import { computed } from 'vue'
import { useTerminalStore } from '@/stores/terminal'
import { useSessionStore } from '@/stores/session'
import { getPanelsByCategory, type PanelMeta } from './registry'

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const terminal = useTerminalStore()
const session = useSessionStore()

// Icon mapping per panel id
const panelIcons: Record<string, string> = {
  watchlist: '📋', 'quote-detail': '🔍', candlestick: '📈', 'market-overview': '📊',
  'market-depth': '📉', heatmap: '🔥', 'ticker-tape': '📰', 'crypto-overview': '🪙',
  'order-entry': '💹', 'order-blotter': '📝', execution: '⚡', 'basket-order': '🧺',
  'broker-status': '🔌', 'action-center': '🔔',
  position: '📦', 'position-detail': '📋', 'portfolio-summary': '💰', 'trade-history': '📜',
  'risk-dashboard': '🛡️', rebalance: '⚖️', 'broker-config': '⚙️',
  'equity-curve': '📈', 'surface-chart': '🌊', correlation: '🔗', distribution: '📊',
  drawing: '✏️', 'monte-carlo': '🎲',
  'stock-research': '🔬', financials: '📑', 'peer-comparison': '⚔️',
  'analyst-estimates': '🎯', 'insider-trading': '👤', 'congress-trading': '🏛️', sentiment: '💬',
  'backtest-result': '🧪', 'factor-analysis': '📐', 'model-registry': '🤖',
  'prediction-dashboard': '🔮', 'alpha-mining': '⛏️', 'rl-monitor': '🧠',
  'prediction-market': '🎰', geopolitics: '🌍', 'gov-data': '🏦', satellite: '🛰️',
  'ai-chat': '💬', news: '📰', 'system-monitor': '🖥️', 'schedule-panel': '⏰',
  'notify-panel': '📬', settings: '⚙️',
}

// Dynamic categories from registry
const panelCategories = computed(() => {
  const groups = getPanelsByCategory()
  // Move welcome to the end
  delete groups['系统']
  const result: { title: string; items: PanelMeta[] }[] = []
  for (const [cat, panels] of Object.entries(groups)) {
    result.push({
      title: cat,
      items: panels.filter(p => p.id !== 'welcome'),
    })
  }
  return result
})

function openPanel(id: string) {
  terminal.openPanel(id, { source: 'welcome' })
}
</script>

<template>
  <div class="welcome-panel">
    <div class="welcome-header">
      <h1 class="welcome-title">QuantFlow Terminal</h1>
      <p class="welcome-subtitle">双模式量化金融终端 · Ctrl+K 搜索面板 · 共 {{ panelCategories.reduce((s, c) => s + c.items.length, 0) }} 个面板</p>
    </div>

    <div class="panel-grid">
      <section v-for="cat in panelCategories" :key="cat.title" class="category-section">
        <h2 class="category-title">{{ cat.title }}</h2>
        <div class="category-grid">
          <button
            v-for="item in cat.items"
            :key="item.id"
            class="panel-card"
            @click="openPanel(item.id)"
          >
            <span class="card-icon">{{ panelIcons[item.id] || '📌' }}</span>
            <div class="card-body">
              <span class="card-label">{{ item.label }}</span>
              <span class="card-desc">{{ item.description }}</span>
            </div>
          </button>
        </div>
      </section>
    </div>
  </div>
</template>

<style scoped>
.welcome-panel {
  padding: 24px;
  background: var(--color-bg-panel);
  height: 100%;
  overflow-y: auto;
}
.welcome-header {
  text-align: center;
  margin-bottom: 28px;
}
.welcome-title {
  font-size: 22px;
  font-weight: 700;
  color: var(--color-text-primary);
  margin: 0 0 4px;
}
.welcome-subtitle {
  font-size: 12px;
  color: var(--color-text-tertiary);
  margin: 0;
}
.panel-grid {
  display: flex;
  flex-direction: column;
  gap: 20px;
  max-width: 900px;
  margin: 0 auto;
}
.category-section { }
.category-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--color-text-secondary);
  margin: 0 0 8px;
  padding-bottom: 4px;
  border-bottom: 1px solid var(--color-border-subtle);
}
.category-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  gap: 6px;
}
.panel-card {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 12px;
  background: var(--color-bg-subtle);
  border: 1px solid var(--color-border);
  border-radius: 6px;
  cursor: pointer;
  text-align: left;
  transition: border-color 0.15s, background 0.15s;
}
.panel-card:hover {
  border-color: var(--color-accent);
  background: var(--color-accent-soft);
}
.card-icon { font-size: 18px; flex-shrink: 0; }
.card-body { display: flex; flex-direction: column; min-width: 0; }
.card-label { font-size: 13px; font-weight: 500; color: var(--color-text-primary); }
.card-desc {
  font-size: 11px; color: var(--color-text-tertiary);
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
}
</style>
