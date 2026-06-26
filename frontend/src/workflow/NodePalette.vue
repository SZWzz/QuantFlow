<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { ListNodes } from '@/lib/wails'

interface NodeMeta { node_type: string; category: string }

const searchQuery = ref('')
const nodes = ref<NodeMeta[]>([])
const loading = ref(false)

async function loadNodes() {
  loading.value = true
  try {
    const result = await ListNodes()
    nodes.value = Array.isArray(result) ? result : []
  } catch { nodes.value = [] }
  finally { loading.value = false }
}

onMounted(loadNodes)

const categories = computed(() => {
  const q = searchQuery.value.toLowerCase().trim()
  const filtered = q
    ? nodes.value.filter(n => n.node_type.toLowerCase().includes(q) || n.category.toLowerCase().includes(q))
    : nodes.value

  const grouped: Record<string, NodeMeta[]> = {}
  for (const n of filtered) {
    if (!grouped[n.category]) grouped[n.category] = []
    grouped[n.category].push(n)
  }
  return grouped
})

const categoryLabels: Record<string, string> = {
  data: '数据', indicator: '指标', signal: '信号', output: '输出', control: '控制',
  alpha: 'Alpha', strategy: '策略', backtest: '回测', ai: 'AI',
  trading: '交易', notify: '通知', schedule: '调度',
  portfolio: '组合', risk: '风控', utility: '工具',
  ml: '机器学习', research: '研究', alternative_data: '另类数据',
}

const categoryColors: Record<string, string> = {
  data: '#58a6ff', indicator: '#3fb950', signal: '#f0883e', output: '#a371f7', control: '#e94560',
  alpha: '#f59e0b', strategy: '#06b6d4', backtest: '#8b5cf6', ai: '#ec4899',
  trading: '#22c55e', notify: '#f97316', schedule: '#6366f1',
  portfolio: '#14b8a6', risk: '#ef4444', utility: '#64748b',
  ml: '#a855f7', research: '#0ea5e9', alternative_data: '#84cc16',
}

function onDragStart(event: DragEvent, nodeType: string) {
  if (event.dataTransfer) {
    event.dataTransfer.setData('application/node-type', nodeType)
    event.dataTransfer.effectAllowed = 'move'
  }
}
</script>

<template>
  <div class="node-palette">
    <div class="palette-header">
      <h3>{{ $t('workflow.node_palette') }}</h3>
      <input
        v-model="searchQuery"
        type="text"
        class="search-input"
        :placeholder="$t('common.search') + '...'"
      />
    </div>

    <div class="palette-list">
      <div v-for="(group, cat) in categories" :key="cat" class="category-group">
        <div class="category-label">
          <span class="cat-dot" :style="{ background: categoryColors[cat] || 'var(--color-text-tertiary)' }" />
          {{ categoryLabels[cat] || cat }}
        </div>
        <div
          v-for="node in group"
          :key="node.node_type"
          class="node-item"
          draggable="true"
          @dragstart="onDragStart($event, node.node_type)"
        >
          <div class="node-icon" :style="{ borderColor: categoryColors[cat] || 'var(--color-text-tertiary)' }">
            <span :style="{ color: categoryColors[cat] || 'var(--color-text-tertiary)' }">⬡</span>
          </div>
          <span class="node-name">{{ node.node_type }}</span>
        </div>
      </div>

      <div v-if="Object.keys(categories).length === 0" class="no-results">
        {{ $t('common.no_data') }}
      </div>
    </div>
  </div>
</template>

<style scoped>
.node-palette {
  width: 200px;
  background: var(--color-bg-panel);
  border-right: 1px solid var(--color-border);
  display: flex;
  flex-direction: column;
  flex-shrink: 0;
}

.palette-header {
  padding: 10px;
  border-bottom: 1px solid var(--color-border);
}

.palette-header h3 {
  font-size: 11px;
  text-transform: uppercase;
  color: var(--color-text-tertiary);
  letter-spacing: 0.5px;
  margin-bottom: 8px;
}

.search-input {
  width: 100%;
  padding: 5px 8px;
  background: var(--color-bg-input);
  border: 1px solid var(--color-border);
  border-radius: 4px;
  color: var(--color-text-primary);
  font-size: 11px;
  outline: none;
}
.search-input:focus { border-color: var(--color-accent); }

.palette-list { flex: 1; overflow-y: auto; padding: 6px; }

.category-group { margin-bottom: 8px; }

.category-label {
  display: flex; align-items: center; gap: 6px;
  padding: 4px 6px; font-size: 10px; color: var(--color-text-tertiary);
  text-transform: uppercase; letter-spacing: 0.5px;
}
.cat-dot { width: 6px; height: 6px; border-radius: 50%; }

.node-item {
  display: flex; align-items: center; gap: 8px; padding: 6px 8px;
  border-radius: 4px; cursor: grab; transition: background 0.1s;
}
.node-item:hover { background: rgba(88,166,255,0.08); }
.node-item:active { cursor: grabbing; }

.node-icon { width: 24px; height: 24px; border: 1.5px solid var(--color-border); border-radius: 4px; display: flex; align-items: center; justify-content: center; font-size: 12px; }
.node-name { font-size: 11px; color: var(--color-text-primary); }

.no-results { padding: 16px; text-align: center; color: var(--color-text-tertiary); font-size: 12px; }
</style>
