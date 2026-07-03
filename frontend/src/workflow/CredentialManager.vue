<script setup lang="ts">
import { ref, onMounted } from 'vue'

interface Credential {
  id: number
  name: string
  type: string
  keys: Record<string, string>
}

const creds = ref<Credential[]>([])
const loading = ref(false)
const showAdd = ref(false)
const formName = ref('')
const formType = ref('api_key')
const formKeys = ref('{"api_key":"","secret":""}')

async function loadCreds() {
  loading.value = true
  try {
    const app = (window as any).go?.main?.App
    if (!app?.ListCredentials) return
    creds.value = (await app.ListCredentials()) || []
  } catch { creds.value = [] }
  finally { loading.value = false }
}

async function saveCred() {
  try {
    const keys = JSON.parse(formKeys.value)
    const app = (window as any).go?.main?.App
    await app.SaveCredential(formName.value, formType.value, keys)
    formName.value = ''
    formKeys.value = '{"api_key":"","secret":""}'
    showAdd.value = false
    await loadCreds()
  } catch (e: any) {
    alert('保存失败: ' + (e?.message || String(e)))
  }
}

async function deleteCred(name: string) {
  if (!confirm('确定删除凭证 "' + name + '"？')) return
  const app = (window as any).go?.main?.App
  await app.DeleteCredential(name)
  await loadCreds()
}

onMounted(loadCreds)
</script>

<template>
  <div class="credential-manager">
    <div class="cred-header">
      <h3>凭证管理</h3>
      <button class="add-btn" @click="showAdd = !showAdd">{{ showAdd ? '取消' : '+ 添加' }}</button>
    </div>

    <div v-if="showAdd" class="add-form">
      <input v-model="formName" placeholder="名称 (如: Binance Main)" class="cred-input" />
      <select v-model="formType" class="cred-select">
        <option value="api_key">API Key</option>
        <option value="oauth">OAuth</option>
        <option value="basic_auth">Basic Auth</option>
      </select>
      <textarea v-model="formKeys" rows="4" placeholder='{"api_key":"...","secret":"..."}' class="cred-textarea" />
      <button class="save-btn" @click="saveCred" :disabled="!formName">保存</button>
    </div>

    <div v-if="creds.length === 0 && !loading" class="empty-state">暂无凭证</div>
    <div v-for="c in creds" :key="c.id" class="cred-item">
      <div class="cred-info">
        <span class="cred-name">{{ c.name }}</span>
        <span class="cred-type">{{ c.type }}</span>
        <span class="cred-keys">{{ Object.keys(c.keys).join(', ') }}</span>
      </div>
      <button class="del-btn" @click="deleteCred(c.name)">删除</button>
    </div>
  </div>
</template>

<style scoped>
.credential-manager { padding: 12px; }
.cred-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 12px; }
.cred-header h3 { font-size: 13px; color: var(--color-text-primary); margin: 0; }
.add-btn { padding: 4px 12px; border: 1px solid var(--color-accent); border-radius: var(--radius-sm); background: rgba(88,166,255,.1); color: var(--color-accent); cursor: pointer; font-size: 11px; }
.add-form { display: flex; flex-direction: column; gap: 6px; margin-bottom: 12px; padding: 10px; background: var(--color-bg-subtle); border-radius: var(--radius-md); }
.cred-input, .cred-select, .cred-textarea { padding: 6px 8px; background: var(--color-bg-input); border: 1px solid var(--color-border); border-radius: var(--radius-sm); color: var(--color-text-primary); font-size: 12px; outline: none; }
.cred-input:focus, .cred-textarea:focus { border-color: var(--color-accent); }
.cred-textarea { font-family: monospace; resize: vertical; }
.save-btn { padding: 6px 12px; background: var(--color-accent); border: none; border-radius: var(--radius-sm); color: #fff; cursor: pointer; font-size: 12px; }
.save-btn:disabled { opacity: .5; cursor: not-allowed; }
.empty-state { padding: 20px; text-align: center; color: var(--color-text-tertiary); font-size: 12px; }
.cred-item { display: flex; justify-content: space-between; align-items: center; padding: 8px 0; border-bottom: 1px solid var(--color-border); }
.cred-info { display: flex; flex-direction: column; gap: 2px; }
.cred-name { font-size: 12px; color: var(--color-text-primary); font-weight: 500; }
.cred-type { font-size: 10px; color: var(--color-text-tertiary); }
.cred-keys { font-size: 10px; color: var(--color-text-secondary); }
.del-btn { padding: 2px 8px; border: 1px solid rgba(248,81,73,.3); border-radius: var(--radius-sm); background: transparent; color: #f85149; cursor: pointer; font-size: 10px; }
.del-btn:hover { background: rgba(248,81,73,.1); }
</style>
