<script setup lang="ts">
import { useTerminalStore } from '@/stores/terminal'
import { useSessionStore } from '@/stores/session'

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const terminal = useTerminalStore()
const session = useSessionStore()

const panelCategories = [
  {
    title: '市场行情',
    items: [
      { id: 'watchlist', icon: '📋', label: 'Watchlist', desc: '自选股监控' },
      { id: 'market-overview', icon: '📊', label: 'Market Overview', desc: '大盘总览' },
      { id: 'candlestick', icon: '📈', label: 'Candlestick', desc: 'K线图表' },
      { id: 'market-depth', icon: '📉', label: 'Market Depth', desc: '深度行情' },
      { id: 'ticker-tape', icon: '📰', label: 'Ticker Tape', desc: '滚动报价' },
    ]
  },
  {
    title: '另类数据 & AI',
    items: [
      { id: 'prediction-market', icon: '🔮', label: 'Prediction Market', desc: '预测市场概率' },
      { id: 'geopolitics', icon: '🌍', label: 'Geopolitics', desc: '地缘政治风险' },
      { id: 'gov-data', icon: '🏛️', label: 'Gov Data', desc: '宏观经济指标' },
      { id: 'satellite', icon: '🛰️', label: 'Satellite', desc: '卫星能源数据' },
      { id: 'ai-chat', icon: '🤖', label: 'AI Chat', desc: 'AI 对话助手' },
    ]
  },
  {
    title: '交易 & 组合',
    items: [
      { id: 'order-entry', icon: '💹', label: 'Order Entry', desc: '下单面板' },
      { id: 'position', icon: '📦', label: 'Position', desc: '持仓管理' },
      { id: 'broker-status', icon: '🔌', label: 'Broker Status', desc: '券商状态' },
      { id: 'portfolio-summary', icon: '💰', label: 'Portfolio', desc: '组合总览' },
      { id: 'order-blotter', icon: '📝', label: 'Order Blotter', desc: '订单流水' },
    ]
  },
]

function openPanel(id: string) {
  terminal.openPanel(id, { source: 'welcome' })
}
</script>

<template>
  <div class="welcome-panel" :data-panel-id="panelId">
    <div class="welcome-content">
      <!-- Hero -->
      <div class="welcome-hero">
        <div class="hero-logo">QF</div>
        <h1>QuantFlow Terminal</h1>
        <p class="hero-sub">双模式量化金融终端 — 彭博式面板终端 × 可视化工作流编排</p>
      </div>

      <!-- Main CTA: Command Bar -->
      <div class="cta-section">
        <div class="cta-box">
          <div class="cta-prompt">
            <kbd class="cta-key">Ctrl</kbd>
            <span class="cta-plus">+</span>
            <kbd class="cta-key">K</kbd>
          </div>
          <p class="cta-label">打开命令栏，搜索并打开任意面板</p>
          <p class="cta-hint">支持拼音、英文、中文搜索 · 50 个面板等你探索</p>
        </div>
      </div>

      <!-- Quick Start Steps -->
      <div class="steps-row">
        <div class="step">
          <div class="step-num">1</div>
          <div class="step-text">
            <strong>搜索</strong>
            <span>按 <kbd>Ctrl+K</kbd> 打开命令栏</span>
          </div>
        </div>
        <div class="step-arrow">→</div>
        <div class="step">
          <div class="step-num">2</div>
          <div class="step-text">
            <strong>打开</strong>
            <span>输入面板名，回车打开</span>
          </div>
        </div>
        <div class="step-arrow">→</div>
        <div class="step">
          <div class="step-num">3</div>
          <div class="step-text">
            <strong>布局</strong>
            <span><kbd>Ctrl+1~4</kbd> 切换预设</span>
          </div>
        </div>
        <div class="step-arrow">→</div>
        <div class="step">
          <div class="step-num">4</div>
          <div class="step-text">
            <strong>切换</strong>
            <span>点击 <kbd>Workflow</kbd> 进入编排模式</span>
          </div>
        </div>
      </div>

      <!-- Panel Categories -->
      <div class="categories">
        <div
          v-for="cat in panelCategories" :key="cat.title"
          class="category"
        >
          <h3 class="cat-title">{{ cat.title }}</h3>
          <div class="cat-grid">
            <button
              v-for="p in cat.items" :key="p.id"
              class="cat-card"
              @click="openPanel(p.id)"
            >
              <span class="card-icon">{{ p.icon }}</span>
              <span class="card-label">{{ p.label }}</span>
            </button>
          </div>
        </div>
      </div>

      <!-- Footer -->
      <p class="welcome-footer">
        也可以从顶部 <strong>PushPin Bar</strong> 钉选常用面板 · 点击
        <strong class="wf-link" @click="session.ui.mode = 'workflow'">Workflow</strong>
        切换到可视化策略编排模式
      </p>
    </div>
  </div>
</template>

<style scoped>
.welcome-panel {
  height: 100%;
  background: var(--color-bg-panel);
  overflow: auto;
  display: flex;
  justify-content: center;
}

.welcome-content {
  max-width: 720px;
  width: 100%;
  padding: 32px 24px;
}

/* ── Hero ──────────────────────────────────────────────────── */
.welcome-hero {
  text-align: center;
  margin-bottom: 24px;
}

.hero-logo {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 48px;
  height: 48px;
  margin-bottom: 12px;
  background: var(--color-brand);
  color: #fff;
  border-radius: var(--radius-lg);
  font-size: 20px;
  font-weight: 800;
  letter-spacing: -1px;
}

h1 {
  font-size: 24px;
  font-weight: 700;
  color: var(--color-text-primary);
  margin-bottom: 4px;
  letter-spacing: -0.5px;
}

.hero-sub {
  font-size: var(--font-sm);
  color: var(--color-text-secondary);
}

/* ── CTA ───────────────────────────────────────────────────── */
.cta-section {
  margin-bottom: 24px;
}

.cta-box {
  padding: 20px;
  background: var(--color-bg-subtle);
  border: 1px dashed var(--color-accent);
  border-radius: var(--radius-lg);
  text-align: center;
}

.cta-prompt {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  margin-bottom: 8px;
}

.cta-key {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 32px;
  padding: 4px 10px;
  font-size: 16px;
  font-weight: 700;
  font-family: 'JetBrains Mono', monospace;
  color: #fff;
  background: var(--color-accent);
  border-radius: var(--radius-sm);
}

.cta-plus {
  font-size: 18px;
  color: var(--color-text-tertiary);
  font-weight: 300;
}

.cta-label {
  font-size: var(--font-base);
  color: var(--color-text-primary);
  font-weight: 500;
}

.cta-hint {
  font-size: var(--font-xs);
  color: var(--color-text-tertiary);
  margin-top: 4px;
}

/* ── Steps ─────────────────────────────────────────────────── */
.steps-row {
  display: flex;
  align-items: flex-start;
  gap: 4px;
  margin-bottom: 28px;
  padding: 12px;
  background: var(--color-bg-subtle);
  border-radius: var(--radius-md);
}

.step {
  flex: 1;
  display: flex;
  align-items: flex-start;
  gap: 8px;
  min-width: 0;
}

.step-num {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 22px;
  height: 22px;
  flex-shrink: 0;
  font-size: 11px;
  font-weight: 700;
  color: var(--color-accent);
  background: var(--color-accent-soft);
  border-radius: 50%;
}

.step-text {
  display: flex;
  flex-direction: column;
  gap: 1px;
  min-width: 0;
}

.step-text strong {
  font-size: var(--font-xs);
  color: var(--color-text-primary);
}

.step-text span {
  font-size: 10px;
  color: var(--color-text-tertiary);
}

.step-text kbd {
  display: inline-block;
  padding: 1px 4px;
  font-size: 9px;
  font-family: 'JetBrains Mono', monospace;
  color: var(--color-text-primary);
  background: var(--color-bg-elevated);
  border: 1px solid var(--color-border-strong);
  border-radius: 2px;
}

.step-arrow {
  font-size: 14px;
  color: var(--color-text-tertiary);
  padding-top: 2px;
  flex-shrink: 0;
}

/* ── Categories ────────────────────────────────────────────── */
.categories {
  display: flex;
  flex-direction: column;
  gap: 16px;
  margin-bottom: 20px;
}

.cat-title {
  font-size: 11px;
  font-weight: 600;
  color: var(--color-text-tertiary);
  text-transform: uppercase;
  letter-spacing: 0.5px;
  margin-bottom: 6px;
  padding-left: 2px;
}

.cat-grid {
  display: grid;
  grid-template-columns: repeat(5, 1fr);
  gap: 4px;
}

.cat-card {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 10px;
  background: var(--color-bg-subtle);
  border: 1px solid transparent;
  border-radius: var(--radius-sm);
  cursor: pointer;
  font-family: inherit;
  font-size: var(--font-xs);
  color: var(--color-text-secondary);
  transition: all var(--transition-fast);
  white-space: nowrap;
}

.cat-card:hover {
  border-color: var(--color-accent);
  color: var(--color-text-primary);
  background: var(--color-bg-elevated);
}

.card-icon {
  font-size: 14px;
  flex-shrink: 0;
}

.card-label {
  overflow: hidden;
  text-overflow: ellipsis;
}

/* ── Footer ────────────────────────────────────────────────── */
.welcome-footer {
  text-align: center;
  font-size: var(--font-xs);
  color: var(--color-text-tertiary);
  line-height: 1.6;
}

.wf-link {
  color: var(--color-brand);
  cursor: pointer;
  transition: opacity var(--transition-fast);
}

.wf-link:hover {
  opacity: 0.8;
}
</style>
