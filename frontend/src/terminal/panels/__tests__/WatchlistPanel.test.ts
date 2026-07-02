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
      global: { stubs: { VChart: true, echarts: true } },
    })
    expect(wrapper.exists()).toBe(true)
  })

  it('should show empty state when no symbols', () => {
    localStorage.setItem('quantflow-watchlist', JSON.stringify([]))
    const wrapper = mount(WatchlistPanel, {
      props: { panelId: 'test', params: {} },
      global: { stubs: { VChart: true, echarts: true } },
    })
    expect(wrapper.find('.empty-state').exists()).toBe(true)
  })

  it('should render symbols from localStorage', () => {
    localStorage.setItem('quantflow-watchlist', JSON.stringify(['600519', '000001']))
    const wrapper = mount(WatchlistPanel, {
      props: { panelId: 'test', params: {} },
      global: { stubs: { VChart: true, echarts: true } },
    })
    expect(wrapper.findAll('.table-row').length).toBe(2)
  })

  it('should remove symbol and dispatch event', () => {
    localStorage.setItem('quantflow-watchlist', JSON.stringify(['600519', '000001']))
    const wrapper = mount(WatchlistPanel, {
      props: { panelId: 'test', params: {} },
      global: { stubs: { VChart: true, echarts: true } },
    })
    const removeBtns = wrapper.findAll('.remove-btn')
    expect(removeBtns.length).toBe(2)
    removeBtns[0].trigger('click')
    expect(localStorage.getItem('quantflow-watchlist')).toBe(JSON.stringify(['000001']))
    expect(mockDispatchEvent).toHaveBeenCalled()
  })

  it('should respond to watchlist-changed event', async () => {
    const wrapper = mount(WatchlistPanel, {
      props: { panelId: 'test', params: {} },
      global: { stubs: { VChart: true, echarts: true } },
    })
    localStorage.setItem('quantflow-watchlist', JSON.stringify(['300750']))
    window.dispatchEvent(new CustomEvent('watchlist-changed'))
    await new Promise(r => setTimeout(r, 50))
    expect(wrapper.findAll('.table-row').length).toBe(1)
  })
})
