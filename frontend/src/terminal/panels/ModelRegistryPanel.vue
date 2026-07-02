<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed } from 'vue'
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
  { id: 'google', name: 'Google Gemini', icon: '🌐', needKey: true },
  { id: 'mistral', name: 'Mistral AI', icon: '🌬️', needKey: true },
  { id: 'groq', name: 'Groq', icon: '⚡', needKey: true },
  { id: 'siliconflow', name: 'SiliconFlow', icon: '🔬', needKey: true },
  { id: 'zhipu', name: '智谱 AI', icon: '🎯', needKey: true },
  { id: 'openrouter', name: 'OpenRouter', icon: '🔀', needKey: true },
  { id: 'opencode', name: 'OpenCode Zen', icon: '🟢', needKey: true },
  { id: 'custom', name: '自定义', icon: '⚙️', needKey: true },
  { id: 'ollama', name: 'Ollama', icon: '🦙', needKey: false },
]

const form = ref({
  openai: { apiKey: '', baseUrl: '' },
  anthropic: { apiKey: '', baseUrl: '' },
  deepseek: { apiKey: '', baseUrl: '' },
  google: { apiKey: '', baseUrl: '' },
  mistral: { apiKey: '', baseUrl: '' },
  groq: { apiKey: '', baseUrl: '' },
  siliconflow: { apiKey: '', baseUrl: '' },
  zhipu: { apiKey: '', baseUrl: '' },
  openrouter: { apiKey: '', baseUrl: '' },
  opencode: { apiKey: '', baseUrl: '' },
  custom: { apiKey: '', baseUrl: '' },
  ollama: { baseUrl: '' },
})

const models = ref<any[]>([])
const loadingModels = ref(false)
const modelsError = ref('')
const testingProvider = ref('')
const testResults = ref<Record<string, { ok: boolean; msg: string }>>({})
const saveMsg = ref('')
const selectProvider = ref('') // provider id being selected (shows model picker)
let saveTimer: ReturnType<typeof setTimeout> | null = null

const defaultUrls: Record<string, string> = {
  openai: 'https://api.openai.com',
  anthropic: 'https://api.anthropic.com',
  deepseek: 'https://api.deepseek.com',
  google: 'https://generativelanguage.googleapis.com',
  mistral: 'https://api.mistral.ai',
  groq: 'https://api.groq.com/openai/v1',
  siliconflow: 'https://api.siliconflow.cn/v1',
  zhipu: 'https://open.bigmodel.cn/api/paas/v4',
  openrouter: 'https://openrouter.ai/api/v1',
  opencode: 'https://opencode.ai/zen/v1',
  custom: '',
  ollama: 'http://localhost:11434',
}

const defaultUrlHint: Record<string, string> = {
  openai: 'https://api.openai.com',
  anthropic: 'https://api.anthropic.com',
  deepseek: 'https://api.deepseek.com',
  google: 'https://generativelanguage.googleapis.com',
  mistral: 'https://api.mistral.ai',
  groq: 'https://api.groq.com/openai/v1',
  siliconflow: 'https://api.siliconflow.cn/v1',
  zhipu: 'https://open.bigmodel.cn/api/paas/v4',
  openrouter: 'https://openrouter.ai/api/v1',
  opencode: 'https://opencode.ai/zen/v1',
  custom: 'https://your-proxy.com/v1',
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
  form.value.google.apiKey = s.llmGoogleKey
  form.value.google.baseUrl = s.llmGoogleBaseUrl
  form.value.mistral.apiKey = s.llmMistralKey
  form.value.mistral.baseUrl = s.llmMistralBaseUrl
  form.value.groq.apiKey = s.llmGroqKey
  form.value.groq.baseUrl = s.llmGroqBaseUrl
  form.value.siliconflow.apiKey = s.llmSiliconflowKey
  form.value.siliconflow.baseUrl = s.llmSiliconflowBaseUrl
  form.value.zhipu.apiKey = s.llmZhipuKey
  form.value.zhipu.baseUrl = s.llmZhipuBaseUrl
  form.value.openrouter.apiKey = s.llmOpenrouterKey
  form.value.openrouter.baseUrl = s.llmOpenrouterBaseUrl
  form.value.opencode.apiKey = s.llmOpencodeKey
  form.value.opencode.baseUrl = s.llmOpencodeBaseUrl
  form.value.custom.apiKey = s.llmCustomKey
  form.value.custom.baseUrl = s.llmCustomBaseUrl
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
  settingsStore.update('llmGoogleKey', f.google.apiKey)
  settingsStore.update('llmGoogleBaseUrl', f.google.baseUrl)
  settingsStore.update('llmMistralKey', f.mistral.apiKey)
  settingsStore.update('llmMistralBaseUrl', f.mistral.baseUrl)
  settingsStore.update('llmGroqKey', f.groq.apiKey)
  settingsStore.update('llmGroqBaseUrl', f.groq.baseUrl)
  settingsStore.update('llmSiliconflowKey', f.siliconflow.apiKey)
  settingsStore.update('llmSiliconflowBaseUrl', f.siliconflow.baseUrl)
  settingsStore.update('llmZhipuKey', f.zhipu.apiKey)
  settingsStore.update('llmZhipuBaseUrl', f.zhipu.baseUrl)
  settingsStore.update('llmOpenrouterKey', f.openrouter.apiKey)
  settingsStore.update('llmOpenrouterBaseUrl', f.openrouter.baseUrl)
  settingsStore.update('llmOpencodeKey', f.opencode.apiKey)
  settingsStore.update('llmOpencodeBaseUrl', f.opencode.baseUrl)
  settingsStore.update('llmCustomKey', f.custom.apiKey)
  settingsStore.update('llmCustomBaseUrl', f.custom.baseUrl)
  settingsStore.update('llmOllamaBaseUrl', f.ollama.baseUrl)
  saveMsg.value = t('settings.llm_save_hint')
  if (saveTimer) clearTimeout(saveTimer)
  saveTimer = setTimeout(() => { saveMsg.value = ''; saveTimer = null }, 3000)
}

onUnmounted(() => { if (saveTimer) clearTimeout(saveTimer) })

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
        { id: 'google/gemini-2.5-flash', provider: 'google', display_name: 'Gemini 2.5 Flash' },
        { id: 'google/gemini-2.5-pro', provider: 'google', display_name: 'Gemini 2.5 Pro' },
        { id: 'mistral/mistral-large-2506', provider: 'mistral', display_name: 'Mistral Large' },
        { id: 'groq/llama-3.3-70b', provider: 'groq', display_name: 'Llama 3.3 70B (Groq)' },
        { id: 'siliconflow/deepseek-v3', provider: 'siliconflow', display_name: 'DeepSeek-V3 (SiliconFlow)' },
        { id: 'zhipu/glm-5', provider: 'zhipu', display_name: 'GLM-5' },
        { id: 'zhipu/glm-5-flash', provider: 'zhipu', display_name: 'GLM-5 Flash' },
        { id: 'openrouter/anthropic/claude-opus-4', provider: 'openrouter', display_name: 'Claude Opus 4 (OpenRouter)' },
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

async function fetchProviderModels(pid: string) {
  const f = form.value[pid as keyof typeof form.value] as any
  const app = (window as any).go?.main?.App
  if (!app?.ListProviderModels) return
  try {
    const result = await app.ListProviderModels(pid, f.apiKey || '', f.baseUrl || defaultUrls[pid] || '')
    const fetchedModels = (Array.isArray(result) ? result : []).map((m: any) => ({
      ...m,
      id: m.id || `${pid}/${m.ID || ''}`,
      provider: pid,
      display_name: m.display_name || m.id?.split('/').pop() || m.ID || m.id,
    }))
    models.value = models.value.filter((m: any) => m.provider !== pid)
    models.value.push(...fetchedModels)
  } catch (e: any) {
    console.error(`[ModelRegistry] fetch ${pid} models:`, e)
  }
}

function addCustomModels() {
  const raw = settingsStore.settings.llmCustomModels || ''
  const ids = raw.split(',').map(s => s.trim()).filter(Boolean)
  const name = settingsStore.settings.llmCustomName || 'Custom'
  models.value = models.value.filter((m: any) => m.provider !== 'custom')
  for (const id of ids) {
    models.value.push({
      id: `custom/${id}`,
      provider: 'custom',
      display_name: `${name}: ${id}`,
    })
  }
}

function confirmSelect(modelId: string, pid: string) {
  settingsStore.update('llmDefaultModel', modelId)
  saveToStore()
  selectProvider.value = ''
  const m = models.value.find(x => x.id === modelId)
  saveMsg.value = `✅ 已选定 ${m?.display_name || modelId}`
  if (saveTimer) clearTimeout(saveTimer)
  saveTimer = setTimeout(() => { saveMsg.value = ''; saveTimer = null }, 3000)
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
      if (result?.ok) {
        await fetchProviderModels(pid)
        if (pid === 'custom') addCustomModels()
      }
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
          <div v-if="p.id === 'custom'" class="field">
            <label>{{ t('settings.llm_custom_name') }}</label>
            <input
              type="text" class="form-input"
              :value="settingsStore.settings.llmCustomName"
              placeholder="My Proxy"
              @input="(e) => settingsStore.update('llmCustomName', (e.target as HTMLInputElement).value)"
            />
          </div>
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
          <div v-if="p.id === 'custom'" class="field">
            <label>{{ t('settings.llm_custom_models') }}</label>
            <input
              type="text" class="form-input"
              :value="settingsStore.settings.llmCustomModels"
              placeholder="gpt-4o, claude-sonnet-4, deepseek-chat"
              @input="(e) => settingsStore.update('llmCustomModels', (e.target as HTMLInputElement).value)"
            />
          </div>
        </div>

        <div class="card-footer">
          <button
            class="btn btn-sm"
            :disabled="testingProvider === p.id"
            @click="testProvider(p.id)"
          >
            {{ testingProvider === p.id ? '...' : (p.id === 'custom' ? '⚙️ ' + t('settings.llm_load_models') : '🔌 ' + t('settings.llm_test')) }}
          </button>
          <button v-if="p.id === 'custom'" class="btn btn-sm" @click="addCustomModels">
            📋 {{ t('settings.llm_apply_models') }}
          </button>
          <button
            v-if="testResults[p.id]?.ok"
            class="btn btn-sm btn-select"
            @click="selectProvider = p.id"
          >
            ✅ {{ t('common.select') }}
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
            {{ loadingModels ? '...' : '📡 ' + t('settings.llm_refresh_all') }}
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

    <!-- Model selection overlay (after test success) -->
    <div v-if="selectProvider" class="detail-overlay" @click="selectProvider = ''">
      <div class="detail-panel" @click.stop>
        <h3>选择 {{ providers.find(p => p.id === selectProvider)?.name }} 模型</h3>
        <div v-if="models.filter(m => m.provider === selectProvider).length === 0" class="status">
          {{ t('common.no_data') }}
        </div>
        <div
          v-for="m in models.filter(m => m.provider === selectProvider)"
          :key="m.id"
          class="select-row"
        >
          <span class="select-model-name">{{ m.display_name || m.id?.split('/')[1] || m.id }}</span>
          <span class="select-model-id">{{ m.id }}</span>
          <button class="btn btn-sm btn-select" @click="confirmSelect(m.id, selectProvider)">
            ✅ 设为默认
          </button>
        </div>
        <button @click="selectProvider = ''" class="btn" style="margin-top: 12px">
          {{ t('common.close') }}
        </button>
      </div>
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
  border-radius: var(--radius-sm);
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
  border-radius: var(--radius-lg);
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
  border-radius: var(--radius-sm);
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
  border-radius: var(--radius-sm);
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
  border-radius: var(--radius-sm);
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
  border-radius: var(--radius-sm);
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
.status-badge { padding: 2px 6px; border-radius: var(--radius-sm); font-size: 11px; }
.status-ready { background: rgba(34,197,94,0.15); color: var(--color-down); }

/* ── Buttons ── */
.btn {
  padding: 5px 12px;
  border: 1px solid var(--color-border-strong);
  border-radius: var(--radius-sm);
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
.btn-select {
  background: rgba(34,197,94,0.15);
  border-color: var(--color-down);
  color: var(--color-down);
}

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
  border-radius: var(--radius-lg);
  max-width: 400px;
  width: 90%;
  max-height: 80vh;
  overflow-y: auto;
}
.detail-panel dl {
  display: grid;
  grid-template-columns: 120px 1fr;
  gap: 8px;
  margin: 12px 0;
}
.detail-panel dt { font-weight: 600; font-size: 12px; }
.detail-panel dd { font-size: 12px; margin: 0; }
.select-row {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 0;
  border-bottom: 1px solid var(--color-border-subtle);
}
.select-model-name { flex: 1; font-size: 12px; font-weight: 500; }
.select-model-id { font-size: 10px; color: var(--color-text-tertiary); font-family: 'SF Mono', monospace; }
</style>
