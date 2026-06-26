<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useMLStore, type MLModel } from '@/stores/ml'

const mlStore = useMLStore()

const searchQuery = ref('')
const typeFilter = ref('')
const categoryFilter = ref('')
const statusFilter = ref('')
const detailVisible = ref(false)
const detailModel = ref<MLModel | null>(null)

const filteredModels = computed(() => {
  let list = mlStore.models
  if (searchQuery.value) {
    const q = searchQuery.value.toLowerCase()
    list = list.filter(m => m.name.toLowerCase().includes(q))
  }
  if (typeFilter.value) list = list.filter(m => m.model_type === typeFilter.value)
  if (categoryFilter.value) list = list.filter(m => m.category === categoryFilter.value)
  if (statusFilter.value) list = list.filter(m => m.status === statusFilter.value)
  return list
})

function showDetail(model: MLModel) {
  detailModel.value = model
  detailVisible.value = true
}

function handleArchive(model: MLModel) { mlStore.archiveModel(model.id) }
function handleDelete(model: MLModel) { mlStore.deleteModel(model.id) }
function handleDragToWorkflow(model: MLModel) {
  window.dispatchEvent(new CustomEvent('quantflow:drag-node', {
    detail: { nodeType: 'predict', params: { model_id: model.id } }
  }))
}

onMounted(() => { mlStore.fetchModels() })
</script>

<template>
  <div class="model-registry-panel">
    <div class="toolbar">
      <input v-model="searchQuery" placeholder="Search models..." class="search-input" />
      <select v-model="typeFilter" class="filter-select">
        <option value="">{{ $t('ml.all_types') }}</option>
        <option value="xgboost">XGBoost</option>
        <option value="lightgbm">LightGBM</option>
        <option value="lstm">LSTM</option>
        <option value="transformer">Transformer</option>
      </select>
      <select v-model="categoryFilter" class="filter-select">
        <option value="">{{ $t('ml.all_categories') }}</option>
        <option value="prediction">Prediction</option>
        <option value="alpha_mining">Alpha Mining</option>
        <option value="rl">RL</option>
        <option value="risk">Risk</option>
      </select>
      <select v-model="statusFilter" class="filter-select">
        <option value="">{{ $t('ml.all_status') }}</option>
        <option value="ready">Ready</option>
        <option value="training">Training</option>
        <option value="failed">{{ $t('ml.failed') }}</option>
        <option value="archived">{{ $t('ml.archived') }}</option>
      </select>
    </div>

    <div v-if="mlStore.loading" class="chart-fallback">{{ $t('common.loading') }}</div>
    <table v-else class="model-table">
      <thead>
        <tr>
          <th>{{ $t('common.name') }}</th>
          <th>{{ $t('common.type') }}</th>
          <th>{{ $t('ml.category') }}</th>
          <th>{{ $t('common.status') }}</th>
          <th>{{ $t('ml.created') }}</th>
          <th>{{ $t('common.actions') }}</th>
        </tr>
      </thead>
      <tbody>
        <tr v-if="filteredModels.length === 0"><td colspan="6" class="no-data">{{ $t('common.no_data') }}</td></tr>
        <tr v-for="model in filteredModels" :key="model.id" @click="showDetail(model)" class="model-row">
          <td>{{ model.name }}</td>
          <td>{{ model.model_type }}</td>
          <td>{{ model.category }}</td>
          <td><span :class="'status-badge status-' + model.status">{{ model.status }}</span></td>
          <td>{{ model.created_at?.slice(0, 10) }}</td>
          <td>
            <button class="btn btn-sm" @click.stop="handleDragToWorkflow(model)">{{ $t('ml.add_to_workflow') }}</button>
            <button class="btn btn-sm btn-warning" @click.stop="handleArchive(model)">{{ $t('ml.archive') }}</button>
            <button class="btn btn-sm btn-danger" @click.stop="handleDelete(model)">{{ $t('common.delete') }}</button>
          </td>
        </tr>
      </tbody>
    </table>

    <div v-if="detailVisible && detailModel" class="detail-overlay" @click="detailVisible = false">
      <div class="detail-panel" @click.stop>
        <h3>Model Details</h3>
        <dl>
          <dt>{{ $t('common.name') }}</dt><dd>{{ detailModel.name }}</dd>
          <dt>{{ $t('common.type') }}</dt><dd>{{ detailModel.model_type }}</dd>
          <dt>{{ $t('ml.category') }}</dt><dd>{{ detailModel.category }}</dd>
          <dt>{{ $t('common.status') }}</dt><dd>{{ detailModel.status }}</dd>
          <dt>{{ $t('ml.hyperparams') }}</dt><dd><pre>{{ JSON.stringify(detailModel.hyperparams, null, 2) }}</pre></dd>
          <dt>Metrics</dt><dd><pre>{{ JSON.stringify(detailModel.metrics, null, 2) }}</pre></dd>
        </dl>
        <button @click="detailVisible = false">{{ $t('common.close') }}</button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.model-registry-panel { display: flex; flex-direction: column; height: 100%; padding: 8px; }
.toolbar { display: flex; gap: 8px; margin-bottom: 8px; flex-wrap: wrap; }
.search-input { padding: 4px 8px; border: 1px solid var(--border-color); border-radius: 4px; }
.filter-select { padding: 4px 8px; }
.model-table { width: 100%; border-collapse: collapse; }
.model-table th, .model-table td { padding: 6px 8px; text-align: left; border-bottom: 1px solid var(--border-color); }
.model-row { cursor: pointer; }
.model-row:hover { background: var(--hover-bg); }
.status-badge { padding: 2px 6px; border-radius: 3px; font-size: 0.85em; }
.status-ready { background: #d4edda; color: #155724; }
.status-training { background: #fff3cd; color: #856404; }
.status-failed { background: #f8d7da; color: #721c24; }
.status-archived { background: #e2e3e5; color: #383d41; }
.btn { padding: 4px 8px; border: 1px solid var(--border-color); border-radius: 4px; cursor: pointer; }
.btn-sm { font-size: 0.85em; }
.btn-warning { background: #ffc107; }
.btn-danger { background: #dc3545; color: white; }
.detail-overlay { position: fixed; top: 0; left: 0; right: 0; bottom: 0; background: rgba(0,0,0,0.5); display: flex; align-items: center; justify-content: center; z-index: 1000; }
.detail-panel { background: var(--bg-color); padding: 20px; border-radius: 8px; max-width: 500px; width: 90%; max-height: 80vh; overflow-y: auto; }
.detail-panel dl { display: grid; grid-template-columns: 120px 1fr; gap: 8px; }
.detail-panel dt { font-weight: bold; }
.detail-panel pre { background: var(--code-bg); padding: 8px; border-radius: 4px; font-size: 0.85em; overflow-x: auto; }
.chart-fallback { display: flex; align-items: center; justify-content: center; height: 100%; color: var(--color-text-tertiary); padding: 40px; text-align: center; }
.no-data { color: var(--color-text-tertiary); text-align: center; padding: 40px; }
</style>
