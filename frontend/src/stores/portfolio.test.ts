import { describe, it, expect, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { usePortfolioStore } from './portfolio'

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
    // Access internal timer via any cast
    const storeAny = store as any
    storeAny.startAutoRefresh()
    expect(storeAny.timer).not.toBeNull()
    storeAny.stopAutoRefresh()
    expect(storeAny.timer).toBeNull()
  })
})
