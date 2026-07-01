<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { usePanelCache } from '@/lib/composables/usePanelCache'

defineProps<{ panelId: string; params?: Record<string, any> }>()

interface ScheduleTask {
  id: string; name: string; cron_expr: string; workflow_id: string
  enabled: boolean; timeout_sec: number; last_run_status: string
}

const { t } = useI18n()
const { fetchWithCache } = usePanelCache()
const tasks = ref<ScheduleTask[]>([])
const loading = ref(false)
const showModal = ref(false)
const editTask = ref({ name: '', cron_expr: '0 9 * * *', workflow_id: '', timeout_sec: 1800, editingId: '' })
const cronPresets = [
  { label: t('schedule.every_hour'), expr: '0 * * * *' }, { label: t('schedule.daily_9'), expr: '0 9 * * *' },
  { label: t('schedule.weekdays_925'), expr: '25 9 * * 1-5' }, { label: t('schedule.every_5min'), expr: '*/5 * * * *' },
  { label: t('schedule.weekly'), expr: '0 9 * * 1' },
]

async function loadTasks() {
  // TODO: move to store
  loading.value = true
  try { const { data: r } = await fetchWithCache<any>('schedule_tasks', () => (window as any).go?.main?.App?.ListScheduleTasks(), 5 * 60 * 1000); tasks.value = Array.isArray(r) ? r : [] }
  catch(e) { console.error('[Schedule] fetch:', e); tasks.value = [] }
  finally { loading.value = false }
}

function openNew() {
  editTask.value = { name: '', cron_expr: '0 9 * * *', workflow_id: '', timeout_sec: 1800, editingId: '' }
  showModal.value = true
}

async function saveTask() {
  // TODO: move to store
  const t = editTask.value
  await (window as any).go.main.App.SaveScheduleTask({
    id: t.editingId, name: t.name, cron_expr: t.cron_expr,
    workflow_id: t.workflow_id, timeout_sec: t.timeout_sec, enabled: true,
  })
  showModal.value = false; loadTasks()
}

async function toggleTask(id: string, enabled: boolean) {
  // TODO: move to store
  await (window as any).go.main.App.ToggleScheduleTask(id, !enabled); loadTasks()
}

async function deleteTask(id: string) {
  // TODO: move to store
  await (window as any).go.main.App.DeleteScheduleTask(id); loadTasks()
}

onMounted(loadTasks)
</script>

<template>
  <div class="schedule-panel">
    <div class="toolbar"><span class="task-count">{{ tasks.length }} {{ t('schedule.tasks') }}</span><button class="new-btn" @click="openNew">{{ t('schedule.new_task') }}</button></div>
    <div v-if="loading" class="empty-state">{{ t('common.loading') }}</div>
    <div v-else-if="!tasks.length" class="empty-state">{{ t('schedule.no_tasks') }}</div>
    <div v-else class="task-list">
      <div v-for="task in tasks" :key="task.id" class="task-row">
        <div class="task-info"><span class="task-name">{{ task.name }}</span><span class="task-cron">{{ task.cron_expr }}</span></div>
        <div class="task-actions">
          <button :class="['toggle-btn', task.enabled ? 'on' : 'off']" @click="toggleTask(task.id, task.enabled)">{{ task.enabled ? t('common.on') : t('common.off') }}</button>
          <button class="delete-btn" @click="deleteTask(task.id)">🗑</button>
        </div>
      </div>
    </div>
    <div v-if="showModal" class="modal-overlay" @click.self="showModal = false">
      <div class="modal"><h3 class="modal-title">{{ t('schedule.new_task') }}</h3>
        <div class="form-group"><label>{{ t('schedule.name') }}</label><input v-model="editTask.name" class="form-input" placeholder="Task name" /></div>
        <div class="form-group"><label>{{ t('schedule.cron') }}</label><input v-model="editTask.cron_expr" class="form-input mono" /></div>
        <div class="presets"><button v-for="p in cronPresets" :key="p.expr" class="preset-btn" @click="editTask.cron_expr = p.expr">{{ p.label }}</button></div>
        <div class="form-group"><label>{{ t('schedule.workflow') }}</label><input v-model="editTask.workflow_id" class="form-input" placeholder="wf-001" /></div>
        <div class="form-group"><label>{{ t('schedule.timeout') }}</label><input v-model.number="editTask.timeout_sec" type="number" class="form-input" /></div>
        <div class="modal-actions"><button class="cancel-btn" @click="showModal = false">{{ t('schedule.cancel') }}</button><button class="save-btn" @click="saveTask">{{ t('schedule.save') }}</button></div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.schedule-panel { padding: 12px; background: var(--color-bg-panel); height: 100%; overflow-y: auto; font-variant-numeric: tabular-nums; }
.toolbar { display: flex; justify-content: space-between; align-items: center; margin-bottom: 10px; }
.task-count { font-size: 12px; color: var(--color-text-tertiary); }
.new-btn { padding: 6px 14px; background: var(--color-accent-soft); border: none; border-radius: 4px; color: var(--color-accent); font-size: 12px; font-weight: 600; cursor: pointer; }
.empty-state { padding: 40px; text-align: center; color: var(--color-text-tertiary); font-size: 13px; }
.task-row { display: flex; justify-content: space-between; align-items: center; padding: 10px; background: var(--color-bg-subtle); border-radius: 4px; margin-bottom: 6px; }
.task-info { display: flex; flex-direction: column; gap: 3px; }
.task-name { font-size: 13px; font-weight: 600; color: var(--color-text-primary); }
.task-cron { font-size: 12px; font-family: monospace; color: var(--color-accent); }
.task-actions { display: flex; gap: 6px; align-items: center; }
.toggle-btn { padding: 4px 10px; border: none; border-radius: 4px; font-size: 11px; font-weight: 600; cursor: pointer; }
.toggle-btn.on { background: var(--color-down); color: var(--color-down); }
.toggle-btn.off { background: var(--color-bg-panel); color: var(--color-text-tertiary); }
.delete-btn { background: none; border: none; cursor: pointer; font-size: 14px; }
.modal-overlay { position: fixed; inset: 0; background: rgba(0,0,0,0.6); display: flex; align-items: center; justify-content: center; z-index: 100; }
.modal { background: var(--color-bg-panel); border: 1px solid var(--color-accent-soft); border-radius: 8px; padding: 20px; width: 420px; }
.modal-title { font-size: 16px; font-weight: 600; color: var(--color-text-primary); margin-bottom: 16px; }
.form-group { margin-bottom: 10px; }
.form-group label { display: block; font-size: 10px; color: var(--color-text-tertiary); text-transform: uppercase; margin-bottom: 4px; }
.form-input { width: 100%; padding: 6px 8px; background: var(--color-bg-input); border: 1px solid var(--color-accent-soft); border-radius: 4px; color: var(--color-text-primary); font-size: 13px; outline: none; box-sizing: border-box; }
.form-input:focus { border-color: var(--color-accent); }
.form-input.mono { font-family: monospace; }
.presets { display: flex; gap: 4px; flex-wrap: wrap; margin-bottom: 10px; }
.preset-btn { padding: 3px 8px; background: var(--color-bg-input); border: 1px solid var(--color-accent-soft); border-radius: 3px; color: var(--color-text-tertiary); font-size: 10px; cursor: pointer; }
.preset-btn:hover { border-color: var(--color-accent); color: var(--color-accent); }
.modal-actions { display: flex; gap: 8px; justify-content: flex-end; margin-top: 16px; }
.cancel-btn { padding: 6px 14px; background: var(--color-bg-input); border: 1px solid var(--color-accent-soft); border-radius: 4px; color: var(--color-text-tertiary); font-size: 12px; cursor: pointer; }
.save-btn { padding: 6px 14px; background: var(--color-accent-soft); border: none; border-radius: 4px; color: var(--color-accent); font-size: 12px; font-weight: 600; cursor: pointer; }
</style>
