<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { GetVersion } from '@/lib/wails'

interface NodeMeta { node_type: string; category: string }

const searchQuery = ref('')
const nodes = ref<NodeMeta[]>([
  { node_type: 'data_loader', category: 'data' },
  { node_type: 'sma', category: 'indicator' },
  { node_type: 'cross_signal', category: 'signal' },
  { node_type: 'log_output', category: 'output' },
  { node_type: 'loop', category: 'control' },
])

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
  data: 'Data',
  indicator: 'Indicator',
  signal: 'Signal',
  output: 'Output',
  control: 'Control',
}

const categoryColors: Record<string, string> = {
  data: '#58a6ff',
  indicator: '#3fb950',
  signal: '#f0883e',
  output: '#a371f7',
  control: '#e94560',
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
      <h3>Nodes</h3>
      <input
        v-model="searchQuery"
        type="text"
        class="search-input"
        placeholder="Filter nodes..."
      />
    </div>

    <div class="palette-list">
      <div v-for="(group, cat) in categories" :key="cat" class="category-group">
        <div class="category-label">
          <span class="cat-dot" :style="{ background: categoryColors[cat] || '#5a6380' }" />
          {{ categoryLabels[cat] || cat }}
        </div>
        <div
          v-for="node in group"
          :key="node.node_type"
          class="node-item"
          draggable="true"
          @dragstart="onDragStart($event, node.node_type)"
        >
          <div class="node-icon" :style="{ borderColor: categoryColors[cat] || '#5a6380' }">
            <span :style="{ color: categoryColors[cat] || '#5a6380' }">⬡</span>
          </div>
          <span class="node-name">{{ node.node_type }}</span>
        </div>
      </div>

      <div v-if="Object.keys(categories).length === 0" class="no-results">
        No nodes found
      </div>
    </div>
  </div>
</template>

<style scoped>
.node-palette {
  width: 200px;
  background: #161b22;
  border-right: 1px solid #30363d;
  display: flex;
  flex-direction: column;
  flex-shrink: 0;
}

.palette-header {
  padding: 10px;
  border-bottom: 1px solid #30363d;
}

.palette-header h3 {
  font-size: 11px;
  text-transform: uppercase;
  color: #5a6380;
  letter-spacing: 0.5px;
  margin-bottom: 8px;
}

.search-input {
  width: 100%;
  padding: 5px 8px;
  background: #0d1117;
  border: 1px solid #30363d;
  border-radius: 4px;
  color: #c9d1d9;
  font-size: 11px;
  outline: none;
}
.search-input:focus { border-color: #58a6ff; }

.palette-list { flex: 1; overflow-y: auto; padding: 6px; }

.category-group { margin-bottom: 8px; }

.category-label {
  display: flex; align-items: center; gap: 6px;
  padding: 4px 6px; font-size: 10px; color: #5a6380;
  text-transform: uppercase; letter-spacing: 0.5px;
}
.cat-dot { width: 6px; height: 6px; border-radius: 50%; }

.node-item {
  display: flex; align-items: center; gap: 8px; padding: 6px 8px;
  border-radius: 4px; cursor: grab; transition: background 0.1s;
}
.node-item:hover { background: rgba(88,166,255,0.08); }
.node-item:active { cursor: grabbing; }

.node-icon { width: 24px; height: 24px; border: 1.5px solid #30363d; border-radius: 4px; display: flex; align-items: center; justify-content: center; font-size: 12px; }
.node-name { font-size: 11px; color: #c9d1d9; }

.no-results { padding: 16px; text-align: center; color: #5a6380; font-size: 12px; }
</style>
