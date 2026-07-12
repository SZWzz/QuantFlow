import { describe, it, expect, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { nextTick } from 'vue'
import { useSessionStore } from '../session'

describe('useSessionStore', () => {
  beforeEach(() => {
    localStorage.clear()
    setActivePinia(createPinia())
  })

  it('should default to dark/zh/terminal', () => {
    const store = useSessionStore()
    expect(store.ui.theme).toBe('dark')
    expect(store.ui.language).toBe('zh')
    expect(store.ui.mode).toBe('terminal')
  })

  it('should toggle mode between terminal and workflow', () => {
    const store = useSessionStore()
    store.toggleMode()
    expect(store.ui.mode).toBe('workflow')
    store.toggleMode()
    expect(store.ui.mode).toBe('terminal')
  })

  it('should set theme to light', () => {
    const store = useSessionStore()
    store.setTheme('light')
    expect(store.ui.theme).toBe('light')
  })

  it('should persist changes to localStorage', async () => {
    const store = useSessionStore()
    store.setTheme('light')
    await nextTick()
    const saved = JSON.parse(localStorage.getItem('quantflow-session')!)
    expect(saved.theme).toBe('light')
  })

  it('should load persisted session on init', () => {
    localStorage.setItem('quantflow-session', JSON.stringify({ theme: 'light', language: 'en', mode: 'workflow' }))
    setActivePinia(createPinia())
    const store = useSessionStore()
    expect(store.ui.theme).toBe('light')
    expect(store.ui.language).toBe('en')
    expect(store.ui.mode).toBe('workflow')
  })
})
