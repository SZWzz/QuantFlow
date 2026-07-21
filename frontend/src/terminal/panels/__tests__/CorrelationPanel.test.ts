import { describe, it, expect, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import { mockWailsIPC } from '@/test-utils/mocks'
import CorrelationPanel from '../CorrelationPanel.vue'

globalThis.ResizeObserver = class { observe() {} unobserve() {} disconnect() {} } as any

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
    expect(wrapper.find('.panel-title').text()).toContain('相关性分析')
  })

  it('renders placeholder before compute', () => {
    const wrapper = mount(CorrelationPanel, {
      props: { panelId: 'test', params: {} },
      global: { stubs: { VChart: true } },
    })
    expect(wrapper.find('.empty-state').exists()).toBe(true)
  })

  it('computes and hides placeholder', async () => {
    const wrapper = mount(CorrelationPanel, {
      props: { panelId: 'test', params: {} },
      global: { stubs: { VChart: true } },
    })
    expect(wrapper.find('.empty-state').exists()).toBe(true)
    const btn = wrapper.find('.btn-primary')
    expect(btn.exists()).toBe(true)
    await btn.trigger('click')
    expect(wrapper.find('.empty-state').exists()).toBe(false)
  })
})
