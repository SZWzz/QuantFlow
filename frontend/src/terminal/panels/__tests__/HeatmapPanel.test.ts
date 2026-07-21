import { describe, it, expect, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import HeatmapPanel from '../HeatmapPanel.vue'

describe('HeatmapPanel', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('mounts without crashing', () => {
    const wrapper = mount(HeatmapPanel, {
      props: { panelId: 'test-heatmap', params: {} },
    })
    expect(wrapper.exists()).toBe(true)
  })

  it('renders title', () => {
    const wrapper = mount(HeatmapPanel, {
      props: { panelId: 'test-heatmap', params: {} },
    })
    // Title from i18n mock — 'misc.heatmap' -> 'Heatmap'
    expect(wrapper.text()).toContain('Heatmap')
  })

  it('renders market tabs', () => {
    const wrapper = mount(HeatmapPanel, {
      props: { panelId: 'test-heatmap', params: {} },
    })
    const tabs = wrapper.findAll('.panel-tabs .tab')
    expect(tabs.length).toBe(3)
  })
})
