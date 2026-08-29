<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import { useSymbolContext } from '@/stores/symbolContext'
import { usePanelCache } from '@/lib/composables/usePanelCache'
import { useAddToWorkflow } from '@/terminal/composables/useAddToWorkflow'
import { PanelHeader, EmptyState, LoadingState } from '@/terminal/components/panel'
import { useWailsApp } from '@/lib/composables/useWailsApp'

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const ctx = useSymbolContext()
const pg = ctx.getOrCreatePanelGroup(props.panelId)
const { fetchWithCache } = usePanelCache()
const { control: addToWfControl } = useAddToWorkflow(props.panelId)
const app = useWailsApp()

interface NewsItem { title: string; source: string; time: string; url?: string; symbol?: string }

const items = ref<NewsItem[]>([])
const loading = ref(false)
const nameCache = ref<Record<string, string>>({})

async function resolveName(sym: string) {
  if (!sym || nameCache.value[sym] !== undefined) return
  try {
    if (!app) return
    const result = await app.GetQuote('CN', sym)
    const quote = Array.isArray(result) ? result[0] : result
    nameCache.value[sym] = quote?.name || ''
  } catch { nameCache.value[sym] = '' }
}

function getName(sym: string): string {
  return nameCache.value[sym] || ''
}

async function loadNews() {
  loading.value = true
  try {
    const sym = props.params?.symbol || ctx.getGroupSymbol(pg.groupId) || ''
    const { data } = await fetchWithCache<NewsItem[]>(
      `news:${sym}`,
      async () => {
        const result = await app?.GetNews(sym, 20)
        return (Array.isArray(result) ? result : []) as NewsItem[]
      },
      60 * 1000,
    )
    items.value = data || []
  } catch(e) { console.error('[News] fetch:', e); items.value = [] }
  finally { loading.value = false }
}

function openUrl(url?: string) {
  if (!url) return
  if (app?.OpenURL) app.OpenURL(url)
}

onMounted(loadNews)

// Resolve names for symbols in news items
watch(() => items.value, (newItems) => {
  for (const item of newItems) {
    if (item.symbol) resolveName(item.symbol)
  }
}, { deep: true })
</script>

<template>
  <div class="news-panel">
    <PanelHeader
      title="News"
      :controls="addToWfControl ? [addToWfControl] : []"
    />
    <LoadingState v-if="loading" type="card" :rows="4" />
    <EmptyState v-else-if="!items.length" :title="$t('news.no_news')" />
    <div v-else class="news-list">
      <div v-for="(item, i) in items" :key="i" class="news-item" @click="openUrl(item.url)">
        <div class="news-title">{{ item.title }}</div>
        <div class="news-meta">
          <span v-if="item.symbol" class="news-symbol" :title="getName(item.symbol) || item.symbol">{{ item.symbol }}</span>
          <span class="news-source">{{ item.source }}</span>
          <span class="news-time">{{ item.time }}</span>
          <span v-if="item.url" class="news-link">↗</span>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.news-panel { height: 100%; display: flex; flex-direction: column; overflow: hidden; }
.news-list { flex: 1; overflow-y: auto; padding: var(--space-sm); }
.news-item { padding: var(--space-sm) var(--space-xs); border-bottom: 1px solid var(--color-border-subtle); cursor: pointer; transition: background var(--transition-fast); }
.news-item:hover { background: var(--color-bg-hover); }
.news-title { font-size: var(--font-xs); color: var(--color-text-primary); line-height: 1.4; margin-bottom: var(--space-xs); }
.news-meta { display: flex; gap: var(--space-sm); align-items: center; }
.news-symbol { padding: 0 var(--space-xs); background: var(--color-accent-soft); color: var(--color-accent); border-radius: var(--radius-sm); font-size: var(--font-xs); font-weight: 600; }
.news-source { font-size: var(--font-xs); color: var(--color-text-tertiary); }
.news-time { font-size: var(--font-xs); color: var(--color-text-tertiary); }
.news-link { font-size: var(--font-xs); color: var(--color-accent); margin-left: auto; opacity: 0; transition: opacity var(--transition-fast); }
.news-item:hover .news-link { opacity: 0.8; }
</style>
