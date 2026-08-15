<script setup lang="ts">
import { ref } from 'vue'
import { useWorkflowStore } from '@/stores/workflow'

defineEmits<{ (e: 'close'): void }>()
const workflow = useWorkflowStore()
const newName = ref('')
const renamingId = ref<string | null>(null)
const renameValue = ref('')

function doSave() {
  const name = newName.value.trim()
  if (!name) return
  workflow.saveWorkflow(name)
  newName.value = ''
}

function doDelete(id: string) { workflow.deleteWorkflow(id) }

function startRename(id: string, currentName: string) {
  renamingId.value = id; renameValue.value = currentName
}

function doRename() {
  if (renamingId.value && renameValue.value.trim()) {
    workflow.renameWorkflow(renamingId.value, renameValue.value.trim())
  }
  renamingId.value = null
}

function doLoad(id: string) { workflow.loadWorkflow(id) }
</script>

<template>
  <div class="wf-list-overlay" @click.self="$emit('close')">
    <div class="wf-list-drawer">
      <div class="drawer-header">
        <h3>{{ $t('workflow.my_workflows') }}</h3>
        <button class="close-btn" @click="$emit('close')">✕</button>
      </div>
      <div class="save-section">
        <input v-model="newName" type="text" :placeholder="$t('workflow.save_as')" class="save-input" @keyup.enter="doSave" />
        <button class="save-btn" @click="doSave" :disabled="!newName.trim()">{{ $t('workflow.save') }}</button>
      </div>
      <div class="wf-list">
        <div v-if="workflow.workflowList.length === 0" class="empty-hint">{{ $t('workflow.no_workflows') }}</div>
        <div v-for="wf in workflow.workflowList" :key="wf.id" class="wf-item">
          <div class="wf-item-main" @click="doLoad(wf.id)">
            <span class="wf-name" v-if="renamingId !== wf.id">{{ wf.name }}</span>
            <input v-if="renamingId === wf.id" v-model="renameValue" class="rename-input"
              @keyup.enter="doRename" @keyup.escape="renamingId = null" @blur="doRename" @click.stop autofocus />
            <span class="wf-meta">{{ wf.nodeCount }}{{ $t('workflow.nodes_count') }} · {{ wf.updatedAt.slice(0, 10) }}</span>
          </div>
          <div class="wf-item-actions">
            <button class="wf-act-btn" @click.stop="startRename(wf.id, wf.name)" title="重命名">✎</button>
            <button class="wf-act-btn danger" @click.stop="doDelete(wf.id)" title="删除">✕</button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.wf-list-overlay { position: fixed; inset: 0; background: var(--color-overlay); z-index: var(--z-overlay); display: flex; justify-content: flex-end; }
.wf-list-drawer { width: 340px; height: 100%; background: var(--color-bg-panel); border-left: 1px solid var(--color-border); display: flex; flex-direction: column; }
.drawer-header { display: flex; justify-content: space-between; align-items: center; padding: 12px 16px; border-bottom: 1px solid var(--color-border); }
.drawer-header h3 { margin: 0; font-size: 14px; color: var(--color-text-primary); }
.close-btn { background: none; border: none; color: var(--color-text-tertiary); cursor: pointer; font-size: 16px; }
.save-section { padding: 12px 16px; display: flex; gap: 8px; border-bottom: 1px solid var(--color-border); }
.save-input { flex: 1; padding: 6px 10px; background: var(--color-bg-input); border: 1px solid var(--color-border); border-radius: var(--radius-sm); color: var(--color-text-primary); font-size: 12px; outline: none; }
.save-btn { padding: 6px 14px; background: var(--color-accent); color: var(--color-text-inverse); border: none; border-radius: var(--radius-sm); cursor: pointer; font-size: 12px; font-weight: 600; }
.save-btn:disabled { opacity: 0.4; cursor: not-allowed; }
.wf-list { flex: 1; overflow-y: auto; padding: 8px 16px; }
.empty-hint { padding: 24px 0; text-align: center; color: var(--color-text-tertiary); font-size: 12px; }
.wf-item { display: flex; align-items: center; padding: 10px 0; border-bottom: 1px solid var(--color-border-subtle); }
.wf-item-main { flex: 1; cursor: pointer; min-width: 0; }
.wf-item-main:hover .wf-name { color: var(--color-accent); }
.wf-name { font-size: 13px; color: var(--color-text-primary); display: block; }
.rename-input { width: 100%; padding: 2px 6px; background: var(--color-bg-input); border: 1px solid var(--color-accent); border-radius: var(--radius-sm); color: var(--color-text-primary); font-size: 13px; outline: none; }
.wf-meta { font-size: var(--font-xs); color: var(--color-text-tertiary); margin-top: 2px; }
.wf-item-actions { display: flex; gap: 4px; flex-shrink: 0; }
.wf-act-btn { background: none; border: 1px solid var(--color-border); border-radius: var(--radius-sm); color: var(--color-text-tertiary); cursor: pointer; font-size: var(--font-xs); padding: 2px 6px; }
.wf-act-btn:hover { border-color: var(--color-accent); color: var(--color-accent); }
.wf-act-btn.danger:hover { border-color: var(--wf-danger); color: var(--wf-danger); }
</style>
