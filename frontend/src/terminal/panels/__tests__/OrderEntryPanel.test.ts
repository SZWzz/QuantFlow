import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import OrderEntryPanel from '../OrderEntryPanel.vue'
import { clearAllToasts, useToast } from '@/lib/composables/useToast'

function mockApp(overrides: Record<string, any> = {}) {
  const app = {
    GetQuote: vi.fn().mockResolvedValue([{ symbol: '600519', name: '贵州茅台', last: 1650 }]),
    SearchSymbols: vi.fn().mockResolvedValue([]),
    PlaceOrderWithStop: vi.fn().mockResolvedValue({ id: 'ord-1', status: 'filled' }),
    ...overrides,
  }
  ;(window as any).go = { main: { App: app } }
  return app
}

function mountPanel() {
  return mount(OrderEntryPanel, {
    props: { panelId: 'test-order-entry', params: {} },
    global: { stubs: { VChart: true, echarts: true } },
  })
}

describe('OrderEntryPanel', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    clearAllToasts()
    mockApp()
  })

  it('should mount without crashing', () => {
    const wrapper = mountPanel()
    expect(wrapper.exists()).toBe(true)
    expect(wrapper.html()).toBeTruthy()
  })

  it('starts with an empty symbol (no demo default) and disabled submit', () => {
    const wrapper = mountPanel()
    const input = wrapper.find('[data-testid="order-symbol-input"]')
    expect((input.element as HTMLInputElement).value).toBe('')
    expect(wrapper.find('[data-testid="order-place-btn"]').attributes('disabled')).toBeDefined()
  })

  it('quantity chips fill the quantity input', async () => {
    const wrapper = mountPanel()
    await wrapper.find('[data-testid="order-qty-chip-500"]').trigger('click')
    expect((wrapper.find('[data-testid="order-quantity-input"]').element as HTMLInputElement).value).toBe('500')
  })

  it('shows inline confirmation, then submits with await and toast', async () => {
    const app = mockApp()
    const wrapper = mountPanel()
    await wrapper.find('[data-testid="order-symbol-input"]').setValue('600519')
    await flushPromises() // quote auto-fill
    await wrapper.find('[data-testid="order-place-btn"]').trigger('click')

    const confirmView = wrapper.find('[data-testid="order-confirm-view"]')
    expect(confirmView.exists()).toBe(true)
    expect(confirmView.text()).toContain('600519')

    await wrapper.find('[data-testid="order-confirm-btn"]').trigger('click')
    await flushPromises()

    expect(app.PlaceOrderWithStop).toHaveBeenCalledWith('600519', 'buy', 'limit', 'paper', 100, 1650, 0)
    const { toasts } = useToast()
    expect(toasts.value.some(t => t.type === 'success')).toBe(true)
    // Back to edit state after success
    expect(wrapper.find('[data-testid="order-symbol-input"]').exists()).toBe(true)
  })

  it('forwards stopPrice for stop orders', async () => {
    const app = mockApp()
    const wrapper = mountPanel()
    await wrapper.find('[data-testid="order-symbol-input"]').setValue('600519')
    await flushPromises()
    await wrapper.find('[data-testid="order-type-select"]').setValue('stop')
    await wrapper.find('[data-testid="order-stop-price-input"]').setValue(1600)
    await wrapper.find('[data-testid="order-place-btn"]').trigger('click')
    expect(wrapper.find('[data-testid="order-confirm-view"]').text()).toContain('1600')
    await wrapper.find('[data-testid="order-confirm-btn"]').trigger('click')
    await flushPromises()
    expect(app.PlaceOrderWithStop).toHaveBeenCalledWith('600519', 'buy', 'stop', 'paper', 100, 1650, 1600)
  })

  it('keeps confirm state and shows error toast on failure', async () => {
    mockApp({ PlaceOrderWithStop: vi.fn().mockRejectedValue(new Error('撮合引擎不可用')) })
    const wrapper = mountPanel()
    await wrapper.find('[data-testid="order-symbol-input"]').setValue('600519')
    await flushPromises()
    await wrapper.find('[data-testid="order-place-btn"]').trigger('click')
    await wrapper.find('[data-testid="order-confirm-btn"]').trigger('click')
    await flushPromises()
    const { toasts } = useToast()
    expect(toasts.value.some(t => t.type === 'error' && t.message.includes('撮合引擎不可用'))).toBe(true)
    expect(wrapper.find('[data-testid="order-confirm-view"]').exists()).toBe(true)
  })

  it('cancel returns to the form without calling the backend', async () => {
    const app = mockApp()
    const wrapper = mountPanel()
    await wrapper.find('[data-testid="order-symbol-input"]').setValue('600519')
    await flushPromises()
    await wrapper.find('[data-testid="order-place-btn"]').trigger('click')
    await wrapper.find('[data-testid="order-cancel-btn"]').trigger('click')
    expect(wrapper.find('[data-testid="order-symbol-input"]').exists()).toBe(true)
    expect(app.PlaceOrderWithStop).not.toHaveBeenCalled()
  })
})
