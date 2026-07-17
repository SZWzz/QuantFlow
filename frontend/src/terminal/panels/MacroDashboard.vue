<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { GetMacroSnapshot, type MacroSnapshot } from '@/lib/wails'

const country = ref('CN')
const snapshot = ref<MacroSnapshot | null>(null)

onMounted(() => fetchData())

async function fetchData() {
  try {
    snapshot.value = await GetMacroSnapshot(country.value)
  } catch { snapshot.value = null }
}

const countryLabel = (c: string) => ({ CN: '中国', US: '美国' }[c] || c)

function renderCard(title: string, items: any[]) {
  if (!items || items.length === 0) return null
  return { title, items: items.slice(0, 6) }
}
</script>

<template>
  <div class="macro-dashboard">
    <div class="toolbar">
      <h3>宏观仪表盘</h3>
      <div class="country-tabs">
        <button v-for="c in ['CN','US']" :key="c" :class="{active:country===c}" @click="country=c;fetchData()">
          {{ countryLabel(c) }}
        </button>
      </div>
    </div>

    <div v-if="snapshot" class="cards-grid">
      <div v-for="card in [
        {title:'📈 增长', items:snapshot.growth},
        {title:'💰 通胀', items:snapshot.inflation},
        {title:'🏦 货币', items:snapshot.monetary},
        {title:'📋 政策', items:snapshot.policy},
      ].filter(c=>c.items.length>0)" :key="card.title" class="macro-card">
        <h4>{{ card.title }}</h4>
        <div v-for="item in card.items.slice(0, 5)" :key="item.name" class="macro-item">
          <span class="item-name">{{ item.name }}</span>
          <span class="item-value">{{ item.value?.toFixed(2) || '-' }}</span>
          <span class="item-unit">{{ item.unit }}</span>
        </div>
      </div>
    </div>
    <div v-else class="empty">加载中...（需要 Python sidecar 运行）</div>
  </div>
</template>

<style scoped>
.macro-dashboard { padding: 16px; height: 100%; overflow-y: auto; }
.toolbar { display: flex; justify-content: space-between; align-items: center; margin-bottom: 16px; }
.toolbar h3 { font-size: 15px; margin: 0; }
.country-tabs { display: flex; gap: 4px; }
.country-tabs button {
  padding: 4px 16px; border: 1px solid var(--color-border); background: var(--color-bg-subtle);
  color: var(--color-text-secondary); border-radius: var(--radius-sm); cursor: pointer; font-size: 12px; font-weight: 600;
}
.country-tabs button.active { background: var(--color-accent); color: #fff; }
.cards-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(240px, 1fr)); gap: 12px; }
.macro-card {
  padding: 12px; border: 1px solid var(--color-border); border-radius: var(--radius-md);
  background: var(--color-bg-subtle);
}
.macro-card h4 { font-size: 12px; margin-bottom: 8px; color: var(--color-accent); }
.macro-item { display: flex; gap: 8px; padding: 4px 0; border-bottom: 1px solid var(--color-border); font-size: 11px; }
.item-name { flex: 1; color: var(--color-text-primary); }
.item-value { font-weight: 600; font-family: 'JetBrains Mono', monospace; }
.item-unit { color: var(--color-text-tertiary); }
.empty { text-align: center; padding: 48px; color: var(--color-text-tertiary); }
</style>
