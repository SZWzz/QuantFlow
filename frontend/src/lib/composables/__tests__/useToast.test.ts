import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest'
import { useToast, clearAllToasts } from '../useToast'

describe('useToast', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    clearAllToasts()
  })
  afterEach(() => {
    vi.useRealTimers()
  })

  it('should start with empty toasts', () => {
    const toast = useToast()
    expect(toast.toasts.value).toHaveLength(0)
  })

  it('should add a toast', () => {
    const toast = useToast()
    toast.addToast({ type: 'info', title: 'Test', message: 'Hello', duration: 5000 })
    expect(toast.toasts.value).toHaveLength(1)
    expect(toast.toasts.value[0].title).toBe('Test')
    expect(toast.toasts.value[0].message).toBe('Hello')
  })

  it('should remove a toast by id', () => {
    const toast = useToast()
    const id = toast.addToast({ type: 'info', title: 'T', message: 'M', duration: 5000 })
    expect(toast.toasts.value).toHaveLength(1)
    toast.removeToast(id)
    expect(toast.toasts.value).toHaveLength(0)
  })

  it('should auto-remove toast after duration', () => {
    const toast = useToast()
    toast.addToast({ type: 'success', title: 'Done', message: 'OK', duration: 3000 })
    expect(toast.toasts.value).toHaveLength(1)
    vi.advanceTimersByTime(3000)
    expect(toast.toasts.value).toHaveLength(0)
  })

  it('should not auto-remove toast with duration 0', () => {
    const toast = useToast()
    toast.addToast({ type: 'error', title: 'Fail', message: 'Err', duration: 0 })
    expect(toast.toasts.value).toHaveLength(1)
    vi.advanceTimersByTime(10000)
    expect(toast.toasts.value).toHaveLength(1)
  })

  it('should provide shorthand methods', () => {
    const toast = useToast()
    const id1 = toast.success('Success!')
    const id2 = toast.error('Error!')
    const id3 = toast.warning('Warn!')
    const id4 = toast.info('Info!')
    expect(toast.toasts.value).toHaveLength(4)
    expect(toast.toasts.value[0].type).toBe('success')
    expect(toast.toasts.value[1].type).toBe('error')
    expect(toast.toasts.value[2].type).toBe('warning')
    expect(toast.toasts.value[3].type).toBe('info')
  })

  it('should merge duplicate errors within 30s', () => {
    const toast = useToast()
    toast.error('Connection lost')
    expect(toast.toasts.value).toHaveLength(1)
    toast.error('Connection lost')
    // Should still be 1 (merged)
    expect(toast.toasts.value).toHaveLength(1)
  })
})
