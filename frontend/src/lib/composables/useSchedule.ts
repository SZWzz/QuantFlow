import { ref } from 'vue'

interface ScheduleTask {
  id: string
  name: string
  cron_expr: string
  workflow_id: string
  enabled: boolean
  timeout_sec: number
  last_run_status: string
}

const tasks = ref<ScheduleTask[]>([])
const loading = ref(false)

async function fetchScheduleTasks() {
  loading.value = true
  try {
    const result = await (window as any).go?.main?.App?.ListScheduleTasks()
    tasks.value = Array.isArray(result) ? result : []
  } catch (e) {
    console.error('[Schedule] fetch:', e)
    tasks.value = []
  } finally {
    loading.value = false
  }
}

async function saveScheduleTask(task: {
  id?: string; name: string; cron_expr: string
  workflow_id: string; timeout_sec: number; enabled?: boolean
}) {
  await (window as any).go?.main?.App?.SaveScheduleTask({
    ...task,
    enabled: task.enabled ?? true,
  })
}

async function toggleScheduleTask(id: string, enabled: boolean) {
  await (window as any).go?.main?.App?.ToggleScheduleTask(id, !enabled)
}

async function deleteScheduleTask(id: string) {
  await (window as any).go?.main?.App?.DeleteScheduleTask(id)
}

export function useSchedule() {
  return {
    tasks,
    loading,
    fetchScheduleTasks,
    saveScheduleTask,
    toggleScheduleTask,
    deleteScheduleTask,
  }
}
