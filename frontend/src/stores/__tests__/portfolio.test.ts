import { describe, it, expect, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { usePortfolioStore } from '../portfolio'

describe('usePortfolioStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('should start with null summary and empty positions', () => {
    const store = usePortfolioStore()
    expect(store.summary).toBeNull()
    expect(store.positions).toHaveLength(0)
  })

  it('should start and stop auto refresh', () => {
    const store = usePortfolioStore()
    const storeAny = store as any
    storeAny.startAutoRefresh()
    expect(storeAny.timer).not.toBeNull()
    storeAny.stopAutoRefresh()
    expect(storeAny.timer).toBeNull()
  })

  it('should fetch mock orders', async () => {
    const store = usePortfolioStore()
    expect(store.orders).toHaveLength(0)
    await store.fetchOrders()
    expect(store.orders.length).toBeGreaterThan(0)
    expect(store.orders[0]).toHaveProperty('order_id')
    expect(store.orders[0]).toHaveProperty('status')
  })

  it('should fetch mock trades', async () => {
    const store = usePortfolioStore()
    await store.fetchTrades()
    expect(store.trades.length).toBeGreaterThan(0)
    expect(store.trades[0]).toHaveProperty('trade_id')
  })

  it('should fetch mock equity curve', async () => {
    const store = usePortfolioStore()
    await store.fetchEquityCurve()
    expect(store.equityCurve).not.toBeNull()
    expect(store.equityCurve!.length).toBeGreaterThan(200)
    expect(store.equityCurve![0]).toHaveProperty('nav')
    expect(store.equityCurve![0]).toHaveProperty('benchmark')
  })

  it('should cancel order optimistically', async () => {
    const store = usePortfolioStore()
    await store.fetchOrders()
    const firstOrder = store.orders[0]
    await store.cancelOrder(firstOrder.order_id)
    expect(store.orders[0].status).toBe('cancelled')
  })
})
