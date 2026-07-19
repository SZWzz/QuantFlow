<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useTerminalStore } from '@/stores/terminal'
import { useSessionStore } from '@/stores/session'
import { useDataStore } from '@/stores/data'
import { getPanelsByCategory, getPanelMeta, type PanelMeta } from './registry'
import { PANEL_ICONS, getIcon } from '@/lib/icons'
import { PanelHeader } from '@/terminal/components/panel'

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
  '另类数据': 'misc.cat_altdata', '港股': 'misc.cat_hk',
  '美股': 'misc.cat_us', '加密货币': 'misc.cat_crypto',
  '系统': 'misc.cat_system',
}
function catLabel(cn: string): string { return CATEGORY_KEYS[cn] ? t(CATEGORY_KEYS[cn]) : cn }

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()

// Map Chinese category names to CSS data-cat keys
function catKey(cn: string): string {
  const map: Record<string, string> = {
    '市场行情': 'market', '交易执行': 'trading',
    '组合与风控': 'portfolio', '图表分析': 'chart',
    '研究分析': 'research', '量化分析': 'quant',
    '另类数据': 'altdata', '港股': 'hk',
    '美股': 'us', '加密货币': 'crypto',
    '系统': 'system',
  }
  return map[cn] || 'system'
}

function getIconSvg(panelId: string): string {
  const iconName = PANEL_ICONS[panelId]
  if (!iconName) return ''
  return getIcon(iconName)
}

// Dynamic categories from registry
const panelCategories = computed(() => {
  const groups = getPanelsByCategory()
  const result: { title: string; items: PanelMeta[]; key: string }[] = []
  for (const [cat, panels] of Object.entries(groups)) {
    const filtered = panels.filter(p => p.id !== 'welcome' && !p.hidden)
    if (filtered.length === 0) continue
    result.push({
      title: cat,
      items: filtered,
      key: catKey(cat),
    })
  }
  return result
})

// 搜索关键词 + 分类筛选
const searchQuery = ref('')
const activeCat = ref('')

const filteredCategories = computed(() => {
  const q = searchQuery.value.trim().toLowerCase().replace(/\s+/g, '')
  const norm = (s: string) => s.toLowerCase().replace(/\s+/g, '')
  return panelCategories.value
    .filter(c => !activeCat.value || c.key === activeCat.value)
    .map(c => ({
      ...c,
      items: q
        ? c.items.filter(p =>
            norm(p.label).includes(q) ||
            norm(p.description).includes(q) ||
            p.id.toLowerCase().includes(q))
        : c.items,
    }))
    .filter(c => c.items.length > 0)
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

    <div class="welcome-toolbar">
      <div class="search-box">
        <span class="search-icon" v-html="getIcon('search')" />
        <input
          v-model="searchQuery"
          type="text"
          class="search-input"
          :placeholder="$t('common.search') + '...'"
        />
      </div>
      <div class="cat-rail">
        <button :class="['cat-chip', { active: activeCat === '' }]" @click="activeCat = ''">
          {{ $t('common.all') }}
        </button>
        <button
          v-for="cat in panelCategories"
          :key="cat.key"
          :class="['cat-chip', { active: activeCat === cat.key }]"
          @click="activeCat = activeCat === cat.key ? '' : cat.key"
        >
          {{ catLabel(cat.title) }}
          <span class="chip-count">{{ cat.items.length }}</span>
        </button>
      </div>
    </div>

    <div v-if="recentPanels.length > 0 && !searchQuery && !activeCat" class="dashboard-section">
      <div class="section-title">
        <span class="section-dot accent" />
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
        <span class="section-dot market" />
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
        v-for="cat in filteredCategories"
        :key="cat.title"
        class="category-section"
        :data-cat="cat.key"
      >
        <div class="category-header">
          <span class="category-dot" />
          <h2 class="category-title">{{ catLabel(cat.title) }}</h2>
          <span class="category-count">{{ cat.items.length }}</span>
        </div>
        <div class="category-grid">
          <button
            v-for="item in cat.items"
            :key="item.id"
            class="panel-card"
            @click="openPanel(item.id)"
          >
            <span
              class="card-icon"
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

/* Category accent colors via data-cat */
.category-section[data-cat="market"] .category-dot,
.category-section[data-cat="market"] .card-icon { background: var(--cat-market-bg); color: var(--cat-market); }
.category-section[data-cat="trading"] .category-dot,
.category-section[data-cat="trading"] .card-icon { background: var(--cat-trading-bg); color: var(--cat-trading); }
.category-section[data-cat="portfolio"] .category-dot,
.category-section[data-cat="portfolio"] .card-icon { background: var(--cat-portfolio-bg); color: var(--cat-portfolio); }
.category-section[data-cat="chart"] .category-dot,
.category-section[data-cat="chart"] .card-icon { background: var(--cat-chart-bg); color: var(--cat-chart); }
.category-section[data-cat="research"] .category-dot,
.category-section[data-cat="research"] .card-icon { background: var(--cat-research-bg); color: var(--cat-research); }
.category-section[data-cat="quant"] .category-dot,
.category-section[data-cat="quant"] .card-icon { background: var(--cat-quant-bg); color: var(--cat-quant); }
.category-section[data-cat="altdata"] .category-dot,
.category-section[data-cat="altdata"] .card-icon { background: var(--cat-altdata-bg); color: var(--cat-altdata); }
.category-section[data-cat="hk"] .category-dot,
.category-section[data-cat="hk"] .card-icon { background: var(--cat-hk-bg); color: var(--cat-hk); }
.category-section[data-cat="us"] .category-dot,
.category-section[data-cat="us"] .card-icon { background: var(--cat-us-bg); color: var(--cat-us); }
.category-section[data-cat="crypto"] .category-dot,
.category-section[data-cat="crypto"] .card-icon { background: var(--cat-crypto-bg); color: var(--cat-crypto); }
.category-section[data-cat="system"] .category-dot,
.category-section[data-cat="system"] .card-icon { background: var(--cat-system-bg); color: var(--cat-system); }

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
  font-size: var(--font-xl);
  font-weight: 700;
  color: var(--color-text-primary);
  margin: 0;
  letter-spacing: -0.3px;
}

.welcome-subtitle {
  font-size: var(--font-sm);
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

/* ── Toolbar: search + category rail ─────────────────────────── */
.welcome-toolbar {
  max-width: 960px;
  margin: 0 auto 24px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.search-box {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 0 12px;
  background: var(--color-bg-input);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  transition: border-color var(--transition-fast), box-shadow var(--transition-fast);
}
.search-box:focus-within {
  border-color: var(--color-accent);
  box-shadow: 0 0 0 3px var(--color-accent-soft);
}
.search-icon {
  display: inline-flex;
  width: 14px;
  height: 14px;
  color: var(--color-text-tertiary);
  flex-shrink: 0;
}
.search-icon :deep(svg) { width: 100%; height: 100%; }
.search-input {
  flex: 1;
  border: none;
  background: transparent;
  padding: 9px 0;
  font-size: var(--font-sm);
  color: var(--color-text-primary);
  outline: none;
}
.search-input:focus { border: none; box-shadow: none; }
.cat-rail {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
.cat-chip {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 4px 10px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  background: transparent;
  color: var(--color-text-secondary);
  font-size: var(--font-xs);
  font-family: inherit;
  cursor: pointer;
  transition: all var(--transition-fast);
}
.cat-chip:hover {
  border-color: var(--color-accent);
  color: var(--color-accent);
}
.cat-chip.active {
  background: var(--color-accent);
  border-color: var(--color-accent);
  color: var(--color-text-inverse);
}
.cat-chip.active .chip-count {
  background: color-mix(in srgb, var(--color-text-inverse) 20%, transparent);
  color: var(--color-text-inverse);
  border-color: transparent;
}
.chip-count {
  font-size: var(--font-xs);
  padding: 0 6px;
  border-radius: var(--radius-lg);
  background: var(--color-bg-subtle);
  border: 1px solid var(--color-border);
  color: var(--color-text-tertiary);
  line-height: 1.5;
}

.panel-grid {
  display: flex;
  flex-direction: column;
  gap: 28px;
  max-width: 960px;
  margin: 0 auto;
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
}

.category-title {
  font-size: var(--font-sm);
  font-weight: 600;
  color: var(--color-text-secondary);
  margin: 0;
  flex: 1;
}

.category-count {
  font-size: var(--font-xs);
  font-weight: 500;
  color: var(--color-text-tertiary);
  padding: 1px 8px;
  background: var(--color-bg-subtle);
  border-radius: var(--radius-lg);
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
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  cursor: pointer;
  text-align: left;
  transition: border-color var(--transition-normal), background var(--transition-normal), box-shadow var(--transition-normal);
  position: relative;
}

.panel-card:hover {
  border-color: var(--color-accent);
  background: var(--color-accent-soft);
  box-shadow: var(--shadow-md);
}

.card-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  border-radius: var(--radius-md);
  flex-shrink: 0;
}

.card-icon :deep(svg) {
  width: 16px;
  height: 16px;
}

.card-body {
  display: flex;
  flex-direction: column;
  min-width: 0;
  flex: 1;
  gap: 2px;
}

.card-label {
  font-size: var(--font-sm);
  font-weight: 500;
  color: var(--color-text-primary);
  transition: color var(--transition-fast);
}

.card-desc {
  font-size: var(--font-xs);
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
  font-size: var(--font-sm);
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
.section-dot.accent { background: var(--color-accent); }
.section-dot.market { background: var(--color-up); }
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
  border-radius: var(--radius-lg);
  background: var(--color-bg-elevated);
  color: var(--color-text-secondary);
  font-size: var(--font-xs);
  cursor: pointer;
  transition: all var(--transition-fast);
}
.recent-chip:hover {
  border-color: var(--color-accent);
  color: var(--color-accent);
  background: var(--color-accent-soft);
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
  border-radius: var(--radius-lg);
}
.snap-name {
  font-size: var(--font-sm);
  color: var(--color-text-secondary);
}
.snap-price {
  font-size: var(--font-sm);
  font-weight: 600;
  color: var(--color-text-primary);
  font-variant-numeric: tabular-nums;
}
.snap-pct {
  font-size: var(--font-sm);
  font-weight: 500;
}
.snap-pct.up { color: var(--color-up); }
.snap-pct.down { color: var(--color-down); }
</style>
