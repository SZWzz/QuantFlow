<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useSymbolContext } from '@/stores/symbolContext'

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const ctx = useSymbolContext()
const pg = ctx.getOrCreatePanelGroup(props.panelId)

interface NewsItem { title: string; source: string; time: string; url?: string; symbol?: string }

const items = ref<NewsItem[]>([])
const loading = ref(false)

async function loadNews() {
  loading.value = true
  try {
    const sym = props.params?.symbol || ctx.getGroupSymbol(pg.groupId) || ''
    const result = await (window as any).go.main.App.GetNews(sym, 20)
    items.value = Array.isArray(result) ? result : []
  } catch { items.value = [] }
  finally { loading.value = false }
}

onMounted(loadNews)
</script>

<template>
  <div class="news-panel">
    <div v-if="loading" class="empty-state">{{ $t('common.loading') }}</div>
    <div v-else-if="!items.length" class="empty-state">{{ $t('news.no_news') }}</div>
    <div v-else v-for="(item, i) in items" :key="i" class="news-item">
      <div class="news-title">{{ item.title }}</div>
      <div class="news-meta">
        <span v-if="item.symbol" class="news-symbol">{{ item.symbol }}</span>
        <span class="news-source">{{ item.source }}</span>
        <span class="news-time">{{ item.time }}</span>
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
.news-symbol { padding: 1px 4px; background: var(--color-accent-soft); color: #58a6ff; border-radius: 2px; font-size: 10px; font-weight: 600; }
.news-source { font-size: 10px; color: var(--color-text-tertiary); }
.news-time { font-size: 10px; color: #3a4a6c; }
.empty-state { padding: 40px; text-align: center; color: var(--color-text-tertiary); font-size: 13px; }
</style>
