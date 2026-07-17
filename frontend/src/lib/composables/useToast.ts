import { ref } from 'vue'

export interface Toast {
  id: string
  type: 'info' | 'success' | 'warning' | 'error'
  title: string
  message: string
  duration: number
  action?: { label: string; onClick: () => void }
}

const DEDUP_WINDOW = 30000

// Module-level singleton — all callers share the same toast state
const toasts = ref<Toast[]>([])
const dedupMap = new Map<string, number>()
let toastIdCounter = 0

/** Clear all toasts — for testing only. */
export function clearAllToasts() {
  toasts.value = []
  dedupMap.clear()
  toastIdCounter = 0
}

export function useToast() {
  function addToast(t: Omit<Toast, 'id'>): string {
    // Dedup: merge same message within window
    const now = Date.now()
    const dedupKey = `${t.type}:${t.message}`
    const lastSeen = dedupMap.get(dedupKey)
    if (lastSeen && now - lastSeen < DEDUP_WINDOW) {
      const existing = toasts.value.find(toast => toast.message === t.message && toast.type === t.type)
      if (existing) return existing.id
    }
    dedupMap.set(dedupKey, now)

    const id = `toast-${++toastIdCounter}`
    const toast: Toast = { id, ...t }
    toasts.value.push(toast)

    if (t.duration > 0) {
      setTimeout(() => removeToast(id), t.duration)
    }

    return id
  }

  function removeToast(id: string) {
    toasts.value = toasts.value.filter(t => t.id !== id)
  }

  function success(message: string, title = ''): string {
    return addToast({ type: 'success', title: title || '成功', message, duration: 3000 })
  }

  function error(message: string, title = ''): string {
    return addToast({ type: 'error', title: title || '错误', message, duration: 0 })
  }

  function warning(message: string, title = ''): string {
    return addToast({ type: 'warning', title: title || '警告', message, duration: 5000 })
  }

  function info(message: string, title = ''): string {
    return addToast({ type: 'info', title: title || '提示', message, duration: 5000 })
  }

  return { toasts, addToast, removeToast, success, error, warning, info }
}
