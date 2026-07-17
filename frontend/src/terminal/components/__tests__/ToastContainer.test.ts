import { describe, it, expect, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import ToastContainer from '../ToastContainer.vue'
import { useToast, clearAllToasts } from '@/lib/composables/useToast'

describe('ToastContainer', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    clearAllToasts()
  })

  it('should mount without crashing', () => {
    const wrapper = mount(ToastContainer)
    expect(wrapper.exists()).toBe(true)
  })

  it('should render toasts from composable', async () => {
    const toast = useToast()
    toast.info('Hello World', 'Test')
    const wrapper = mount(ToastContainer)
    expect(wrapper.findAll('[data-test="toast"]')).toHaveLength(1)
    expect(wrapper.text()).toContain('Hello World')
  })

  it('should show dismiss button on error toasts', async () => {
    const toast = useToast()
    toast.error('Fatal error')
    const wrapper = mount(ToastContainer)
    expect(wrapper.find('[data-test="toast-dismiss"]').exists()).toBe(true)
  })
})
