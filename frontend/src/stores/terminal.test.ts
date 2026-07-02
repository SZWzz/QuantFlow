import { describe, it, expect, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useTerminalStore } from './terminal'

describe('useTerminalStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('should start with empty panels', () => {
    const store = useTerminalStore()
    expect(store.activePanels).toHaveLength(0)
  })

  it('should open and close panel', () => {
    const store = useTerminalStore()
    const id = store.openPanel('watchlist')
    expect(store.activePanels).toHaveLength(1)
    expect(store.activePanels[0].panelId).toBe('watchlist')
    store.closePanel(id)
    expect(store.activePanels).toHaveLength(0)
  })

  it('should pass params when opening panel', () => {
    const store = useTerminalStore()
    store.openPanel('candlestick', { symbol: '000001' })
    expect(store.activePanels[0].params).toEqual({ symbol: '000001' })
  })

  it('should manage command history with max 20 entries', () => {
    const store = useTerminalStore()
    store.addCommand('cmd1')
    expect(store.commandHistory[0]).toBe('cmd1')
    for (let i = 2; i <= 25; i++) store.addCommand(`cmd${i}`)
    expect(store.commandHistory).toHaveLength(20)
  })

  it('should toggle focus mode', () => {
    const store = useTerminalStore()
    expect(store.focusMode).toBe(false)
    store.toggleFocusMode()
    expect(store.focusMode).toBe(true)
  })
})
