import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

export interface Notification {
  id: number; level: string; title: string; body: string; metadata: string; is_read: boolean; created_at: string
}

export const useNotifyStore = defineStore('notify', () => {
  const notifications = ref<Notification[]>([])
  const unreadCount = ref(0)
  const levelFilter = ref<string>('all')

  async function fetchNotifications(limit = 50, offset = 0) {
    try {
      const result = await (window as any).go.main.App.GetNotifications(limit, offset)
      if (result) {
        notifications.value = result
        unreadCount.value = result.filter((n: Notification) => !n.is_read).length
      }
    } catch (e) { console.warn('GetNotifications not available:', e) }
  }

  async function markRead(id: number) {
    try {
      await (window as any).go.main.App.MarkNotificationRead(id)
      const n = notifications.value.find(x => x.id === id)
      if (n && !n.is_read) { n.is_read = true; unreadCount.value-- }
    } catch (e) { console.warn('MarkNotificationRead not available:', e) }
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
  return { notifications, unreadCount, levelFilter, filteredNotifications, fetchNotifications, markRead, markAllRead, setFilter }
})
