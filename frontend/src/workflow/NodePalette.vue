<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { ListNodes } from '@/lib/wails'
import { useWorkflowStore } from '@/stores/workflow'
import { TEMPLATES } from './templates'
import { nodeLabel } from './nodeLabels'

interface NodeMeta { node_type: string; category: string }

const { t } = useI18n()
const workflow = useWorkflowStore()
const showTemplates = ref(false)

function insertTemplate(tpl: typeof TEMPLATES[0]) {
  const nodeIds: string[] = tpl.nodes.map(n =>
    workflow.addNode(n.node_type, { x: n.x, y: n.y + 100 }, n.params)
  )
  tpl.edges.forEach(e => {
    const srcId = nodeIds[e.from]
    const tgtId = nodeIds[e.to]
    if (srcId && tgtId) {
      workflow.addEdge({
        id: `e-${srcId}-${tgtId}`,
        source: srcId,
        target: tgtId,
        sourceHandle: e.from_port,
        targetHandle: e.to_port,
        type: 'default',
        style: { stroke: '#30363d', strokeWidth: 2 },
      } as any)
    }
  })
}

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

const catLabels: Record<string, string> = {
  data: t('workflow.cat_data'), indicator: t('workflow.cat_indicator'),
  indicators: t('workflow.cat_indicators'), signal: t('workflow.cat_signal'),
  output: t('workflow.cat_output'), control: t('workflow.cat_control'),
  alpha: t('workflow.cat_alpha'), strategy: t('workflow.cat_strategy'),
  backtest: t('workflow.cat_backtest'), ai: t('workflow.cat_ai'),
  trading: t('workflow.cat_trading'), notify: t('workflow.cat_notify'),
  schedule: t('workflow.cat_schedule'), portfolio: t('workflow.cat_portfolio'),
  risk: t('workflow.cat_risk'), utility: t('workflow.cat_utility'),
  ml: t('workflow.cat_ml'), research: t('workflow.cat_research'),
  alternative_data: t('workflow.cat_alternative_data'),
}

const categoryColors: Record<string, string> = {
  data: '#58a6ff', indicator: '#3fb950', indicators: '#22c55e', signal: '#f0883e', output: '#a371f7', control: '#e94560',
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
      <!-- Favorites -->
      <div v-if="workflow.favoriteTypes.size > 0 && !searchQuery" class="category-group">
        <div class="category-label">⭐ {{ $t('workflow.favorites') }}</div>
        <div
          v-for="node in nodes.filter(n => workflow.favoriteTypes.has(n.node_type))"
          :key="'fav-' + node.node_type"
          class="node-item"
          draggable="true"
          @dragstart="onDragStart($event, node.node_type)"
        >
          <span class="node-name">{{ nodeLabel(node.node_type) }}</span>
        </div>
      </div>

      <!-- Recent -->
      <div v-if="workflow.recentTypes.length > 0 && !searchQuery" class="category-group">
        <div class="category-label">🕐 {{ $t('workflow.recent') }}</div>
        <div
          v-for="rtype in workflow.recentTypes"
          :key="'rec-' + rtype"
          class="node-item"
          draggable="true"
          @dragstart="onDragStart($event, rtype)"
        >
          <span class="node-name">{{ nodeLabel(rtype) }}</span>
        </div>
      </div>

      <div v-for="(group, cat) in categories" :key="cat" class="category-group">
        <div class="category-label">
          <span class="cat-dot" :style="{ background: categoryColors[cat] || 'var(--color-text-tertiary)' }" />
          {{ catLabels[cat] || cat }}
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
          <span class="node-name">{{ nodeLabel(node.node_type) }}</span>
        </div>
      </div>

      <div v-if="Object.keys(categories).length === 0" class="no-results">
        {{ $t('common.no_data') }}
      </div>

      <!-- Templates -->
      <div class="templates-section">
        <div class="templates-toggle" @click="showTemplates = !showTemplates">
          <span>📋 {{ $t('workflow.templates') }}</span>
          <span class="toggle-arrow">{{ showTemplates ? '▾' : '▸' }}</span>
        </div>
        <div v-if="showTemplates" class="templates-list">
          <div
            v-for="tpl in TEMPLATES"
            :key="tpl.id"
            class="template-card"
            @click="insertTemplate(tpl)"
          >
            <div class="tpl-header">
              <span class="tpl-icon">{{ tpl.icon }}</span>
              <div class="tpl-meta">
                <span class="tpl-name">{{ tpl.name }}</span>
                <span class="tpl-count">{{ tpl.nodes.length }}{{ $t('workflow.nodes_count') }}</span>
              </div>
            </div>
            <p class="tpl-desc">{{ tpl.description }}</p>
            <div class="tpl-flow">
              <div
                v-for="(step, si) in tpl.nodes"
                :key="si"
                class="flow-step"
                :class="{ last: si === tpl.nodes.length - 1 }"
              >
                <div class="flow-dot" :title="step.node_type" />
                <span class="flow-label">{{ nodeLabel(step.node_type) }}</span>
                <span v-if="si < tpl.nodes.length - 1" class="flow-arrow">→</span>
              </div>
            </div>
          </div>
        </div>
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
  border-radius: var(--radius-sm);
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
  border-radius: var(--radius-sm); cursor: grab; transition: background 0.1s;
}
.node-item:hover { background: rgba(88,166,255,0.08); }
.node-item:active { cursor: grabbing; }

.node-icon { width: 24px; height: 24px; border: 1.5px solid var(--color-border); border-radius: var(--radius-sm); display: flex; align-items: center; justify-content: center; font-size: 12px; }
.node-name { font-size: 11px; color: var(--color-text-primary); }

.no-results { padding: 16px; text-align: center; color: var(--color-text-tertiary); font-size: 12px; }

.templates-section { margin-top: 8px; border-top: 1px solid var(--color-border); padding-top: 6px; }
.templates-toggle { display: flex; justify-content: space-between; align-items: center; padding: 6px; cursor: pointer; font-size: 11px; color: var(--color-text-secondary); }
.templates-toggle:hover { color: var(--color-accent); }
.toggle-arrow { font-size: 10px; }
.templates-list { padding: 0 4px; display: flex; flex-direction: column; gap: 8px; }

.template-card {
  background: var(--color-bg-subtle, #161b22);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md, 6px);
  padding: 10px;
  cursor: pointer;
  transition: border-color 0.15s, background 0.15s;
}
.template-card:hover {
  border-color: var(--color-accent, #58a6ff);
  background: rgba(88,166,255,0.04);
}

.tpl-header { display: flex; align-items: center; gap: 8px; margin-bottom: 6px; }
.tpl-icon { font-size: 18px; flex-shrink: 0; }
.tpl-meta { display: flex; flex-direction: column; min-width: 0; }
.tpl-name { font-size: 12px; font-weight: 600; color: var(--color-text-primary); }
.tpl-count { font-size: 10px; color: var(--color-text-tertiary); }

.tpl-desc { font-size: 10px; color: var(--color-text-secondary); line-height: 1.4; margin: 0 0 8px; }

.tpl-flow {
  display: flex; flex-wrap: wrap; gap: 3px 0;
  padding: 6px 4px;
  background: var(--color-bg-input, #0d1117);
  border-radius: var(--radius-sm, 4px);
}
.flow-step { display: flex; align-items: center; gap: 4px; font-size: 0; }
.flow-step.last .flow-arrow { display: none; }
.flow-dot {
  width: 6px; height: 6px; border-radius: 50%;
  background: var(--color-accent, #58a6ff); flex-shrink: 0;
}
.flow-label { font-size: 9px; color: var(--color-text-tertiary); white-space: nowrap; }
.flow-arrow { font-size: 8px; color: var(--color-text-tertiary); margin: 0 2px; }
</style>
