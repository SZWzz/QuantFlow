<script setup lang="ts">
import { useTerminalStore } from '@/stores/terminal'

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const terminal = useTerminalStore()

const quickPanels = [
  { id: 'watchlist', icon: '📋', label: 'Watchlist', desc: '自选股监控' },
  { id: 'candlestick', icon: '📈', label: 'Candlestick', desc: 'K线图表' },
  { id: 'market-overview', icon: '📊', label: 'Market Overview', desc: '市场总览' },
  { id: 'prediction-market', icon: '🔮', label: 'Prediction Market', desc: '预测市场' },
  { id: 'geopolitics', icon: '🌍', label: 'Geopolitics', desc: '地缘政治风险' },
  { id: 'gov-data', icon: '🏛️', label: 'Gov Data', desc: '宏观经济指标' },
]

function openPanel(id: string) {
  terminal.openPanel(id, { source: 'welcome' })
}

const shortcuts = [
  { keys: 'Ctrl+K', action: 'Command Bar' },
  { keys: 'Ctrl+1-4', action: '布局预设' },
  { keys: 'Ctrl+W', action: 'Workflow Mode' },
]
</script>

<template>
  <div class="welcome-panel" :data-panel-id="panelId">
    <div class="welcome-content">
      <div class="welcome-hero">
        <div class="hero-logo">QF</div>
        <h1>QuantFlow Terminal</h1>
        <p class="hero-sub">双模式量化金融终端 — 彭博式面板终端 × 可视化工作流编排</p>
      </div>

      <div class="welcome-section">
        <h3>快速启动</h3>
        <div class="quick-grid">
          <button
            v-for="p in quickPanels" :key="p.id"
            class="quick-card"
            @click="openPanel(p.id)"
          >
            <span class="quick-icon">{{ p.icon }}</span>
            <span class="quick-label">{{ p.label }}</span>
            <span class="quick-desc">{{ p.desc }}</span>
          </button>
        </div>
      </div>

      <div class="welcome-section">
        <h3>快捷键</h3>
        <div class="shortcuts-list">
          <div v-for="s in shortcuts" :key="s.keys" class="shortcut-row">
            <kbd>{{ s.keys }}</kbd>
            <span>{{ s.action }}</span>
          </div>
        </div>
      </div>

      <p class="welcome-hint">按 <kbd>Ctrl+K</kbd> 搜索并打开面板</p>
    </div>
  </div>
</template>

<style scoped>
.welcome-panel {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  background: var(--color-bg-panel);
  overflow: auto;
}

.welcome-content {
  max-width: 640px;
  padding: var(--space-2xl);
  text-align: center;
}

.welcome-hero {
  margin-bottom: var(--space-2xl);
}

.hero-logo {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 56px;
  height: 56px;
  margin: 0 auto var(--space-lg);
  background: var(--color-brand);
  color: #fff;
  border-radius: var(--radius-lg);
  font-size: 22px;
  font-weight: 800;
  letter-spacing: -1px;
}

h1 {
  font-size: var(--font-xl);
  font-weight: 700;
  color: var(--color-text-primary);
  margin-bottom: var(--space-sm);
}

.hero-sub {
  font-size: var(--font-sm);
  color: var(--color-text-secondary);
  line-height: 1.5;
}

.welcome-section {
  margin-bottom: var(--space-xl);
  text-align: left;
}

.welcome-section h3 {
  font-size: var(--font-xs);
  font-weight: 600;
  color: var(--color-text-tertiary);
  text-transform: uppercase;
  letter-spacing: 1px;
  margin-bottom: var(--space-md);
}

.quick-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: var(--space-sm);
}

.quick-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--space-xs);
  padding: var(--space-lg) var(--space-md);
  background: var(--color-bg-subtle);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  cursor: pointer;
  text-align: center;
  transition: all var(--transition-fast);
  font-family: inherit;
  color: var(--color-text-primary);
}

.quick-card:hover {
  border-color: var(--color-accent);
  background: var(--color-bg-elevated);
  transform: translateY(-1px);
  box-shadow: var(--shadow-sm);
}

.quick-icon { font-size: 20px; }
.quick-label { font-size: var(--font-sm); font-weight: 600; }
.quick-desc { font-size: 10px; color: var(--color-text-tertiary); }

.shortcuts-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-sm);
}

.shortcut-row {
  display: flex;
  align-items: center;
  gap: var(--space-md);
  font-size: var(--font-sm);
  color: var(--color-text-secondary);
}

kbd {
  display: inline-block;
  padding: 2px 8px;
  font-size: 10px;
  font-family: 'JetBrains Mono', monospace;
  color: var(--color-text-primary);
  background: var(--color-bg-subtle);
  border: 1px solid var(--color-border-strong);
  border-radius: var(--radius-sm);
  min-width: 60px;
  text-align: center;
}

.welcome-hint {
  font-size: var(--font-xs);
  color: var(--color-text-tertiary);
  margin-top: var(--space-lg);
}
</style>
