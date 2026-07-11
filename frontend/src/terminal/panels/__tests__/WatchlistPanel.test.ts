import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import WatchlistPanel from '../WatchlistPanel.vue'

// Mock window.go
const mockGetQuote = vi.fn().mockResolvedValue([{ symbol: '600519', name: '贵州茅台', last: 1889.5, change: 12.5, change_pct: 0.67 }])
const mockSearchSymbols = vi.fn().mockResolvedValue([{ symbol: '600519', name: '贵州茅台' }])
const mockDispatchEvent = vi.fn()

beforeEach(() => {
  setActivePinia(createPinia())
  localStorage.clear()
  vi.clearAllMocks()
  ;(window as any).go = { main: { App: { GetQuote: mockGetQuote, SearchSymbols: mockSearchSymbols } } }
  window.dispatchEvent = mockDispatchEvent
})

describe('WatchlistPanel', () => {
  it('should mount without crashing', () => {
    const wrapper = mount(WatchlistPanel, {
      props: { panelId: 'test', params: {} },
      global: { stubs: { VChart: true, echarts: true, PanelHeader: true } },
    })
    expect(wrapper.exists()).toBe(true)
  })

  it('should render default symbols', async () => {
    const wrapper = mount(WatchlistPanel, {
      props: { panelId: 'test', params: {} },
      global: { stubs: { VChart: true, echarts: true, PanelHeader: true } },
    })
    // Wait for async fetch to complete
    await new Promise(r => setTimeout(r, 100))
    expect(wrapper.findAll('.table-row').length).toBeGreaterThanOrEqual(8)
  })

  it('should render symbols from localStorage', async () => {
    localStorage.setItem('quantflow-watchlist', JSON.stringify(['600519', '000001']))
    const wrapper = mount(WatchlistPanel, {
      props: { panelId: 'test', params: {} },
      global: { stubs: { VChart: true, echarts: true, PanelHeader: true } },
    })
    // Wait for async fetch to complete
    await new Promise(r => setTimeout(r, 100))
    expect(wrapper.findAll('.table-row').length).toBe(2)
  })
})
