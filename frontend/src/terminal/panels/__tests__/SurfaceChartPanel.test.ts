import { describe, it, expect, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import SurfaceChartPanel from '../SurfaceChartPanel.vue'

describe('SurfaceChartPanel', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('mounts without crashing', () => {
    const wrapper = mount(SurfaceChartPanel, {
      props: { panelId: 'test', params: {} },
      global: { stubs: { VChart: true } },
    })
    expect(wrapper.exists()).toBe(true)
  })

  it('renders title', () => {
    const wrapper = mount(SurfaceChartPanel, {
      props: { panelId: 'test', params: {} },
      global: { stubs: { VChart: true } },
    })
    expect(wrapper.text()).toContain('Volatility Surface')
  })

  it('has symbol input', () => {
    const wrapper = mount(SurfaceChartPanel, {
      props: { panelId: 'test', params: {} },
      global: { stubs: { VChart: true } },
    })
    const input = wrapper.find('.symbol-input')
    expect(input.exists()).toBe(true)
  })

  it('has maturity slice selector', () => {
    const wrapper = mount(SurfaceChartPanel, {
      props: { panelId: 'test', params: {} },
      global: { stubs: { VChart: true } },
    })
    expect(wrapper.find('.slice-select').exists()).toBe(true)
  })
})
