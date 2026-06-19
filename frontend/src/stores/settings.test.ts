import { describe, it, expect, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useSettingsStore } from './settings'

describe('useSettingsStore', () => {
  beforeEach(() => {
    localStorage.clear()
    setActivePinia(createPinia())
  })

  it('should initialize with defaults', () => {
    const store = useSettingsStore()
    expect(store.settings.language).toBe('zh')
    expect(store.settings.defaultBroker).toBe('paper')
    expect(store.settings.defaultQty).toBe(100)
  })

  it('should update a setting and persist to localStorage', () => {
    const store = useSettingsStore()
    store.update('defaultQty', 200)
    expect(store.settings.defaultQty).toBe(200)
    const saved = JSON.parse(localStorage.getItem('quantflow-settings')!)
    expect(saved.defaultQty).toBe(200)
  })

  it('should reset to defaults', () => {
    const store = useSettingsStore()
    store.update('language', 'en')
    store.reset()
    expect(store.settings.language).toBe('zh')
  })

  it('should load persisted settings on init', () => {
    localStorage.setItem('quantflow-settings', JSON.stringify({ language: 'en', defaultQty: 50 }))
    setActivePinia(createPinia())
    const store = useSettingsStore()
    expect(store.settings.language).toBe('en')
    expect(store.settings.defaultQty).toBe(50)
  })

  it('should handle corrupted localStorage gracefully', () => {
    localStorage.setItem('quantflow-settings', 'not-json')
    setActivePinia(createPinia())
    const store = useSettingsStore()
    expect(store.settings.language).toBe('zh')
  })
})
