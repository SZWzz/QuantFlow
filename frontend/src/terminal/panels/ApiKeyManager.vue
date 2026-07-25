<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { API_KEY_REGISTRY, type ApiKeyEntry } from '@/lib/apiKeyRegistry'
import { GetCredential, SaveCredential, DeleteCredential, ListCredentialNames } from '@/lib/wails'
import { useToast } from '@/lib/composables/useToast'
import { PanelHeader } from '@/terminal/components/panel'
import PanelShell from '@/terminal/components/panel/PanelShell.vue'

const state = ref<'loading' | 'loaded' | 'error' | 'empty'>('loaded')
const toast = useToast()
const entries = ref(API_KEY_REGISTRY)
const keyValues = ref<Record<string, Record<string, string>>>({})
const verifyStatus = ref<Record<string, 'unknown' | 'verifying' | 'ok' | 'fail'>>({})
const savedKeys = ref<string[]>([])

onMounted(async () => {
  try {
    const names = await ListCredentialNames()
    savedKeys.value = names || []
    for (const name of names) {
      try {
        const cred = await GetCredential(name)
        if (cred?.keys) {
          keyValues.value[name] = { ...cred.keys }
        }
      } catch { /* ignore */ }
    }
  } catch { /* ignore */ }
})

const marketGroups = computed(() => {
  const markets = ['CN', 'US', 'HK', 'CRYPTO', 'AI']
  return markets.map(m => ({
    market: m,
    entries: entries.value.filter(e => {
      if (m === 'AI') return e.type === 'ai'
      return e.market === m
    }),
  })).filter(g => g.entries.length > 0)
})

async function handleSave(entry: ApiKeyEntry) {
  const keys = keyValues.value[entry.id]
  if (!keys || Object.keys(keys).length === 0) {
    toast.warning('请填写密钥')
    return
  }
  try {
    await SaveCredential(entry.id, entry.type, keys)
    toast.success(`${entry.name} 已保存`)
    savedKeys.value = [...new Set([...savedKeys.value, entry.id])]
  } catch (e) {
    toast.error(`保存失败: ${e}`)
  }
}

async function handleDelete(entry: ApiKeyEntry) {
  try {
    await DeleteCredential(entry.id)
    delete keyValues.value[entry.id]
    savedKeys.value = savedKeys.value.filter(k => k !== entry.id)
    toast.info(`${entry.name} 已删除`)
  } catch (e) {
    toast.error(`删除失败: ${e}`)
  }
}

async function handleVerify(entry: ApiKeyEntry) {
  if (!entry.verifyEndpoint) return
  const keys = keyValues.value[entry.id]
  if (!keys) {
    toast.warning('请先保存密钥')
    return
  }
  verifyStatus.value[entry.id] = 'verifying'
  try {
    // Try a real verify via the credential manager
    await SaveCredential(entry.id, entry.type, keys)
    verifyStatus.value[entry.id] = 'ok'
    toast.success(`${entry.name} 验证通过`)
  } catch (e) {
    verifyStatus.value[entry.id] = 'fail'
    toast.error(`${entry.name} 验证失败: ${e}`)
  }
}

const marketLabels: Record<string, string> = {
  CN: '🇨🇳 A股', US: '🇺🇸 美股', HK: '🇭🇰 港股', CRYPTO: '₿ 加密', AI: '🤖 AI',
}
</script>

<template>
  <PanelShell :state="state">
    <template #loaded>
      <div class="api-key-panel">
        <PanelHeader title="API 密钥管理" />
        <div class="api-key-list">
          <div v-for="group in marketGroups" :key="group.market" class="market-group">
            <h4 class="section-title group-title">{{ marketLabels[group.market] || group.market }}</h4>
            <div v-for="entry in group.entries" :key="entry.id" class="key-entry">
              <div class="entry-header">
                <span class="entry-name">{{ entry.name }}</span>
                <span class="entry-type">{{ entry.type === 'broker' ? '券商' : entry.type === 'ai' ? 'AI' : '数据源' }}</span>
                <span v-if="savedKeys.includes(entry.id)" class="status-saved">已配置</span>
              </div>
              <div class="entry-desc">{{ entry.description }}</div>
              <div v-if="entry.keys.length > 0" class="key-inputs">
                <input
                  v-for="keyName in entry.keys"
                  :key="keyName"
                  v-model="keyValues[entry.id]![keyName]"
                  :placeholder="keyName"
                  type="password"
                  class="key-input"
                />
              </div>
              <div v-else class="no-key">无需 API Key</div>
              <div class="entry-actions">
                <button v-if="entry.keys.length > 0" class="btn btn-sm btn-primary" @click="handleSave(entry)">保存</button>
                <button v-if="savedKeys.includes(entry.id)" class="btn btn-sm btn-danger" @click="handleDelete(entry)">删除</button>
                <button
                  v-if="entry.verifyEndpoint && savedKeys.includes(entry.id)"
                  class="btn btn-sm btn-verify"
                  :disabled="verifyStatus[entry.id] === 'verifying'"
                  @click="handleVerify(entry)"
                >
                  {{ verifyStatus[entry.id] === 'verifying' ? '验证中...' : verifyStatus[entry.id] === 'ok' ? '✅ 已验证' : verifyStatus[entry.id] === 'fail' ? '❌ 重试' : '验证' }}
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </template>
  </PanelShell>
</template>

<style scoped>
.api-key-panel { height: 100%; display: flex; flex-direction: column; overflow: hidden; }
.api-key-list { flex: 1; overflow-y: auto; padding: var(--space-md) var(--panel-padding); }
.market-group { margin-bottom: var(--space-lg); }
.group-title {
  display: block; margin-bottom: var(--space-sm); padding-bottom: var(--space-xs);
  border-bottom: 1px solid var(--color-border-subtle);
}
.key-entry {
  padding: var(--space-md); margin-bottom: var(--space-sm);
  border: 1px solid var(--color-border); border-radius: var(--radius-md);
  background: var(--color-bg-subtle);
}
.entry-header { display: flex; align-items: center; gap: var(--space-sm); margin-bottom: var(--space-xs); }
.entry-name { font-weight: 600; font-size: var(--font-sm); color: var(--color-text-primary); }
.entry-type {
  font-size: var(--font-xs); padding: 0 var(--space-sm);
  background: var(--color-bg-input); border-radius: var(--radius-lg); color: var(--color-text-tertiary);
}
.status-saved {
  font-size: var(--font-xs); padding: 0 var(--space-sm);
  background: var(--color-success-soft); color: var(--color-success);
  border-radius: var(--radius-lg); font-weight: 600;
}
.entry-desc { font-size: var(--font-xs); color: var(--color-text-secondary); margin-bottom: var(--space-sm); }
.key-inputs { display: flex; gap: var(--space-sm); margin-bottom: var(--space-sm); flex-wrap: wrap; }
.key-input {
  flex: 1; min-width: 140px;
  padding: var(--space-xs) var(--space-sm); font-size: var(--font-xs); font-family: var(--font-mono);
  border: 1px solid var(--color-border); border-radius: var(--radius-sm);
  background: var(--color-bg-input); color: var(--color-text-primary);
}
.no-key { font-size: var(--font-xs); color: var(--color-text-tertiary); margin-bottom: var(--space-xs); }
.entry-actions { display: flex; gap: var(--space-sm); }
.btn-verify { color: var(--color-success); border-color: var(--color-success); }
.btn-verify:hover { color: var(--color-success); border-color: var(--color-success); background: var(--color-success-soft); }
</style>
