import { describe, it, expect, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import CorrelationPanel from '../CorrelationPanel.vue'

// Mock ResizeObserver (required by vue-echarts)
global.ResizeObserver = class {
  observe() {}
  unobserve() {}
  disconnect() {}
} as any

describe('CorrelationPanel', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
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
    expect(wrapper.text()).toContain('Correlation Matrix')
  })

  it('renders placeholder before compute', () => {
    const wrapper = mount(CorrelationPanel, {
      props: { panelId: 'test', params: {} },
      global: { stubs: { VChart: true } },
    })
    expect(wrapper.text()).toContain('Enter symbols and click Compute')
  })

  it('computes and hides placeholder', async () => {
    const wrapper = mount(CorrelationPanel, {
      props: { panelId: 'test', params: {} },
      global: { stubs: { VChart: true } },
    })
    // Before compute — shows placeholder
    expect(wrapper.text()).toContain('Enter symbols and click Compute')
    // Click compute button
    const btn = wrapper.find('button.compute-btn')
    expect(btn.exists()).toBe(true)
    await btn.trigger('click')
    // After compute — placeholder disappears, results rendered
    expect(wrapper.text()).not.toContain('Enter symbols and click Compute')
  })
})
