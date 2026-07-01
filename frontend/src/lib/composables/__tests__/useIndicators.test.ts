import { describe, it, expect } from 'vitest'
import { createIndicatorCache } from '../useIndicators'

describe('createIndicatorCache', () => {
  it('returns cached result on second call', () => {
    const cache = createIndicatorCache()
    let count = 0
    const fn = () => { count++; return 42 }
    const a = cache.getCached('x', fn)
    const b = cache.getCached('x', fn)
    expect(a).toBe(42)
    expect(b).toBe(42)
    expect(count).toBe(1)
  })
  it('recomputes after clear', () => {
    const cache = createIndicatorCache()
    let count = 0
    const fn = () => { count++; return count }
    cache.getCached('x', fn)
    cache.clear()
    cache.getCached('x', fn)
    expect(count).toBe(2)
  })
})
