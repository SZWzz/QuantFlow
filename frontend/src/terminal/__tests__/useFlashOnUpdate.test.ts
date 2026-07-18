import { describe, it, expect, vi, afterEach } from 'vitest'
import { ref, nextTick } from 'vue'
import { useFlashOnUpdate } from '@/lib/composables/useFlashOnUpdate'

describe('useFlashOnUpdate', () => {
  afterEach(() => { vi.useRealTimers() })

  it('sets flash-up when value rises, then clears after duration', async () => {
    vi.useFakeTimers()
    const v = ref(10)
    const { flashClass } = useFlashOnUpdate(v)
    v.value = 11
    await nextTick()
    expect(flashClass.value).toBe('flash-up')
    vi.advanceTimersByTime(650)
    await nextTick()
    expect(flashClass.value).toBe('')
  })

  it('sets flash-down when value falls', async () => {
    vi.useFakeTimers()
    const v = ref(10)
    const { flashClass } = useFlashOnUpdate(v)
    v.value = 9.5
    await nextTick()
    expect(flashClass.value).toBe('flash-down')
  })

  it('does not flash on equal or null transitions', async () => {
    vi.useFakeTimers()
    const v = ref<number | null>(10)
    const { flashClass } = useFlashOnUpdate(v)
    v.value = 10
    await nextTick()
    expect(flashClass.value).toBe('')
    v.value = null
    await nextTick()
    expect(flashClass.value).toBe('')
  })
})
