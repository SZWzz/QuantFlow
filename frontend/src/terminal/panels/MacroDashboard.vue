<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { GetMacroSnapshot, type MacroSnapshot } from '@/lib/wails'
import { PanelHeader, EmptyState, LoadingState } from '@/terminal/components/panel'

const country = ref('CN')
const snapshot = ref<MacroSnapshot | null>(null)
const loading = ref(false)

const countryTabs = [
  { key: 'CN', label: '中国' },
  { key: 'US', label: '美国' },
]

onMounted(() => fetchData())

async function fetchData() {
  loading.value = true
  try {
    snapshot.value = await GetMacroSnapshot(country.value)
  } catch { snapshot.value = null }
  finally { loading.value = false }
}

function switchCountry(c: string) {
  country.value = c
  fetchData()
}
</script>

<template>
  <div class="macro-dashboard">
    <PanelHeader
      title="宏观仪表盘"
      :tabs="countryTabs"
      :active-tab="country"
      @tab-change="switchCountry"
    />

    <LoadingState v-if="loading" type="card" :rows="2" />

    <div v-else-if="snapshot" class="cards-grid">
      <div v-for="card in [
        {title:'📈 增长', items:snapshot.growth},
        {title:'💰 通胀', items:snapshot.inflation},
        {title:'🏦 货币', items:snapshot.monetary},
        {title:'📋 政策', items:snapshot.policy},
      ].filter(c=>c.items.length>0)" :key="card.title" class="macro-card">
        <h4 class="section-title card-title">{{ card.title }}</h4>
        <div v-for="item in card.items.slice(0, 5)" :key="item.name" class="macro-item">
          <span class="item-name">{{ item.name }}</span>
          <span class="item-value">{{ item.value?.toFixed(2) || '-' }}</span>
          <span class="item-unit">{{ item.unit }}</span>
        </div>
      </div>
    </div>

    <EmptyState v-else title="暂无宏观数据" description="需要 Python sidecar 运行" />
  </div>
</template>

<style scoped>
.macro-dashboard { height: 100%; display: flex; flex-direction: column; overflow: hidden; }
.cards-grid {
  flex: 1; min-height: 0; overflow-y: auto;
  display: grid; grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
  gap: var(--space-md); padding: var(--panel-padding);
}
.macro-card {
  padding: var(--space-md); border: 1px solid var(--color-border); border-radius: var(--radius-md);
  background: var(--color-bg-subtle);
}
.card-title { margin-bottom: var(--space-sm); color: var(--color-accent); }
.macro-item { display: flex; gap: var(--space-sm); padding: var(--space-xs) 0; border-bottom: 1px solid var(--color-border); font-size: var(--font-xs); }
.item-name { flex: 1; color: var(--color-text-primary); }
.item-value { font-weight: 600; font-family: var(--font-mono); }
.item-unit { color: var(--color-text-tertiary); }
</style>
