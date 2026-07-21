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

// 主行只保留高频分类（全部 + 6 个），其余收进「更多」展开区，默认折叠
const PRIMARY_CAT_KEYS = ['market', 'trading', 'portfolio', 'chart', 'research', 'crypto']
const showMoreCats = ref(false)

const primaryCategories = computed(() =>
  panelCategories.value.filter(c => PRIMARY_CAT_KEYS.includes(c.key)))
const moreCategories = computed(() =>
  panelCategories.value.filter(c => !PRIMARY_CAT_KEYS.includes(c.key)))
const moreHasActive = computed(() =>
  moreCategories.value.some(c => c.key === activeCat.value))
// 折叠时若选中项在「更多」里，将该 chip 钉在主行，选中态保持可见
const pinnedMoreCat = computed(() =>
  showMoreCats.value ? undefined : moreCategories.value.find(c => c.key === activeCat.value))

function selectCat(key: string) {
  activeCat.value = activeCat.value === key ? '' : key
}

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
          v-for="cat in primaryCategories"
          :key="cat.key"
          :class="['cat-chip', { active: activeCat === cat.key }]"
          @click="selectCat(cat.key)"
        >
          {{ catLabel(cat.title) }}
          <span class="chip-count">{{ cat.items.length }}</span>
        </button>
        <button
          v-if="pinnedMoreCat"
          class="cat-chip active"
          @click="selectCat(pinnedMoreCat.key)"
        >
          {{ catLabel(pinnedMoreCat.title) }}
          <span class="chip-count">{{ pinnedMoreCat.items.length }}</span>
        </button>
        <button
          v-if="moreCategories.length"
          :class="['cat-chip', 'cat-chip-more', { 'has-active': moreHasActive }]"
          :aria-expanded="showMoreCats"
          @click="showMoreCats = !showMoreCats"
        >
          更多
          <span class="chip-chevron" v-html="getIcon(showMoreCats ? 'chevron-up' : 'chevron-down')" />
        </button>
        <template v-if="showMoreCats">
          <button
            v-for="cat in moreCategories"
            :key="cat.key"
            :class="['cat-chip', { active: activeCat === cat.key }]"
            @click="selectCat(cat.key)"
          >
            {{ catLabel(cat.title) }}
            <span class="chip-count">{{ cat.items.length }}</span>
          </button>
        </template>
      </div>
    </div>

    <div v-if="recentPanels.length > 0 && !searchQuery && !activeCat" class="dashboard-section">
      <h2 class="section-title">{{ $t('misc.recent_panels') }}</h2>
      <div class="recent-row">
        <button
          v-for="p in recentPanels"
          :key="p"
          class="recent-item"
          @click="openPanel(p)"
        >
          <span class="recent-icon" v-html="getIconSvg(p)" />
          <span class="recent-label">{{ getPanelMeta(p)?.label || p }}</span>
        </button>
      </div>
    </div>

    <div v-if="shIndex || hkIndex" class="dashboard-section">
      <h2 class="section-title">{{ $t('misc.market_snapshot') }}</h2>
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
  padding: var(--space-2xl);
  background: var(--color-bg-panel);
  height: 100%;
  overflow-y: auto;
}

/* 分类色仅用于图标本体；图标底色保持中性，降低彩色底噪 */
.category-section[data-cat="market"] .card-icon { color: var(--cat-market); }
.category-section[data-cat="trading"] .card-icon { color: var(--cat-trading); }
.category-section[data-cat="portfolio"] .card-icon { color: var(--cat-portfolio); }
.category-section[data-cat="chart"] .card-icon { color: var(--cat-chart); }
.category-section[data-cat="research"] .card-icon { color: var(--cat-research); }
.category-section[data-cat="quant"] .card-icon { color: var(--cat-quant); }
.category-section[data-cat="altdata"] .card-icon { color: var(--cat-altdata); }
.category-section[data-cat="hk"] .card-icon { color: var(--cat-hk); }
.category-section[data-cat="us"] .card-icon { color: var(--cat-us); }
.category-section[data-cat="crypto"] .card-icon { color: var(--cat-crypto); }
.category-section[data-cat="system"] .card-icon { color: var(--cat-system); }

.welcome-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: var(--space-2xl);
  padding-bottom: var(--space-xl);
  border-bottom: 1px solid var(--color-border);
}

.logo-area {
  display: flex;
  align-items: center;
  gap: var(--space-lg);
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
  gap: var(--space-xs);
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
  gap: var(--space-sm);
}

.action-btn {
  display: flex;
  align-items: center;
  gap: var(--space-xs);
  padding: var(--space-sm) var(--space-md);
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
  margin: 0 auto var(--space-xl);
  display: flex;
  flex-direction: column;
  gap: var(--space-md);
}
.search-box {
  display: flex;
  align-items: center;
  gap: var(--space-sm);
  padding: 0 var(--space-md);
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
  padding: var(--space-sm) 0;
  font-size: var(--font-sm);
  color: var(--color-text-primary);
  outline: none;
}
.search-input:focus { border: none; box-shadow: none; }
.cat-rail {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: var(--space-xs);
}
.cat-chip {
  display: inline-flex;
  align-items: center;
  gap: var(--space-xs);
  padding: var(--space-xs) var(--space-sm);
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
/* 计数徽标弱化：无底色、仅 tertiary 文本 */
.chip-count {
  font-size: var(--font-xs);
  color: var(--color-text-tertiary);
  line-height: 1.5;
}
.cat-chip.active .chip-count {
  color: color-mix(in srgb, var(--color-text-inverse) 80%, transparent);
}
.cat-chip-more.has-active {
  border-color: var(--color-accent);
  color: var(--color-accent);
}
.chip-chevron {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 12px;
  height: 12px;
}
.chip-chevron :deep(svg) { width: 100%; height: 100%; }

/* 键盘可达性：焦点环走 border-glow token */
.cat-chip:focus-visible,
.recent-item:focus-visible,
.action-btn:focus-visible {
  outline: none;
  border-color: var(--color-accent);
  box-shadow: 0 0 0 3px var(--color-border-glow);
}

.panel-grid {
  display: flex;
  flex-direction: column;
  gap: var(--space-xl);
  max-width: 960px;
  margin: 0 auto;
}

/* 区块标题行对齐终端面板头语言（参考 WatchlistPanel 组头）：小字、无装饰 */
.category-header {
  display: flex;
  align-items: center;
  gap: var(--space-sm);
  margin-bottom: var(--space-md);
  padding-bottom: var(--space-xs);
  border-bottom: 1px solid var(--color-border-subtle);
}

.category-title {
  font-size: var(--font-xs);
  font-weight: 600;
  color: var(--color-text-secondary);
  margin: 0;
  flex: 1;
}

.category-count {
  font-size: var(--font-xs);
  color: var(--color-text-tertiary);
}

.category-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
  gap: var(--space-sm);
}

.panel-card {
  display: flex;
  align-items: center;
  gap: var(--space-md);
  padding: var(--space-md) var(--space-lg);
  background: var(--color-bg-subtle);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  font-family: inherit;
  cursor: pointer;
  text-align: left;
  transition: border-color var(--transition-normal), background var(--transition-normal), box-shadow var(--transition-normal);
  position: relative;
}

.panel-card:hover {
  border-color: var(--color-border-hover);
  background: var(--color-bg-panel);
  box-shadow: var(--shadow-sm);
}

.panel-card:focus-visible {
  outline: none;
  border-color: var(--color-accent);
  box-shadow: 0 0 0 3px var(--color-border-glow);
}

.card-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  border-radius: var(--radius-md);
  background: var(--color-bg-hover);
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
  gap: var(--space-xs);
}

/* 标题与描述拉开层级：600/font-sm vs tertiary/font-xs 单行省略 */
.card-label {
  font-size: var(--font-sm);
  font-weight: 600;
  color: var(--color-text-primary);
}

.card-desc {
  font-size: var(--font-xs);
  color: var(--color-text-tertiary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
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

.panel-card:hover .card-arrow,
.panel-card:focus-visible .card-arrow {
  opacity: 1;
  transform: translateX(0);
  color: var(--color-text-secondary);
}

/* Dashboard sections */
.dashboard-section {
  max-width: 960px;
  margin: 0 auto var(--space-xl);
  padding-bottom: var(--space-lg);
  border-bottom: 1px solid var(--color-border-subtle);
}
.section-title {
  font-size: var(--font-xs);
  font-weight: 600;
  color: var(--color-text-secondary);
  margin: 0 0 var(--space-sm);
}
/* 最近使用：紧凑横排条目，与主网格卡片明确区分 */
.recent-row {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-xs);
}
.recent-item {
  display: inline-flex;
  align-items: center;
  gap: var(--space-xs);
  padding: var(--space-xs) var(--space-sm);
  border: 1px solid var(--color-border-subtle);
  border-radius: var(--radius-md);
  background: var(--color-bg-subtle);
  color: var(--color-text-secondary);
  font-size: var(--font-xs);
  font-family: inherit;
  cursor: pointer;
  transition: all var(--transition-fast);
}
.recent-item:hover {
  border-color: var(--color-border-hover);
  background: var(--color-bg-hover);
  color: var(--color-text-primary);
}
.recent-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 12px;
  height: 12px;
  color: var(--color-text-tertiary);
  flex-shrink: 0;
}
.recent-icon :deep(svg) { width: 100%; height: 100%; }
.snapshot-row {
  display: flex;
  gap: var(--space-lg);
}
.snapshot-item {
  display: flex;
  align-items: center;
  gap: var(--space-md);
  padding: var(--space-sm) var(--space-lg);
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
