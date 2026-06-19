<script setup lang="ts">
import { ref } from 'vue'

defineProps<{ panelId: string; params?: Record<string, any> }>()

interface NewsItem { id: number; title: string; source: string; time: string; symbol?: string }
const items = ref<NewsItem[]>([
  { id: 1, title: 'Apple Q2 财报超预期，iPhone 出货量增长 8%', source: 'Reuters', time: '2h ago', symbol: 'AAPL' },
  { id: 2, title: 'Fed 暗示 9 月可能开始降息周期', source: 'Bloomberg', time: '3h ago' },
  { id: 3, title: 'NVIDIA 发布新一代 AI 芯片，股价再创新高', source: 'CNBC', time: '5h ago', symbol: 'NVDA' },
  { id: 4, title: '中国央行下调 LPR 利率 10 个基点', source: '财联社', time: '6h ago' },
  { id: 5, title: 'Tesla Cybertruck 季度交付量超预期', source: 'WSJ', time: '8h ago', symbol: 'TSLA' },
])
</script>

<template>
  <div class="news-panel">
    <div v-for="item in items" :key="item.id" class="news-item">
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
.news-panel { padding: 8px; background: #1a1a2e; height: 100%; overflow-y: auto; }
.news-item { padding: 8px 6px; border-bottom: 1px solid #0f2137; cursor: pointer; transition: background 0.1s; }
.news-item:hover { background: rgba(88,166,255,0.05); }
.news-title { font-size: 12px; color: #c9d1d9; line-height: 1.4; margin-bottom: 4px; }
.news-meta { display: flex; gap: 8px; align-items: center; }
.news-symbol { padding: 1px 4px; background: #0f3460; color: #58a6ff; border-radius: 2px; font-size: 10px; font-weight: 600; }
.news-source { font-size: 10px; color: #5a6380; }
.news-time { font-size: 10px; color: #3a4a6c; }
</style>
