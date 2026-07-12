import { describe, it, expect, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useDataStore } from '../data'

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
