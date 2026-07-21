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

  it('groups symbols by market with group headers', async () => {
    const wrapper = mount(WatchlistPanel, {
      props: { panelId: 'test', params: {} },
      global: { stubs: { VChart: true, echarts: true, PanelHeader: true } },
    })
    await new Promise(r => setTimeout(r, 100))
    // Default symbols are all 6-digit CN A-shares → one CN group, expanded by default
    const headers = wrapper.findAll('.group-header')
    expect(headers.length).toBeGreaterThanOrEqual(1)
    expect(headers[0].find('.group-count').text()).toBe('8')
    // Rows of the expanded group are visible in the DOM
    expect(wrapper.findAll('.table-row').length).toBe(8)
  })

  it('clicking sortable th emits sort and reorders rows', async () => {
    // Distinct change_pct per symbol so asc/desc produce visibly different orders
    const pct: Record<string, number> = {
      '600519': 1.5, '000001': -2.5, '300750': 3.5, '601318': 0.5,
      '000858': -1.0, '600036': 2.0, '601166': -0.5, '600276': 4.0,
    }
    mockGetQuote.mockImplementation((_mkt: string, sym: string) =>
      Promise.resolve([{ symbol: sym, name: sym, last: 100, change: 1, change_pct: pct[sym] ?? 0 }]),
    )
    const wrapper = mount(WatchlistPanel, {
      props: { panelId: 'test', params: {} },
      global: { stubs: { VChart: true, echarts: true, PanelHeader: true } },
    })
    await new Promise(r => setTimeout(r, 100))

    const symbolOrder = () =>
      wrapper.findAll('.table-row').map(r => r.findAll('.td')[0].text())
    // Unsorted: original watchlist order
    expect(symbolOrder()).toEqual(['600519', '000001', '300750', '601318', '000858', '600036', '601166', '600276'])

    // Sortable ths are [现价(last), 涨跌幅(changePct), 成交额(turnover)]; click 涨跌幅
    const sortableThs = wrapper.findAll('.th.sortable')
    expect(sortableThs.length).toBe(3)
    await sortableThs[1].trigger('click')
    const asc = symbolOrder()
    expect(asc[0]).toBe('000001') // -2.5 lowest first
    expect(asc[7]).toBe('600276') // 4.0 highest last

    await wrapper.findAll('.th.sortable')[1].trigger('click')
    const desc = symbolOrder()
    expect(desc[0]).toBe('600276') // highest first
    expect(desc[7]).toBe('000001') // lowest last
    expect(desc).toEqual([...asc].reverse())
  })
})
