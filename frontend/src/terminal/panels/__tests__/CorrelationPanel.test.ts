import { describe, it, expect, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import { mockWailsIPC } from '@/test-utils/mocks'
import CorrelationPanel from '../CorrelationPanel.vue'

// Mock ResizeObserver (required by vue-echarts)
globalThis.ResizeObserver = class {
  observe() {}
  unobserve() {}
  disconnect() {}
} as any

describe('CorrelationPanel', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    mockWailsIPC()
  })

  it('mounts without crashing', () => {
    const wrapper = mount(CorrelationPanel, {
      props: { panelId: 'test', params: {} },
      global: { stubs: { VChart: true } },
    })
    expect(wrapper.exists()).toBe(true)
  })

  it('renders title', () => {
    const wrapper = mount(CorrelationPanel, {
      props: { panelId: 'test', params: {} },
      global: { stubs: { VChart: true } },
    })
    expect(wrapper.find('.panel-header h3').text()).toContain('Correlation Matrix')
  })

  it('renders placeholder before compute', () => {
    const wrapper = mount(CorrelationPanel, {
      props: { panelId: 'test', params: {} },
      global: { stubs: { VChart: true } },
    })
    expect(wrapper.find('.placeholder-msg').exists()).toBe(true)
  })

  it('computes and hides placeholder', async () => {
    const wrapper = mount(CorrelationPanel, {
      props: { panelId: 'test', params: {} },
      global: { stubs: { VChart: true } },
    })
    // Before compute — shows placeholder
    expect(wrapper.find('.placeholder-msg').exists()).toBe(true)
    // Click compute button
    const btn = wrapper.find('button.compute-btn')
    expect(btn.exists()).toBe(true)
    await btn.trigger('click')
    // After compute — placeholder disappears, results rendered
    expect(wrapper.find('.placeholder-msg').exists()).toBe(false)
  })
})
