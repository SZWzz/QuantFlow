<script setup lang="ts">
import { ref } from 'vue'

defineProps<{ panelId: string; params?: Record<string, any> }>()

interface ScheduleTask { id: string; name: string; cron_expr: string; workflow_id: string; enabled: boolean; last_run_status: string }

const tasks = ref<ScheduleTask[]>([
  { id: '1', name: 'Morning Scan', cron_expr: '25 9 * * 1-5', workflow_id: 'wf-001', enabled: true, last_run_status: 'success' },
  { id: '2', name: 'EOD Risk Check', cron_expr: '0 15 * * 1-5', workflow_id: 'wf-002', enabled: false, last_run_status: 'error' },
])

const showModal = ref(false)
const editTask = ref({ name: '', cron_expr: '0 9 * * *', workflow_id: '', timeout_sec: 1800 })

const cronPresets = [
  { label: 'Every Hour', expr: '0 * * * *' },
  { label: 'Daily 9:00', expr: '0 9 * * *' },
  { label: 'Weekdays 9:25', expr: '25 9 * * 1-5' },
  { label: 'Every 5min', expr: '*/5 * * * *' },
  { label: 'Weekly Mon', expr: '0 9 * * 1' },
]

function openNew() { editTask.value = { name: '', cron_expr: '0 9 * * *', workflow_id: '', timeout_sec: 1800 }; showModal.value = true }
function saveTask() { showModal.value = false }
function toggleTask(id: string) { const t = tasks.value.find(x => x.id === id); if (t) t.enabled = !t.enabled }
function deleteTask(id: string) { tasks.value = tasks.value.filter(x => x.id !== id) }
</script>

<template>
  <div class="schedule-panel">
    <div class="toolbar"><span class="task-count">{{ tasks.length }} tasks</span><button class="new-btn" @click="openNew">+ New Task</button></div>
    <div class="task-list">
      <div v-for="task in tasks" :key="task.id" class="task-row">
        <div class="task-info">
          <span class="task-name">{{ task.name }}</span>
          <span class="task-cron">{{ task.cron_expr }}</span>
          <span :class="['task-status', task.last_run_status]">{{ task.last_run_status === 'success' ? '✅' : '❌' }}</span>
        </div>
        <div class="task-actions">
          <button :class="['toggle-btn', task.enabled ? 'on' : 'off']" @click="toggleTask(task.id)">{{ task.enabled ? 'ON' : 'OFF' }}</button>
          <button class="delete-btn" @click="deleteTask(task.id)">🗑</button>
        </div>
      </div>
    </div>
    <div v-if="showModal" class="modal-overlay" @click.self="showModal = false">
      <div class="modal">
        <h3 class="modal-title">New Schedule Task</h3>
        <div class="form-group"><label>Name</label><input v-model="editTask.name" class="form-input" placeholder="Task name" /></div>
        <div class="form-group"><label>Cron Expression</label><input v-model="editTask.cron_expr" class="form-input mono" placeholder="* * * * *" /></div>
        <div class="presets"><button v-for="p in cronPresets" :key="p.expr" class="preset-btn" @click="editTask.cron_expr = p.expr">{{ p.label }}</button></div>
        <div class="form-group"><label>Workflow ID</label><input v-model="editTask.workflow_id" class="form-input" placeholder="wf-001" /></div>
        <div class="form-group"><label>Timeout (sec)</label><input v-model.number="editTask.timeout_sec" type="number" class="form-input" /></div>
        <div class="modal-actions">
          <button class="cancel-btn" @click="showModal = false">Cancel</button>
          <button class="save-btn" @click="saveTask">Save</button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.schedule-panel { padding: 12px; background: #1a1a2e; height: 100%; overflow-y: auto; font-variant-numeric: tabular-nums; }
.toolbar { display: flex; justify-content: space-between; align-items: center; margin-bottom: 10px; }
.task-count { font-size: 12px; color: #5a6380; }
.new-btn { padding: 6px 14px; background: #1a3a5c; border: none; border-radius: 4px; color: #58a6ff; font-size: 12px; font-weight: 600; cursor: pointer; }
.task-row { display: flex; justify-content: space-between; align-items: center; padding: 10px; background: #16213e; border-radius: 4px; margin-bottom: 6px; }
.task-info { display: flex; flex-direction: column; gap: 3px; }
.task-name { font-size: 13px; font-weight: 600; color: #e0e0e0; }
.task-cron { font-size: 12px; font-family: monospace; color: #58a6ff; }
.task-actions { display: flex; gap: 6px; align-items: center; }
.toggle-btn { padding: 4px 10px; border: none; border-radius: 4px; font-size: 11px; font-weight: 600; cursor: pointer; }
.toggle-btn.on { background: #0a3d1a; color: #3fb950; }
.toggle-btn.off { background: #1a1a2e; color: #5a6380; }
.delete-btn { background: none; border: none; cursor: pointer; font-size: 14px; }
.modal-overlay { position: fixed; inset: 0; background: rgba(0,0,0,0.6); display: flex; align-items: center; justify-content: center; z-index: 100; }
.modal { background: #1a1a2e; border: 1px solid #1a3a5c; border-radius: 8px; padding: 20px; width: 420px; }
.modal-title { font-size: 16px; font-weight: 600; color: #e0e0e0; margin-bottom: 16px; }
.form-group { margin-bottom: 10px; }
.form-group label { display: block; font-size: 10px; color: #5a6380; text-transform: uppercase; margin-bottom: 4px; }
.form-input { width: 100%; padding: 6px 8px; background: #0f2137; border: 1px solid #1a3a5c; border-radius: 4px; color: #c9d1d9; font-size: 13px; outline: none; box-sizing: border-box; }
.form-input:focus { border-color: #58a6ff; }
.form-input.mono { font-family: monospace; }
.presets { display: flex; gap: 4px; flex-wrap: wrap; margin-bottom: 10px; }
.preset-btn { padding: 3px 8px; background: #0f2137; border: 1px solid #1a3a5c; border-radius: 3px; color: #5a6380; font-size: 10px; cursor: pointer; }
.preset-btn:hover { border-color: #58a6ff; color: #58a6ff; }
.modal-actions { display: flex; gap: 8px; justify-content: flex-end; margin-top: 16px; }
.cancel-btn { padding: 6px 14px; background: #0f2137; border: 1px solid #1a3a5c; border-radius: 4px; color: #5a6380; font-size: 12px; cursor: pointer; }
.save-btn { padding: 6px 14px; background: #1a3a5c; border: none; border-radius: 4px; color: #58a6ff; font-size: 12px; font-weight: 600; cursor: pointer; }
</style>
