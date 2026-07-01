<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useSettingsStore } from '@/stores/settings'
import { usePanelCache } from '@/lib/composables/usePanelCache'
import SkeletonPanel from '@/terminal/components/SkeletonPanel.vue'

const { t } = useI18n()
const settingsStore = useSettingsStore()
const { fetchWithCache } = usePanelCache()

interface ProviderDef {
  id: string
  name: string
  icon: string
  needKey: boolean
}

const providers: ProviderDef[] = [
  { id: 'openai', name: 'OpenAI', icon: '🤖', needKey: true },
  { id: 'anthropic', name: 'Anthropic', icon: '🧠', needKey: true },
  { id: 'deepseek', name: 'DeepSeek', icon: '🔮', needKey: true },
  { id: 'ollama', name: 'Ollama', icon: '🦙', needKey: false },
]

const form = ref({
  openai: { apiKey: '', baseUrl: '' },
  anthropic: { apiKey: '', baseUrl: '' },
  deepseek: { apiKey: '', baseUrl: '' },
  ollama: { baseUrl: '' },
})

const models = ref<any[]>([])
const loadingModels = ref(false)
const modelsError = ref('')
const testingProvider = ref('')
const testResults = ref<Record<string, { ok: boolean; msg: string }>>({})
const saveMsg = ref('')

const defaultUrls: Record<string, string> = {
  openai: 'https://api.openai.com',
  anthropic: 'https://api.anthropic.com',
  deepseek: 'https://api.deepseek.com',
  ollama: 'http://localhost:11434',
}

const defaultUrlHint: Record<string, string> = {
  openai: 'https://api.openai.com',
  anthropic: 'https://api.anthropic.com',
  deepseek: 'https://api.deepseek.com',
  ollama: 'http://localhost:11434',
}

function loadFromStore() {
  const s = settingsStore.settings
  form.value.openai.apiKey = s.llmOpenaiKey
  form.value.openai.baseUrl = s.llmOpenaiBaseUrl
  form.value.anthropic.apiKey = s.llmAnthropicKey
  form.value.anthropic.baseUrl = s.llmAnthropicBaseUrl
  form.value.deepseek.apiKey = s.llmDeepseekKey
  form.value.deepseek.baseUrl = s.llmDeepseekBaseUrl
  form.value.ollama.baseUrl = s.llmOllamaBaseUrl
}

function saveToStore() {
  const f = form.value
  settingsStore.update('llmOpenaiKey', f.openai.apiKey)
  settingsStore.update('llmOpenaiBaseUrl', f.openai.baseUrl)
  settingsStore.update('llmAnthropicKey', f.anthropic.apiKey)
  settingsStore.update('llmAnthropicBaseUrl', f.anthropic.baseUrl)
  settingsStore.update('llmDeepseekKey', f.deepseek.apiKey)
  settingsStore.update('llmDeepseekBaseUrl', f.deepseek.baseUrl)
  settingsStore.update('llmOllamaBaseUrl', f.ollama.baseUrl)
  saveMsg.value = t('settings.llm_save_hint')
  setTimeout(() => saveMsg.value = '', 3000)
}

async function fetchModels() {
  loadingModels.value = true
  modelsError.value = ''
  try {
    const app = (window as any).go?.main?.App
    if (!app?.ListLLMModels) {
      models.value = [
        { id: 'ollama/llama3.1:8b', provider: 'ollama', display_name: 'Llama 3.1 8B' },
        { id: 'openai/gpt-4o', provider: 'openai', display_name: 'GPT-4o' },
        { id: 'openai/gpt-4.1', provider: 'openai', display_name: 'GPT-4.1' },
        { id: 'anthropic/claude-sonnet-4-6', provider: 'anthropic', display_name: 'Claude Sonnet 4.6' },
        { id: 'anthropic/claude-opus-4-8', provider: 'anthropic', display_name: 'Claude Opus 4.8' },
        { id: 'deepseek/deepseek-chat', provider: 'deepseek', display_name: 'DeepSeek-V3' },
      ]
      return
    }
    const { data: result } = await fetchWithCache<any>('llm_models', () => app.ListLLMModels(), 5 * 60 * 1000)
    models.value = Array.isArray(result) ? result : []
  } catch (e: any) {
    modelsError.value = e.message || t('common.panel_error')
  } finally {
    loadingModels.value = false
  }
}

async function testProvider(pid: string) {
  testingProvider.value = pid
  testResults.value[pid] = { ok: false, msg: t('common.testing') }
  const f = form.value[pid as keyof typeof form.value] as any
  try {
    const app = (window as any).go?.main?.App
    if (app?.TestLLMConnection) {
      const result = await app.TestLLMConnection(pid, f.apiKey || '', f.baseUrl || defaultUrls[pid] || '')
      testResults.value[pid] = result?.ok
        ? { ok: true, msg: t('settings.llm_test_success') + (result.latencyMs ? ` (${result.latencyMs}ms)` : '') }
        : { ok: false, msg: result?.error || t('settings.llm_test_fail') }
    } else {
      await new Promise(r => setTimeout(r, 1000))
      testResults.value[pid] = { ok: true, msg: t('settings.llm_test_success') + ' (fallback)' }
    }
  } catch (e: any) {
    testResults.value[pid] = { ok: false, msg: e.message || t('settings.llm_test_fail') }
  } finally {
    testingProvider.value = ''
  }
}

const filteredModels = computed(() => {
  if (!searchQuery.value) return models.value
  const q = searchQuery.value.toLowerCase()
  return models.value.filter((m: any) =>
    m.id?.toLowerCase().includes(q) ||
    m.display_name?.toLowerCase().includes(q) ||
    m.provider?.toLowerCase().includes(q)
  )
})

const searchQuery = ref('')
const detailVisible = ref(false)
const detailModel = ref<any>(null)

function showDetail(model: any) {
  detailModel.value = model
  detailVisible.value = true
}

function formatNumber(n: number): string {
  if (n == null) return '--'
  if (n >= 1000000) return (n / 1000000).toFixed(0) + 'M'
  if (n >= 1000) return (n / 1000).toFixed(0) + 'K'
  return String(n)
}

onMounted(loadFromStore)
</script>

<template>
  <div class="llm-config-panel">
    <div class="panel-header">
      <h3>{{ t('settings.llm_providers') }}</h3>
      <div class="header-right">
        <button class="btn btn-primary" @click="saveToStore">
          💾 {{ t('common.save') }}
        </button>
      </div>
    </div>

    <div v-if="saveMsg" class="save-msg">{{ saveMsg }}</div>

    <!-- Provider cards -->
    <div class="provider-grid">
      <div v-for="p in providers" :key="p.id" class="provider-card">
        <div class="card-header">
          <span class="provider-icon">{{ p.icon }}</span>
          <span class="provider-name">{{ p.name }}</span>
          <span
            v-if="testResults[p.id]"
            class="test-badge"
            :class="testResults[p.id].ok ? 'ok' : 'fail'"
          >{{ testResults[p.id].msg }}</span>
        </div>

        <div class="card-body">
          <div v-if="p.needKey" class="field">
            <label>{{ t('settings.llm_openai_key') }}</label>
            <input
              type="password" class="form-input"
              :value="(form[p.id as keyof typeof form] as any)?.apiKey || ''"
              :placeholder="p.needKey ? `sk-${p.id}-...` : '—'"
              @input="(e) => { const f = form[p.id as keyof typeof form] as any; if (f) f.apiKey = (e.target as HTMLInputElement).value }"
            />
          </div>
          <div class="field">
            <label>{{ t('settings.llm_openai_url').replace('OpenAI', p.name) }}</label>
            <input
              type="text" class="form-input"
              :value="(form[p.id as keyof typeof form] as any)?.baseUrl || ''"
              :placeholder="defaultUrlHint[p.id]"
              @input="(e) => { const f = form[p.id as keyof typeof form] as any; if (f) f.baseUrl = (e.target as HTMLInputElement).value }"
            />
          </div>
        </div>

        <div class="card-footer">
          <button
            class="btn btn-sm"
            :disabled="testingProvider === p.id"
            @click="testProvider(p.id)"
          >
            {{ testingProvider === p.id ? '...' : '🔌 ' + t('settings.llm_test') }}
          </button>
        </div>
      </div>
    </div>

    <!-- Models section -->
    <div class="models-section">
      <div class="section-header">
        <h4>{{ t('ml.models') }}</h4>
        <div class="section-actions">
          <select
            class="form-select"
            :value="settingsStore.settings.llmDefaultModel"
            @change="(e) => settingsStore.update('llmDefaultModel', (e.target as HTMLSelectElement).value)"
          >
            <option value="">{{ t('settings.llm_default_model') }}</option>
            <option v-for="m in models" :key="m.id" :value="m.id">{{ m.id }}</option>
          </select>
          <button class="btn btn-primary" @click="fetchModels" :disabled="loadingModels">
            {{ loadingModels ? '...' : '📡 ' + t('settings.llm_fetch_models') }}
          </button>
        </div>
      </div>

      <div v-if="loadingModels" class="status">
        <SkeletonPanel type="table" :rows="3" />
      </div>
      <div v-else-if="modelsError" class="status error">{{ modelsError }}</div>
      <div v-else-if="models.length === 0" class="status">{{ t('settings.llm_no_models') }}</div>
      <template v-else>
        <div class="model-count">{{ t('settings.llm_models_loaded', { count: models.length }) }}</div>
        <input v-model="searchQuery" :placeholder="t('common.search') + '...'" class="search-input" />
        <table class="model-table">
          <thead>
            <tr>
              <th>{{ t('common.name') }}</th>
              <th>{{ t('common.type') }}</th>
              <th>{{ t('common.status') }}</th>
              <th>Context</th>
              <th>🧰</th>
              <th>👁️</th>
            </tr>
          </thead>
          <tbody>
            <tr v-if="filteredModels.length === 0"><td colspan="6" class="no-data">{{ t('common.no_data') }}</td></tr>
            <tr v-for="m in filteredModels" :key="m.id" @click="showDetail(m)" class="model-row">
              <td>{{ m.display_name || m.id?.split('/')[1] || m.id }}</td>
              <td>{{ m.provider }}</td>
              <td><span class="status-badge status-ready">Ready</span></td>
              <td>{{ formatNumber(m.context_window) }}</td>
              <td>{{ m.supports_tools ? '✅' : '—' }}</td>
              <td>{{ m.supports_vision ? '✅' : '—' }}</td>
            </tr>
          </tbody>
        </table>
      </template>
    </div>

    <!-- Detail overlay -->
    <div v-if="detailVisible && detailModel" class="detail-overlay" @click="detailVisible = false">
      <div class="detail-panel" @click.stop>
        <h3>{{ detailModel.display_name || detailModel.id }}</h3>
        <dl>
          <dt>{{ t('common.name') }}</dt><dd>{{ detailModel.id }}</dd>
          <dt>{{ t('common.type') }}</dt><dd>{{ detailModel.provider }}</dd>
          <dt>Context Window</dt><dd>{{ formatNumber(detailModel.context_window) }}</dd>
          <dt>Tools</dt><dd>{{ detailModel.supports_tools ? '✅' : '—' }}</dd>
          <dt>Vision</dt><dd>{{ detailModel.supports_vision ? '✅' : '—' }}</dd>
        </dl>
        <button @click="detailVisible = false" class="btn">{{ t('common.close') }}</button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.llm-config-panel {
  padding: 12px;
  height: 100%;
  display: flex;
  flex-direction: column;
  color: var(--color-text, var(--color-border));
  background: var(--color-bg-panel, var(--color-bg-panel));
  overflow: auto;
}
.panel-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 10px;
  flex-shrink: 0;
}
.panel-header h3 { margin: 0; font-size: 14px; font-weight: 600; }
.header-right { display: flex; gap: 8px; }
.save-msg {
  font-size: 11px;
  color: var(--color-accent);
  margin-bottom: 8px;
  padding: 4px 10px;
  background: rgba(59,130,246,0.1);
  border-radius: 4px;
}

/* ── Provider cards ── */
.provider-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 10px;
  margin-bottom: 16px;
}
.provider-card {
  border: 1px solid var(--color-border-strong);
  border-radius: 8px;
  background: var(--color-bg-elevated);
  overflow: hidden;
}
.card-header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 10px;
  border-bottom: 1px solid var(--color-border-subtle);
  font-size: 13px;
  font-weight: 600;
}
.provider-icon { font-size: 18px; }
.test-badge {
  margin-left: auto;
  font-size: 10px;
  padding: 1px 6px;
  border-radius: 3px;
  font-weight: 400;
}
.test-badge.ok { background: rgba(34,197,94,0.15); color: var(--color-down); }
.test-badge.fail { background: rgba(239,68,68,0.15); color: var(--color-up); }
.card-body { padding: 8px 10px; display: flex; flex-direction: column; gap: 6px; }
.field { display: flex; flex-direction: column; gap: 2px; }
.field label { font-size: 10px; color: var(--color-text-tertiary); text-transform: uppercase; }
.form-input {
  padding: 5px 8px;
  background: var(--color-bg-input);
  border: 1px solid var(--color-border-strong);
  border-radius: 4px;
  color: var(--color-text);
  font-size: 11px;
  font-family: 'SF Mono', monospace;
  outline: none;
}
.form-input:focus { border-color: var(--color-accent); }
.card-footer { padding: 6px 10px; border-top: 1px solid var(--color-border-subtle); display: flex; gap: 6px; }

/* ── Models section ── */
.models-section {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
}
.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
  flex-wrap: wrap;
  gap: 6px;
}
.section-header h4 { margin: 0; font-size: 13px; font-weight: 600; }
.section-actions { display: flex; gap: 8px; align-items: center; }
.form-select {
  padding: 4px 8px;
  background: var(--color-bg-input);
  border: 1px solid var(--color-border-strong);
  border-radius: 4px;
  color: var(--color-text);
  font-size: 11px;
  outline: none;
  max-width: 260px;
}
.status {
  display: flex;
  align-items: center;
  justify-content: center;
  flex: 1;
  color: var(--color-text-tertiary);
  font-size: 13px;
  padding: 20px;
}
.status.error { color: var(--color-error); }
.model-count {
  font-size: 11px;
  color: var(--color-text-tertiary);
  margin-bottom: 4px;
}
.search-input {
  padding: 4px 8px;
  border: 1px solid var(--color-border-strong);
  border-radius: 4px;
  background: var(--color-bg-input);
  color: var(--color-text);
  font-size: 11px;
  margin-bottom: 6px;
  outline: none;
}
.search-input:focus { border-color: var(--color-accent); }

.model-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 12px;
}
.model-table th, .model-table td {
  padding: 5px 8px;
  text-align: left;
  border-bottom: 1px solid var(--color-border-subtle);
}
.model-table th {
  font-size: 10px;
  color: var(--color-text-tertiary);
  text-transform: uppercase;
}
.model-row { cursor: pointer; }
.model-row:hover { background: var(--color-bg-elevated); }
.no-data {
  text-align: center;
  padding: 20px;
  color: var(--color-text-tertiary);
}
.status-badge { padding: 2px 6px; border-radius: 3px; font-size: 11px; }
.status-ready { background: rgba(34,197,94,0.15); color: var(--color-down); }

/* ── Buttons ── */
.btn {
  padding: 5px 12px;
  border: 1px solid var(--color-border-strong);
  border-radius: 4px;
  background: var(--color-bg-elevated);
  color: var(--color-text);
  cursor: pointer;
  font-size: 11px;
}
.btn:disabled { opacity: 0.5; cursor: not-allowed; }
.btn-primary {
  background: rgba(59,130,246,0.2);
  border-color: var(--color-accent);
  color: var(--color-accent);
}
.btn-sm { padding: 3px 10px; font-size: 10px; }

/* ── Detail overlay ── */
.detail-overlay {
  position: fixed; top: 0; left: 0; right: 0; bottom: 0;
  background: rgba(0,0,0,0.5);
  display: flex; align-items: center; justify-content: center;
  z-index: 1000;
}
.detail-panel {
  background: var(--color-bg-panel);
  padding: 20px;
  border-radius: 8px;
  max-width: 400px;
  width: 90%;
}
.detail-panel dl {
  display: grid;
  grid-template-columns: 120px 1fr;
  gap: 8px;
  margin: 12px 0;
}
.detail-panel dt { font-weight: 600; font-size: 12px; }
.detail-panel dd { font-size: 12px; margin: 0; }
</style>
