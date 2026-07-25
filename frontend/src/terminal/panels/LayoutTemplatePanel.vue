<script setup lang="ts">
import PanelShell from '@/terminal/components/panel/PanelShell.vue'
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
  <PanelShell state="loaded">
    <template #loaded>
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
  </PanelShell>
</template>

<style scoped>
.layout-template-panel { display: flex; flex-direction: column; height: 100%; overflow: hidden; }
.save-input { display: flex; gap: var(--space-xs); padding: var(--space-sm) var(--panel-padding); border-bottom: 1px solid var(--color-border); }
.save-input input { flex: 1; background: transparent; border: 1px solid var(--color-border-strong); color: var(--color-text-primary); padding: var(--space-xs) var(--space-sm); border-radius: var(--radius-sm); }
.save-input input:focus { outline: none; border-color: var(--color-accent); }
.save-input button { background: var(--color-accent); color: var(--color-text-primary); border: none; padding: var(--space-xs) var(--space-md); border-radius: var(--radius-sm); cursor: pointer; }
.save-input button:disabled { opacity: 0.5; cursor: not-allowed; }
.layout-list { flex: 1; overflow-y: auto; padding: var(--space-xs) 0; }
.layout-item { display: flex; align-items: center; justify-content: space-between; padding: var(--space-sm) var(--panel-padding); cursor: pointer; border-bottom: 1px solid var(--color-border-subtle); }
.layout-item:hover { background: var(--color-bg-subtle); }
.layout-info { display: flex; flex-direction: column; gap: var(--space-xs); flex: 1; }
.layout-hotkey { font-size: var(--font-xs); color: var(--color-text-tertiary); }
.layout-name { font-size: var(--font-sm); color: var(--color-text-primary); }
.delete-btn { background: none; border: none; color: var(--color-text-secondary); cursor: pointer; padding: var(--space-xs) var(--space-sm); border-radius: var(--radius-sm); }
.delete-btn:hover { background: var(--color-up-soft); color: var(--color-up); }
.loading, .empty { padding: var(--space-xl); text-align: center; color: var(--color-text-tertiary); }
.hint { padding: var(--space-sm) var(--panel-padding); font-size: var(--font-xs); color: var(--color-text-secondary); border-top: 1px solid var(--color-border-subtle); }
</style>
