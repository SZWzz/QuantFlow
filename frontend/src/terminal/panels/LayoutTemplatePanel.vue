<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { confirmDialog } from '@/lib/wails'
import { useTerminalStore } from '@/stores/terminal'
import PanelHeader from '@/terminal/components/panel/PanelHeader.vue'
import EmptyState from '@/terminal/components/panel/EmptyState.vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
const terminal = useTerminalStore()

const loading = ref(true)
const saving = ref(false)
const newName = ref('')
const showSaveInput = ref(false)

async function loadList() {
  loading.value = true
  try {
    await terminal.refreshLayouts()
  } finally {
    loading.value = false
  }
}

async function handleSave() {
  const name = newName.value.trim()
  if (!name) return
  saving.value = true
  try {
    await terminal.saveLayout(name)
    newName.value = ''
    showSaveInput.value = false
  } finally {
    saving.value = false
  }
}

async function handleLoad(name: string) {
  await terminal.loadLayout(name)
}

async function handleDelete(name: string) {
  const ok = await confirmDialog(t('layout.confirmDelete', { name }))
  if (!ok) return
  await terminal.deleteLayout(name)
}

onMounted(loadList)
</script>

<template>
  <div class="layout-template-panel">
    <PanelHeader
      :title="t('layout.title')"
      :controls="[
        { icon: 'plus', action: () => { showSaveInput = !showSaveInput; if (showSaveInput) newName = '' }, title: t('layout.saveNew'), loading: saving },
        { icon: 'refresh', action: loadList, title: t('common.refresh') },
      ]"
    />

    <div v-if="showSaveInput" class="save-input">
      <input
        v-model="newName"
        :placeholder="t('layout.namePlaceholder')"
        @keyup.enter="handleSave"
        data-testid="layout-name-input"
      />
      <button @click="handleSave" :disabled="!newName.trim()" data-testid="save-layout">
        {{ t('common.save') }}
      </button>
    </div>

    <div v-if="loading" class="loading">{{ t('common.loading') }}</div>

    <div v-else-if="terminal.savedLayouts.length === 0" class="empty">
      <EmptyState :title="t('layout.empty')" />
    </div>

    <div v-else class="layout-list">
      <div
        v-for="(name, idx) in terminal.savedLayouts"
        :key="name"
        class="layout-item"
      >
        <div class="layout-info" @click="handleLoad(name)">
          <span class="layout-hotkey">Ctrl+Shift+{{ idx + 1 }}</span>
          <span class="layout-name">{{ name }}</span>
        </div>
        <button class="delete-btn" @click="handleDelete(name)" :title="t('common.delete')">
          ✕
        </button>
      </div>
    </div>

    <div class="hint">
      {{ t('layout.hint') }}
    </div>
  </div>
</template>

<style scoped>
.layout-template-panel {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
}
.save-input {
  display: flex;
  gap: 4px;
  padding: 8px;
  border-bottom: 1px solid var(--border-color, #333);
}
.save-input input {
  flex: 1;
  background: transparent;
  border: 1px solid var(--border-color, #555);
  color: var(--text-color, #fff);
  padding: 4px 8px;
  border-radius: 4px;
}
.save-input input:focus {
  outline: none;
  border-color: var(--accent-color, #4a9eff);
}
.save-input button {
  background: var(--accent-color, #4a9eff);
  color: #fff;
  border: none;
  padding: 4px 12px;
  border-radius: 4px;
  cursor: pointer;
}
.save-input button:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
.layout-list {
  flex: 1;
  overflow-y: auto;
  padding: 4px 0;
}
.layout-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 12px;
  cursor: pointer;
  border-bottom: 1px solid var(--border-color, #222);
}
.layout-item:hover {
  background: var(--hover-bg, rgba(255,255,255,0.05));
}
.layout-info {
  display: flex;
  flex-direction: column;
  gap: 2px;
  flex: 1;
}
.layout-hotkey {
  font-size: 11px;
  color: var(--text-secondary, #888);
}
.layout-name {
  font-size: 14px;
  color: var(--text-color, #eee);
}
.delete-btn {
  background: none;
  border: none;
  color: var(--text-secondary, #666);
  cursor: pointer;
  padding: 4px 8px;
  border-radius: 4px;
}
.delete-btn:hover {
  background: rgba(255, 0, 0, 0.1);
  color: #ff4444;
}
.loading,
.empty {
  padding: 24px;
  text-align: center;
  color: var(--text-secondary, #888);
}
.hint {
  padding: 8px 12px;
  font-size: 11px;
  color: var(--text-secondary, #666);
  border-top: 1px solid var(--border-color, #222);
}
</style>
