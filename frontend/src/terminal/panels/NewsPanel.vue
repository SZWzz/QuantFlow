<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import { useSymbolContext } from '@/stores/symbolContext'
import { usePanelCache } from '@/lib/composables/usePanelCache'
import { useAddToWorkflow } from '@/terminal/composables/useAddToWorkflow'
import { PanelHeader } from '@/terminal/components/panel'

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const ctx = useSymbolContext()
const pg = ctx.getOrCreatePanelGroup(props.panelId)
const { fetchWithCache } = usePanelCache()
const { control: addToWfControl } = useAddToWorkflow(props.panelId)

interface NewsItem { title: string; source: string; time: string; url?: string; symbol?: string }

const items = ref<NewsItem[]>([])
const loading = ref(false)
const nameCache = ref<Record<string, string>>({})

async function resolveName(sym: string) {
  if (!sym || nameCache.value[sym] !== undefined) return
  try {
    const app = (window as any).go?.main?.App
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
    const { data } = await fetchWithCache(
      `news:${sym}`,
      async () => {
        const result = await (window as any).go?.main?.App?.GetNews(sym, 20)
        return Array.isArray(result) ? result : []
      },
      60 * 1000,
    )
    items.value = data
  } catch(e) { console.error('[News] fetch:', e); items.value = [] }
  finally { loading.value = false }
}

function openUrl(url?: string) {
  if (!url) return
  const app = (window as any).go?.main?.App
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
    <div v-if="loading" class="empty-state">{{ $t('common.loading') }}</div>
    <div v-else-if="!items.length" class="empty-state">{{ $t('news.no_news') }}</div>
    <div v-else v-for="(item, i) in items" :key="i" class="news-item" @click="openUrl(item.url)">
      <div class="news-title">{{ item.title }}</div>
      <div class="news-meta">
        <span v-if="item.symbol" class="news-symbol" :title="getName(item.symbol) || item.symbol">{{ item.symbol }}</span>
        <span class="news-source">{{ item.source }}</span>
        <span class="news-time">{{ item.time }}</span>
        <span v-if="item.url" class="news-link">↗</span>
      </div>
    </div>
  </div>
</template>

<style scoped>
.news-panel { padding: 8px; background: var(--color-bg-panel); height: 100%; overflow-y: auto; }
.news-item { padding: 8px 6px; border-bottom: 1px solid var(--color-bg-input); cursor: pointer; transition: background 0.1s; }
.news-item:hover { background: rgba(88,166,255,0.05); }
.news-title { font-size: 12px; color: var(--color-text-primary); line-height: 1.4; margin-bottom: 4px; }
.news-meta { display: flex; gap: 8px; align-items: center; }
.news-symbol { padding: 1px 4px; background: var(--color-accent-soft); color: var(--color-accent); border-radius: 2px; font-size: 10px; font-weight: 600; }
.news-source { font-size: 10px; color: var(--color-text-tertiary); }
.news-time { font-size: 10px; color: var(--color-text-secondary); }
.news-link { font-size: 10px; color: var(--color-accent, var(--color-accent)); margin-left: auto; opacity: 0; transition: opacity 0.15s; }
.news-item:hover .news-link { opacity: 0.8; }
</style>
