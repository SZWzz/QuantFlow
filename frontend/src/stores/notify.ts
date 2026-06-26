import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

export interface Notification {
  id: number; level: string; title: string; body: string; metadata: string; is_read: boolean; created_at: string
}

export const useNotifyStore = defineStore('notify', () => {
  const notifications = ref<Notification[]>([])
  const unreadCount = ref(0)
  const levelFilter = ref<string>('all')
  const error = ref<string | null>(null)

  async function fetchNotifications(limit = 50, offset = 0) {
    error.value = null
    try {
      const app = (window as any).go?.main?.App
      if (!app) { error.value = 'Bridge unavailable'; return }
      const result = await app.GetNotifications(limit, offset)
      if (result) {
        notifications.value = result
        unreadCount.value = result.filter((n: Notification) => !n.is_read).length
      }
    } catch (e) { error.value = String(e) }
  }

  async function markRead(id: number) {
    error.value = null
    try {
      const app = (window as any).go?.main?.App
      if (!app) { error.value = 'Bridge unavailable'; return }
      await app.MarkNotificationRead(id)
      const n = notifications.value.find(x => x.id === id)
      if (n && !n.is_read) { n.is_read = true; unreadCount.value-- }
    } catch (e) { error.value = String(e) }
  }

  async function markAllRead() {
    for (const n of notifications.value) { if (!n.is_read) { await markRead(n.id) } }
    unreadCount.value = 0
  }

  const filteredNotifications = computed(() => {
    if (levelFilter.value === 'all') return notifications.value
    return notifications.value.filter(n => n.level === levelFilter.value)
  })
  function setFilter(level: string) { levelFilter.value = level }
  return { notifications, unreadCount, levelFilter, filteredNotifications, error, fetchNotifications, markRead, markAllRead, setFilter }
})
