import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useDataStore } from '../data'

// Mock the wails wrappers used by the data store
vi.mock('@/lib/wails', () => ({
  fetchMarketOverview: vi.fn().mockResolvedValue({
    indices: [
      { code: '000001', name: '上证指数', price: 3200, change_pct: 0.5, prev_close: 3184 },
      { code: '399001', name: '深证成指', price: 11000, change_pct: -0.2, prev_close: 11022 },
    ],
    breadth: { advancers: 1500, decliners: 800, unchanged: 200 },
    sentiment: { limit_up: 30, limit_down: 5, northbound_flow: 2.5, total_volume: 1200 },
  }),
  fetchMinuteLine: vi.fn(),
  GetIndustryRanks: vi.fn().mockResolvedValue([
    { name: '银行', change_pct: 1.2 },
    { name: '科技', change_pct: -0.5 },
    { name: '消费', change_pct: 0.8 },
  ]),
}))

describe('useDataStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('should start with empty quotes map', () => {
    const store = useDataStore()
    expect(store.quotes.size).toBe(0)
  })

  it('should update and retrieve quote', () => {
    const store = useDataStore()
    const snap = { symbol: '000001', last: 10.5, bid: 10.4, ask: 10.6, volume: 1000, change: 0.1, changePct: 1.0, timestamp: Date.now() }
    store.updateQuote('000001', snap)
    expect(store.getQuote('000001')).toEqual(snap)
  })

  it('should return undefined for missing quote', () => {
    const store = useDataStore()
    expect(store.getQuote('nope')).toBeUndefined()
  })

  it('should set and get OHLCV cache', () => {
    const store = useDataStore()
    const bars = [{ date: '2024-01-01', open: 10, high: 11, low: 9, close: 10.5, volume: 5000 }]
    store.setOHLCV('key1', bars)
    expect(store.getOHLCV('key1')).toEqual(bars)
  })

  it('should toggle offline mode', () => {
    const store = useDataStore()
    expect(store.isOffline).toBe(false)
    store.toggleOffline()
    expect(store.isOffline).toBe(true)
  })

  it('should start with null marketOverview', () => {
    const store = useDataStore()
    expect(store.marketOverview).toBeNull()
  })

  it('should fetch market overview with mock data', async () => {
    const store = useDataStore()
    await store.fetchMarketOverview()
    expect(store.marketOverview).not.toBeNull()
    expect(store.marketOverview!.indices.length).toBeGreaterThan(0)
    expect(store.marketOverview!.breadth.advancers).toBeGreaterThan(0)
    expect(store.marketOverview!.sectors.length).toBeGreaterThan(0)
  })
})
