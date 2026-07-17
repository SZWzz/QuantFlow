import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import { useToast, clearAllToasts } from '@/lib/composables/useToast'
import { useTerminalStore } from '@/stores/terminal'
import ToastContainer from '../ToastContainer.vue'

describe('Error Visibility Integration', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    clearAllToasts()
  })

  it('error from store triggers toast via composable', () => {
    const toast = useToast()
    const id = toast.error('数据源超时: Tencent')
    expect(toast.toasts.value).toHaveLength(1)
    expect(toast.toasts.value[0].type).toBe('error')
    expect(toast.toasts.value[0].message).toContain('Tencent')
  })

  it('success toasts auto-dismiss', () => {
    vi.useFakeTimers()
    const toast = useToast()
    toast.success('回测完成')
    expect(toast.toasts.value).toHaveLength(1)
    vi.advanceTimersByTime(3000)
    expect(toast.toasts.value).toHaveLength(0)
    vi.useRealTimers()
  })

  it('error toasts do not auto-dismiss', () => {
    vi.useFakeTimers()
    const toast = useToast()
    toast.error('Python sidecar 断连')
    expect(toast.toasts.value).toHaveLength(1)
    vi.advanceTimersByTime(60000)
    expect(toast.toasts.value).toHaveLength(1)
    vi.useRealTimers()
  })

  it('ToastContainer renders toasts from shared composable', () => {
    const toast = useToast()
    toast.warning('API Key 验证失败')

    const wrapper = mount(ToastContainer)
    expect(wrapper.findAll('[data-test="toast"]')).toHaveLength(1)
    expect(wrapper.text()).toContain('API Key')
  })

  it('connection status updates in store', () => {
    const store = useTerminalStore()
    store.updateConnectionStatus({
      markets: { 'A股': '实时', '港股': '延迟' },
      python: '运行中',
    })
    expect(store.connectionStatus.markets['A股']).toBe('实时')
    expect(store.connectionStatus.python).toBe('运行中')
  })

  it('status bar shows connection status from store', () => {
    const store = useTerminalStore()
    store.updateConnectionStatus({
      markets: { 'A股': '2 适配器' },
      brokers: { 'Paper Trading': '已连接' },
      python: '运行中',
    })
    expect(store.connectionStatus.markets['A股']).toBe('2 适配器')
    expect(store.connectionStatus.brokers['Paper Trading']).toBe('已连接')
  })
})
