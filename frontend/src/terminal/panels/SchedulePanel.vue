<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useSchedule } from '@/lib/composables/useSchedule'
import { PanelHeader, EmptyState, LoadingState } from '@/terminal/components/panel'

defineProps<{ panelId: string; params?: Record<string, any> }>()

const { t } = useI18n()
const { tasks, loading, fetchScheduleTasks, saveScheduleTask, toggleScheduleTask, deleteScheduleTask } = useSchedule()
const showModal = ref(false)
const editTask = ref({ name: '', cron_expr: '0 9 * * *', workflow_id: '', timeout_sec: 1800, editingId: '' })
const cronPresets = [
  { label: t('schedule.every_hour'), expr: '0 * * * *' }, { label: t('schedule.daily_9'), expr: '0 9 * * *' },
  { label: t('schedule.weekdays_925'), expr: '25 9 * * 1-5' }, { label: t('schedule.every_5min'), expr: '*/5 * * * *' },
  { label: t('schedule.weekly'), expr: '0 9 * * 1' },
]

function openNew() {
  editTask.value = { name: '', cron_expr: '0 9 * * *', workflow_id: '', timeout_sec: 1800, editingId: '' }
  showModal.value = true
}

async function saveTask() {
  const t = editTask.value
  await saveScheduleTask({
    id: t.editingId, name: t.name, cron_expr: t.cron_expr,
    workflow_id: t.workflow_id, timeout_sec: t.timeout_sec, enabled: true,
  })
  showModal.value = false; fetchScheduleTasks()
}

async function toggleTask(id: string, enabled: boolean) {
  await toggleScheduleTask(id, enabled); fetchScheduleTasks()
}

async function deleteTask(id: string) {
  await deleteScheduleTask(id); fetchScheduleTasks()
}

onMounted(fetchScheduleTasks)
</script>

<template>
  <div class="schedule-panel">
    <PanelHeader
      :title="t('schedule.title')"
      :subtitle="`${tasks.length} ${t('schedule.tasks')}`"
    >
      <template #controls>
        <button class="btn btn-sm btn-primary" @click="openNew">{{ t('schedule.new_task') }}</button>
      </template>
    </PanelHeader>

    <LoadingState v-if="loading" type="card" :rows="3" />
    <EmptyState v-else-if="!tasks.length" :title="t('schedule.no_tasks')" />
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
.schedule-panel { height: 100%; display: flex; flex-direction: column; overflow: hidden; font-variant-numeric: tabular-nums; }
.task-list { flex: 1; min-height: 0; overflow-y: auto; padding: var(--space-md) var(--panel-padding); }

.task-row {
  display: flex; justify-content: space-between; align-items: center;
  padding: var(--space-sm) var(--space-md); background: var(--color-bg-subtle);
  border-radius: var(--radius-sm); margin-bottom: var(--space-sm);
}
.task-info { display: flex; flex-direction: column; gap: var(--space-xs); }
.task-name { font-size: var(--font-sm); font-weight: 600; color: var(--color-text-primary); }
.task-cron { font-size: var(--font-xs); font-family: var(--font-mono); color: var(--color-accent); }
.task-actions { display: flex; gap: var(--space-sm); align-items: center; }
.toggle-btn { padding: var(--space-xs) var(--space-sm); border: none; border-radius: var(--radius-sm); font-size: var(--font-xs); font-weight: 600; cursor: pointer; }
.toggle-btn.on { background: var(--color-down); color: var(--color-text-inverse); }
.toggle-btn.off { background: var(--color-bg-elevated); color: var(--color-text-tertiary); }
.delete-btn { background: none; border: none; cursor: pointer; font-size: var(--font-base); }

/* 浮层弹窗（floating layer） */
.modal-overlay { position: fixed; inset: 0; background: var(--color-overlay); display: flex; align-items: center; justify-content: center; z-index: var(--z-overlay); }
.modal { background: var(--color-bg-panel); border: 1px solid var(--color-border); border-radius: var(--radius-lg); padding: var(--space-lg); width: 420px; }
.modal-title { font-size: var(--font-lg); font-weight: 600; color: var(--color-text-primary); margin-bottom: var(--space-md); }
.form-group { margin-bottom: var(--space-sm); }
.form-group label { display: block; font-size: var(--font-xs); color: var(--color-text-tertiary); text-transform: uppercase; margin-bottom: var(--space-xs); }
.form-input { width: 100%; padding: var(--space-xs) var(--space-sm); background: var(--color-bg-input); border: 1px solid var(--color-border); border-radius: var(--radius-sm); color: var(--color-text-primary); font-size: var(--font-sm); outline: none; box-sizing: border-box; }
.form-input:focus { border-color: var(--color-accent); }
.form-input.mono { font-family: var(--font-mono); }
.presets { display: flex; gap: var(--space-xs); flex-wrap: wrap; margin-bottom: var(--space-sm); }
.preset-btn { padding: var(--space-xs) var(--space-sm); background: var(--color-bg-input); border: 1px solid var(--color-border); border-radius: var(--radius-sm); color: var(--color-text-tertiary); font-size: var(--font-xs); cursor: pointer; }
.preset-btn:hover { border-color: var(--color-accent); color: var(--color-accent); }
.modal-actions { display: flex; gap: var(--space-sm); justify-content: flex-end; margin-top: var(--space-md); }
.cancel-btn { padding: var(--space-xs) var(--space-md); background: var(--color-bg-input); border: 1px solid var(--color-border); border-radius: var(--radius-sm); color: var(--color-text-tertiary); font-size: var(--font-xs); cursor: pointer; }
.save-btn { padding: var(--space-xs) var(--space-md); background: var(--color-accent-soft); border: none; border-radius: var(--radius-sm); color: var(--color-accent); font-size: var(--font-xs); font-weight: 600; cursor: pointer; }
</style>
