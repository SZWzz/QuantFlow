<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useTerminalStore } from '@/stores/terminal'
import { useSessionStore } from '@/stores/session'
import { useDataStore } from '@/stores/data'
import { getPanelsByCategory, getPanelMeta, type PanelMeta } from './registry'
import { PANEL_ICONS, getIcon } from '@/lib/icons'

const { t } = useI18n()
const terminal = useTerminalStore()
const session = useSessionStore()
const dataStore = useDataStore()

const recentPanels = computed(() => terminal.recentPanels.slice(-8).reverse())
const shIndex = computed(() => dataStore.marketOverview?.indices?.find(i => i.symbol === '000001'))
const hkIndex = computed(() => dataStore.marketOverview?.indices?.find(i => i.symbol === 'HSI'))

const CATEGORY_KEYS: Record<string, string> = {
  '市场行情': 'misc.cat_market', '交易执行': 'misc.cat_trading',
  '组合与风控': 'misc.cat_portfolio', '图表分析': 'misc.cat_chart',
  '研究分析': 'misc.cat_research', '量化分析': 'misc.cat_quant',
  '另类数据': 'misc.cat_altdata', '系统': 'misc.cat_system',
}
function catLabel(cn: string): string { return CATEGORY_KEYS[cn] ? t(CATEGORY_KEYS[cn]) : cn }

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()

const categoryColors: Record<string, { bg: string; border: string; accent: string }> = {
  '市场行情': { bg: 'rgba(59, 130, 246, 0.08)', border: 'rgba(59, 130, 246, 0.2)', accent: '#3b82f6' },
  '交易执行': { bg: 'rgba(233, 69, 96, 0.08)', border: 'rgba(233, 69, 96, 0.2)', accent: '#e94560' },
  '组合与风控': { bg: 'rgba(139, 92, 246, 0.08)', border: 'rgba(139, 92, 246, 0.2)', accent: '#8b5cf6' },
  '图表分析': { bg: 'rgba(6, 182, 212, 0.08)', border: 'rgba(6, 182, 212, 0.2)', accent: '#06b6d4' },
  '研究分析': { bg: 'rgba(245, 158, 11, 0.08)', border: 'rgba(245, 158, 11, 0.2)', accent: '#f59e0b' },
  '量化分析': { bg: 'rgba(34, 197, 94, 0.08)', border: 'rgba(34, 197, 94, 0.2)', accent: '#22c55e' },
  '另类数据': { bg: 'rgba(236, 72, 153, 0.08)', border: 'rgba(236, 72, 153, 0.2)', accent: '#ec4899' },
  '港股': { bg: 'rgba(255, 107, 53, 0.08)', border: 'rgba(255, 107, 53, 0.2)', accent: '#ff6b35' },
  '美股': { bg: 'rgba(30, 144, 255, 0.08)', border: 'rgba(30, 144, 255, 0.2)', accent: '#1e90ff' },
  '加密货币': { bg: 'rgba(247, 147, 26, 0.08)', border: 'rgba(247, 147, 26, 0.2)', accent: '#f7931a' },
  '系统': { bg: 'rgba(148, 163, 184, 0.08)', border: 'rgba(148, 163, 184, 0.2)', accent: '#94a3b8' },
}

function getCategoryColor(cat: string) {
  return categoryColors[cat] || categoryColors['系统']
}

function getIconSvg(panelId: string): string {
  const iconName = PANEL_ICONS[panelId]
  if (!iconName) return ''
  return getIcon(iconName)
}

// Dynamic categories from registry
const panelCategories = computed(() => {
  const groups = getPanelsByCategory()
  const result: { title: string; items: PanelMeta[]; color: ReturnType<typeof getCategoryColor> }[] = []
  for (const [cat, panels] of Object.entries(groups)) {
    const filtered = panels.filter(p => p.id !== 'welcome')
    if (filtered.length === 0) continue
    result.push({
      title: cat,
      items: filtered,
      color: getCategoryColor(cat),
    })
  }
  return result
})

function openPanel(id: string) {
  terminal.openPanel(id, { source: 'welcome' })
}

let marketTimer: ReturnType<typeof setInterval> | null = null
onMounted(() => {
  dataStore.fetchMarketOverview(session.ui.activeMarket)
  marketTimer = setInterval(() => dataStore.fetchMarketOverview(session.ui.activeMarket), 60000)
})
onUnmounted(() => { if (marketTimer) clearInterval(marketTimer) })
</script>

<template>
  <div class="welcome-panel">
    <div class="welcome-header">
      <div class="logo-area">
        <div class="logo-badge">
          <span class="logo-icon" v-html="getIcon('welcome')" />
        </div>
        <div class="logo-text">
          <h1 class="welcome-title">{{ $t('misc.welcome') }}</h1>
          <p class="welcome-subtitle">{{ $t('misc.welcome_subtitle') }}{{ panelCategories.reduce((s, c) => s + c.items.length, 0) }}{{ $t('misc.panel_count') }}</p>
        </div>
      </div>
      <div class="welcome-actions">
        <button class="action-btn" @click="terminal.openPanel('settings')">
          <span class="btn-icon" v-html="getIcon('settings')" />
          {{ $t('settings.title') }}
        </button>
      </div>
    </div>

    <div v-if="recentPanels.length > 0" class="dashboard-section">
      <div class="section-title">
        <span class="section-dot" style="background:#3b82f6" />
        {{ $t('misc.recent_panels') }}
      </div>
      <div class="recent-row">
        <button
          v-for="p in recentPanels"
          :key="p"
          class="recent-chip"
          @click="openPanel(p)"
        >
          {{ getPanelMeta(p)?.label || p }}
        </button>
      </div>
    </div>

    <div v-if="shIndex || hkIndex" class="dashboard-section">
      <div class="section-title">
        <span class="section-dot" style="background:#22c55e" />
        {{ $t('misc.market_snapshot') }}
      </div>
      <div class="snapshot-row">
        <div v-if="shIndex" class="snapshot-item">
          <span class="snap-name">{{ shIndex.name }}</span>
          <span class="snap-price">{{ (shIndex.last || 0).toFixed(0) }}</span>
          <span class="snap-pct" :class="(shIndex.changePct || 0) >= 0 ? 'up' : 'down'">
            {{ (shIndex.changePct || 0) >= 0 ? '+' : '' }}{{ (shIndex.changePct || 0).toFixed(2) }}%
          </span>
        </div>
        <div v-if="hkIndex" class="snapshot-item">
          <span class="snap-name">{{ hkIndex.name }}</span>
          <span class="snap-price">{{ (hkIndex.last || 0).toFixed(0) }}</span>
          <span class="snap-pct" :class="(hkIndex.changePct || 0) >= 0 ? 'up' : 'down'">
            {{ (hkIndex.changePct || 0) >= 0 ? '+' : '' }}{{ (hkIndex.changePct || 0).toFixed(2) }}%
          </span>
        </div>
      </div>
    </div>

    <div class="panel-grid">
      <section
        v-for="(cat, catIdx) in panelCategories"
        :key="cat.title"
        class="category-section"
        :style="{ animationDelay: `${catIdx * 50}ms` }"
      >
        <div class="category-header">
          <span class="category-dot" :style="{ background: cat.color.accent }" />
          <h2 class="category-title">{{ catLabel(cat.title) }}</h2>
          <span class="category-count">{{ cat.items.length }}</span>
        </div>
        <div class="category-grid">
          <button
            v-for="(item, idx) in cat.items"
            :key="item.id"
            class="panel-card"
            :style="{ animationDelay: `${catIdx * 50 + idx * 30}ms` }"
            @click="openPanel(item.id)"
          >
            <span
              class="card-icon"
              :style="{ color: cat.color.accent, background: cat.color.bg }"
              v-html="getIconSvg(item.id)"
            />
            <div class="card-body">
              <span class="card-label">{{ item.label }}</span>
              <span class="card-desc">{{ item.description }}</span>
            </div>
            <span class="card-arrow" v-html="getIcon('chevron-right')" />
          </button>
        </div>
      </section>
    </div>
  </div>
</template>

<style scoped>
.welcome-panel {
  padding: 32px;
  background: var(--color-bg-panel);
  height: 100%;
  overflow-y: auto;
}

.welcome-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 32px;
  padding-bottom: 20px;
  border-bottom: 1px solid var(--color-border);
}

.logo-area {
  display: flex;
  align-items: center;
  gap: 16px;
}

.logo-badge {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 44px;
  height: 44px;
  background: var(--gradient-accent);
  border: 1px solid var(--color-border-glow);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-glow);
}

.logo-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 20px;
  height: 20px;
  color: var(--color-accent);
}

.logo-icon :deep(svg) {
  width: 100%;
  height: 100%;
}

.logo-text {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.welcome-title {
  font-size: 24px;
  font-weight: 700;
  color: var(--color-text-primary);
  margin: 0;
  letter-spacing: -0.3px;
  background: linear-gradient(135deg, var(--color-text-primary) 0%, var(--color-accent) 100%);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
}

.welcome-subtitle {
  font-size: 13px;
  color: var(--color-text-tertiary);
  margin: 0;
}

.welcome-actions {
  display: flex;
  gap: 8px;
}

.action-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 14px;
  border: 1px solid var(--color-border);
  background: var(--color-bg-subtle);
  color: var(--color-text-secondary);
  border-radius: var(--radius-md);
  font-size: var(--font-sm);
  font-family: inherit;
  cursor: pointer;
  transition: all var(--transition-fast);
}

.action-btn:hover {
  border-color: var(--color-accent);
  background: var(--color-accent-soft);
  color: var(--color-text-primary);
  box-shadow: 0 0 8px var(--color-accent-glow);
}

.btn-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 14px;
  height: 14px;
}

.btn-icon :deep(svg) {
  width: 100%;
  height: 100%;
}

.panel-grid {
  display: flex;
  flex-direction: column;
  gap: 28px;
  max-width: 960px;
  margin: 0 auto;
}

.category-section {
  opacity: 0;
  animation: fadeIn 0.4s ease forwards;
}

@keyframes fadeIn {
  from { opacity: 0; transform: translateY(8px); }
  to { opacity: 1; transform: translateY(0); }
}

.category-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
  padding-bottom: 8px;
  border-bottom: 1px solid var(--color-border-subtle);
}

.category-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex-shrink: 0;
  box-shadow: 0 0 6px currentColor;
}

.category-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--color-text-secondary);
  margin: 0;
  flex: 1;
}

.category-count {
  font-size: 11px;
  font-weight: 500;
  color: var(--color-text-tertiary);
  padding: 1px 8px;
  background: var(--color-bg-subtle);
  border-radius: 10px;
  border: 1px solid var(--color-border);
}

.category-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
  gap: 8px;
}

.panel-card {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 14px;
  background: var(--color-bg-subtle);
  background-image: var(--gradient-card);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  cursor: pointer;
  text-align: left;
  transition: all var(--transition-normal);
  opacity: 0;
  animation: fadeIn 0.3s ease forwards;
  position: relative;
  overflow: hidden;
}

.panel-card::before {
  content: '';
  position: absolute;
  inset: 0;
  background: linear-gradient(135deg, transparent 0%, rgba(255,255,255,0.02) 100%);
  opacity: 0;
  transition: opacity var(--transition-normal);
}

.panel-card:hover {
  border-color: var(--color-accent);
  background: var(--color-accent-soft);
  transform: translateY(-1px);
  box-shadow: var(--shadow-md), 0 0 12px var(--color-accent-glow);
}

.panel-card:hover::before {
  opacity: 1;
}

.card-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  border-radius: var(--radius-md);
  flex-shrink: 0;
  transition: all var(--transition-normal);
}

.card-icon :deep(svg) {
  width: 16px;
  height: 16px;
}

.panel-card:hover .card-icon {
  transform: scale(1.1);
}

.card-body {
  display: flex;
  flex-direction: column;
  min-width: 0;
  flex: 1;
  gap: 2px;
}

.card-label {
  font-size: 13px;
  font-weight: 500;
  color: var(--color-text-primary);
  transition: color var(--transition-fast);
}

.card-desc {
  font-size: 11px;
  color: var(--color-text-tertiary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  transition: color var(--transition-fast);
}

.panel-card:hover .card-desc {
  color: var(--color-text-secondary);
}

.card-arrow {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 14px;
  height: 14px;
  color: var(--color-text-tertiary);
  opacity: 0;
  transform: translateX(-4px);
  transition: all var(--transition-normal);
  flex-shrink: 0;
}

.card-arrow :deep(svg) {
  width: 100%;
  height: 100%;
}

.panel-card:hover .card-arrow {
  opacity: 1;
  transform: translateX(0);
  color: var(--color-accent);
}

/* Dashboard sections */
.dashboard-section {
  margin-bottom: 20px;
  padding-bottom: 16px;
  border-bottom: 1px solid var(--color-border);
}
.section-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  font-weight: 600;
  color: var(--color-text-primary);
  margin-bottom: 10px;
}
.section-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex-shrink: 0;
}
.recent-row {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
.recent-chip {
  display: inline-flex;
  align-items: center;
  padding: 4px 12px;
  border: 1px solid var(--color-border-strong);
  border-radius: 12px;
  background: var(--color-bg-elevated);
  color: var(--color-text-secondary);
  font-size: 11px;
  cursor: pointer;
  transition: all var(--transition-fast);
}
.recent-chip:hover {
  border-color: var(--color-accent);
  color: var(--color-accent);
  background: rgba(59,130,246,0.1);
}
.snapshot-row {
  display: flex;
  gap: 16px;
}
.snapshot-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 16px;
  background: var(--color-bg-elevated);
  border: 1px solid var(--color-border-subtle);
  border-radius: 8px;
}
.snap-name {
  font-size: 12px;
  color: var(--color-text-secondary);
}
.snap-price {
  font-size: 14px;
  font-weight: 600;
  color: var(--color-text-primary);
  font-variant-numeric: tabular-nums;
}
.snap-pct {
  font-size: 12px;
  font-weight: 500;
}
.snap-pct.up { color: #dc2626; }
.snap-pct.down { color: #16a34a; }
</style>
