import { describe, it, expect, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useNotifyStore } from '../notify'

describe('useNotifyStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('should start with empty notifications', () => {
    const store = useNotifyStore()
    expect(store.notifications).toHaveLength(0)
    expect(store.unreadCount).toBe(0)
  })

  it('should set filter level', () => {
    const store = useNotifyStore()
    store.setFilter('error')
    expect(store.levelFilter).toBe('error')
  })

  it('should filter notifications by level', () => {
    const store = useNotifyStore()
    store.notifications.push(
      { id: 1, level: 'info', title: 't1', body: 'b1', metadata: '{}', is_read: false, created_at: '2024-01-01' },
      { id: 2, level: 'error', title: 't2', body: 'b2', metadata: '{}', is_read: false, created_at: '2024-01-02' },
    )
    store.setFilter('error')
    expect(store.filteredNotifications).toHaveLength(1)
    expect(store.filteredNotifications[0].level).toBe('error')
  })
})
