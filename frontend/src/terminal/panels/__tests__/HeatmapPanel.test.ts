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
    expect(wrapper.text()).toContain('Market Heatmap')
  })

  it('renders heatmap cells', async () => {
    const wrapper = mount(HeatmapPanel, {
      props: { panelId: 'test-heatmap', params: {} },
    })
    // Wait for async data load
    await new Promise(r => setTimeout(r, 50))
    const cells = wrapper.findAll('.heatmap-cell')
    expect(cells.length).toBeGreaterThanOrEqual(1)
  })
})
