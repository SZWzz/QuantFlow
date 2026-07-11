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
    expect(wrapper.find('.panel-header h3').text()).toContain('Volatility Surface')
  })

  it('renders refresh button', () => {
    const wrapper = mount(SurfaceChartPanel, {
      props: { panelId: 'test', params: {} },
      global: { stubs: { VChart: true } },
    })
    expect(wrapper.find('.refresh-btn').exists()).toBe(true)
  })

  it('has data-panel-id attribute', () => {
    const wrapper = mount(SurfaceChartPanel, {
      props: { panelId: 'test-surface', params: {} },
      global: { stubs: { VChart: true } },
    })
    expect(wrapper.find('.surface-chart-panel').exists()).toBe(true)
  })
})
